package schemas

import (
	"encoding/json"
	"strings"
	"testing"
)

// --- WorldGenerationMeta ---

func TestWorldGenerationMeta_Validate_Valid(t *testing.T) {
	m := WorldGenerationMeta{
		Theme:             "mystery",
		Setting:           "space station",
		EstimatedDuration: 20,
		HasGM:             true,
		HasNPC:            true,
	}
	if err := m.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWorldGenerationMeta_Validate_TooShort(t *testing.T) {
	m := WorldGenerationMeta{EstimatedDuration: 5}
	err := m.Validate()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "minimum 10") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWorldGenerationMeta_Validate_TooLong(t *testing.T) {
	m := WorldGenerationMeta{EstimatedDuration: 60}
	err := m.Validate()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "maximum 30") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- PlayerRoleSchema ---

func TestPlayerRoleSchema_Validate_Valid(t *testing.T) {
	r := PlayerRoleSchema{
		ID:            "role-1",
		CharacterName: "Detective",
		PersonalGoals: []PersonalGoalSchema{
			{ID: "g1", Description: "Find the truth"},
		},
	}
	if err := r.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPlayerRoleSchema_Validate_NoGoals(t *testing.T) {
	r := PlayerRoleSchema{
		ID:            "role-1",
		PersonalGoals: []PersonalGoalSchema{},
	}
	err := r.Validate()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "personalGoals") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- WorldGenerationGameStructure ---

func TestGameStructure_Validate_Valid(t *testing.T) {
	gs := WorldGenerationGameStructure{
		EndConditions: []EndConditionSchema{
			{ID: "ec-1", IsFallback: true},
		},
	}
	if err := gs.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGameStructure_Validate_NoEndConditions(t *testing.T) {
	gs := WorldGenerationGameStructure{EndConditions: []EndConditionSchema{}}
	err := gs.Validate()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "endConditions: minimum 1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGameStructure_Validate_NoFallback(t *testing.T) {
	gs := WorldGenerationGameStructure{
		EndConditions: []EndConditionSchema{
			{ID: "ec-1", IsFallback: false},
		},
	}
	err := gs.Validate()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "isFallback") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- WorldGeneration full validation ---

func makeValidWorldGeneration() WorldGeneration {
	return WorldGeneration{
		Meta: WorldGenerationMeta{
			Theme:             "mystery",
			Setting:           "manor",
			EstimatedDuration: 20,
		},
		World: WorldGenerationWorld{
			Title:      "Mystery Manor",
			Synopsis:   "A dark tale",
			Atmosphere: "tense",
		},
		GameStructure: WorldGenerationGameStructure{
			Concept:      "whodunit",
			CoreConflict: "murder",
			EndConditions: []EndConditionSchema{
				{ID: "ec-1", IsFallback: true, TriggerType: "timeout"},
			},
			BriefingText: "Welcome",
		},
		Map: WorldGenerationMap{
			Rooms: []RoomSchema{
				{ID: "r1", Name: "Lobby", Type: "public"},
				{ID: "r2", Name: "Library", Type: "public"},
				{ID: "r3", Name: "Kitchen", Type: "public"},
				{ID: "r4", Name: "Garden", Type: "public"},
			},
			Connections: []ConnectionSchema{
				{RoomA: "r1", RoomB: "r2", Bidirectional: true},
				{RoomA: "r2", RoomB: "r3", Bidirectional: true},
				{RoomA: "r3", RoomB: "r4", Bidirectional: true},
			},
		},
		Characters: WorldGenerationCharacters{
			PlayerRoles: []PlayerRoleSchema{
				{
					ID:            "role-1",
					CharacterName: "Detective",
					PersonalGoals: []PersonalGoalSchema{{ID: "g1", Description: "goal"}},
				},
				{
					ID:            "role-2",
					CharacterName: "Butler",
					PersonalGoals: []PersonalGoalSchema{{ID: "g2", Description: "goal"}},
				},
			},
			NPCs: []NPCSchema{
				{ID: "npc-1", Name: "Guard", CurrentRoomID: "r1"},
			},
		},
		Information: WorldGenerationInformation{
			SemiPublic: []SemiPublicInfoSchema{
				{ID: "sp1", TargetPlayerIDs: []string{"role-1", "role-2"}, Content: "shared"},
			},
		},
		Clues: []ClueSchema{
			{ID: "c1", Name: "Note", RoomID: "r1"},
			{ID: "c2", Name: "Footprint", RoomID: "r2"},
			{ID: "c3", Name: "Letter", RoomID: "r3"},
			{ID: "c4", Name: "Key", RoomID: "r4"},
		},
		Gimmicks: []GimmickSchema{
			{ID: "gm1", RoomID: "r1"},
		},
	}
}

func TestWorldGeneration_Validate_Valid(t *testing.T) {
	wg := makeValidWorldGeneration()
	if err := wg.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWorldGeneration_Validate_MetaError(t *testing.T) {
	wg := makeValidWorldGeneration()
	wg.Meta.EstimatedDuration = 5
	err := wg.Validate()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "meta:") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWorldGeneration_Validate_NotEnoughRooms(t *testing.T) {
	wg := makeValidWorldGeneration()
	wg.Map.Rooms = []RoomSchema{{ID: "r1", Name: "Only"}}
	err := wg.Validate()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "map.rooms") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWorldGeneration_Validate_NotEnoughClues(t *testing.T) {
	wg := makeValidWorldGeneration()
	wg.Clues = []ClueSchema{{ID: "c1", Name: "One", RoomID: "r1"}}
	err := wg.Validate()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "clues:") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWorldGeneration_Validate_NoSemiPublic(t *testing.T) {
	wg := makeValidWorldGeneration()
	wg.Information.SemiPublic = nil
	err := wg.Validate()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "semiPublic") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWorldGeneration_Validate_NPCInvalidRoom(t *testing.T) {
	wg := makeValidWorldGeneration()
	wg.Characters.NPCs[0].CurrentRoomID = "nonexistent"
	err := wg.Validate()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "npcs[0]") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWorldGeneration_Validate_ClueInvalidRoom(t *testing.T) {
	wg := makeValidWorldGeneration()
	wg.Clues[0].RoomID = "nonexistent"
	err := wg.Validate()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "clues[0]") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWorldGeneration_Validate_GimmickInvalidRoom(t *testing.T) {
	wg := makeValidWorldGeneration()
	wg.Gimmicks[0].RoomID = "nonexistent"
	err := wg.Validate()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "gimmicks[0]") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWorldGeneration_Validate_IsolatedRoom(t *testing.T) {
	wg := makeValidWorldGeneration()
	wg.Map.Rooms = append(wg.Map.Rooms, RoomSchema{ID: "r-isolated", Name: "Isolated"})
	err := wg.Validate()
	if err == nil {
		t.Fatalf("expected error for isolated room")
	}
	if !strings.Contains(err.Error(), "isolated") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWorldGeneration_Validate_InvalidNPCRef(t *testing.T) {
	wg := makeValidWorldGeneration()
	wg.Map.Rooms[0].NPCIDs = []string{"nonexistent-npc"}
	err := wg.Validate()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "npcId") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWorldGeneration_Validate_InvalidClueRef(t *testing.T) {
	wg := makeValidWorldGeneration()
	wg.Map.Rooms[0].ClueIDs = []string{"nonexistent-clue"}
	err := wg.Validate()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "clueId") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWorldGeneration_Validate_InvalidSemiPublicTarget(t *testing.T) {
	wg := makeValidWorldGeneration()
	wg.Information.SemiPublic[0].TargetPlayerIDs = []string{"nonexistent-role"}
	err := wg.Validate()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "targetPlayerIds") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWorldGeneration_Validate_PlayerRoleNoGoals(t *testing.T) {
	wg := makeValidWorldGeneration()
	wg.Characters.PlayerRoles[0].PersonalGoals = nil
	err := wg.Validate()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "playerRoles[0]") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- Map Connectivity BFS ---

func TestValidateMapConnectivity_EmptyMap(t *testing.T) {
	err := validateMapConnectivity(WorldGenerationMap{})
	if err != nil {
		t.Errorf("expected nil error for empty map, got: %v", err)
	}
}

func TestValidateMapConnectivity_SingleRoom(t *testing.T) {
	m := WorldGenerationMap{
		Rooms: []RoomSchema{{ID: "r1"}},
	}
	if err := validateMapConnectivity(m); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateMapConnectivity_Connected(t *testing.T) {
	m := WorldGenerationMap{
		Rooms: []RoomSchema{{ID: "r1"}, {ID: "r2"}, {ID: "r3"}},
		Connections: []ConnectionSchema{
			{RoomA: "r1", RoomB: "r2"},
			{RoomA: "r2", RoomB: "r3"},
		},
	}
	if err := validateMapConnectivity(m); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateMapConnectivity_Disconnected(t *testing.T) {
	m := WorldGenerationMap{
		Rooms: []RoomSchema{{ID: "r1"}, {ID: "r2"}, {ID: "r3"}},
		Connections: []ConnectionSchema{
			{RoomA: "r1", RoomB: "r2"},
		},
	}
	err := validateMapConnectivity(m)
	if err == nil {
		t.Fatalf("expected error for disconnected map")
	}
	if !strings.Contains(err.Error(), "r3") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- ParseAIGameEvent ---

func TestParseAIGameEvent_Narration(t *testing.T) {
	raw := json.RawMessage(`{"type":"narration","data":{"text":"The wind howls","mood":"tense"}}`)
	event, err := ParseAIGameEvent(raw)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if event.AIEventType() != "narration" {
		t.Errorf("expected narration, got %s", event.AIEventType())
	}
	ne, ok := event.(AINarrationEvent)
	if !ok {
		t.Fatalf("expected AINarrationEvent")
	}
	if ne.Data.Text != "The wind howls" {
		t.Errorf("Text: got %q", ne.Data.Text)
	}
}

func TestParseAIGameEvent_AllTypes(t *testing.T) {
	cases := []struct {
		name     string
		json     string
		expected string
	}{
		{"narration", `{"type":"narration","data":{"text":"t","mood":"m"}}`, "narration"},
		{"npc_dialogue", `{"type":"npc_dialogue","data":{"npcId":"n","npcName":"N","text":"t","emotion":"e"}}`, "npc_dialogue"},
		{"npc_give_item", `{"type":"npc_give_item","data":{"npcId":"n","npcName":"N","playerId":"p","playerName":"P","item":{"id":"i","name":"I"}}}`, "npc_give_item"},
		{"npc_receive_item", `{"type":"npc_receive_item","data":{"npcId":"n","npcName":"N","playerId":"p","playerName":"P","item":{"id":"i","name":"I"}}}`, "npc_receive_item"},
		{"npc_reveal", `{"type":"npc_reveal","data":{"npcId":"n","npcName":"N","revelation":"r"}}`, "npc_reveal"},
		{"clue_found", `{"type":"clue_found","data":{"playerId":"p","playerName":"P","clue":{"id":"c","name":"C","description":"D"},"location":"L"}}`, "clue_found"},
		{"story_event", `{"type":"story_event","data":{"title":"T","description":"D","consequences":[]}}`, "story_event"},
		{"examine_result", `{"type":"examine_result","data":{"playerId":"p","playerName":"P","target":"T","description":"D","clueFound":false}}`, "examine_result"},
		{"action_result", `{"type":"action_result","data":{"playerId":"p","playerName":"P","action":"A","result":"R","triggeredEvents":[]}}`, "action_result"},
		{"player_move", `{"type":"player_move","data":{"playerId":"p","playerName":"P","from":"F","to":"T"}}`, "player_move"},
		{"game_end", `{"type":"game_end","data":{"reason":"R","commonResult":"C"}}`, "game_end"},
		{"time_warning", `{"type":"time_warning","data":{"remainingMinutes":5}}`, "time_warning"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			event, err := ParseAIGameEvent(json.RawMessage(tc.json))
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if event.AIEventType() != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, event.AIEventType())
			}
		})
	}
}

func TestParseAIGameEvent_Unknown(t *testing.T) {
	raw := json.RawMessage(`{"type":"unknown_type","data":{}}`)
	_, err := ParseAIGameEvent(raw)
	if err == nil {
		t.Fatalf("expected error for unknown type")
	}
	if !strings.Contains(err.Error(), "unknown event type") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseAIGameEvent_InvalidJSON(t *testing.T) {
	raw := json.RawMessage(`not valid json`)
	_, err := ParseAIGameEvent(raw)
	if err == nil {
		t.Fatalf("expected error for invalid JSON")
	}
}

// --- GameResponse ---

func TestGameResponse_ParsedEvents(t *testing.T) {
	resp := GameResponse{
		Events: []json.RawMessage{
			json.RawMessage(`{"type":"narration","data":{"text":"Hello","mood":"calm"}}`),
			json.RawMessage(`{"type":"story_event","data":{"title":"T","description":"D","consequences":[]}}`),
		},
	}
	events, err := resp.ParsedEvents()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].AIEventType() != "narration" {
		t.Errorf("event 0: expected narration, got %s", events[0].AIEventType())
	}
	if events[1].AIEventType() != "story_event" {
		t.Errorf("event 1: expected story_event, got %s", events[1].AIEventType())
	}
}

func TestGameResponse_ParsedEvents_Error(t *testing.T) {
	resp := GameResponse{
		Events: []json.RawMessage{
			json.RawMessage(`{"type":"unknown","data":{}}`),
		},
	}
	_, err := resp.ParsedEvents()
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestGameResponse_JSONRoundTrip(t *testing.T) {
	resp := GameResponse{
		Events: []json.RawMessage{
			json.RawMessage(`{"type":"narration","data":{"text":"t","mood":"m"}}`),
		},
		StateChanges: []json.RawMessage{
			json.RawMessage(`{"type":"discover_clue","playerId":"p1","clueId":"c1"}`),
		},
	}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got GameResponse
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.Events) != 1 {
		t.Errorf("Events length: got %d", len(got.Events))
	}
	if len(got.StateChanges) != 1 {
		t.Errorf("StateChanges length: got %d", len(got.StateChanges))
	}
}

// --- StateChange types ---

func TestStateChange_Interfaces(t *testing.T) {
	changes := []StateChange{
		StateChangeDiscoverClue{Type: "discover_clue"},
		StateChangeAddItem{Type: "add_item"},
		StateChangeRemoveItem{Type: "remove_item"},
		StateChangeTriggerGimmick{Type: "trigger_gimmick"},
		StateChangeTriggerEvent{Type: "trigger_event"},
		StateChangeUpdateNPCTrust{Type: "update_npc_trust"},
	}
	expected := []string{
		"discover_clue", "add_item", "remove_item",
		"trigger_gimmick", "trigger_event", "update_npc_trust",
	}
	for i, sc := range changes {
		if sc.StateChangeType() != expected[i] {
			t.Errorf("change %d: expected %q, got %q", i, expected[i], sc.StateChangeType())
		}
	}
}

// --- NPCResponse ---

func TestNPCResponse_Validate_Valid(t *testing.T) {
	r := NPCResponse{
		Dialogue:    "Hello",
		Emotion:     "friendly",
		TrustChange: 0.5,
	}
	if err := r.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNPCResponse_Validate_TrustTooLow(t *testing.T) {
	r := NPCResponse{TrustChange: -1.5}
	err := r.Validate()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "minimum -1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNPCResponse_Validate_TrustTooHigh(t *testing.T) {
	r := NPCResponse{TrustChange: 1.5}
	err := r.Validate()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "maximum 1") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNPCResponse_ParsedEvents(t *testing.T) {
	r := NPCResponse{
		Events: []json.RawMessage{
			json.RawMessage(`{"type":"npc_dialogue","data":{"npcId":"n","npcName":"N","text":"Hello","emotion":"calm"}}`),
		},
	}
	events, err := r.ParsedEvents()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].AIEventType() != "npc_dialogue" {
		t.Errorf("expected npc_dialogue, got %s", events[0].AIEventType())
	}
}

// --- Ending ---

func TestEnding_JSONRoundTrip(t *testing.T) {
	ending := Ending{
		CommonResult: "The mystery was solved",
		PlayerEndings: []PlayerEndingSchema{
			{
				PlayerID: "p1",
				Summary:  "The detective won",
				GoalResults: []GoalResultSchema{
					{GoalID: "g1", Description: "Find culprit", Achieved: true, Evaluation: "Correctly identified"},
				},
				Narrative: "A thrilling conclusion",
			},
		},
	}

	data, err := json.Marshal(ending)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Ending
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.CommonResult != "The mystery was solved" {
		t.Errorf("CommonResult: got %q", got.CommonResult)
	}
	if len(got.PlayerEndings) != 1 {
		t.Fatalf("PlayerEndings length: got %d", len(got.PlayerEndings))
	}
	if !got.PlayerEndings[0].GoalResults[0].Achieved {
		t.Errorf("expected Achieved=true")
	}
}
