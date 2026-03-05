# NetworkServer (`internal/server/network/`)

## 책임

WebSocket 연결 관리, 메시지 직렬화/역직렬화, 연결-플레이어 매핑.

## 의존하는 모듈

없음

## 의존받는 모듈

SessionManager, MessageRouter, ActionProcessor (간접)

## 네트워크 아키텍처

```
클라이언트-서버: Host가 WebSocket 서버

    Client A ──WebSocket──┐
    Client B ──WebSocket──┤── Host (NetworkServer)
    Client C ──WebSocket──┘

모든 메시지가 Host를 반드시 거침 → 완벽한 정보 필터링 가능
```

## 연결 수립 흐름

```
1. Host가 WebSocket 서버 시작
   http.HandleFunc("/ws", ns.HandleConnection)
   http.ListenAndServe(":3000", nil)

2. 참가자가 WebSocket 연결
   Client: new WebSocket('ws://host:3000/ws')

3. 연결 수립 완료 (즉시)
   Host ←── WebSocket ──→ Client
```

## 인터페이스

```go
import (
    "github.com/gorilla/websocket"
    "net/http"
    "sync"
)

type NetworkServer struct {
    upgrader      websocket.Upgrader
    clients       map[string]*websocket.Conn  // playerId → conn
    mu            sync.RWMutex
    config        NetworkConfig
    onConnHandler func(conn *websocket.Conn)
    onDiscHandler func(playerId string)
    onMsgHandler  func(playerId string, message ClientMessage)
}

func NewNetworkServer(config NetworkConfig) *NetworkServer

type NetworkConfig struct {
    Port int  // WebSocket 서버 포트 (기본값: 3000)
}

// ── 생명주기 ──
func (ns *NetworkServer) Start(roomCode string) error
func (ns *NetworkServer) Stop() error

// ── 연결 이벤트 (SessionManager가 구독) ──
func (ns *NetworkServer) OnConnection(handler func(conn *websocket.Conn))
func (ns *NetworkServer) OnDisconnection(handler func(playerId string))
func (ns *NetworkServer) OnMessage(handler func(playerId string, message ClientMessage))

// ── 메시지 전송 ──
func (ns *NetworkServer) SendTo(playerId string, message ServerMessage)
func (ns *NetworkServer) SendToMany(playerIds []string, message ServerMessage)
func (ns *NetworkServer) SendToAll(message ServerMessage)

// ── 연결 매핑 ──
func (ns *NetworkServer) BindPlayerToSocket(playerId string, conn *websocket.Conn)
func (ns *NetworkServer) UnbindPlayer(playerId string)

// ── HTTP 핸들러 (gorilla/websocket 업그레이드) ──
func (ns *NetworkServer) HandleConnection(w http.ResponseWriter, r *http.Request)
```

## 설계 결정

- **WebSocket 사용 (클라이언트-서버).** 호스트가 서버 역할. 서버 비용 $0.
- **Go WebSocket 구현: `github.com/gorilla/websocket`.** 경량, 아카이브 상태이나 안정적이며 널리 사용됨. 향후 `github.com/coder/websocket` 등 대안 검토 가능. 별도 중개 서버 없이 직접 연결.
- **연결 1개 = 플레이어 1명.** 재연결 시 동일 playerId로 리바인딩.
- **메시지 직렬화는 JSON.** `conn.WriteJSON()` / `conn.ReadJSON()` 사용.
- **포트 설정 가능.** 기본값 3000. `--port` 옵션으로 변경 가능.
- **공유 상태 보호.** `clients` 맵은 `sync.RWMutex`로 보호. 읽기는 `RLock`, 쓰기는 `Lock`.
- **TLS/WSS (NFR-018).** MVP에서는 로컬 네트워크 환경을 기본으로 하되, NFR-018에 따라 WSS 지원을 위한 TLS 설정 옵션을 제공한다. `--tls-cert`, `--tls-key` 플래그로 인증서를 지정하면 WSS로 동작한다. 외부 API 호출(AI provider) 시에는 항상 HTTPS 사용.
- **Graceful Shutdown (FR-083).** `Stop()` 호출 시 모든 연결된 클라이언트에게 종료 알림(`system_message`)을 전송한 뒤 WebSocket 연결을 정리하고 HTTP 서버를 종료. `context.Context` 기반 타임아웃(5초) 적용.
- **메시지 유효성 검사.** 수신 메시지에 대해 다음을 검증: 최대 메시지 크기 64KB (초과 시 연결 종료), 채팅 컨텐츠 최대 500자, 형식이 잘못된 JSON은 즉시 거부 후 에러 응답.

## 내부 흐름

```
HTTP /ws 엔드포인트 → HandleConnection()
    │
    ▼
websocket.Upgrader.Upgrade() → *websocket.Conn
    │
    ▼
onConnHandler 콜백 → SessionManager가 처리
    │
    ▼
conn.ReadJSON() 루프 → ClientMessage 수신 및 타입 검증
    │
    ▼
onMsgHandler 콜백 → 메시지 타입에 따라 라우팅:
    ├── join, rejoin, start_game, ready, cancel_game → SessionManager
    └── 그 외 (chat, move, examine, ...) → ActionProcessor
    │
    ▼
(응답) SendTo / SendToMany / SendToAll → conn.WriteJSON() → WebSocket 전송
```
