# ClientState (`internal/client/state/`)

## 책임

클라이언트 측 상태 관리. 서버에서 받은 정보만 저장.
Bubble Tea에서 상태는 `AppModel` struct의 필드로 직접 관리한다. React의 `useReducer` 대신 `applyServerMessage()` 순수 함수가 메시지를 받아 새로운 `ClientState`를 반환한다.

## 의존하는 모듈

없음 (상태 컨테이너)

## 상태 구조

```go
// internal/client/state/state.go

type ConnectionStatus string

const (
    StatusConnecting    ConnectionStatus = "connecting"
    StatusConnected     ConnectionStatus = "connected"
    StatusReconnecting  ConnectionStatus = "reconnecting"
    StatusDisconnected  ConnectionStatus = "disconnected"
)

type GamePhase string

const (
    PhaseConnecting GamePhase = "connecting"
    PhaseNickname   GamePhase = "nickname"  // 연결 후 닉네임 입력 대기 (클라이언트 전용)
    PhaseLobby      GamePhase = "lobby"
    PhaseGenerating GamePhase = "generating"
    PhaseBriefing   GamePhase = "briefing"
    PhasePlaying    GamePhase = "playing"
    PhaseEnding     GamePhase = "ending"
    PhaseFinished   GamePhase = "finished"
)

type ClientState struct {
    // ── 연결 ──
    ConnectionStatus ConnectionStatus
    PlayerID         string
    Nickname         string  // CLI 인자 또는 닉네임 입력 화면에서 설정
    RoomCode         string
    GamePhase        GamePhase
    IsHost           bool

    // ── 로비 ──
    LobbyPlayers []LobbyPlayer
    MaxPlayers   int  // lobby_update에서 수신

    // ── 브리핑 ──
    BriefingPublic     *PublicInfo
    BriefingSecrets    []string
    BriefingSemiPublic []SemiPublicInfo  // briefing_private에서 수신 (FR-051)

    // ── 게임 ──
    WorldTitle      string  // briefing_public.info.title 수신 시 설정
    MyRole          *PlayerRole
    CurrentRoom     *RoomView
    MapOverview     *MapView
    Inventory       []Item
    DiscoveredClues []Clue

    // ── 채팅/이벤트 ──
    Messages []DisplayMessage

    // ── 투표 ──
    ActiveVote       *ActiveVoteState
    LastVoteResult   *VoteResult
    ActiveEndProposal *EndProposalState

    // ── 합의 (consensus) ──
    ActiveSolve *ActiveSolveState

    // ── 세계 생성 진행 ──
    GenerationMessage string
    GenerationProgress float64  // 0.0 ~ 1.0

    // ── 엔딩 ──
    EndingData        *EndingData
    FeedbackRequested bool  // feedback_request 수신 시 true → 피드백 입력 UI 표시

    // ── 에러 ──
    LastError *ClientError
}
```

## 클라이언트 전용 타입

Store에서 사용하는 클라이언트 전용 타입. 서버의 `ActiveVote`/`EndProposal`은 timer, votes 등 서버 런타임 필드를 포함하므로, 클라이언트는 표시에 필요한 필드만 가진 별도 타입을 사용한다.

```go
// internal/client/state/types.go

type ActiveVoteState struct {
    Reason         string
    Candidates     []string
    TimeoutSeconds int
    VotedCount     int
    TotalVoters    int
}

type EndProposalState struct {
    ProposerID     string
    ProposerName   string
    TimeoutSeconds int
}

type VoteResult struct {
    Results []VoteResultEntry
    Outcome string
}

type ActiveSolveState struct {
    Prompt         string
    TimeoutSeconds int
    SubmittedCount int
    TotalPlayers   int
}

type EndingData struct {
    CommonResult   string
    PersonalEnding PlayerEnding  // protocol.md와 일치: 구조화된 PlayerEnding
    SecretReveal   SecretReveal
}

type ClientError struct {
    Code    string
    Message string
}

// GameEvent는 클라이언트 측 게임 이벤트 표현.
// 서버의 GameEvent 인터페이스와 달리, JSON 역직렬화 시 flat struct로 처리한다.
// event-renderers.md 참조: event.Data["fieldName"].(type) 방식으로 접근.
type GameEvent struct {
    ID        string         `json:"id"`
    Type      string         `json:"type"`
    Timestamp int64          `json:"timestamp"`
    Data      map[string]any `json:"data"`
}

// DisplayMessage는 ChatLog에 표시되는 모든 메시지 타입을 통합한다.
type DisplayMessage struct {
    ID        string
    Kind      string  // "chat" | "event" | "system"
    // Kind == "chat"
    SenderID       string
    SenderName     string
    Content        string
    Scope          string  // "room" | "global"
    SenderLocation *string  // scope가 'global'일 때만 포함 (protocol.md: *string)
    // Kind == "event"
    Event GameEvent  // 클라이언트 전용 concrete struct (위 정의 참조)
    // Kind == "system"
    // Content 필드 재사용
    Timestamp int64
}
```

