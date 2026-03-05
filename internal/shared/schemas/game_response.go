package schemas

import (
	"encoding/json"
	"fmt"
)

// AIGameEvent is the common interface for AI-generated game events.
// The Type field discriminates between event kinds.
// Note: BaseEvent fields (id, timestamp, visibility) are assigned by the server,
// not included in AI responses.
type AIGameEvent interface {
	AIEventType() string
}

// --- AI Event Types ---

type AINarrationEventData struct {
	Text string `json:"text"`
	Mood string `json:"mood"`
}

type AINarrationEvent struct {
	Type string               `json:"type"`
	Data AINarrationEventData `json:"data"`
}

func (e AINarrationEvent) AIEventType() string { return "narration" }

type AINPCDialogueEventData struct {
	NPCID   string `json:"npcId"`
	NPCName string `json:"npcName"`
	Text    string `json:"text"`
	Emotion string `json:"emotion"`
}

type AINPCDialogueEvent struct {
	Type string                 `json:"type"`
	Data AINPCDialogueEventData `json:"data"`
}

func (e AINPCDialogueEvent) AIEventType() string { return "npc_dialogue" }

type AINPCGiveItemEventData struct {
	NPCID      string     `json:"npcId"`
	NPCName    string     `json:"npcName"`
	PlayerID   string     `json:"playerId"`
	PlayerName string     `json:"playerName"`
	Item       ItemSchema `json:"item"`
}

type AINPCGiveItemEvent struct {
	Type string                 `json:"type"`
	Data AINPCGiveItemEventData `json:"data"`
}

func (e AINPCGiveItemEvent) AIEventType() string { return "npc_give_item" }

type AINPCReceiveItemEventData struct {
	NPCID      string     `json:"npcId"`
	NPCName    string     `json:"npcName"`
	PlayerID   string     `json:"playerId"`
	PlayerName string     `json:"playerName"`
	Item       ItemSchema `json:"item"`
}

type AINPCReceiveItemEvent struct {
	Type string                    `json:"type"`
	Data AINPCReceiveItemEventData `json:"data"`
}

func (e AINPCReceiveItemEvent) AIEventType() string { return "npc_receive_item" }

type AINPCRevealClue struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type AINPCRevealEventData struct {
	NPCID      string           `json:"npcId"`
	NPCName    string           `json:"npcName"`
	Revelation string           `json:"revelation"`
	Clue       *AINPCRevealClue `json:"clue"`
}

type AINPCRevealEvent struct {
	Type string               `json:"type"`
	Data AINPCRevealEventData `json:"data"`
}

func (e AINPCRevealEvent) AIEventType() string { return "npc_reveal" }

type AIClueFoundEventData struct {
	PlayerID   string          `json:"playerId"`
	PlayerName string          `json:"playerName"`
	Clue       AINPCRevealClue `json:"clue"`
	Location   string          `json:"location"`
}

type AIClueFoundEvent struct {
	Type string               `json:"type"`
	Data AIClueFoundEventData `json:"data"`
}

func (e AIClueFoundEvent) AIEventType() string { return "clue_found" }

type AIStoryEventData struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Consequences []string `json:"consequences"`
}

type AIStoryEvent struct {
	Type string           `json:"type"`
	Data AIStoryEventData `json:"data"`
}

func (e AIStoryEvent) AIEventType() string { return "story_event" }

type AIExamineResultEventData struct {
	PlayerID    string `json:"playerId"`
	PlayerName  string `json:"playerName"`
	Target      string `json:"target"`
	Description string `json:"description"`
	ClueFound   bool   `json:"clueFound"`
}

type AIExamineResultEvent struct {
	Type string                   `json:"type"`
	Data AIExamineResultEventData `json:"data"`
}

func (e AIExamineResultEvent) AIEventType() string { return "examine_result" }

type AIActionResultEventData struct {
	PlayerID        string   `json:"playerId"`
	PlayerName      string   `json:"playerName"`
	Action          string   `json:"action"`
	Result          string   `json:"result"`
	TriggeredEvents []string `json:"triggeredEvents"`
}

type AIActionResultEvent struct {
	Type string                  `json:"type"`
	Data AIActionResultEventData `json:"data"`
}

func (e AIActionResultEvent) AIEventType() string { return "action_result" }

type AIPlayerMoveEventData struct {
	PlayerID   string `json:"playerId"`
	PlayerName string `json:"playerName"`
	From       string `json:"from"`
	To         string `json:"to"`
}

