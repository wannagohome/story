package game

import (
	"testing"

	"github.com/anthropics/story/internal/server/eventbus"
	"github.com/anthropics/story/internal/server/mapengine"
	"github.com/anthropics/story/internal/shared/types"
)

func makeTestWorld() types.World {
	return types.World{
		Title:    "Test",
		Synopsis: "Test world",
		Map: types.GameMap{
			Rooms: []types.Room{
				{ID: "r1", Name: "Lobby", Type: "public", Items: []types.Item{{ID: "item1", Name: "Sword"}}},
				{ID: "r2", Name: "Library", Type: "public"},
				{ID: "r3", Name: "Kitchen", Type: "public"},
			},
			Connections: []types.Connection{
				{RoomA: "r1", RoomB: "r2", Bidirectional: true},
				{RoomA: "r2", RoomB: "r3", Bidirectional: true},
			},
		},
		NPCs: []types.NPC{
			{ID: "npc1", Name: "Guard", CurrentRoomID: "r1", InitialTrust: 0.5},
		},
		Clues: []types.Clue{
			{ID: "clue1", Name: "Note", RoomID: "r1"},
		},
		Gimmicks: []types.Gimmick{
			{ID: "gim1", RoomID: "r2"},
		},
		Information: types.InformationLayers{
			SemiPublic: []types.SemiPublicInfo{
				{ID: "sp1", TargetPlayerIDs: []string{"p1"}, Content: "secret info"},
			},
		},
	}
}

func makeTestGSM() (*GameStateManager, *eventbus.EventBus) {
	bus := eventbus.NewEventBus()
	me := mapengine.NewMapEngine()
	gsm := NewGameStateManager(bus, me)

	// Add players
	gsm.AddPlayer(&types.Player{ID: "p1", Nickname: "Alice", Status: "connected"})
	gsm.AddPlayer(&types.Player{ID: "p2", Nickname: "Bob", Status: "connected"})

	// Initialize world
	roles := map[string]types.PlayerRole{
		"p1": {ID: "role1", CharacterName: "Detective"},
		"p2": {ID: "role2", CharacterName: "Butler"},
	}
	gsm.InitializeWorld(makeTestWorld(), roles)

	return gsm, bus
}

func TestGameStateManager_InitializeWorld(t *testing.T) {
	gsm, _ := makeTestGSM()

	// Check players have roles
	p1 := gsm.GetPlayer("p1")
	if p1.Role == nil {
		t.Fatal("p1 should have a role")
	}
	if p1.Role.CharacterName != "Detective" {
		t.Errorf("expected Detective, got %s", p1.Role.CharacterName)
	}

	// Check players placed in first room
	if p1.CurrentRoomID != "r1" {
		t.Errorf("expected r1, got %s", p1.CurrentRoomID)
	}

	// Check clue states initialized
	state := gsm.GetFullState()
	if len(state.ClueStates) != 1 {
		t.Errorf("expected 1 clue state, got %d", len(state.ClueStates))
	}

	// Check NPC states initialized
	if len(state.NPCStates) != 1 {
		t.Errorf("expected 1 NPC state, got %d", len(state.NPCStates))
	}

	// Check NPC trust initialized for players
	npcState := state.NPCStates["npc1"]
	if npcState.TrustLevels["p1"] != 0.5 {
		t.Errorf("expected trust 0.5, got %f", npcState.TrustLevels["p1"])
	}

	// Check gimmick states initialized
	if len(state.GimmickStates) != 1 {
		t.Errorf("expected 1 gimmick state, got %d", len(state.GimmickStates))
	}
}

func TestGameStateManager_MovePlayer(t *testing.T) {
	gsm, _ := makeTestGSM()

	gsm.MovePlayer("p1", "r2")
	p1 := gsm.GetPlayer("p1")
	if p1.CurrentRoomID != "r2" {
		t.Errorf("expected r2, got %s", p1.CurrentRoomID)
	}
	if len(p1.MoveHistory) != 1 {
		t.Errorf("expected 1 move record, got %d", len(p1.MoveHistory))
	}
}

func TestGameStateManager_AddRemoveItem(t *testing.T) {
	gsm, _ := makeTestGSM()

	gsm.AddItemToPlayer("p1", types.Item{ID: "newitem", Name: "Key"})
	p1 := gsm.GetPlayer("p1")
	if len(p1.Inventory) != 1 {
		t.Fatalf("expected 1 item, got %d", len(p1.Inventory))
	}
	if p1.Inventory[0].Name != "Key" {
		t.Errorf("expected Key, got %s", p1.Inventory[0].Name)
	}

	gsm.RemoveItemFromPlayer("p1", "newitem")
	p1 = gsm.GetPlayer("p1")
	if len(p1.Inventory) != 0 {
		t.Errorf("expected 0 items, got %d", len(p1.Inventory))
	}
}

func TestGameStateManager_DiscoverClue(t *testing.T) {
	gsm, _ := makeTestGSM()

	gsm.DiscoverClue("p1", "clue1")
	state := gsm.GetFullState()
	cs := state.ClueStates["clue1"]
	if !cs.IsDiscovered {
		t.Error("clue should be discovered")
	}
	if len(cs.DiscoveredBy) != 1 || cs.DiscoveredBy[0] != "p1" {
		t.Errorf("unexpected discoveredBy: %v", cs.DiscoveredBy)
	}

	p1 := gsm.GetPlayer("p1")
	if len(p1.DiscoveredClueIDs) != 1 {
		t.Errorf("expected 1 discovered clue, got %d", len(p1.DiscoveredClueIDs))
	}
}

