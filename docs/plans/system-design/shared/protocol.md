# Client-Server Protocol (`internal/shared/protocol/`)

WebSocket을 통한 클라이언트-서버 간 메시지 프로토콜. 양방향 각각 타입별 Go struct. Host가 WebSocket 서버 역할을 하며, 모든 메시지는 Host를 경유한다.

---

## Client → Server (`ClientMessage`)

```go
// internal/shared/protocol/client.go
//
// 메시지 파싱 패턴: type 필드를 먼저 읽은 뒤 구체 타입으로 역직렬화.
//
//   var raw struct{ Type string `json:"type"` }
//   json.Unmarshal(data, &raw)
//   switch raw.Type { case "join": ... }

// 세션
type JoinMessage struct {
	Type     string `json:"type"`     // "join"
	Nickname string `json:"nickname"`
}

type RejoinMessage struct {
	Type     string `json:"type"`     // "rejoin"
	PlayerID string `json:"playerId"` // 재연결 시
}

type StartGameMessage struct {
	Type         string `json:"type"`                   // "start_game" — 호스트만
	ThemeKeyword string `json:"themeKeyword,omitempty"` // FR-089: 선호 테마/장르 키워드 (선택)
}

type ReadyMessage struct {
	Type  string `json:"type"`  // "ready"
	Phase string `json:"phase"` // "briefing_read" | "game_ready"
	// Phase == "briefing_read": 공개 브리핑을 읽었음을 서버에 알림 (FR-078 AC2)
	// Phase == "game_ready":    개인 브리핑을 읽고 게임 시작 준비가 되었음을 알림 (FR-078 AC4)
}

type CancelGameMessage struct {
	Type string `json:"type"` // "cancel_game" — 호스트만
}

// 채팅
type ChatClientMessage struct {
	Type    string `json:"type"`    // "chat"
	Content string `json:"content"` // 같은 방 채팅
}

type ShoutMessage struct {
	Type    string `json:"type"`    // "shout"
	Content string `json:"content"` // 글로벌 채팅
}

// 행동
type MoveMessage struct {
	Type         string `json:"type"`         // "move"
	TargetRoomID string `json:"targetRoomId"` // 방 이름을 전송. 서버의 MapEngine이 내부 ID로 변환한다.
	// 클라이언트는 방 이름/NPC 이름을 전송하며, 서버의 MapEngine이 내부 ID로 변환한다.
}

type ExamineMessage struct {
	Type   string  `json:"type"`             // "examine"
	Target *string `json:"target,omitempty"`
}

type DoMessage struct {
	Type   string `json:"type"`   // "do"
	Action string `json:"action"`
}

type TalkMessage struct {
	Type    string `json:"type"`    // "talk"
	NPCID   string `json:"npcId"`   // NPC 이름을 전송. 서버의 MapEngine.GetNPCByName()이 내부 ID로 변환한다.
	Message string `json:"message"`
}

type GiveMessage struct {
	Type   string `json:"type"`   // "give"
	NPCID  string `json:"npcId"`
	ItemID string `json:"itemId"`
}

// 투표 / 합의
type VoteMessage struct {
	Type     string `json:"type"`     // "vote"
	TargetID string `json:"targetId"`
}

type SolveMessage struct {
	Type   string `json:"type"`   // "solve" — /solve 해결안 제시 (합의 시스템)
	Answer string `json:"answer"`
}

type ProposeEndMessage struct {
	Type string `json:"type"` // "propose_end" — /end 발의
}

type EndVoteMessage struct {
	Type  string `json:"type"`  // "end_vote"
	Agree bool   `json:"agree"` // /end 투표 응답
}

// 정보 조회
type RequestLookMessage struct {
	Type string `json:"type"` // "request_look" — 현재 방 설명 재표시
	// 서버는 `request_look` 수신 시 현재 방의 RoomView를 포함한 `room_changed` 메시지를 응답으로 전송한다. 이동 없이 현재 방 정보를 재전송하는 형태.
}

type RequestInventoryMessage struct {
	Type string `json:"type"` // "request_inventory"
}

type RequestRoleMessage struct {
	Type string `json:"type"` // "request_role"
}

type RequestMapMessage struct {
	Type string `json:"type"` // "request_map"
}

type RequestWhoMessage struct {
	Type string `json:"type"` // "request_who"
}

type RequestHelpMessage struct {
	Type string `json:"type"` // "request_help"
}

// 피드백
type SubmitFeedbackMessage struct {
	Type            string  `json:"type"`            // "submit_feedback"
	FunRating       int     `json:"funRating"`       // 1~5
	ImmersionRating int     `json:"immersionRating"` // 1~5
	Comment         *string `json:"comment"`         // 선택
}

type SkipFeedbackMessage struct {
	Type string `json:"type"` // "skip_feedback"
}

// ClientMessage는 클라이언트→서버 메시지의 공용 구조체.
// JSON 역직렬화 시 type 필드로 분기하여 해당 필드만 사용한다.
type ClientMessage struct {
	Type         string `json:"type"`
	// join
	Nickname     string `json:"nickname,omitempty"`
	// chat
	Content      string `json:"content,omitempty"`
	Scope        string `json:"scope,omitempty"`      // "room" | "global"
	// move
	TargetRoomID string `json:"targetRoomId,omitempty"` // 실제로는 방 이름, 서버에서 ID로 변환
	// examine
	Target       *string `json:"target,omitempty"`
	// do
	Action       string `json:"action,omitempty"`
	// talk
	NPCID        string `json:"npcId,omitempty"`       // 실제로는 NPC 이름
	Message      string `json:"message,omitempty"`
	// vote
	TargetID     string `json:"targetId,omitempty"`
	// endvote
	Agree        bool   `json:"agree,omitempty"`
	// solve
	Answer       string `json:"answer,omitempty"`
	// give (P1)
	ItemName     string `json:"itemName,omitempty"`
	// ready
	Phase        string `json:"phase,omitempty"` // "briefing_read" | "game_ready"
	// feedback
	FunRating       int     `json:"funRating,omitempty"`
	ImmersionRating int     `json:"immersionRating,omitempty"`
	Comment         *string `json:"comment,omitempty"`
}
```

