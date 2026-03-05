# Game Events (`internal/shared/events/`)

게임 내 모든 변화를 나타내는 타입화된 이벤트. Go **interface** 패턴 사용.

---

## Base Event

모든 이벤트의 공통 필드.

```go
// internal/shared/events.go

type BaseEvent struct {
	ID         string          `json:"id"`
	Timestamp  int64           `json:"timestamp"`
	Visibility EventVisibility `json:"visibility"`
}

// EventVisibility — scope 값에 따라 필드 선택적 포함
type EventVisibility struct {
	Scope     string   `json:"scope"`               // "all" | "room" | "player" (PRD Section 6.2)
	RoomID    string   `json:"roomId,omitempty"`    // scope == "room"일 때
	PlayerIDs []string `json:"playerIds,omitempty"` // scope == "player"일 때
}
```

## Event Union

```go
// GameEvent interface — 모든 이벤트가 구현
type GameEvent interface {
	EventType() string
	GetBaseEvent() BaseEvent
}

// 참고: 투표(vote_started/progress/ended)는 GameEvent가 아닌 ServerMessage로 처리.
// 투표는 시스템 레벨 조작이므로 EventBus를 거치지 않고 직접 전송됨.
// → shared/protocol.md 참조
```

## 이벤트 타입별 정의

### GM & 스토리

```go
type NarrationData struct {
	Text string `json:"text"`
	Mood string `json:"mood"` // "tense", "calm", "urgent" 등
}

type NarrationEvent struct {
	BaseEvent
	Type string        `json:"type"` // "narration"
	Data NarrationData `json:"data"`
}

func (e NarrationEvent) EventType() string       { return "narration" }
func (e NarrationEvent) GetBaseEvent() BaseEvent { return e.BaseEvent }

type StoryEventData struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Consequences []string `json:"consequences"`
}

type StoryEventEvent struct {
	BaseEvent
	Type string         `json:"type"` // "story_event"
	Data StoryEventData `json:"data"`
}

func (e StoryEventEvent) EventType() string       { return "story_event" }
func (e StoryEventEvent) GetBaseEvent() BaseEvent { return e.BaseEvent }

type TimeWarningData struct {
	RemainingMinutes int `json:"remainingMinutes"`
}

type TimeWarningEvent struct {
	BaseEvent
	Type string          `json:"type"` // "time_warning"
	Data TimeWarningData `json:"data"`
}

func (e TimeWarningEvent) EventType() string       { return "time_warning" }
func (e TimeWarningEvent) GetBaseEvent() BaseEvent { return e.BaseEvent }
```

### NPC

```go
type NPCDialogueData struct {
	NPCID      string `json:"npcId"`
	NPCName    string `json:"npcName"`
	PlayerID   string `json:"playerId"`   // 대화를 시작한 플레이어 ID (동시 대화 구분용)
	PlayerName string `json:"playerName"` // 대화를 시작한 플레이어 닉네임
	Text       string `json:"text"`
	Emotion    string `json:"emotion"`
	// 참고: PlayerID, PlayerName은 AI 출력(AINPCDialogueEventData)에 포함되지 않는다.
	// 서버(ActionProcessor)가 요청 플레이어 정보를 주입한 후 이벤트를 발행한다.
	// schemas.md AINPCDialogueEventData 참조.
}

type NPCDialogueEvent struct {
	BaseEvent
	Type string          `json:"type"` // "npc_dialogue"
	Data NPCDialogueData `json:"data"`
}

func (e NPCDialogueEvent) EventType() string       { return "npc_dialogue" }
func (e NPCDialogueEvent) GetBaseEvent() BaseEvent { return e.BaseEvent }

type NPCGiveItemData struct {
	NPCID      string `json:"npcId"`
	NPCName    string `json:"npcName"`
	PlayerID   string `json:"playerId"`
	PlayerName string `json:"playerName"`
	Item       Item   `json:"item"`
}

type NPCGiveItemEvent struct {
	BaseEvent
	Type string          `json:"type"` // "npc_give_item"
	Data NPCGiveItemData `json:"data"`
}

func (e NPCGiveItemEvent) EventType() string       { return "npc_give_item" }
func (e NPCGiveItemEvent) GetBaseEvent() BaseEvent { return e.BaseEvent }

type NPCReceiveItemData struct {
	NPCID      string `json:"npcId"`
	NPCName    string `json:"npcName"`
	PlayerID   string `json:"playerId"`
	PlayerName string `json:"playerName"`
	Item       Item   `json:"item"`
}

type NPCReceiveItemEvent struct {
	BaseEvent
	Type string             `json:"type"` // "npc_receive_item"
	Data NPCReceiveItemData `json:"data"`
}

func (e NPCReceiveItemEvent) EventType() string       { return "npc_receive_item" }
func (e NPCReceiveItemEvent) GetBaseEvent() BaseEvent { return e.BaseEvent }

type NPCRevealData struct {
	NPCID      string `json:"npcId"`
	NPCName    string `json:"npcName"`
	Revelation string `json:"revelation"`
	Clue       *Clue  `json:"clue"`
}

type NPCRevealEvent struct {
	BaseEvent
	Type string        `json:"type"` // "npc_reveal"
	Data NPCRevealData `json:"data"`
}

func (e NPCRevealEvent) EventType() string       { return "npc_reveal" }
func (e NPCRevealEvent) GetBaseEvent() BaseEvent { return e.BaseEvent }

type NPCMovedData struct {
	NPCID   string `json:"npcId"`
	NPCName string `json:"npcName"`
	From    string `json:"from"` // 이전 방 이름
	To      string `json:"to"`   // 이동한 방 이름
}

type NPCMovedEvent struct {
	BaseEvent
	Type string       `json:"type"` // "npc_moved" — FR-058 AC3: NPC 이동 시 전체 알림
	Data NPCMovedData `json:"data"`
}

func (e NPCMovedEvent) EventType() string       { return "npc_moved" }
func (e NPCMovedEvent) GetBaseEvent() BaseEvent { return e.BaseEvent }
```