## applyServerMessage

React reducer를 대체하는 순수 함수. `ServerMessage`를 받아 새로운 `ClientState`를 반환한다.
`AppModel.Update()`에서 `ServerMsgReceived` 처리 시 호출된다.

```go
// internal/client/state/apply.go

// applyServerMessage는 ServerMessage를 ClientState에 적용하여 새 상태를 반환한다.
// ServerMessage는 RawServerMessage.Type으로 분기 후 각 타입별 구조체로 역직렬화한다.
// 아래 의사코드는 Type별 필드 접근을 간결하게 표현한 것이며,
// 실제 구현에서는 타입 단언(type assertion) 또는 switch 분기를 사용한다.
//
// GameEvent 역직렬화: 서버에서 수신한 GameEvent JSON의 `data` 하위 객체를
// 최상위 Data map으로 승격하여 저장한다.
// 즉, {"type":"narration","data":{"text":"..."}} 수신 시 event.Data["text"]로 접근 가능하다.
func applyServerMessage(state ClientState, msg ServerMessage) ClientState {
    switch msg.Type {
    // ── 세션 ──
    case "joined":
        state.ConnectionStatus = StatusConnected
        state.PlayerID = msg.PlayerID
        state.RoomCode = msg.RoomCode
        state.IsHost   = msg.IsHost
        state.GamePhase = PhaseLobby

    case "lobby_update":
        state.LobbyPlayers = msg.Players
        state.MaxPlayers   = msg.MaxPlayers

    case "error":
        state.LastError = &ClientError{Code: msg.Code, Message: msg.Message}

    case "player_disconnected":
        state = addSystemMessage(state, msg.Nickname+"이(가) 접속이 끊어졌습니다")

    case "player_reconnected":
        state = addSystemMessage(state, msg.Nickname+"이(가) 재접속했습니다")

    case "generation_progress":
        state.GamePhase            = PhaseGenerating
        state.GenerationMessage    = msg.Message
        state.GenerationProgress   = msg.Progress

    // ── 브리핑 ──
    case "briefing_public":
        state.GamePhase      = PhaseBriefing
        state.BriefingPublic = &msg.Info
        state.WorldTitle     = msg.Info.Title

    case "briefing_private":
        state.MyRole           = &msg.Role
        state.BriefingSecrets  = msg.Secrets
        state.BriefingSemiPublic = msg.SemiPublicInfo

    case "game_started":
        state.GamePhase    = PhasePlaying
        state.CurrentRoom  = &msg.InitialRoom
        // MapOverview는 game_started 직후 서버가 브로드캐스트하는 map_info 메시지로 채워진다.
        // (data-flow.md Phase 4 참조: 서버는 game_started 후 초기 map_info를 전송한다.)

    case "room_changed":
        state.CurrentRoom = &msg.Room

    // ── 게임 진행 ──
    case "chat_message":
        state = addMessage(state, DisplayMessage{
            ID:             generateID(),
            Kind:           "chat",
            SenderID:       msg.SenderID,
            SenderName:     msg.SenderName,
            Content:        msg.Content,
            Scope:          msg.Scope,
            SenderLocation: msg.SenderLocation,
            Timestamp:      msg.Timestamp,
        })

    case "game_event":
        state = addMessage(state, DisplayMessage{
            ID:        msg.Event.ID,
            Kind:      "event",
            Event:     msg.Event,
            Timestamp: msg.Event.Timestamp,
        })

    case "system_message":
        state = addSystemMessage(state, msg.Content)

    case "player_joined_room":
        state = addSystemMessage(state, msg.Nickname+"이(가) 들어왔습니다")

    case "player_left_room":
        state = addSystemMessage(state, msg.Nickname+"이(가) "+msg.Destination+"(으)로 나갔습니다")

    // ── 정보 조회 응답 ──
    case "inventory":
        state.Inventory       = msg.Items
        state.DiscoveredClues = msg.Clues

    case "role_info":
        state.MyRole = &msg.Role  // RoleInfoMessage.Role는 PlayerRole (비포인터), MyRole는 *PlayerRole

    case "map_info":
        state.MapOverview = &msg.Map  // MapInfoMessage.Map는 MapView (비포인터), MapOverview는 *MapView

    case "who_info":
        lines := make([]string, 0, len(msg.Players))
        for _, p := range msg.Players {
            line := p.Nickname + " -> " + p.RoomName
            if p.Status == "disconnected" {
                line += " (접속끊김)"
            }
            lines = append(lines, line)
        }
        state = addSystemMessage(state, strings.Join(lines, "\n"))

    case "help_info":
        lines := make([]string, 0, len(msg.Commands))
        for _, c := range msg.Commands {
            lines = append(lines, c.Command+" - "+c.Description)
        }
        state = addSystemMessage(state, strings.Join(lines, "\n"))

    // ── 투표 ──
    case "vote_started":
        state.ActiveVote = &ActiveVoteState{
            Reason:         msg.Reason,
            Candidates:     msg.Candidates,
            TimeoutSeconds: msg.TimeoutSeconds,
        }

    case "vote_progress":
        if state.ActiveVote != nil {
            state.ActiveVote.VotedCount  = msg.VotedCount
            state.ActiveVote.TotalVoters = msg.TotalVoters
        }

    case "vote_ended":
        state.ActiveVote      = nil
        state.LastVoteResult  = &VoteResult{Results: msg.Results, Outcome: msg.Outcome}

    // ── 합의 (consensus) ──
    case "solve_started":
        state.ActiveSolve = &ActiveSolveState{
            Prompt:         msg.Prompt,
            TimeoutSeconds: msg.TimeoutSeconds,
        }

    case "solve_progress":
        if state.ActiveSolve != nil {
            state.ActiveSolve.SubmittedCount = msg.SubmittedCount
            state.ActiveSolve.TotalPlayers   = msg.TotalPlayers
        }

    case "solve_result":
        state.ActiveSolve = nil
        state = addSystemMessage(state, "합의 결과: "+msg.Outcome)

    case "end_proposed":
        state.ActiveEndProposal = &EndProposalState{
            ProposerID:     msg.ProposerID,
            ProposerName:   msg.ProposerName,
            TimeoutSeconds: msg.TimeoutSeconds,
        }

    case "end_vote_result":
        state.ActiveEndProposal = nil
        outcome := "게임 종료 제안이 부결되었습니다."
        if msg.Passed {
            outcome = "게임 종료가 결정되었습니다."
        }
        state = addSystemMessage(state, outcome)

    case "feedback_request":
        // 서버가 피드백 입력을 요청 (엔딩 표시 후)
        // GamePhase는 이미 PhaseEnding (game_ending에서 설정됨)
        // FeedbackRequested 플래그로 엔딩 화면 → 피드백 입력 UI 전환
        state.FeedbackRequested = true

    case "feedback_ack":
        state = addSystemMessage(state, "피드백이 전송되었습니다. 감사합니다!")

    // ── 종료 ──
    case "game_ending":
        state.GamePhase = PhaseEnding
        state.EndingData = &EndingData{
            CommonResult:   msg.CommonResult,
            PersonalEnding: msg.PersonalEnding,
            SecretReveal:   msg.SecretReveal,
        }

    case "game_cancelled":
        state.GamePhase = PhaseFinished
        state.LastError = &ClientError{Code: "GAME_CANCELLED", Message: msg.Reason}
        state = addSystemMessage(state, "게임이 취소되었습니다: "+msg.Reason)

    case "game_finished":
        state.GamePhase = PhaseFinished
    }

    return state
}
```

