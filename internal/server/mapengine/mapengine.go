package mapengine

import (
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/story/internal/shared/types"
)

var (
	ErrNotAdjacent = errors.New("not_adjacent")
	ErrRoomNotFound = errors.New("room_not_found")
	ErrSameRoom     = errors.New("same_room")
)

// MoveResult represents the result of a player movement between rooms.
type MoveResult struct {
	FromRoom *types.Room
	ToRoom   *types.Room
}

// MapEngine manages the game map graph, providing adjacency queries
// and room lookups.
type MapEngine struct {
	adjacencyMap map[string]map[string]struct{}
	rooms        map[string]*types.Room
	roomsList    []*types.Room
}

// NewMapEngine creates a new empty MapEngine.
func NewMapEngine() *MapEngine {
	return &MapEngine{
		adjacencyMap: make(map[string]map[string]struct{}),
		rooms:        make(map[string]*types.Room),
	}
}

// Initialize populates the map engine from a GameMap definition.
func (me *MapEngine) Initialize(gameMap types.GameMap) {
	me.adjacencyMap = make(map[string]map[string]struct{})
	me.rooms = make(map[string]*types.Room)
	me.roomsList = nil

	for i := range gameMap.Rooms {
		room := &gameMap.Rooms[i]
		me.rooms[room.ID] = room
		me.roomsList = append(me.roomsList, room)
		if me.adjacencyMap[room.ID] == nil {
			me.adjacencyMap[room.ID] = make(map[string]struct{})
		}
	}

	for _, conn := range gameMap.Connections {
		if me.adjacencyMap[conn.RoomA] == nil {
			me.adjacencyMap[conn.RoomA] = make(map[string]struct{})
		}
		if me.adjacencyMap[conn.RoomB] == nil {
			me.adjacencyMap[conn.RoomB] = make(map[string]struct{})
		}

		me.adjacencyMap[conn.RoomA][conn.RoomB] = struct{}{}
		if conn.Bidirectional {
			me.adjacencyMap[conn.RoomB][conn.RoomA] = struct{}{}
		}
	}
}

// IsAdjacent returns true if roomA and roomB are directly connected.
func (me *MapEngine) IsAdjacent(roomA, roomB string) bool {
	neighbors, ok := me.adjacencyMap[roomA]
	if !ok {
		return false
	}
	_, adjacent := neighbors[roomB]
	return adjacent
}

// GetAdjacentRooms returns all rooms directly connected to the given room ID.
func (me *MapEngine) GetAdjacentRooms(roomId string) []*types.Room {
	neighbors, ok := me.adjacencyMap[roomId]
	if !ok {
		return nil
	}

	var result []*types.Room
	for neighborID := range neighbors {
		if room, exists := me.rooms[neighborID]; exists {
			result = append(result, room)
		}
	}
	return result
}

// GetRoomByName returns the first room matching the given name (case-insensitive).
func (me *MapEngine) GetRoomByName(name string) *types.Room {
	lower := strings.ToLower(name)
	for _, room := range me.rooms {
		if strings.ToLower(room.Name) == lower {
			return room
		}
	}
	return nil
}

// GetRoomByID returns the room with the given ID.
func (me *MapEngine) GetRoomByID(id string) *types.Room {
	return me.rooms[id]
}

// GetNPCByName returns the first NPC matching the given name (case-insensitive).
func (me *MapEngine) GetNPCByName(name string, npcs []types.NPC) *types.NPC {
	lower := strings.ToLower(name)
	for i := range npcs {
		if strings.ToLower(npcs[i].Name) == lower {
			return &npcs[i]
		}
	}
	return nil
}

// ValidateConnectivity checks that all rooms are reachable from the first room
// using BFS. Returns an error listing unreachable rooms.
func (me *MapEngine) ValidateConnectivity() error {
	if len(me.roomsList) == 0 {
		return nil
	}

	visited := make(map[string]bool)
	queue := []string{me.roomsList[0].ID}
	visited[me.roomsList[0].ID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for neighborID := range me.adjacencyMap[current] {
			if !visited[neighborID] {
				visited[neighborID] = true
				queue = append(queue, neighborID)
			}
		}
	}

	var unreachable []string
	for _, room := range me.roomsList {
		if !visited[room.ID] {
			unreachable = append(unreachable, room.Name)
		}
	}

	if len(unreachable) > 0 {
		return fmt.Errorf("unreachable rooms: %s", strings.Join(unreachable, ", "))
	}
	return nil
}

// ValidateRoomCount checks that there are enough rooms for the given player count.
// Requires at least playerCount + 1 rooms (one common area plus one per player).
func (me *MapEngine) ValidateRoomCount(playerCount int) error {
	required := playerCount + 1
	if len(me.roomsList) < required {
		return fmt.Errorf("need at least %d rooms for %d players, got %d", required, playerCount, len(me.roomsList))
	}
	return nil
}

// GetAllRooms returns all rooms in the map.
func (me *MapEngine) GetAllRooms() []*types.Room {
	result := make([]*types.Room, len(me.roomsList))
	copy(result, me.roomsList)
	return result
}
