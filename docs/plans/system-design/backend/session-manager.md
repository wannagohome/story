# SessionManager (`internal/server/session/`)

## 책임

게임 세션 생명주기 관리, 로비, 플레이어 입퇴장, 상태 전이.

> 처리하는 ClientMessage: `join`, `rejoin`, `start_game`, `ready`, `cancel_game`
> 게임 진행 중 메시지(chat, move, examine 등)는 ActionProcessor가 처리.

## 의존하는 모듈

NetworkServer, EventBus, GameStateManager, AILayer

> `OnWorldGenerated()` 호출 시 `GameStateManager.InitializeWorld(world, roleAssignments)`를 호출하여 게임 상태를 초기화한 후 briefing 단계로 전이한다.
> `StartGame()`은 `aiLayer.GenerateWorld()`를 호출하여 세계 생성을 시작한다.

## 인터페이스

```go
type SessionManager struct {
    game      *Game
    network   *network.NetworkServer
    eventBus  *eventbus.EventBus
    gameState *game.GameStateManager // OnWorldGenerated() 시 InitializeWorld() 호출에 사용
    aiLayer   *ai.AILayer            // StartGame() 시 GenerateWorld() 호출에 사용
}

func NewSessionManager(
    net *network.NetworkServer,
    bus *eventbus.EventBus,
    gs  *game.GameStateManager,
    ail *ai.AILayer,
) *SessionManager

// ── 세션 생성 ──
func (sm *SessionManager) CreateSession() string  // roomCode 반환

// ── 플레이어 관리 ──
func (sm *SessionManager) AddPlayer(conn *websocket.Conn, nickname string) (*Player, error)
func (sm *SessionManager) RemovePlayer(playerId string)
func (sm *SessionManager) GetPlayers() []*Player

// ── 상태 전이 ──
func (sm *SessionManager) StartGame() error        // lobby → generating. aiLayer.GenerateWorld()를 호출하여 세계 생성 시작
func (sm *SessionManager) OnWorldGenerated()       // generating → briefing
func (sm *SessionManager) OnAllPlayersReady()      // briefing → playing
func (sm *SessionManager) StartEnding()            // playing → ending
func (sm *SessionManager) FinishGame()             // ending → finished

// ── 세션 취소 (FR-082) ──
func (sm *SessionManager) CancelSession() error    // lobby → (종료). 호스트만 가능, 전원에게 알림 후 연결 정리

// 참고 사항:
// - 호스트는 CreateSession() 시 자동으로 첫 번째 플레이어로 등록된다. 최대 인원 확인 시 호스트를 포함한다.
// - P1: rejoin 메시지 처리는 MVP에서는 미구현. 연결 해제 시 비활성 상태로 전환만 수행.
// - P1: FinishGame() 시 GameState + World + EventLog를 ~/.story/sessions/<roomCode>.json으로 직렬화하여 저장 (FR-069)

// ── Graceful Shutdown (FR-083) ──
func (sm *SessionManager) Shutdown(ctx context.Context) error  // 시그널(Ctrl+C) 수신 시 호출

// ── 조회 ──
func (sm *SessionManager) GetGameStatus() GameStatus
func (sm *SessionManager) GetRoomCode() string
func (sm *SessionManager) IsHost(playerId string) bool

// 에러 타입
type JoinError string

const (
    JoinErrorGameStarted       JoinError = "GAME_ALREADY_STARTED"
    JoinErrorRoomFull          JoinError = "ROOM_FULL"
    JoinErrorDuplicateNickname JoinError = "DUPLICATE_NICKNAME"
    JoinErrorInvalidNickname   JoinError = "INVALID_NICKNAME"
    // JoinErrorInvalidNickname: 닉네임 길이가 1~20자 범위를 벗어나거나 제어 문자(control characters)를 포함할 때 반환.
)

func (e JoinError) Error() string { return string(e) }

type StartError string

const (
    StartErrorNotHost          StartError = "NOT_HOST"
    StartErrorNotEnoughPlayers StartError = "NOT_ENOUGH_PLAYERS"
)

func (e StartError) Error() string { return string(e) }
```

## 상태 전이 다이어그램

```
lobby ──(호스트 시작)──► generating ──(세계 생성 완료)──► briefing
                                                           │
                                               (모든 플레이어 ready)
                                                           │
                                                           ▼
finished ◄──(엔딩 완료)── ending ◄──(종료 조건 달성)── playing
```

| 전이 | 트리거 | 검증 |
|------|--------|------|
| lobby → generating | 호스트가 `start_game` 전송 | 최소 2명, 호스트만 가능 |
| generating → briefing | WorldGenerator 완료 | 세계 생성 + 검증 통과 |
| briefing → playing | 모든 플레이어 `ready` 전송 | 전원 ready 확인 |
| playing → ending | EndConditionEngine 트리거 | 종료 조건 충족 |
| ending → finished | 엔딩 서술 완료 | 모든 엔딩 메시지 전송 완료 |
| lobby → (종료) | 호스트가 `cancel_game` 전송 (FR-082) | 호스트만 가능. 모든 참가자에게 알림 후 연결 정리 |

## 룸 코드 생성

```go
// internal/server/session/room_code.go

var words = []string{"WOLF", "MOON", "STAR", "DARK", "FIRE", "IRON", "SILK" /* ~200 단어 */}
// 대문자 영어 4글자.

func generateRoomCode() string {
    word   := words[rand.Intn(len(words))]
    number := rand.Intn(10000)
    return fmt.Sprintf("%s-%04d", word, number)  // 예: WOLF-7423
}
```

200 단어 x 10000 숫자 = 약 200만 조합. 동시 활성 세션이 수천 개가 아닌 한 충분.

## Graceful Shutdown (FR-083)

```
시그널(SIGINT/SIGTERM) 수신
    │
    ▼
Shutdown(ctx)
    │
    ├── 게임 진행 중이면 → 저장 시도 (EventLog 등)
    ├── 모든 플레이어에게 system_message("서버가 종료됩니다") 전송
    ├── 모든 WebSocket 연결 정리 (Close)
    └── NetworkServer.Stop()
```

호스트 프로세스 종료 시: 모든 참가자에게 알림 후 연결 정리.
참가자 클라이언트 종료 시: 해당 플레이어만 퇴장 처리 (기존 RemovePlayer 흐름).

## 입장 시 흐름

```
WebSocket 연결 수립 → OnConnection 핸들러
    │
    ▼
Client → { type: 'join', nickname: 'Alice' }
    │
    ▼
AddPlayer()
    ├── 게임 상태가 'lobby'인지 확인
    ├── 닉네임 유효성 확인: 길이 1~20자, 제어 문자 미포함 (실패 시 JoinErrorInvalidNickname)
    ├── 닉네임 중복 확인
    ├── 최대 인원 확인 (호스트 포함한 인원수 기준)
    ├── Player 객체 생성
    └── 연결 바인딩 (playerId ↔ *websocket.Conn)
    │
    ▼
Server → { type: 'joined', playerId, roomCode }  (본인에게)
Server → { type: 'lobby_update', players }         (전체에게)
```