## 메시지 관리

```go
// internal/client/state/messages.go

const maxMessages = 200

func addMessage(state ClientState, msg DisplayMessage) ClientState {
    state.Messages = append(state.Messages, msg)
    if len(state.Messages) > maxMessages {
        state.Messages = state.Messages[len(state.Messages)-maxMessages:]
    }
    return state
}

func addSystemMessage(state ClientState, content string) ClientState {
    return addMessage(state, DisplayMessage{
        ID:        generateID(),
        Kind:      "system",
        Content:   content,
        Timestamp: now(),
    })
}
```

## AppModel에서의 접근

Bubble Tea에서 상태는 `AppModel`의 필드로 존재한다. React의 `useStore()` 훅 없이 `View()` 함수가 `m.state`를 직접 참조한다.

```go
// internal/client/screens/app.go

type AppModel struct {
    screen  Screen
    state   ClientState  // 모든 상태가 여기에
    network *NetworkClient
    // 화면별 UI 서브 모델 — screens.md 참조
    connecting ConnectingModel  // spinner 포함
    nickname   NicknameModel   // 닉네임 입력
    lobby      LobbyModel
    generating GeneratingModel  // progress bar 포함
    briefing   BriefingModel
    game       GameModel
    ending     EndingModel
}

// View()는 m.state를 직접 읽는다. 별도 Context나 훅 불필요.
func (m AppModel) View() string {
    switch m.screen {
    case ScreenGame:
        return m.game.View(m.state)  // state 전달
    // ...
    }
}
```

상태 변경은 반드시 `Update()` 내부에서만 발생하며, 항상 새로운 `ClientState` 값을 반환한다 (불변성 유지).