### 플레이어 행동

```go
type ExamineResultData struct {
	PlayerID    string `json:"playerId"`
	PlayerName  string `json:"playerName"`
	Target      string `json:"target"`
	Description string `json:"description"`
	ClueFound   bool   `json:"clueFound"`
}

type ExamineResultEvent struct {
	BaseEvent
	Type string            `json:"type"` // "examine_result"
	Data ExamineResultData `json:"data"`
}

func (e ExamineResultEvent) EventType() string       { return "examine_result" }
func (e ExamineResultEvent) GetBaseEvent() BaseEvent { return e.BaseEvent }

type ActionResultData struct {
	PlayerID        string   `json:"playerId"`
	PlayerName      string   `json:"playerName"`
	Action          string   `json:"action"`
	Result          string   `json:"result"`
	TriggeredEvents []string `json:"triggeredEvents"`
}

type ActionResultEvent struct {
	BaseEvent
	Type string           `json:"type"` // "action_result"
	Data ActionResultData `json:"data"`
}

func (e ActionResultEvent) EventType() string       { return "action_result" }
func (e ActionResultEvent) GetBaseEvent() BaseEvent { return e.BaseEvent }

type ClueFoundData struct {
	PlayerID   string `json:"playerId"`
	PlayerName string `json:"playerName"`
	Clue       Clue   `json:"clue"`
	Location   string `json:"location"`
}

type ClueFoundEvent struct {
	BaseEvent
	Type string        `json:"type"` // "clue_found"
	Data ClueFoundData `json:"data"`
}

func (e ClueFoundEvent) EventType() string       { return "clue_found" }
func (e ClueFoundEvent) GetBaseEvent() BaseEvent { return e.BaseEvent }

type PlayerMoveData struct {
	PlayerID   string `json:"playerId"`
	PlayerName string `json:"playerName"`
	From       string `json:"from"`
	To         string `json:"to"`
}

type PlayerMoveEvent struct {
	BaseEvent
	Type string         `json:"type"` // "player_move"
	Data PlayerMoveData `json:"data"`
}

func (e PlayerMoveEvent) EventType() string       { return "player_move" }
func (e PlayerMoveEvent) GetBaseEvent() BaseEvent { return e.BaseEvent }
```

### 종료

```go
// 종료 알림용. 개인별 엔딩/비밀 공개는 game_ending ServerMessage로 개별 전송.
// → shared/protocol.md 참조
//
// 설계 결정: PRD 6.2절의 game_end.ending은 개인화된 엔딩을 포함하지만,
// 개인별 엔딩/비밀 공개는 game_ending ServerMessage로 개별 전송하는 것이 정보 격리 원칙에 부합.
// 따라서 GameEndEventData에는 공통 결과(commonResult)만 포함하고,
// 개인화된 엔딩은 protocol.md의 game_ending 메시지를 통해 각 플레이어에게 개별 전송.
//
// Note: PRD Section 6.2는 game_end.data에 ending: Ending을 정의하나, 정보 격리를 위해 개인 엔딩은 별도 game_ending ServerMessage로 전송. PRD와의 의도적 차이.
type GameEndEventData struct {
	Reason       string `json:"reason"`
	CommonResult string `json:"commonResult"`
}

type GameEndEvent struct {
	BaseEvent
	Type string           `json:"type"` // "game_end"
	Data GameEndEventData `json:"data"`
}

func (e GameEndEvent) EventType() string       { return "game_end" }
func (e GameEndEvent) GetBaseEvent() BaseEvent { return e.BaseEvent }
```

## 가시성 매트릭스

| 이벤트 타입 | 기본 scope | 비고 |
|-------------|-----------|------|
| `narration` | `all` 또는 `room` | AI가 결정 |
| `story_event` | `all` 또는 `room` | AI가 결정 |
| `npc_dialogue` | `room` | NPC가 있는 방 |
| `npc_give_item` | `room` | NPC가 있는 방 |
| `npc_receive_item` | `room` | NPC가 있는 방 |
| `npc_reveal` | `room` | NPC가 있는 방 |
| `npc_moved` | `all` | FR-058 AC3: 전체 공개 |
| `examine_result` | `room` | 실행한 플레이어의 방 |
| `action_result` | `room` | 실행한 플레이어의 방 |
| `clue_found` | `room` | 발견한 플레이어의 방 |
| `player_move` | `all` | 항상 전체 공개 |
| `game_end` | `all` | 항상 전체 공개 |
| `time_warning` | `all` | 항상 전체 공개 |
