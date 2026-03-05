package ai

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/anthropics/story/internal/ai/provider"
	"github.com/anthropics/story/internal/shared/schemas"
	"github.com/anthropics/story/internal/shared/types"
)

// mockProvider implements provider.AIProvider for testing.
type mockProvider struct {
	structuredResponse json.RawMessage
	textResponse       string
	err                error
}

func (m *mockProvider) GenerateStructured(_ context.Context, _ provider.StructuredRequest) (json.RawMessage, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.structuredResponse, nil
}

func (m *mockProvider) GenerateText(_ context.Context, _ provider.TextRequest) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.textResponse, nil
}

// sampleWorldGeneration returns a valid WorldGeneration JSON for testing.
func sampleWorldGeneration() json.RawMessage {
	wg := schemas.WorldGeneration{
		Meta: schemas.WorldGenerationMeta{
			Theme:             "mystery",
			Setting:           "abandoned mansion",
			EstimatedDuration: 20,
			HasGM:             true,
			HasNPC:            true,
		},
		World: schemas.WorldGenerationWorld{
			Title:      "The Last Supper",
			Synopsis:   "A dinner party gone wrong. Someone poisoned the host.",
			Atmosphere: "Tense and suspicious",
		},
		GameStructure: schemas.WorldGenerationGameStructure{
			Concept:          "Murder mystery dinner party",
			CoreConflict:     "Who poisoned the host?",
			ProgressionStyle: "investigation",
			EndConditions: []schemas.EndConditionSchema{
				{ID: "ec-1", Description: "Correct accusation", TriggerType: "ai_judgment", TriggerCriteria: map[string]interface{}{"type": "accusation"}, IsFallback: false},
				{ID: "ec-2", Description: "Time runs out", TriggerType: "timeout", TriggerCriteria: map[string]interface{}{"minutes": 20}, IsFallback: true},
			},
			WinConditions: []schemas.WinConditionSchema{
				{Description: "Identify the poisoner", EvaluationCriteria: "Correct accusation"},
			},
			RequiredSystems: []string{"ai_judge"},
			BriefingText:    "Welcome to the dinner party.\nSomeone has been poisoned.\nFind the culprit.",
		},
		Map: schemas.WorldGenerationMap{
			Rooms: []schemas.RoomSchema{
				{ID: "room-1", Name: "Dining Hall", Description: "The main dining room.", Type: "public", Items: []schemas.ItemSchema{}, NPCIDs: []string{"npc-1"}, ClueIDs: []string{"clue-1", "clue-2"}},
				{ID: "room-2", Name: "Kitchen", Description: "A large kitchen.", Type: "private", Items: []schemas.ItemSchema{}, NPCIDs: []string{}, ClueIDs: []string{"clue-3", "clue-4"}},
				{ID: "room-3", Name: "Study", Description: "A quiet study.", Type: "private", Items: []schemas.ItemSchema{}, NPCIDs: []string{}, ClueIDs: []string{"clue-5", "clue-6"}},
				{ID: "room-4", Name: "Garden", Description: "A dark garden.", Type: "public", Items: []schemas.ItemSchema{}, NPCIDs: []string{}, ClueIDs: []string{"clue-7", "clue-8"}},
				{ID: "room-5", Name: "Wine Cellar", Description: "A cold cellar.", Type: "private", Items: []schemas.ItemSchema{}, NPCIDs: []string{}, ClueIDs: []string{}},
			},
			Connections: []schemas.ConnectionSchema{
				{RoomA: "room-1", RoomB: "room-2", Bidirectional: true},
				{RoomA: "room-1", RoomB: "room-3", Bidirectional: true},
				{RoomA: "room-1", RoomB: "room-4", Bidirectional: true},
				{RoomA: "room-2", RoomB: "room-5", Bidirectional: true},
			},
		},
		Characters: schemas.WorldGenerationCharacters{
			PlayerRoles: []schemas.PlayerRoleSchema{
				{
					ID: "role-1", CharacterName: "Lady Rose", Background: "The host's wife",
					PersonalGoals: []schemas.PersonalGoalSchema{{ID: "goal-1", Description: "Find the poisoner", EvaluationHint: "Correct accusation", EntityRefs: []string{"npc-1"}}},
					Secret: "She knew about the poison", SpecialRole: nil,
					Relationships: []schemas.RelationshipSchema{{TargetCharacterName: "Lord Grey", Description: "Rival"}},
				},
				{
					ID: "role-2", CharacterName: "Lord Grey", Background: "A business partner",
					PersonalGoals: []schemas.PersonalGoalSchema{{ID: "goal-2", Description: "Protect his secret deal", EvaluationHint: "Keep the deal hidden", EntityRefs: []string{"role-1"}}},
					Secret: "He made a secret deal with the host", SpecialRole: nil,
					Relationships: []schemas.RelationshipSchema{{TargetCharacterName: "Lady Rose", Description: "Suspect"}},
				},
				{
					ID: "role-3", CharacterName: "Dr. Black", Background: "The family doctor",
					PersonalGoals: []schemas.PersonalGoalSchema{{ID: "goal-3", Description: "Discover the poison type", EvaluationHint: "Examine the body", EntityRefs: []string{"clue-1"}}},
					Secret: "He prescribed the medication", SpecialRole: nil,
					Relationships: []schemas.RelationshipSchema{{TargetCharacterName: "Lady Rose", Description: "Patient"}},
				},
			},
			NPCs: []schemas.NPCSchema{
				{
					ID: "npc-1", Name: "Butler Jenkins", CurrentRoomID: "room-1",
					Persona: "A loyal but nervous butler", KnownInfo: []string{"The dinner menu", "Guest list"},
					HiddenInfo: []string{"He saw someone enter the kitchen at midnight"},
					BehaviorPrinciple: "Loyal to the household", InitialTrust: 0.5,
				},
			},
		},
		Information: schemas.WorldGenerationInformation{
			Public: schemas.WorldGenerationPublicInfo{
				Title:    "The Last Supper",
				Synopsis: "A dinner party gone wrong.",
				CharacterList: []schemas.CharacterListEntrySchema{
					{Name: "Lady Rose", PublicDescription: "The host's wife"},
					{Name: "Lord Grey", PublicDescription: "A business partner"},
					{Name: "Dr. Black", PublicDescription: "The family doctor"},
				},
				NPCList: []schemas.NPCListEntrySchema{
					{Name: "Butler Jenkins", Location: "Dining Hall"},
				},
				GameRules: "Investigate, accuse, and vote.",
			},
			SemiPublic: []schemas.SemiPublicInfoSchema{
				{ID: "sp-1", TargetPlayerIDs: []string{"role-1", "role-2"}, Content: "Both know about the financial troubles"},
			},
			Private: []schemas.PrivateInfoSchema{
				{PlayerID: "role-1", AdditionalSecrets: []string{"Saw a shadow in the garden"}},
				{PlayerID: "role-2", AdditionalSecrets: []string{"Has the contract in his pocket"}},
				{PlayerID: "role-3", AdditionalSecrets: []string{"The medication was tampered with"}},
			},
		},
		Clues: []schemas.ClueSchema{
			{ID: "clue-1", Name: "Poison Vial", Description: "An empty vial", RoomID: "room-1", DiscoverCondition: "examine the table", RelatedClueIDs: []string{"clue-2"}},
			{ID: "clue-2", Name: "Stained Napkin", Description: "A napkin with stains", RoomID: "room-1", DiscoverCondition: "examine the chairs", RelatedClueIDs: []string{"clue-1"}},
			{ID: "clue-3", Name: "Recipe Book", Description: "A suspicious recipe", RoomID: "room-2", DiscoverCondition: "search the shelves", RelatedClueIDs: []string{}},
			{ID: "clue-4", Name: "Dirty Glass", Description: "A glass with residue", RoomID: "room-2", DiscoverCondition: "examine the sink", RelatedClueIDs: []string{"clue-1"}},
			{ID: "clue-5", Name: "Letter", Description: "A threatening letter", RoomID: "room-3", DiscoverCondition: "search the desk", RelatedClueIDs: []string{}},
			{ID: "clue-6", Name: "Hidden Journal", Description: "A private journal", RoomID: "room-3", DiscoverCondition: "examine the bookshelf", RelatedClueIDs: []string{"clue-5"}},
			{ID: "clue-7", Name: "Footprints", Description: "Fresh footprints in mud", RoomID: "room-4", DiscoverCondition: "examine the ground", RelatedClueIDs: []string{}},
			{ID: "clue-8", Name: "Broken Flower", Description: "A crushed flower bed", RoomID: "room-4", DiscoverCondition: "examine the garden bed", RelatedClueIDs: []string{"clue-7"}},
		},
		Gimmicks: []schemas.GimmickSchema{},
	}

	data, _ := json.Marshal(wg)
	return data
}