func TestGameStateManager_UpdateNPCTrust(t *testing.T) {
	gsm, _ := makeTestGSM()

	gsm.UpdateNPCTrust("npc1", "p1", 0.3)
	state := gsm.GetFullState()
	trust := state.NPCStates["npc1"].TrustLevels["p1"]
	if trust != 0.8 {
		t.Errorf("expected 0.8, got %f", trust)
	}

	// Test clamping at 1
	gsm.UpdateNPCTrust("npc1", "p1", 0.5)
	state = gsm.GetFullState()
	trust = state.NPCStates["npc1"].TrustLevels["p1"]
	if trust != 1.0 {
		t.Errorf("expected 1.0 (clamped), got %f", trust)
	}
}

func TestGameStateManager_TriggerGimmick(t *testing.T) {
	gsm, _ := makeTestGSM()

	gsm.TriggerGimmick("gim1")
	state := gsm.GetFullState()
	gs := state.GimmickStates["gim1"]
	if !gs.IsTriggered {
		t.Error("gimmick should be triggered")
	}
	if gs.TriggeredAt == nil {
		t.Error("triggeredAt should be set")
	}
}

func TestGameStateManager_AddConversation(t *testing.T) {
	gsm, _ := makeTestGSM()

	gsm.AddConversation("npc1", types.ConversationRecord{
		PlayerID: "p1", Message: "Hello", Response: "Hi", Timestamp: 1000,
	})
	state := gsm.GetFullState()
	history := state.NPCStates["npc1"].ConversationHistory
	if len(history) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(history))
	}

	// Test 20-entry limit
	for i := 0; i < 25; i++ {
		gsm.AddConversation("npc1", types.ConversationRecord{
			PlayerID: "p1", Message: "msg", Response: "resp",
		})
	}
	state = gsm.GetFullState()
	history = state.NPCStates["npc1"].ConversationHistory
	if len(history) != 20 {
		t.Errorf("expected 20 conversations (limit), got %d", len(history))
	}
}

func TestGameStateManager_GoalProgress(t *testing.T) {
	gsm, _ := makeTestGSM()

	gsm.RecordGoalProgress("p1", "goal1", "found clue")
	gsm.RecordGoalProgress("p1", "goal1", "talked to NPC")
	progress := gsm.GetGoalProgress("p1")
	if len(progress) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(progress))
	}
	if len(progress[0].Evidence) != 2 {
		t.Errorf("expected 2 evidence, got %d", len(progress[0].Evidence))
	}
}

func TestGameStateManager_GetPlayersInRoom(t *testing.T) {
	gsm, _ := makeTestGSM()

	players := gsm.GetPlayersInRoom("r1")
	if len(players) != 2 {
		t.Errorf("expected 2 players in r1, got %d", len(players))
	}

	gsm.MovePlayer("p1", "r2")
	players = gsm.GetPlayersInRoom("r1")
	if len(players) != 1 {
		t.Errorf("expected 1 player in r1 after move, got %d", len(players))
	}
}

func TestGameStateManager_GetRoomView(t *testing.T) {
	gsm, _ := makeTestGSM()

	rv := gsm.GetRoomView("p1")
	if rv.ID != "r1" {
		t.Errorf("expected r1, got %s", rv.ID)
	}
	if rv.Name != "Lobby" {
		t.Errorf("expected Lobby, got %s", rv.Name)
	}
	if len(rv.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(rv.Items))
	}
	if len(rv.NPCs) != 1 {
		t.Errorf("expected 1 NPC, got %d", len(rv.NPCs))
	}
}

func TestGameStateManager_GetMapView(t *testing.T) {
	gsm, _ := makeTestGSM()

	mv := gsm.GetMapView("p1")
	if mv.MyRoomID != "r1" {
		t.Errorf("expected r1, got %s", mv.MyRoomID)
	}
	if len(mv.Rooms) != 3 {
		t.Errorf("expected 3 rooms, got %d", len(mv.Rooms))
	}
}

func TestGameStateManager_GetSemiPublicInfo(t *testing.T) {
	gsm, _ := makeTestGSM()

	info := gsm.GetSemiPublicInfoForPlayer("p1")
	if len(info) != 1 {
		t.Fatalf("expected 1 semi-public info for p1, got %d", len(info))
	}

	info = gsm.GetSemiPublicInfoForPlayer("p2")
	if len(info) != 0 {
		t.Errorf("expected 0 semi-public info for p2, got %d", len(info))
	}
}

func TestGameStateManager_GetNPCsInRoom(t *testing.T) {
	gsm, _ := makeTestGSM()

	npcs := gsm.GetNPCsInRoom("r1")
	if len(npcs) != 1 {
		t.Errorf("expected 1 NPC in r1, got %d", len(npcs))
	}

	npcs = gsm.GetNPCsInRoom("r2")
	if len(npcs) != 0 {
		t.Errorf("expected 0 NPCs in r2, got %d", len(npcs))
	}
}

func TestGameStateManager_RemovePlayer(t *testing.T) {
	gsm, _ := makeTestGSM()

	gsm.RemovePlayer("p1")
	p1 := gsm.GetPlayer("p1")
	if p1.Status != "disconnected" {
		t.Errorf("expected disconnected, got %s", p1.Status)
	}

	// Disconnected player should not appear in GetAllPlayerIDs
	ids := gsm.GetAllPlayerIDs()
	for _, id := range ids {
		if id == "p1" {
			t.Error("disconnected player should not be in GetAllPlayerIDs")
		}
	}
}
