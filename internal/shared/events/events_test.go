package events

import (
	"encoding/json"
	"testing"

	"github.com/anthropics/story/internal/shared/types"
)

func TestNarrationEvent_Interface(t *testing.T) {
	e := NarrationEvent{
		BaseEvent: types.BaseEvent{
			ID:        "evt-1",
			Timestamp: 100,
			Visibility: types.EventVisibility{
				Scope: "all",
			},
		},
		Type: "narration",
		Data: NarrationData{Text: "The wind howls.", Mood: "tense"},
	}

	var ge types.GameEvent = e
	if ge.EventType() != "narration" {
		t.Errorf("expected narration, got %s", ge.EventType())
	}
	base := ge.GetBaseEvent()
	if base.ID != "evt-1" {
		t.Errorf("base ID: got %q", base.ID)
	}
}

func TestNarrationEvent_JSONRoundTrip(t *testing.T) {
	e := NarrationEvent{
		BaseEvent: types.BaseEvent{ID: "evt-1", Timestamp: 100, Visibility: types.EventVisibility{Scope: "all"}},
		Type:      "narration",
		Data:      NarrationData{Text: "Storm approaches", Mood: "urgent"},
	}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got NarrationEvent
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Data.Text != "Storm approaches" {
		t.Errorf("Text: got %q", got.Data.Text)
	}
	if got.Data.Mood != "urgent" {
		t.Errorf("Mood: got %q", got.Data.Mood)
	}
	if got.Type != "narration" {
		t.Errorf("Type: got %q", got.Type)
	}
}

func TestStoryEventEvent_Interface(t *testing.T) {
	e := StoryEventEvent{
		Type: "story_event",
		Data: StoryEventData{Title: "Blackout", Description: "desc", Consequences: []string{"dark"}},
	}
	if e.EventType() != "story_event" {
		t.Errorf("expected story_event, got %s", e.EventType())
	}
}

func TestTimeWarningEvent_Interface(t *testing.T) {
	e := TimeWarningEvent{
		Type: "time_warning",
		Data: TimeWarningData{RemainingMinutes: 5},
	}
	if e.EventType() != "time_warning" {
		t.Errorf("expected time_warning, got %s", e.EventType())
	}
}

func TestNPCDialogueEvent_JSONRoundTrip(t *testing.T) {
	e := NPCDialogueEvent{
		BaseEvent: types.BaseEvent{ID: "evt-2", Timestamp: 200, Visibility: types.EventVisibility{Scope: "room", RoomID: "r1"}},
		Type:      "npc_dialogue",
		Data: NPCDialogueData{
			NPCID:      "npc-1",
			NPCName:    "Bartender",
			PlayerID:   "p1",
			PlayerName: "Alice",
			Text:       "Hello there",
			Emotion:    "friendly",
		},
	}

	data, _ := json.Marshal(e)
	var got NPCDialogueEvent
	json.Unmarshal(data, &got)
	if got.Data.NPCName != "Bartender" {
		t.Errorf("NPCName: got %q", got.Data.NPCName)
	}
	if got.Data.PlayerName != "Alice" {
		t.Errorf("PlayerName: got %q", got.Data.PlayerName)
	}
}

func TestNPCGiveItemEvent_Interface(t *testing.T) {
	e := NPCGiveItemEvent{
		Type: "npc_give_item",
		Data: NPCGiveItemData{
			NPCID:   "npc-1",
			NPCName: "Guard",
			Item:    types.Item{ID: "i1", Name: "Key"},
		},
	}
	if e.EventType() != "npc_give_item" {
		t.Errorf("expected npc_give_item, got %s", e.EventType())
	}
}

func TestNPCReceiveItemEvent_Interface(t *testing.T) {
	e := NPCReceiveItemEvent{
		Type: "npc_receive_item",
		Data: NPCReceiveItemData{
			NPCID:   "npc-1",
			NPCName: "Guard",
			Item:    types.Item{ID: "i1", Name: "Bribe"},
		},
	}
	if e.EventType() != "npc_receive_item" {
		t.Errorf("expected npc_receive_item, got %s", e.EventType())
	}
}

func TestNPCRevealEvent_JSONRoundTrip(t *testing.T) {
	clue := types.Clue{ID: "c1", Name: "Secret Note"}
	e := NPCRevealEvent{
		Type: "npc_reveal",
		Data: NPCRevealData{
			NPCID:      "npc-1",
			NPCName:    "Informant",
			Revelation: "I know the truth",
			Clue:       &clue,
		},
	}

	data, _ := json.Marshal(e)
	var got NPCRevealEvent
	json.Unmarshal(data, &got)
	if got.Data.Clue == nil || got.Data.Clue.Name != "Secret Note" {
		t.Errorf("Clue mismatch")
	}
}