// sampleGameContext returns a GameContext suitable for testing runtime AI calls.
func sampleGameContext() types.GameContext {
	return types.GameContext{
		World: types.World{
			Title:      "The Last Supper",
			Synopsis:   "A dinner party gone wrong.",
			Atmosphere: "Tense",
			GameStructure: types.GameStructure{
				CoreConflict:      "Who poisoned the host?",
				EstimatedDuration: 20,
				EndConditions: []types.EndCondition{
					{ID: "ec-1", Description: "Correct accusation", TriggerType: "ai_judgment", IsFallback: false},
					{ID: "ec-2", Description: "Time runs out", TriggerType: "timeout", IsFallback: true},
				},
			},
			PlayerRoles: []types.PlayerRole{
				{ID: "role-1", CharacterName: "Lady Rose", Background: "The host's wife", Secret: "She knew about the poison",
					PersonalGoals: []types.PersonalGoal{{ID: "goal-1", Description: "Find the poisoner"}}},
			},
			NPCs: []types.NPC{
				{ID: "npc-1", Name: "Butler Jenkins", CurrentRoomID: "room-1",
					Persona: "A loyal but nervous butler", KnownInfo: []string{"The dinner menu"},
					HiddenInfo: []string{"He saw someone enter the kitchen at midnight"},
					BehaviorPrinciple: "Loyal to the household", InitialTrust: 0.5},
			},
			Clues: []types.Clue{
				{ID: "clue-1", Name: "Poison Vial", Description: "An empty vial", RoomID: "room-1", DiscoverCondition: "examine the table"},
			},
			Map: types.GameMap{
				Rooms: []types.Room{
					{ID: "room-1", Name: "Dining Hall", Description: "The main dining room.", Type: "public"},
				},
			},
		},
		CurrentState: types.GameState{
			ElapsedTime: 300,
			ClueStates: map[string]types.ClueState{
				"clue-1": {IsDiscovered: false},
			},
			NPCStates: map[string]types.NPCState{
				"npc-1": {
					TrustLevels:         map[string]float64{"player-1": 0.5},
					ConversationHistory: []types.ConversationRecord{},
					GimmickTriggered:    false,
				},
			},
		},
		RequestingPlayer: types.Player{
			ID:       "player-1",
			Nickname: "Alice",
			Role: &types.PlayerRole{
				ID: "role-1", CharacterName: "Lady Rose",
				PersonalGoals: []types.PersonalGoal{{ID: "goal-1", Description: "Find the poisoner"}},
			},
			CurrentRoomID: "room-1",
		},
		CurrentRoom: types.Room{
			ID: "room-1", Name: "Dining Hall", Description: "The main dining room.", Type: "public",
			Items: []types.Item{{ID: "item-1", Name: "Candelabra", Description: "A silver candelabra"}},
		},
		PlayersInRoom: []types.Player{
			{ID: "player-1", Nickname: "Alice"},
		},
	}
}