type AIPlayerMoveEvent struct {
	Type string                `json:"type"`
	Data AIPlayerMoveEventData `json:"data"`
}

func (e AIPlayerMoveEvent) AIEventType() string { return "player_move" }

type AIGameEndEventData struct {
	Reason       string `json:"reason"`
	CommonResult string `json:"commonResult"`
}

type AIGameEndEvent struct {
	Type string             `json:"type"`
	Data AIGameEndEventData `json:"data"`
}

func (e AIGameEndEvent) AIEventType() string { return "game_end" }

type AITimeWarningEventData struct {
	RemainingMinutes int `json:"remainingMinutes"`
}

type AITimeWarningEvent struct {
	Type string                 `json:"type"`
	Data AITimeWarningEventData `json:"data"`
}

func (e AITimeWarningEvent) AIEventType() string { return "time_warning" }

// ParseAIGameEvent deserializes a JSON event by its type field.
func ParseAIGameEvent(data json.RawMessage) (AIGameEvent, error) {
	var raw struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	switch raw.Type {
	case "narration":
		var e AINarrationEvent
		return e, json.Unmarshal(data, &e)
	case "npc_dialogue":
		var e AINPCDialogueEvent
		return e, json.Unmarshal(data, &e)
	case "npc_give_item":
		var e AINPCGiveItemEvent
		return e, json.Unmarshal(data, &e)
	case "npc_receive_item":
		var e AINPCReceiveItemEvent
		return e, json.Unmarshal(data, &e)
	case "npc_reveal":
		var e AINPCRevealEvent
		return e, json.Unmarshal(data, &e)
	case "clue_found":
		var e AIClueFoundEvent
		return e, json.Unmarshal(data, &e)
	case "story_event":
		var e AIStoryEvent
		return e, json.Unmarshal(data, &e)
	case "examine_result", "examination_result", "room_description":
		var e AIExamineResultEvent
		return e, json.Unmarshal(data, &e)
	case "action_result":
		var e AIActionResultEvent
		return e, json.Unmarshal(data, &e)
	case "player_move":
		var e AIPlayerMoveEvent
		return e, json.Unmarshal(data, &e)
	case "game_end":
		var e AIGameEndEvent
		return e, json.Unmarshal(data, &e)
	case "time_warning":
		var e AITimeWarningEvent
		return e, json.Unmarshal(data, &e)
	default:
		return nil, fmt.Errorf("unknown event type: %s", raw.Type)
	}
}

// --- State Changes ---

// StateChange represents a game state mutation requested by AI.
type StateChange interface {
	StateChangeType() string
}

type StateChangeDiscoverClue struct {
	Type     string `json:"type"`
	PlayerID string `json:"playerId"`
	ClueID   string `json:"clueId"`
}

func (s StateChangeDiscoverClue) StateChangeType() string { return "discover_clue" }

type StateChangeAddItem struct {
	Type     string     `json:"type"`
	PlayerID string     `json:"playerId"`
	Item     ItemSchema `json:"item"`
}

func (s StateChangeAddItem) StateChangeType() string { return "add_item" }

type StateChangeRemoveItem struct {
	Type     string `json:"type"`
	PlayerID string `json:"playerId"`
	ItemID   string `json:"itemId"`
}

func (s StateChangeRemoveItem) StateChangeType() string { return "remove_item" }

type StateChangeTriggerGimmick struct {
	Type      string `json:"type"`
	GimmickID string `json:"gimmickId"`
}

func (s StateChangeTriggerGimmick) StateChangeType() string { return "trigger_gimmick" }

type StateChangeTriggerEvent struct {
	Type             string `json:"type"`
	EventDescription string `json:"eventDescription"`
}

func (s StateChangeTriggerEvent) StateChangeType() string { return "trigger_event" }

type StateChangeUpdateNPCTrust struct {
	Type  string  `json:"type"`
	NPCID string  `json:"npcId"`
	Delta float64 `json:"delta"`
}

func (s StateChangeUpdateNPCTrust) StateChangeType() string { return "update_npc_trust" }

// GameResponse is the top-level AI response for game actions.
type GameResponse struct {
	Events       []json.RawMessage `json:"events"`
	StateChanges []json.RawMessage `json:"stateChanges"`
}

// ParsedEvents deserializes all events in the response.
func (r *GameResponse) ParsedEvents() ([]AIGameEvent, error) {
	events := make([]AIGameEvent, 0, len(r.Events))
	for _, raw := range r.Events {
		e, err := ParseAIGameEvent(raw)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}
