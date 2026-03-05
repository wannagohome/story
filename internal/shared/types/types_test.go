package types

import (
	"encoding/json"
	"testing"
)

// --- Game ---

func TestGameStatus_Constants(t *testing.T) {
	statuses := []GameStatus{
		GameStatusLobby,
		GameStatusGenerating,
		GameStatusBriefing,
		GameStatusPlaying,
		GameStatusEnding,
		GameStatusFinished,
	}
	expected := []string{"lobby", "generating", "briefing", "playing", "ending", "finished"}
	for i, s := range statuses {
		if string(s) != expected[i] {
			t.Errorf("expected %q, got %q", expected[i], s)
		}
	}
}

func TestGame_JSONRoundTrip(t *testing.T) {
	startedAt := int64(1000)
	game := Game{
		ID:       "game-1",
		RoomCode: "ABCD",
		HostID:   "player-1",
		Status:   GameStatusLobby,
		Settings: GameSettings{
			MaxPlayers:     8,
			TimeoutMinutes: 20,
			HasGM:          true,
			HasNPC:         true,
		},
		World:   nil,
		Players: map[string]*Player{},
		EventLog: []interface{}{
			map[string]interface{}{"type": "test"},
		},
		CreatedAt: 999,
		StartedAt: &startedAt,
		EndedAt:   nil,
	}

	data, err := json.Marshal(game)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Game
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.ID != game.ID {
		t.Errorf("ID: got %q, want %q", got.ID, game.ID)
	}
	if got.Status != GameStatusLobby {
		t.Errorf("Status: got %q, want %q", got.Status, GameStatusLobby)
	}
	if got.Settings.MaxPlayers != 8 {
		t.Errorf("MaxPlayers: got %d, want 8", got.Settings.MaxPlayers)
	}
	if got.StartedAt == nil || *got.StartedAt != 1000 {
		t.Errorf("StartedAt: got %v, want 1000", got.StartedAt)
	}
	if got.EndedAt != nil {
		t.Errorf("EndedAt: got %v, want nil", got.EndedAt)
	}
}

// --- BaseEvent ---

func TestBaseEvent_JSONRoundTrip(t *testing.T) {
	ev := BaseEvent{
		ID:        "evt-1",
		Timestamp: 12345,
		Visibility: EventVisibility{
			Scope:     "room",
			RoomID:    "room-1",
			PlayerIDs: nil,
		},
	}

	data, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got BaseEvent
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ID != "evt-1" {
		t.Errorf("ID: got %q, want %q", got.ID, "evt-1")
	}
	if got.Visibility.Scope != "room" {
		t.Errorf("Scope: got %q, want %q", got.Visibility.Scope, "room")
	}
	if got.Visibility.RoomID != "room-1" {
		t.Errorf("RoomID: got %q, want %q", got.Visibility.RoomID, "room-1")
	}
}

func TestEventVisibility_OmitEmpty(t *testing.T) {
	vis := EventVisibility{Scope: "all"}
	data, _ := json.Marshal(vis)
	s := string(data)
	if containsKey(s, "roomId") {
		t.Errorf("expected roomId to be omitted, got %s", s)
	}
	if containsKey(s, "playerIds") {
		t.Errorf("expected playerIds to be omitted, got %s", s)
	}
}

// --- World ---

func TestWorld_JSONRoundTrip(t *testing.T) {
	commonGoal := "Survive the station"
	w := World{
		Title:      "Test World",
		Synopsis:   "A test synopsis",
		Atmosphere: "tense",
		GameStructure: GameStructure{
			Concept:          "test concept",
			CoreConflict:     "conflict",
			ProgressionStyle: "linear",
			CommonGoal:       &commonGoal,
			EndConditions: []EndCondition{
				{ID: "ec-1", Description: "timeout", TriggerType: "timeout", IsFallback: true},
			},
			WinConditions:   []WinCondition{{Description: "win", EvaluationCriteria: "criteria"}},
			RequiredSystems: []RequiredSystem{RequiredSystemVote},
			BriefingText:    "brief",
		},
		Map: GameMap{
			Rooms: []Room{
				{ID: "r1", Name: "Room 1", Description: "desc", Type: "public"},
			},
			Connections: []Connection{},
		},
		PlayerRoles: []PlayerRole{},
		NPCs:        []NPC{},
		Clues:       []Clue{},
		Gimmicks:    []Gimmick{},
		Information:  InformationLayers{},
	}

	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got World
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Title != "Test World" {
		t.Errorf("Title: got %q, want %q", got.Title, "Test World")
	}
	if got.GameStructure.CommonGoal == nil || *got.GameStructure.CommonGoal != "Survive the station" {
		t.Errorf("CommonGoal mismatch")
	}
}