func TestGenerateWorld(t *testing.T) {
	mock := &mockProvider{structuredResponse: sampleWorldGeneration()}
	layer := NewAILayerWithProvider(mock)

	settings := types.GameSettings{
		MaxPlayers:     3,
		TimeoutMinutes: 20,
		HasGM:          true,
		HasNPC:         true,
	}

	world, err := layer.GenerateWorld(context.Background(), settings, 3, "mystery")
	if err != nil {
		t.Fatalf("GenerateWorld failed: %v", err)
	}

	if world.Title != "The Last Supper" {
		t.Errorf("expected title 'The Last Supper', got '%s'", world.Title)
	}
	if len(world.PlayerRoles) != 3 {
		t.Errorf("expected 3 player roles, got %d", len(world.PlayerRoles))
	}
	if len(world.Map.Rooms) != 5 {
		t.Errorf("expected 5 rooms, got %d", len(world.Map.Rooms))
	}
	if len(world.NPCs) != 1 {
		t.Errorf("expected 1 NPC, got %d", len(world.NPCs))
	}
	if len(world.Clues) != 8 {
		t.Errorf("expected 8 clues, got %d", len(world.Clues))
	}
	if world.GameStructure.EstimatedDuration != 20 {
		t.Errorf("expected duration 20, got %d", world.GameStructure.EstimatedDuration)
	}
	if len(world.Information.SemiPublic) != 1 {
		t.Errorf("expected 1 semi-public info, got %d", len(world.Information.SemiPublic))
	}
}