## Server → Client (`ServerMessage`)

```go
// internal/shared/protocol/server.go

// ── 세션 관리 ──

type JoinedMessage struct {
	Type     string `json:"type"`     // "joined"
	PlayerID string `json:"playerId"`
	RoomCode string `json:"roomCode"`
	IsHost   bool   `json:"isHost"`   // 클라이언트가 호스트 여부를 판단하는 데 사용
}

type LobbyUpdateMessage struct {
	Type       string        `json:"type"`       // "lobby_update"
	Players    []LobbyPlayer `json:"players"`
	MaxPlayers int           `json:"maxPlayers"` // FR-004 AC3: 현재 인원/최대 인원 표시용
}

type GenerationProgressMessage struct {
	Type     string  `json:"type"`     // "generation_progress"
	Step     string  `json:"step"`     // 현재 단계 식별자 (예: "world", "characters", "map")
	Message  string  `json:"message"`  // 사용자에게 표시할 진행 메시지
	Progress float64 `json:"progress"` // 0.0 ~ 1.0
}

type ErrorMessage struct {
	Type    string    `json:"type"`    // "error"
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

type PlayerDisconnectedMessage struct {
	Type     string `json:"type"`     // "player_disconnected" — FR-007 AC1: 다른 플레이어에게 알림
	PlayerID string `json:"playerId"`
	Nickname string `json:"nickname"`
}

type PlayerReconnectedMessage struct {
	Type     string `json:"type"`     // "player_reconnected"
	PlayerID string `json:"playerId"`
	Nickname string `json:"nickname"`
}

// ── 게임 시작 (브리핑) ──

type BriefingPublicMessage struct {
	Type string     `json:"type"` // "briefing_public"
	Info PublicInfo `json:"info"`
}

type BriefingPrivateMessage struct {
	Type           string           `json:"type"`           // "briefing_private"
	Role           PlayerRole       `json:"role"`
	Secrets        []string         `json:"secrets"`
	SemiPublicInfo []SemiPublicInfo `json:"semiPublicInfo"` // FR-051: 조건부 공개 정보 (반공개 정보 중 해당 플레이어 대상분)
}

type GameStartedMessage struct {
	Type        string   `json:"type"`        // "game_started"
	InitialRoom RoomView `json:"initialRoom"` // initialRoom.players에 같은 방 플레이어 목록 포함
}

// ── 게임 진행 ──

type ChatServerMessage struct {
	Type           string  `json:"type"`                     // "chat_message"
	SenderID       string  `json:"senderId"`
	SenderName     string  `json:"senderName"`
	Content        string  `json:"content"`
	Scope          string  `json:"scope"`                    // "room" | "global"
	SenderLocation *string `json:"senderLocation,omitempty"` // scope가 'global'일 때만 포함 (발신자의 현재 방 이름)
	Timestamp      int64   `json:"timestamp"`                // 메시지 전송 시각 (FR-036 AC3)
}

type GameEventMessage struct {
	Type  string          `json:"type"`  // "game_event"
	Event json.RawMessage `json:"event"` // 클라이언트에서 type 필드 기반으로 역직렬화
}

// 클라이언트는 event의 type 필드를 먼저 읽고, schemas.ParseAIGameEvent와 유사한 패턴으로 구체 타입으로 역직렬화한다.

type RoomChangedMessage struct {
	Type string   `json:"type"` // "room_changed" — 이동한 플레이어 본인에게 (room.players에 플레이어 목록 포함)
	Room RoomView `json:"room"`
}

type PlayerJoinedRoomMessage struct {
	Type     string `json:"type"`     // "player_joined_room"
	PlayerID string `json:"playerId"`
	Nickname string `json:"nickname"`
}

type PlayerLeftRoomMessage struct {
	Type        string `json:"type"`        // "player_left_room"
	PlayerID    string `json:"playerId"`
	Nickname    string `json:"nickname"`
	Destination string `json:"destination"`
}

type SystemMessage struct {
	Type    string `json:"type"`    // "system_message"
	Content string `json:"content"`
}

// ── 정보 조회 응답 ──

type InventoryMessage struct {
	Type  string `json:"type"`  // "inventory"
	Items []Item `json:"items"`
	Clues []Clue `json:"clues"`
}

type RoleInfoMessage struct {
	Type string     `json:"type"` // "role_info"
	Role PlayerRole `json:"role"`
}

type MapInfoMessage struct {
	Type string  `json:"type"` // "map_info"
	Map  MapView `json:"map"`
}

type WhoInfoMessage struct {
	Type    string               `json:"type"`    // "who_info"
	Players []PlayerLocationInfo `json:"players"`
}

type HelpInfoMessage struct {
	Type     string        `json:"type"`     // "help_info"
	Commands []CommandInfo `json:"commands"`
}

// ── 투표 ──

type VoteStartedMessage struct {
	Type           string   `json:"type"`           // "vote_started"
	Reason         string   `json:"reason"`
	Candidates     []string `json:"candidates"`
	TimeoutSeconds int      `json:"timeoutSeconds"`
}

type VoteProgressMessage struct {
	Type        string `json:"type"`        // "vote_progress"
	VotedCount  int    `json:"votedCount"`
	TotalVoters int    `json:"totalVoters"`
}

type VoteEndedMessage struct {
	Type    string            `json:"type"`    // "vote_ended"
	Results []VoteResultEntry `json:"results"`
	Outcome string            `json:"outcome"`
}

type EndProposedMessage struct {
	Type           string `json:"type"`           // "end_proposed"
	ProposerID     string `json:"proposerId"`
	ProposerName   string `json:"proposerName"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
}