func TestRequiredSystem_Constants(t *testing.T) {
	if RequiredSystemVote != "vote" {
		t.Errorf("expected vote, got %s", RequiredSystemVote)
	}
	if RequiredSystemConsensus != "consensus" {
		t.Errorf("expected consensus, got %s", RequiredSystemConsensus)
	}
	if RequiredSystemAIJudge != "ai_judge" {
		t.Errorf("expected ai_judge, got %s", RequiredSystemAIJudge)
	}
}

// --- Map ---

func TestGameMap_JSONRoundTrip(t *testing.T) {
	m := GameMap{
		Rooms: []Room{
			{ID: "r1", Name: "Bridge", Description: "The bridge", Type: "public", Items: []Item{{ID: "i1", Name: "Key", Description: "A key", IsKey: true}}, NPCIDs: []string{"npc1"}, ClueIDs: []string{"c1"}},
		},
		Connections: []Connection{
			{RoomA: "r1", RoomB: "r2", Bidirectional: true},
		},
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got GameMap
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.Rooms) != 1 || got.Rooms[0].Name != "Bridge" {
		t.Errorf("unexpected rooms: %+v", got.Rooms)
	}
	if !got.Connections[0].Bidirectional {
		t.Errorf("expected bidirectional")
	}
}

// --- Player ---