func TestNPCRevealEvent_NilClue(t *testing.T) {
	e := NPCRevealEvent{
		Type: "npc_reveal",
		Data: NPCRevealData{NPCID: "npc-1", NPCName: "Guard", Revelation: "Nothing"},
	}
	data, _ := json.Marshal(e)
	var got NPCRevealEvent
	json.Unmarshal(data, &got)
	if got.Data.Clue != nil {
		t.Errorf("expected nil clue")
	}
}

func TestNPCMovedEvent_Interface(t *testing.T) {
	e := NPCMovedEvent{
		Type: "npc_moved",
		Data: NPCMovedData{NPCID: "npc-1", NPCName: "Guard", From: "Gate", To: "Hallway"},
	}
	if e.EventType() != "npc_moved" {
		t.Errorf("expected npc_moved, got %s", e.EventType())
	}
}

func TestExamineResultEvent_JSONRoundTrip(t *testing.T) {
	e := ExamineResultEvent{
		Type: "examine_result",
		Data: ExamineResultData{
			PlayerID:    "p1",
			PlayerName:  "Alice",
			Target:      "desk",
			Description: "A dusty desk with papers",
			ClueFound:   true,
		},
	}

	data, _ := json.Marshal(e)
	var got ExamineResultEvent
	json.Unmarshal(data, &got)
	if !got.Data.ClueFound {
		t.Errorf("expected ClueFound=true")
	}
}

func TestActionResultEvent_Interface(t *testing.T) {
	e := ActionResultEvent{
		Type: "action_result",
		Data: ActionResultData{
			PlayerID:        "p1",
			PlayerName:      "Bob",
			Action:          "pull lever",
			Result:          "The door opens",
			TriggeredEvents: []string{"gimmick_1"},
		},
	}
	if e.EventType() != "action_result" {
		t.Errorf("expected action_result, got %s", e.EventType())
	}
}

func TestClueFoundEvent_JSONRoundTrip(t *testing.T) {
	e := ClueFoundEvent{
		Type: "clue_found",
		Data: ClueFoundData{
			PlayerID:   "p1",
			PlayerName: "Alice",
			Clue:       types.Clue{ID: "c1", Name: "Hidden message"},
			Location:   "Library",
		},
	}

	data, _ := json.Marshal(e)
	var got ClueFoundEvent
	json.Unmarshal(data, &got)
	if got.Data.Clue.Name != "Hidden message" {
		t.Errorf("Clue.Name: got %q", got.Data.Clue.Name)
	}
}

func TestPlayerMoveEvent_Interface(t *testing.T) {
	e := PlayerMoveEvent{
		Type: "player_move",
		Data: PlayerMoveData{PlayerID: "p1", PlayerName: "Alice", From: "Lobby", To: "Bridge"},
	}
	if e.EventType() != "player_move" {
		t.Errorf("expected player_move, got %s", e.EventType())
	}
}

func TestGameEndEvent_JSONRoundTrip(t *testing.T) {
	e := GameEndEvent{
		BaseEvent: types.BaseEvent{ID: "evt-end", Timestamp: 9999, Visibility: types.EventVisibility{Scope: "all"}},
		Type:      "game_end",
		Data:      GameEndEventData{Reason: "timeout", CommonResult: "The station exploded"},
	}

	data, _ := json.Marshal(e)
	var got GameEndEvent
	json.Unmarshal(data, &got)
	if got.Data.Reason != "timeout" {
		t.Errorf("Reason: got %q", got.Data.Reason)
	}
	if got.Data.CommonResult != "The station exploded" {
		t.Errorf("CommonResult: got %q", got.Data.CommonResult)
	}
}

func TestAllEvents_ImplementGameEvent(t *testing.T) {
	// Compile-time check that all events implement GameEvent
	events := []types.GameEvent{
		NarrationEvent{Type: "narration"},
		StoryEventEvent{Type: "story_event"},
		TimeWarningEvent{Type: "time_warning"},
		NPCDialogueEvent{Type: "npc_dialogue"},
		NPCGiveItemEvent{Type: "npc_give_item"},
		NPCReceiveItemEvent{Type: "npc_receive_item"},
		NPCRevealEvent{Type: "npc_reveal"},
		NPCMovedEvent{Type: "npc_moved"},
		ExamineResultEvent{Type: "examine_result"},
		ActionResultEvent{Type: "action_result"},
		ClueFoundEvent{Type: "clue_found"},
		PlayerMoveEvent{Type: "player_move"},
		GameEndEvent{Type: "game_end"},
	}

	expectedTypes := []string{
		"narration", "story_event", "time_warning",
		"npc_dialogue", "npc_give_item", "npc_receive_item", "npc_reveal", "npc_moved",
		"examine_result", "action_result", "clue_found", "player_move",
		"game_end",
	}

	for i, e := range events {
		if e.EventType() != expectedTypes[i] {
			t.Errorf("event %d: expected %q, got %q", i, expectedTypes[i], e.EventType())
		}
	}
}