type EndVoteResultMessage struct {
	Type      string `json:"type"`      // "end_vote_result"
	Agreed    int    `json:"agreed"`
	Disagreed int    `json:"disagreed"`
	Passed    bool   `json:"passed"`
}

// ── 합의 (consensus 시스템) ──

type SolveStartedMessage struct {
	Type           string `json:"type"`           // "solve_started" — 합의 시스템 활성화 시
	Prompt         string `json:"prompt"`          // "해결안을 제시하세요"
	TimeoutSeconds int    `json:"timeoutSeconds"`
}

type SolveProgressMessage struct {
	Type           string `json:"type"`           // "solve_progress"
	SubmittedCount int    `json:"submittedCount"`
	TotalPlayers   int    `json:"totalPlayers"`
}

type SolveResultMessage struct {
	Type    string `json:"type"`    // "solve_result"
	Answers []struct {
		PlayerID   string `json:"playerId"`
		PlayerName string `json:"playerName"`
		Answer     string `json:"answer"`
	} `json:"answers"`
	Outcome string `json:"outcome"` // AI 판정 결과
}

// ── 종료 ──

type GameEndingMessage struct {
	Type           string       `json:"type"`           // "game_ending"
	CommonResult   string       `json:"commonResult"`
	PersonalEnding PlayerEnding `json:"personalEnding"` // FR-067 AC2, FR-092 AC5: 구조화된 개인 엔딩 (Summary, GoalResults, Narrative 포함)
	SecretReveal   SecretReveal `json:"secretReveal"`
}

