package mapengine

import (
	"testing"

	"github.com/anthropics/story/internal/shared/types"
)

func newTestMap() types.GameMap {
	return types.GameMap{
		Rooms: []types.Room{
			{ID: "room-1", Name: "Grand Hall", Description: "A large hall", Type: "public"},
			{ID: "room-2", Name: "Library", Description: "Shelves of books", Type: "public"},
			{ID: "room-3", Name: "Kitchen", Description: "A busy kitchen", Type: "private"},
			{ID: "room-4", Name: "Garden", Description: "A serene garden", Type: "public"},
		},
		Connections: []types.Connection{
			{RoomA: "room-1", RoomB: "room-2", Bidirectional: true},
			{RoomA: "room-1", RoomB: "room-3", Bidirectional: true},
			{RoomA: "room-2", RoomB: "room-4", Bidirectional: true},
			{RoomA: "room-3", RoomB: "room-4", Bidirectional: true},
		},
	}
}

func TestInitializeAndGetAllRooms(t *testing.T) {
	me := NewMapEngine()
	me.Initialize(newTestMap())

	rooms := me.GetAllRooms()
	if len(rooms) != 4 {
		t.Fatalf("expected 4 rooms, got %d", len(rooms))
	}
}

func TestIsAdjacent(t *testing.T) {
	me := NewMapEngine()
	me.Initialize(newTestMap())

	if !me.IsAdjacent("room-1", "room-2") {
		t.Error("room-1 should be adjacent to room-2")
	}
	if !me.IsAdjacent("room-2", "room-1") {
		t.Error("room-2 should be adjacent to room-1 (bidirectional)")
	}
	if me.IsAdjacent("room-1", "room-4") {
		t.Error("room-1 should not be adjacent to room-4")
	}
}

func TestIsAdjacentUnidirectional(t *testing.T) {
	gameMap := types.GameMap{
		Rooms: []types.Room{
			{ID: "a", Name: "A"},
			{ID: "b", Name: "B"},
		},
		Connections: []types.Connection{
			{RoomA: "a", RoomB: "b", Bidirectional: false},
		},
	}
	me := NewMapEngine()
	me.Initialize(gameMap)

	if !me.IsAdjacent("a", "b") {
		t.Error("a should be adjacent to b")
	}
	if me.IsAdjacent("b", "a") {
		t.Error("b should not be adjacent to a (unidirectional)")
	}
}

func TestGetAdjacentRooms(t *testing.T) {
	me := NewMapEngine()
	me.Initialize(newTestMap())

	adj := me.GetAdjacentRooms("room-1")
	if len(adj) != 2 {
		t.Fatalf("expected 2 adjacent rooms for room-1, got %d", len(adj))
	}

	ids := make(map[string]bool)
	for _, r := range adj {
		ids[r.ID] = true
	}
	if !ids["room-2"] || !ids["room-3"] {
		t.Errorf("expected room-2 and room-3, got %v", ids)
	}
}

func TestGetAdjacentRoomsNonExistent(t *testing.T) {
	me := NewMapEngine()
	me.Initialize(newTestMap())

	adj := me.GetAdjacentRooms("nonexistent")
	if adj != nil {
		t.Fatalf("expected nil for nonexistent room, got %v", adj)
	}
}

func TestGetRoomByName(t *testing.T) {
	me := NewMapEngine()
	me.Initialize(newTestMap())

	room := me.GetRoomByName("Library")
	if room == nil || room.ID != "room-2" {
		t.Fatalf("expected room-2, got %v", room)
	}

	// Case insensitive
	room = me.GetRoomByName("library")
	if room == nil || room.ID != "room-2" {
		t.Fatalf("expected room-2 (case insensitive), got %v", room)
	}

	// Not found
	room = me.GetRoomByName("Dungeon")
	if room != nil {
		t.Fatalf("expected nil for nonexistent room, got %v", room)
	}
}

func TestGetRoomByID(t *testing.T) {
	me := NewMapEngine()
	me.Initialize(newTestMap())

	room := me.GetRoomByID("room-3")
	if room == nil || room.Name != "Kitchen" {
		t.Fatalf("expected Kitchen, got %v", room)
	}

	room = me.GetRoomByID("nonexistent")
	if room != nil {
		t.Fatalf("expected nil, got %v", room)
	}
}

func TestGetNPCByName(t *testing.T) {
	me := NewMapEngine()

	npcs := []types.NPC{
		{ID: "npc-1", Name: "Innkeeper", Persona: "friendly"},
		{ID: "npc-2", Name: "Guard", Persona: "stern"},
	}

	npc := me.GetNPCByName("innkeeper", npcs)
	if npc == nil || npc.ID != "npc-1" {
		t.Fatalf("expected npc-1, got %v", npc)
	}

	npc = me.GetNPCByName("Ghost", npcs)
	if npc != nil {
		t.Fatalf("expected nil, got %v", npc)
	}
}

func TestValidateConnectivity(t *testing.T) {
	me := NewMapEngine()
	me.Initialize(newTestMap())

	if err := me.ValidateConnectivity(); err != nil {
		t.Fatalf("expected connectivity to pass, got: %v", err)
	}
}

func TestValidateConnectivityDisconnected(t *testing.T) {
	gameMap := types.GameMap{
		Rooms: []types.Room{
			{ID: "a", Name: "Room A"},
			{ID: "b", Name: "Room B"},
			{ID: "c", Name: "Room C"},
		},
		Connections: []types.Connection{
			{RoomA: "a", RoomB: "b", Bidirectional: true},
			// Room C is disconnected
		},
	}

	me := NewMapEngine()
	me.Initialize(gameMap)

	err := me.ValidateConnectivity()
	if err == nil {
		t.Fatal("expected connectivity validation to fail")
	}
	if err.Error() != "unreachable rooms: Room C" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestValidateConnectivityEmpty(t *testing.T) {
	me := NewMapEngine()
	me.Initialize(types.GameMap{})

	if err := me.ValidateConnectivity(); err != nil {
		t.Fatalf("expected empty map to pass connectivity, got: %v", err)
	}
}

func TestValidateRoomCount(t *testing.T) {
	me := NewMapEngine()
	me.Initialize(newTestMap())

	// 4 rooms, 3 players -> need 4, have 4 -> OK
	if err := me.ValidateRoomCount(3); err != nil {
		t.Fatalf("expected room count to pass, got: %v", err)
	}

	// 4 rooms, 4 players -> need 5, have 4 -> fail
	if err := me.ValidateRoomCount(4); err == nil {
		t.Fatal("expected room count validation to fail")
	}
}