func TestPlayer_JSONRoundTrip(t *testing.T) {
	specialRole := "culprit"
	achieved := true
	p := Player{
		ID:            "p1",
		Nickname:      "Alice",
		IsHost:        true,
		Status:        "connected",
		CurrentRoomID: "r1",
		Role: &PlayerRole{
			ID:            "role-1",
			CharacterName: "Detective",
			Background:    "A veteran detective",
			PersonalGoals: []PersonalGoal{
				{ID: "g1", Description: "Find the culprit", EvaluationHint: "hint", EntityRefs: []string{"npc-1"}, IsAchieved: &achieved},
			},
			Secret:      "Knows the victim",
			SpecialRole: &specialRole,
			Relationships: []Relationship{
				{TargetCharacterName: "Bob", Description: "rivals"},
			},
		},
		Inventory:         []Item{{ID: "i1", Name: "Flashlight"}},
		DiscoveredClueIDs: []string{"c1"},
		MoveHistory: []MoveRecord{
			{RoomID: "r1", RoomName: "Lobby", EnteredAt: 100},
		},
		ConnectedAt: 50,
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Player
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Nickname != "Alice" {
		t.Errorf("Nickname: got %q", got.Nickname)
	}
	if got.Role == nil || got.Role.CharacterName != "Detective" {
		t.Errorf("Role mismatch")
	}
	if got.Role.PersonalGoals[0].IsAchieved == nil || !*got.Role.PersonalGoals[0].IsAchieved {
		t.Errorf("IsAchieved mismatch")
	}
}

func TestMoveRecord_LeftAtOmitEmpty(t *testing.T) {
	mr := MoveRecord{RoomID: "r1", RoomName: "Room", EnteredAt: 100}
	data, _ := json.Marshal(mr)
	if containsKey(string(data), "leftAt") {
		t.Errorf("expected leftAt to be omitted")
	}
}

// --- NPC ---

func TestNPC_JSONRoundTrip(t *testing.T) {
	npc := NPC{
		ID:                "npc-1",
		Name:              "Bartender",
		CurrentRoomID:     "r1",
		Persona:           "Gruff but kind",
		KnownInfo:         []string{"info1"},
		HiddenInfo:        []string{"secret1"},
		BehaviorPrinciple: "Never lies",
		Gimmick: &NPCGimmick{
			Description:      "Drops glass",
			TriggerCondition: "trust > 0.8",
			Effect:           "Reveals clue",
		},
		InitialTrust: 0.5,
	}

	data, err := json.Marshal(npc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got NPC
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Name != "Bartender" {
		t.Errorf("Name: got %q", got.Name)
	}
	if got.Gimmick == nil || got.Gimmick.Description != "Drops glass" {
		t.Errorf("Gimmick mismatch")
	}
	if got.InitialTrust != 0.5 {
		t.Errorf("InitialTrust: got %f", got.InitialTrust)
	}
}

func TestNPC_NilGimmick(t *testing.T) {
	npc := NPC{ID: "npc-2", Name: "Guard"}
	data, _ := json.Marshal(npc)
	var got NPC
	json.Unmarshal(data, &got)
	if got.Gimmick != nil {
		t.Errorf("expected nil gimmick")
	}
}

// --- Items ---

func TestItem_JSONRoundTrip(t *testing.T) {
	ownerID := "p1"
	item := Item{
		ID:          "i1",
		Name:        "Key",
		Description: "A rusty key",
		OwnerID:     &ownerID,
		IsKey:       true,
	}

	data, _ := json.Marshal(item)
	var got Item
	json.Unmarshal(data, &got)
	if got.OwnerID == nil || *got.OwnerID != "p1" {
		t.Errorf("OwnerID mismatch")
	}
	if !got.IsKey {
		t.Errorf("expected IsKey=true")
	}
}

func TestClue_JSONRoundTrip(t *testing.T) {
	clue := Clue{
		ID:                "c1",
		Name:              "Note",
		Description:       "A crumpled note",
		RoomID:            "r1",
		DiscoverCondition: "examine desk",
		RelatedClueIDs:    []string{"c2"},
	}

	data, _ := json.Marshal(clue)
	var got Clue
	json.Unmarshal(data, &got)
	if got.Name != "Note" {
		t.Errorf("Name: got %q", got.Name)
	}
	if len(got.RelatedClueIDs) != 1 {
		t.Errorf("RelatedClueIDs length: got %d", len(got.RelatedClueIDs))
	}
}

func TestGimmick_JSONRoundTrip(t *testing.T) {
	g := Gimmick{
		ID:               "g1",
		Description:      "Hidden passage",
		RoomID:           "r1",
		TriggerCondition: "pull lever",
		Effect:           "Opens passage",
	}

	data, _ := json.Marshal(g)
	var got Gimmick
	json.Unmarshal(data, &got)
	if got.Effect != "Opens passage" {
		t.Errorf("Effect: got %q", got.Effect)
	}
}

// --- Information ---

func TestInformationLayers_JSONRoundTrip(t *testing.T) {
	info := InformationLayers{
		Public: PublicInfo{
			Title:    "Test",
			Synopsis: "Synopsis",
			CharacterList: []CharacterListEntry{
				{Name: "Alice", PublicDescription: "A detective"},
			},
			Relationships: "Complex",
			MapOverview:   "5 rooms",
			NPCList: []NPCListEntry{
				{Name: "Guard", Location: "Gate"},
			},
			GameRules: "Standard",
		},
		SemiPublic: []SemiPublicInfo{
			{ID: "sp1", TargetPlayerIDs: []string{"p1", "p2"}, Content: "Shared secret"},
		},
		Private: []PrivateInfo{
			{PlayerID: "p1", AdditionalSecrets: []string{"secret1"}},
		},
	}

	data, _ := json.Marshal(info)
	var got InformationLayers
	json.Unmarshal(data, &got)
	if got.Public.Title != "Test" {
		t.Errorf("Title: got %q", got.Public.Title)
	}
	if len(got.SemiPublic) != 1 {
		t.Errorf("SemiPublic length: got %d", len(got.SemiPublic))
	}
	if len(got.Private) != 1 {
		t.Errorf("Private length: got %d", len(got.Private))
	}
}

// --- Ending ---

func TestGameEndData_JSONRoundTrip(t *testing.T) {
	specialRole := "culprit"
	endData := GameEndData{
		CommonResult: "The station was saved",
		PlayerEndings: []PlayerEnding{
			{
				PlayerID: "p1",
				Summary:  "Found the culprit",
				GoalResults: []GoalResult{
					{GoalID: "g1", Description: "Find culprit", Achieved: true, Evaluation: "Correctly identified"},
				},
				Narrative: "The detective solved the case",
			},
		},
		SecretReveal: SecretReveal{
			PlayerSecrets: []PlayerSecretEntry{
				{PlayerID: "p1", CharacterName: "Detective", Secret: "Knew the victim", SpecialRole: &specialRole},
			},
			SemiPublicReveal: []SemiPublicRevealEntry{
				{Info: "Shared info", SharedBetween: []string{"p1", "p2"}},
			},
			UndiscoveredClues: []UndiscoveredClueEntry{
				{Clue: Clue{ID: "c2", Name: "Hidden note"}, RoomName: "Library"},
			},
			NPCSecrets: []NPCSecretEntry{
				{NPCName: "Guard", HiddenInfo: []string{"Was bribed"}},
			},
			UntriggeredGimmicks: []GimmickReveal{
				{GimmickID: "g1", Name: "Trap", Description: "A hidden trap", RoomID: "r2", Condition: "step on tile"},
			},
		},
	}

	data, err := json.Marshal(endData)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got GameEndData
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.CommonResult != "The station was saved" {
		t.Errorf("CommonResult: got %q", got.CommonResult)
	}
	if len(got.PlayerEndings) != 1 {
		t.Errorf("PlayerEndings length: got %d", len(got.PlayerEndings))
	}
	if got.SecretReveal.PlayerSecrets[0].SpecialRole == nil {
		t.Errorf("expected SpecialRole to be non-nil")
	}
}

// --- GameState ---

func TestGameState_JSONRoundTrip(t *testing.T) {
	triggeredAt := int64(500)
	gs := GameState{
		Players: map[string]*Player{
			"p1": {ID: "p1", Nickname: "Alice"},
		},
		ClueStates: map[string]ClueState{
			"c1": {IsDiscovered: true, DiscoveredBy: []string{"p1"}},
		},
		NPCStates: map[string]NPCState{
			"npc1": {
				TrustLevels: map[string]float64{"p1": 0.7},
				ConversationHistory: []ConversationRecord{
					{PlayerID: "p1", Message: "Hello", Response: "Hi", Timestamp: 100},
				},
				GimmickTriggered: false,
			},
		},
		GimmickStates: map[string]GimmickState{
			"g1": {IsTriggered: true, TriggeredAt: &triggeredAt},
		},
		ElapsedTime: 300,
	}

	data, _ := json.Marshal(gs)
	var got GameState
	json.Unmarshal(data, &got)
	if got.ElapsedTime != 300 {
		t.Errorf("ElapsedTime: got %d", got.ElapsedTime)
	}
	if got.ClueStates["c1"].IsDiscovered != true {
		t.Errorf("ClueState mismatch")
	}
	if got.GimmickStates["g1"].TriggeredAt == nil || *got.GimmickStates["g1"].TriggeredAt != 500 {
		t.Errorf("GimmickState TriggeredAt mismatch")
	}
}

// --- GameContext ---

func TestGameContext_JSONRoundTrip(t *testing.T) {
	ctx := GameContext{
		World: World{Title: "Test"},
		CurrentState: GameState{
			Players:     map[string]*Player{},
			ElapsedTime: 100,
		},
		RecentEvents:     []interface{}{},
		ActionLog:        []interface{}{},
		RequestingPlayer: Player{ID: "p1", Nickname: "Alice"},
		CurrentRoom:      Room{ID: "r1", Name: "Lobby"},
		PlayersInRoom:    []Player{{ID: "p1", Nickname: "Alice"}},
	}

	data, _ := json.Marshal(ctx)
	var got GameContext
	json.Unmarshal(data, &got)
	if got.World.Title != "Test" {
		t.Errorf("World.Title: got %q", got.World.Title)
	}
	if got.CurrentRoom.Name != "Lobby" {
		t.Errorf("CurrentRoom.Name: got %q", got.CurrentRoom.Name)
	}
}

// --- Result ---

func TestResult_JSONRoundTrip(t *testing.T) {
	r := Result[string, string]{
		Ok:    true,
		Value: "success",
	}

	data, _ := json.Marshal(r)
	var got Result[string, string]
	json.Unmarshal(data, &got)
	if !got.Ok || got.Value != "success" {
		t.Errorf("Result mismatch: %+v", got)
	}
}

func TestResult_Error(t *testing.T) {
	r := Result[string, string]{
		Ok:    false,
		Error: "something went wrong",
	}

	data, _ := json.Marshal(r)
	var got Result[string, string]
	json.Unmarshal(data, &got)
	if got.Ok || got.Error != "something went wrong" {
		t.Errorf("Result error mismatch: %+v", got)
	}
}

// --- Feedback ---

func TestFeedback_JSONRoundTrip(t *testing.T) {
	comment := "Great game!"
	fb := Feedback{
		PlayerID:        "p1",
		FunRating:       5,
		ImmersionRating: 4,
		Comment:         &comment,
		SubmittedAt:     12345,
	}

	data, _ := json.Marshal(fb)
	var got Feedback
	json.Unmarshal(data, &got)
	if got.FunRating != 5 {
		t.Errorf("FunRating: got %d", got.FunRating)
	}
	if got.Comment == nil || *got.Comment != "Great game!" {
		t.Errorf("Comment mismatch")
	}
}

func TestFeedback_NilComment(t *testing.T) {
	fb := Feedback{PlayerID: "p1", FunRating: 3, ImmersionRating: 3, SubmittedAt: 100}
	data, _ := json.Marshal(fb)
	var got Feedback
	json.Unmarshal(data, &got)
	if got.Comment != nil {
		t.Errorf("expected nil comment")
	}
}

// helper
func containsKey(jsonStr, key string) bool {
	var m map[string]interface{}
	json.Unmarshal([]byte(jsonStr), &m)
	_, ok := m[key]
	return ok
}