type FeedbackRequestMessage struct {
	Type string `json:"type"` // "feedback_request" — 엔딩 표시 후 피드백 입력 요청
}

type FeedbackAckMessage struct {
	Type string `json:"type"` // "feedback_ack" — 피드백 수신 확인
}

type GameCancelledMessage struct {
	Type   string `json:"type"`   // "game_cancelled" — FR-082 AC2: 호스트 취소 시 전원 알림
	Reason string `json:"reason"`
}

type GameFinishedMessage struct {
	Type string `json:"type"` // "game_finished"
}

// ServerMessage는 서버→클라이언트 메시지의 공용 구조체.
// Type 필드로 메시지 종류를 식별하고, json.RawMessage로 페이로드를 지연 파싱한다.
type RawServerMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"-"` // Type에 따라 구체 타입으로 역직렬화
}
```

## 보조 타입

```go
// internal/shared/protocol/types.go

type LobbyPlayer struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	IsHost   bool   `json:"isHost"`
}

// GameStateManager에서 정의 (server/game/). 여기서는 프로토콜 참조용.
// Room과 달리 clueIds, npcIds를 포함하지 않음 — 정보 비대칭 보장.
type RoomViewItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RoomViewPlayer struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
}

type RoomViewNPC struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RoomView struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Type        string           `json:"type"` // "public" | "private"
	Items       []RoomViewItem   `json:"items"`   // 이름만 (설명은 /examine으로)
	Players     []RoomViewPlayer `json:"players"`
	NPCs        []RoomViewNPC    `json:"npcs"`
}

type MapViewRoom struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"` // "public" | "private"
	PlayerCount int      `json:"playerCount"`
	PlayerNames []string `json:"playerNames"`
}

type MapView struct {
	Rooms       []MapViewRoom `json:"rooms"`
	Connections []Connection  `json:"connections"`
	MyRoomID    string        `json:"myRoomId"`
}

type PlayerLocationInfo struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	RoomID   string `json:"roomId"`
	RoomName string `json:"roomName"`
	Status   string `json:"status"` // "connected" | "disconnected" | "inactive"
}

type CommandInfo struct {
	Command     string `json:"command"`
	Description string `json:"description"`
	Usage       string `json:"usage"`
}

type VoteResultEntry struct {
	CandidateID   string `json:"candidateId"`
	CandidateName string `json:"candidateName"`
	Votes         int    `json:"votes"`
}

type ErrorCode string

