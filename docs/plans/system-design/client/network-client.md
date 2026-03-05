# NetworkClient (`internal/client/network/`)

> **Import 경로:** `charm.land/bubbletea/v2`, `github.com/gorilla/websocket` v1.5.3

## 책임

WebSocket 연결 관리, 메시지 전송/수신. 수신한 서버 메시지를 `tea.Msg`로 Bubble Tea v2 런타임에 전달.

## 의존하는 모듈

없음 (AppModel이 NetworkClient를 참조)

## 라이브러리

`github.com/gorilla/websocket`

## 인터페이스

```go
// internal/client/network/client.go

type NetworkClient struct {
    conn      *websocket.Conn
    serverURL string
    playerID  string
    send      chan []byte  // 전송 큐
}

// ConnectCmd는 WebSocket 연결을 시작하는 tea.Cmd를 반환한다.
func (nc *NetworkClient) ConnectCmd() tea.Cmd

// SendCmd는 ClientMessage를 전송하는 tea.Cmd를 반환한다.
func (nc *NetworkClient) SendCmd(msg ClientMessage) tea.Cmd

// ListenCmd는 서버 메시지를 기다리는 long-running tea.Cmd를 반환한다.
func (nc *NetworkClient) ListenCmd() tea.Cmd

// Disconnect는 연결을 닫는다.
func (nc *NetworkClient) Disconnect()
```

## Bubble Tea 통합

Bubble Tea에서 비동기 I/O는 `tea.Cmd`로 처리한다. WebSocket 수신은 goroutine에서 실행되며, 메시지가 오면 `tea.Msg`로 Bubble Tea 런타임에 전달된다.

```go
// internal/client/network/client.go

// ServerMsgReceived는 서버에서 메시지를 수신했을 때 전달되는 tea.Msg다.
type ServerMsgReceived struct {
    Msg ServerMessage
}

// ConnectError는 연결 실패 시 전달되는 tea.Msg다.
type ConnectError struct {
    Err error
}

// ConnectSuccess는 연결 성공 시 전달되는 tea.Msg다.
type ConnectSuccess struct{}

// Disconnected는 연결이 끊어졌을 때 전달되는 tea.Msg다.
type Disconnected struct{}

// ParseError는 서버 메시지 파싱 실패 시 전달되는 tea.Msg다.
type ParseError struct {
    Err error
}

// SendError는 메시지 직렬화 실패 시 전달되는 tea.Msg다.
type SendError struct {
    Err error
}
```

## 내부 동작

### 연결

```go
// ConnectCmd는 gorilla/websocket으로 연결을 수립하고 결과를 tea.Msg로 반환한다.
func (nc *NetworkClient) ConnectCmd() tea.Cmd {
    return func() tea.Msg {
        conn, _, err := websocket.DefaultDialer.Dial(nc.serverURL, nil)
        if err != nil {
            return ConnectError{Err: err}
        }
        nc.conn = conn
        // 전송 goroutine 시작
        go nc.writeLoop()
        return ConnectSuccess{}
    }
}
```

### 메시지 수신 → tea.Msg

```go
// ListenCmd는 WebSocket에서 다음 메시지를 읽는 tea.Cmd다.
// 메시지 수신 후 즉시 반환하므로, AppModel.Update()에서 다시 ListenCmd()를 반환해야 한다.
//
// 수신 JSON을 먼저 RawServerMessage로 파싱하여 Type 필드를 확인한 후,
// Type에 따라 구체 ServerMessage 타입으로 역직렬화한다.
// RawServerMessage 정의는 shared/protocol.md 참조.
func (nc *NetworkClient) ListenCmd() tea.Cmd {
    return func() tea.Msg {
        _, data, err := nc.conn.ReadMessage()
        if err != nil {
            return Disconnected{}
        }
        var msg ServerMessage
        if err := json.Unmarshal(data, &msg); err != nil {
            // 파싱 실패: 재귀 호출 대신 ParseError 반환하여 스택 오버플로 방지.
            // AppModel.Update()에서 ParseError를 받으면 무시하고 ListenCmd()를 재발행.
            return ParseError{Err: err}
        }
        return ServerMsgReceived{Msg: msg}
    }
}
```

