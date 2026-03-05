package events

import "github.com/anthropics/story/internal/shared/types"

// Re-export for convenience
type BaseEvent = types.BaseEvent
type EventVisibility = types.EventVisibility

// --- Narration ---

type NarrationData struct {
	Text string `json:"text"`
	Mood string `json:"mood"`
}

type NarrationEvent struct {
	BaseEvent
	Type string        `json:"type"`
	Data NarrationData `json:"data"`
}

func (e NarrationEvent) EventType() string       { return "narration" }
func (e NarrationEvent) GetBaseEvent() BaseEvent { return e.BaseEvent }

// --- StoryEvent ---

type StoryEventData struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Consequences []string `json:"consequences"`
}

type StoryEventEvent struct {
	BaseEvent
	Type string         `json:"type"`
	Data StoryEventData `json:"data"`
}

func (e StoryEventEvent) EventType() string       { return "story_event" }
func (e StoryEventEvent) GetBaseEvent() BaseEvent { return e.BaseEvent }

// --- TimeWarning ---

type TimeWarningData struct {
	RemainingMinutes int `json:"remainingMinutes"`
}

type TimeWarningEvent struct {
	BaseEvent
	Type string          `json:"type"`
	Data TimeWarningData `json:"data"`
}

func (e TimeWarningEvent) EventType() string       { return "time_warning" }
func (e TimeWarningEvent) GetBaseEvent() BaseEvent { return e.BaseEvent }

// --- NPC Events ---

type NPCDialogueData struct {
	NPCID      string `json:"npcId"`
	NPCName    string `json:"npcName"`
	PlayerID   string `json:"playerId"`
	PlayerName string `json:"playerName"`
	Text       string `json:"text"`
	Emotion    string `json:"emotion"`
}

type NPCDialogueEvent struct {
	BaseEvent
	Type string           `json:"type"`
	Data NPCDialogueData `json:"data"`
}

func (e NPCDialogueEvent) EventType() string       { return "npc_dialogue" }
func (e NPCDialogueEvent) GetBaseEvent() BaseEvent { return e.BaseEvent }

type NPCGiveItemData struct {
	NPCID      string     `json:"npcId"`
	NPCName    string     `json:"npcName"`
	PlayerID   string     `json:"playerId"`
	PlayerName string     `json:"playerName"`
	Item       types.Item `json:"item"`
}

type NPCGiveItemEvent struct {
	BaseEvent
	Type string           `json:"type"`
	Data NPCGiveItemData `json:"data"`
}

func (e NPCGiveItemEvent) EventType() string       { return "npc_give_item" }
func (e NPCGiveItemEvent) GetBaseEvent() BaseEvent { return e.BaseEvent }

type NPCReceiveItemData struct {
	NPCID      string     `json:"npcId"`
	NPCName    string     `json:"npcName"`
	PlayerID   string     `json:"playerId"`
	PlayerName string     `json:"playerName"`
	Item       types.Item `json:"item"`
}

type NPCReceiveItemEvent struct {
	BaseEvent
	Type string              `json:"type"`
	Data NPCReceiveItemData `json:"data"`
}

func (e NPCReceiveItemEvent) EventType() string       { return "npc_receive_item" }
func (e NPCReceiveItemEvent) GetBaseEvent() BaseEvent { return e.BaseEvent }

type NPCRevealData struct {
	NPCID      string      `json:"npcId"`
	NPCName    string      `json:"npcName"`
	Revelation string      `json:"revelation"`
	Clue       *types.Clue `json:"clue"`
}

type NPCRevealEvent struct {
	BaseEvent
	Type string         `json:"type"`
	Data NPCRevealData `json:"data"`
}

func (e NPCRevealEvent) EventType() string       { return "npc_reveal" }
func (e NPCRevealEvent) GetBaseEvent() BaseEvent { return e.BaseEvent }

type NPCMovedData struct {
	NPCID   string `json:"npcId"`
	NPCName string `json:"npcName"`
	From    string `json:"from"`
	To      string `json:"to"`
}

type NPCMovedEvent struct {
	BaseEvent
	Type string       `json:"type"`
	Data NPCMovedData `json:"data"`
}

func (e NPCMovedEvent) EventType() string       { return "npc_moved" }
func (e NPCMovedEvent) GetBaseEvent() BaseEvent { return e.BaseEvent }

// --- Player Action Events ---

type ExamineResultData struct {
	PlayerID    string `json:"playerId"`
	PlayerName  string `json:"playerName"`
	Target      string `json:"target"`
	Description string `json:"description"`
	ClueFound   bool   `json:"clueFound"`
}

type ExamineResultEvent struct {
	BaseEvent
	Type string            `json:"type"`
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
	Type string           `json:"type"`
	Data ActionResultData `json:"data"`
}

func (e ActionResultEvent) EventType() string       { return "action_result" }
func (e ActionResultEvent) GetBaseEvent() BaseEvent { return e.BaseEvent }

type ClueFoundData struct {
	PlayerID   string     `json:"playerId"`
	PlayerName string     `json:"playerName"`
	Clue       types.Clue `json:"clue"`
	Location   string     `json:"location"`
}

type ClueFoundEvent struct {
	BaseEvent
	Type string        `json:"type"`
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
	Type string         `json:"type"`
	Data PlayerMoveData `json:"data"`
}

func (e PlayerMoveEvent) EventType() string       { return "player_move" }
func (e PlayerMoveEvent) GetBaseEvent() BaseEvent { return e.BaseEvent }

// --- Game End ---

type GameEndEventData struct {
	Reason       string `json:"reason"`
	CommonResult string `json:"commonResult"`
}

type GameEndEvent struct {
	BaseEvent
	Type string           `json:"type"`
	Data GameEndEventData `json:"data"`
}

func (e GameEndEvent) EventType() string       { return "game_end" }
func (e GameEndEvent) GetBaseEvent() BaseEvent { return e.BaseEvent }