const (
	ErrorCodeInvalidRoomCode    ErrorCode = "INVALID_ROOM_CODE"
	ErrorCodeGameAlreadyStarted ErrorCode = "GAME_ALREADY_STARTED"
	ErrorCodeRoomFull           ErrorCode = "ROOM_FULL"
	ErrorCodeDuplicateNickname  ErrorCode = "DUPLICATE_NICKNAME"
	ErrorCodeNotHost            ErrorCode = "NOT_HOST"
	ErrorCodeNotEnoughPlayers   ErrorCode = "NOT_ENOUGH_PLAYERS"
	ErrorCodeInvalidMove        ErrorCode = "INVALID_MOVE"
	ErrorCodeNPCNotInRoom       ErrorCode = "NPC_NOT_IN_ROOM"
	ErrorCodeItemNotFound       ErrorCode = "ITEM_NOT_FOUND"
	ErrorCodeUnknownCommand     ErrorCode = "UNKNOWN_COMMAND"
	ErrorCodeNotSupported       ErrorCode = "NOT_SUPPORTED"        // 미지원 기능 (예: DM)
	ErrorCodeVoteNotActive      ErrorCode = "VOTE_NOT_ACTIVE"      // 진행 중인 투표 없음
	ErrorCodeVotingDisabled     ErrorCode = "VOTING_DISABLED"      // 이 게임에 투표 시스템 없음
	ErrorCodeEndVoteAlreadyOpen ErrorCode = "END_VOTE_ALREADY_OPEN" // 이미 종료 투표 진행 중
	ErrorCodeEmptyMessage       ErrorCode = "EMPTY_MESSAGE"         // 빈 메시지 전송 시도
	ErrorCodeMessageTooLong     ErrorCode = "MESSAGE_TOO_LONG"     // 메시지 길이 초과
	ErrorCodeInvalidNickname    ErrorCode = "INVALID_NICKNAME"     // 닉네임 유효성 검증 실패 (길이/문자)
	ErrorCodeInvalidAPIKey      ErrorCode = "INVALID_API_KEY"      // API 키 유효성 검증 실패
	ErrorCodeSaveFailed         ErrorCode = "SAVE_FAILED"          // 세션 데이터 저장 실패
	ErrorCodeConnectionLost     ErrorCode = "CONNECTION_LOST"      // WebSocket 연결 끊김
	ErrorCodeWSConnectionFailed    ErrorCode = "WS_CONNECTION_FAILED"    // WebSocket 서버 연결 실패
	ErrorCodeBriefingNotComplete   ErrorCode = "BRIEFING_NOT_COMPLETE"   // 아직 전원이 ready 전송 완료 전 — 역할 조회 등 브리핑 이후 기능 사용 불가
)
```

## 메시지 흐름 다이어그램

```
Client                              Server
  │                                    │
  │──── { type: 'join', ... } ────────>│
  │<─── { type: 'joined', ... } ──────│
  │<─── { type: 'lobby_update' } ─────│
  │                                    │
  │──── { type: 'start_game' } ──────>│  (호스트만)
  │<─── { type: 'generation_progress' }│  (반복)
  │<─── { type: 'briefing_public' } ──│
  │<─── { type: 'briefing_private' } ─│
  │──── { type: 'ready', phase: 'briefing_read' } ─>│  (공개 브리핑 읽음 확인)
  │<─── { type: 'briefing_private' } ────────────-─│
  │──── { type: 'ready', phase: 'game_ready' } ───>│  (게임 시작 준비 완료)
  │<─── { type: 'game_started' } ─────│
  │                                    │
  │──── { type: 'chat', ... } ───────>│
  │<─── { type: 'chat_message' } ─────│  (같은 방만)
  │                                    │
  │──── { type: 'examine', ... } ────>│
  │<─── { type: 'game_event' } ───────│  (examine_result)
  │<─── { type: 'game_event' } ───────│  (clue_found, optional)
  │                                    │
  │<─── { type: 'game_ending' } ──────│
  │<─── { type: 'feedback_request' } ─│
  │──── { type: 'submit_feedback' } ─>│  (또는 'skip_feedback')
  │<─── { type: 'feedback_ack' } ─────│
  │<─── { type: 'game_finished' } ────│
```