AppModel은 `ServerMsgReceived`를 처리한 후 다시 `ListenCmd()`를 반환하여 연속 수신한다:

```go
// internal/client/screens/app.go

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case network.ConnectSuccess:
        if m.state.PlayerID != "" {
            // 재연결: rejoin 전송
            return m, tea.Batch(m.network.ListenCmd(), m.network.SendCmd(ClientMessage{Type: "rejoin", PlayerID: m.state.PlayerID}))
        }
        // 닉네임이 CLI 인자로 제공된 경우 즉시 join, 아닌 경우 NicknameScreen으로 전환.
        if m.state.Nickname == "" {
            m.state.GamePhase = PhaseNickname
            return m, m.network.ListenCmd()
        }
        // 초기 접속: join 전송 후 서버 응답(joined) 대기. ScreenLobby 전환은 joined 수신 시 syncScreenFromPhase()에서 처리.
        return m, tea.Batch(m.network.ListenCmd(), m.network.SendCmd(ClientMessage{Type: "join", Nickname: m.state.Nickname}))

    case network.ServerMsgReceived:
        m.state = applyServerMessage(m.state, msg.Msg)
        m = m.syncScreenFromPhase()
        // 처리 후 즉시 다음 메시지 대기
        return m, m.network.ListenCmd()

    case network.Disconnected:
        m.state.ConnectionStatus = StatusReconnecting
        return m, m.reconnectCmd(1)
    // ...
    }
}
```

### 메시지 전송

```go
// SendCmd는 ClientMessage를 직렬화하여 WebSocket으로 전송하는 tea.Cmd다.
func (nc *NetworkClient) SendCmd(msg ClientMessage) tea.Cmd {
    return func() tea.Msg {
        data, err := json.Marshal(msg)
        if err != nil {
            return SendError{Err: err}
        }
        nc.send <- data
        return nil
    }
}

// writeLoop는 send 채널의 메시지를 WebSocket으로 실제 전송한다.
func (nc *NetworkClient) writeLoop() {
    for data := range nc.send {
        if err := nc.conn.WriteMessage(websocket.TextMessage, data); err != nil {
            return
        }
    }
}
```

### 연결 해제

```go
func (nc *NetworkClient) Disconnect() {
    close(nc.send)
    if nc.conn != nil {
        nc.conn.Close()
    }
}
```

## 재연결 전략

> 참고: 서버 측 상태 복원(FR-007)은 P1. MVP에서는 rejoin 시 단순 재연결만 지원하며, 전체 상태 복원은 포함되지 않을 수 있음.

WebSocket 연결이 끊어진 경우 (cross-cutting.md 참조):
1. 최대 3회 재연결 시도, 지수 백오프 (1s→2s→4s, 총 ~7초)
2. `serverURL`로 직접 재연결
3. 3회 실패 시 사용자에게 에러 표시

```go
// internal/client/network/reconnect.go

// 지수 백오프 딜레이 계산
func backoffDelay(attempt int) time.Duration {
    return time.Duration(1<<(attempt-1)) * time.Second // 1s, 2s, 4s
}

func (m AppModel) reconnectCmd(attempt int) tea.Cmd {
    if attempt > 3 {
        return func() tea.Msg {
            return ReconnectFailed{}
        }
    }
    delay := backoffDelay(attempt)
    return tea.Sequence(
        tea.Tick(delay, func(t time.Time) tea.Msg {
            return reconnectTick{attempt: attempt}
        }),
    )
}

// AppModel.Update()에서 처리
case reconnectTick:
    return m, tea.Batch(
        m.network.ConnectCmd(),
        m.retryOnFailCmd(msg.attempt),
    )

case ReconnectFailed:
    m.state.LastError = &ClientError{
        Code:    "CONNECTION_LOST",
        Message: "서버와의 연결이 끊어졌습니다. 게임에 다시 참가해주세요.",
    }
    return m, nil
```

재연결 성공 시 `rejoin` 메시지 전송은 위의 `ConnectSuccess` 핸들러에서 `PlayerID` 존재 여부로 분기하여 처리한다.