func TestEvaluateExamine(t *testing.T) {
	response := schemas.GameResponse{
		Events:       []json.RawMessage{json.RawMessage(`{"type":"examine_result","data":{"playerId":"player-1","playerName":"Alice","target":"table","description":"You examine the table carefully.","clueFound":false}}`)},
		StateChanges: []json.RawMessage{},
	}
	respJSON, _ := json.Marshal(response)
	mock := &mockProvider{structuredResponse: respJSON}
	layer := NewAILayerWithProvider(mock)

	gameCtx := sampleGameContext()
	result, err := layer.EvaluateExamine(context.Background(), gameCtx, "table")
	if err != nil {
		t.Fatalf("EvaluateExamine failed: %v", err)
	}
	if len(result.Events) != 1 {
		t.Errorf("expected 1 event, got %d", len(result.Events))
	}
}

func TestEvaluateAction(t *testing.T) {
	response := schemas.GameResponse{
		Events:       []json.RawMessage{json.RawMessage(`{"type":"action_result","data":{"playerId":"player-1","playerName":"Alice","action":"open the drawer","result":"The drawer creaks open.","triggeredEvents":[]}}`)},
		StateChanges: []json.RawMessage{},
	}
	respJSON, _ := json.Marshal(response)
	mock := &mockProvider{structuredResponse: respJSON}
	layer := NewAILayerWithProvider(mock)

	gameCtx := sampleGameContext()
	result, err := layer.EvaluateAction(context.Background(), gameCtx, "open the drawer")
	if err != nil {
		t.Fatalf("EvaluateAction failed: %v", err)
	}
	if len(result.Events) != 1 {
		t.Errorf("expected 1 event, got %d", len(result.Events))
	}
}

func TestTalkToNPC(t *testing.T) {
	npcResp := schemas.NPCResponse{
		Dialogue:        "Good evening, my lady.",
		Emotion:         "nervous",
		InternalThought: "I must be careful.",
		InfoRevealed:    []string{},
		TrustChange:     0.1,
	}
	respJSON, _ := json.Marshal(npcResp)
	mock := &mockProvider{structuredResponse: respJSON}
	layer := NewAILayerWithProvider(mock)

	gameCtx := sampleGameContext()
	result, err := layer.TalkToNPC(context.Background(), gameCtx, "npc-1", "What happened last night?")
	if err != nil {
		t.Fatalf("TalkToNPC failed: %v", err)
	}
	if result.Dialogue != "Good evening, my lady." {
		t.Errorf("unexpected dialogue: %s", result.Dialogue)
	}
	if result.TrustChange != 0.1 {
		t.Errorf("expected trust change 0.1, got %f", result.TrustChange)
	}
}

func TestTalkToNPC_NotFound(t *testing.T) {
	mock := &mockProvider{structuredResponse: json.RawMessage(`{}`)}
	layer := NewAILayerWithProvider(mock)

	gameCtx := sampleGameContext()
	_, err := layer.TalkToNPC(context.Background(), gameCtx, "nonexistent", "Hello")
	if err == nil {
		t.Fatal("expected error for nonexistent NPC")
	}
}

func TestJudgeEndCondition(t *testing.T) {
	judgment := json.RawMessage(`{"shouldEnd": true, "reason": "The accusation was correct."}`)
	mock := &mockProvider{structuredResponse: judgment}
	layer := NewAILayerWithProvider(mock)

	gameCtx := sampleGameContext()
	condition := types.EndCondition{
		ID:          "ec-1",
		Description: "Correct accusation",
		TriggerType: "ai_judgment",
	}
	shouldEnd, err := layer.JudgeEndCondition(context.Background(), gameCtx, condition)
	if err != nil {
		t.Fatalf("JudgeEndCondition failed: %v", err)
	}
	if !shouldEnd {
		t.Error("expected shouldEnd to be true")
	}
}

func TestJudgeEndCondition_NotMet(t *testing.T) {
	judgment := json.RawMessage(`{"shouldEnd": false, "reason": "Insufficient evidence."}`)
	mock := &mockProvider{structuredResponse: judgment}
	layer := NewAILayerWithProvider(mock)

	gameCtx := sampleGameContext()
	condition := types.EndCondition{
		ID:          "ec-1",
		Description: "Correct accusation",
		TriggerType: "ai_judgment",
	}
	shouldEnd, err := layer.JudgeEndCondition(context.Background(), gameCtx, condition)
	if err != nil {
		t.Fatalf("JudgeEndCondition failed: %v", err)
	}
	if shouldEnd {
		t.Error("expected shouldEnd to be false")
	}
}

func TestGenerateEndings(t *testing.T) {
	ending := schemas.Ending{
		CommonResult: "The truth was finally revealed.",
		PlayerEndings: []schemas.PlayerEndingSchema{
			{
				PlayerID: "role-1",
				Summary:  "Lady Rose uncovered the truth.",
				GoalResults: []schemas.GoalResultSchema{
					{GoalID: "goal-1", Description: "Find the poisoner", Achieved: true, Evaluation: "She correctly identified the culprit."},
				},
				Narrative: "Lady Rose stood victorious.",
			},
		},
	}
	respJSON, _ := json.Marshal(ending)
	mock := &mockProvider{structuredResponse: respJSON}
	layer := NewAILayerWithProvider(mock)

	gameCtx := sampleGameContext()
	result, err := layer.GenerateEndings(context.Background(), gameCtx, "condition_met")
	if err != nil {
		t.Fatalf("GenerateEndings failed: %v", err)
	}
	if result.CommonResult != "The truth was finally revealed." {
		t.Errorf("unexpected common result: %s", result.CommonResult)
	}
	if len(result.PlayerEndings) != 1 {
		t.Errorf("expected 1 player ending, got %d", len(result.PlayerEndings))
	}
}

func TestGenerateNarration(t *testing.T) {
	resp := json.RawMessage(`{"text": "A cold wind sweeps through the hall."}`)
	mock := &mockProvider{structuredResponse: resp}
	layer := NewAILayerWithProvider(mock)

	gameCtx := sampleGameContext()
	narration, err := layer.GenerateNarration(context.Background(), gameCtx, "game_start")
	if err != nil {
		t.Fatalf("GenerateNarration failed: %v", err)
	}
	if narration != "A cold wind sweeps through the hall." {
		t.Errorf("unexpected narration: %s", narration)
	}
}

func TestContainsHiddenKeywords(t *testing.T) {
	tests := []struct {
		name       string
		dialogue   string
		hiddenInfo string
		expected   bool
	}{
		{
			name:       "no leak",
			dialogue:   "Good evening, my lady.",
			hiddenInfo: "He saw someone enter the kitchen at midnight",
			expected:   false,
		},
		{
			name:       "leak detected - keyword match",
			dialogue:   "Someone entered the kitchen late last night.",
			hiddenInfo: "He saw someone enter the kitchen at midnight",
			expected:   true,
		},
		{
			name:       "short words ignored",
			dialogue:   "He saw it at the door.",
			hiddenInfo: "He saw it",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsHiddenKeywords(tt.dialogue, tt.hiddenInfo)
			if result != tt.expected {
				t.Errorf("containsHiddenKeywords(%q, %q) = %v, want %v",
					tt.dialogue, tt.hiddenInfo, result, tt.expected)
			}
		})
	}
}

func TestTransformToWorld(t *testing.T) {
	var gen schemas.WorldGeneration
	if err := json.Unmarshal(sampleWorldGeneration(), &gen); err != nil {
		t.Fatalf("failed to unmarshal sample: %v", err)
	}

	world := transformToWorld(&gen)

	if world.Title != "The Last Supper" {
		t.Errorf("expected title 'The Last Supper', got '%s'", world.Title)
	}
	if world.GameStructure.EstimatedDuration != 20 {
		t.Errorf("expected duration from meta (20), got %d", world.GameStructure.EstimatedDuration)
	}
	if len(world.GameStructure.EndConditions) != 2 {
		t.Errorf("expected 2 end conditions, got %d", len(world.GameStructure.EndConditions))
	}
	if len(world.Map.Connections) != 4 {
		t.Errorf("expected 4 connections, got %d", len(world.Map.Connections))
	}
	if world.NPCs[0].InitialTrust != 0.5 {
		t.Errorf("expected NPC initial trust 0.5, got %f", world.NPCs[0].InitialTrust)
	}
}
