package game

import (
	"sync"
	"time"

	"github.com/anthropics/story/internal/server/eventbus"
	"github.com/anthropics/story/internal/server/mapengine"
	"github.com/anthropics/story/internal/shared/protocol"
	"github.com/anthropics/story/internal/shared/types"
)

type GameStateManager struct {
	state      types.GameState
	world      types.World
	eventLog   []types.GameEvent
	mapEngine  *mapengine.MapEngine
	eventBus   *eventbus.EventBus
	goalProgress map[string][]types.GoalProgressEntry // playerID -> entries
	mu         sync.RWMutex
}

func NewGameStateManager(bus *eventbus.EventBus, me *mapengine.MapEngine) *GameStateManager {
	return &GameStateManager{
		eventBus:     bus,
		mapEngine:    me,
		goalProgress: make(map[string][]types.GoalProgressEntry),
		state: types.GameState{
			Players:       make(map[string]*types.Player),
			ClueStates:    make(map[string]types.ClueState),
			NPCStates:     make(map[string]types.NPCState),
			GimmickStates: make(map[string]types.GimmickState),
		},
	}
}

func (gsm *GameStateManager) InitializeWorld(world types.World, roleAssignments map[string]types.PlayerRole) {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	gsm.world = world
	gsm.mapEngine.Initialize(world.Map)

	// Initialize clue states
	for _, clue := range world.Clues {
		gsm.state.ClueStates[clue.ID] = types.ClueState{
			IsDiscovered: false,
			DiscoveredBy: []string{},
		}
	}

	// Initialize NPC states
	for _, npc := range world.NPCs {
		gsm.state.NPCStates[npc.ID] = types.NPCState{
			TrustLevels:         make(map[string]float64),
			ConversationHistory: []types.ConversationRecord{},
			GimmickTriggered:    false,
		}
	}

	// Initialize gimmick states
	for _, gimmick := range world.Gimmicks {
		gsm.state.GimmickStates[gimmick.ID] = types.GimmickState{
			IsTriggered: false,
		}
	}

	// Assign roles and place players in initial rooms
	for playerID, role := range roleAssignments {
		if p, ok := gsm.state.Players[playerID]; ok {
			roleCopy := role
			p.Role = &roleCopy
			// Place in first room by default
			if len(world.Map.Rooms) > 0 {
				p.CurrentRoomID = world.Map.Rooms[0].ID
			}
			// Initialize NPC trust for this player
			for _, npc := range world.NPCs {
				npcState := gsm.state.NPCStates[npc.ID]
				npcState.TrustLevels[playerID] = npc.InitialTrust
				gsm.state.NPCStates[npc.ID] = npcState
			}
		}
	}
}

func (gsm *GameStateManager) AddPlayer(player *types.Player) {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()
	gsm.state.Players[player.ID] = player
}

func (gsm *GameStateManager) RemovePlayer(playerID string) {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()
	if p, ok := gsm.state.Players[playerID]; ok {
		p.Status = "disconnected"
	}
}

func (gsm *GameStateManager) MovePlayer(playerID string, targetRoomID string) {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	player, ok := gsm.state.Players[playerID]
	if !ok {
		return
	}

	oldRoomID := player.CurrentRoomID
	now := time.Now().UnixMilli()

	// Update move history
	if len(player.MoveHistory) > 0 {
		last := &player.MoveHistory[len(player.MoveHistory)-1]
		last.LeftAt = &now
	}

	player.CurrentRoomID = targetRoomID
	room := gsm.mapEngine.GetRoomByID(targetRoomID)
	roomName := ""
	if room != nil {
		roomName = room.Name
	}
	player.MoveHistory = append(player.MoveHistory, types.MoveRecord{
		RoomID:    targetRoomID,
		RoomName:  roomName,
		EnteredAt: now,
	})

	gsm.eventBus.PublishStateChanged(eventbus.StateChangedData{
		ChangeType: "player_moved",
		Data: map[string]string{
			"playerID":     playerID,
			"fromRoomID":   oldRoomID,
			"toRoomID":     targetRoomID,
		},
	})
}

func (gsm *GameStateManager) AddItemToPlayer(playerID string, item types.Item) {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	if player, ok := gsm.state.Players[playerID]; ok {
		ownerID := playerID
		item.OwnerID = &ownerID
		player.Inventory = append(player.Inventory, item)
	}
}

func (gsm *GameStateManager) RemoveItemFromPlayer(playerID string, itemID string) {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	if player, ok := gsm.state.Players[playerID]; ok {
		for i, item := range player.Inventory {
			if item.ID == itemID {
				player.Inventory = append(player.Inventory[:i], player.Inventory[i+1:]...)
				break
			}
		}
	}
}

func (gsm *GameStateManager) DiscoverClue(playerID string, clueID string) {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	state, ok := gsm.state.ClueStates[clueID]
	if !ok {
		return
	}
	state.IsDiscovered = true
	state.DiscoveredBy = append(state.DiscoveredBy, playerID)
	gsm.state.ClueStates[clueID] = state

	if player, ok := gsm.state.Players[playerID]; ok {
		player.DiscoveredClueIDs = append(player.DiscoveredClueIDs, clueID)
	}
}

func (gsm *GameStateManager) UpdateNPCTrust(npcID string, playerID string, delta float64) {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	state, ok := gsm.state.NPCStates[npcID]
	if !ok {
		return
	}
	current := state.TrustLevels[playerID]
	newTrust := current + delta
	if newTrust > 1 {
		newTrust = 1
	}
	if newTrust < -1 {
		newTrust = -1
	}
	state.TrustLevels[playerID] = newTrust
	gsm.state.NPCStates[npcID] = state
}

func (gsm *GameStateManager) TriggerGimmick(gimmickID string) {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	state, ok := gsm.state.GimmickStates[gimmickID]
	if !ok {
		return
	}
	now := time.Now().UnixMilli()
	state.IsTriggered = true
	state.TriggeredAt = &now
	gsm.state.GimmickStates[gimmickID] = state
}

func (gsm *GameStateManager) AddConversation(npcID string, record types.ConversationRecord) {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	state, ok := gsm.state.NPCStates[npcID]
	if !ok {
		return
	}
	state.ConversationHistory = append(state.ConversationHistory, record)
	// Keep only last 20 entries
	if len(state.ConversationHistory) > 20 {
		state.ConversationHistory = state.ConversationHistory[len(state.ConversationHistory)-20:]
	}
	gsm.state.NPCStates[npcID] = state
}

func (gsm *GameStateManager) RecordGoalProgress(playerID string, goalID string, evidence string) {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	entries := gsm.goalProgress[playerID]
	for i, e := range entries {
		if e.GoalID == goalID {
			entries[i].Evidence = append(entries[i].Evidence, evidence)
			gsm.goalProgress[playerID] = entries
			return
		}
	}
	gsm.goalProgress[playerID] = append(entries, types.GoalProgressEntry{
		GoalID:   goalID,
		Evidence: []string{evidence},
	})
}

func (gsm *GameStateManager) GetGoalProgress(playerID string) []types.GoalProgressEntry {
	gsm.mu.RLock()
	defer gsm.mu.RUnlock()
	return gsm.goalProgress[playerID]
}

func (gsm *GameStateManager) AppendEventLog(event types.GameEvent) {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()
	gsm.eventLog = append(gsm.eventLog, event)
}

// View generation

func (gsm *GameStateManager) GetRoomView(playerID string) protocol.RoomView {
	gsm.mu.RLock()
	defer gsm.mu.RUnlock()

	player, ok := gsm.state.Players[playerID]
	if !ok {
		return protocol.RoomView{}
	}

	room := gsm.mapEngine.GetRoomByID(player.CurrentRoomID)
	if room == nil {
		return protocol.RoomView{}
	}

	var items []protocol.RoomViewItem
	for _, item := range room.Items {
		items = append(items, protocol.RoomViewItem{ID: item.ID, Name: item.Name})
	}

	var players []protocol.RoomViewPlayer
	for _, p := range gsm.state.Players {
		if p.CurrentRoomID == room.ID && p.Status == "connected" {
			players = append(players, protocol.RoomViewPlayer{ID: p.ID, Nickname: p.Nickname})
		}
	}

	var npcs []protocol.RoomViewNPC
	for _, npc := range gsm.world.NPCs {
		if npc.CurrentRoomID == room.ID {
			npcs = append(npcs, protocol.RoomViewNPC{ID: npc.ID, Name: npc.Name})
		}
	}

	return protocol.RoomView{
		ID:          room.ID,
		Name:        room.Name,
		Description: room.Description,
		Type:        room.Type,
		Items:       items,
		Players:     players,
		NPCs:        npcs,
	}
}

func (gsm *GameStateManager) GetMapView(playerID string) protocol.MapView {
	gsm.mu.RLock()
	defer gsm.mu.RUnlock()

	player := gsm.state.Players[playerID]
	myRoomID := ""
	if player != nil {
		myRoomID = player.CurrentRoomID
	}

	var rooms []protocol.MapViewRoom
	for _, room := range gsm.world.Map.Rooms {
		var playerNames []string
		count := 0
		for _, p := range gsm.state.Players {
			if p.CurrentRoomID == room.ID && p.Status == "connected" {
				playerNames = append(playerNames, p.Nickname)
				count++
			}
		}
		rooms = append(rooms, protocol.MapViewRoom{
			ID:          room.ID,
			Name:        room.Name,
			Type:        room.Type,
			PlayerCount: count,
			PlayerNames: playerNames,
		})
	}

	return protocol.MapView{
		Rooms:       rooms,
		Connections: gsm.world.Map.Connections,
		MyRoomID:    myRoomID,
	}
}

// Query methods

func (gsm *GameStateManager) GetPlayer(playerID string) *types.Player {
	gsm.mu.RLock()
	defer gsm.mu.RUnlock()
	return gsm.state.Players[playerID]
}

func (gsm *GameStateManager) GetAllPlayerIDs() []string {
	gsm.mu.RLock()
	defer gsm.mu.RUnlock()
	ids := make([]string, 0, len(gsm.state.Players))
	for id, p := range gsm.state.Players {
		if p.Status == "connected" {
			ids = append(ids, id)
		}
	}
	return ids
}

func (gsm *GameStateManager) GetPlayersInRoom(roomID string) []*types.Player {
	gsm.mu.RLock()
	defer gsm.mu.RUnlock()
	var result []*types.Player
	for _, p := range gsm.state.Players {
		if p.CurrentRoomID == roomID && p.Status == "connected" {
			result = append(result, p)
		}
	}
	return result
}

func (gsm *GameStateManager) GetNPCsInRoom(roomID string) []types.NPC {
	gsm.mu.RLock()
	defer gsm.mu.RUnlock()
	var result []types.NPC
	for _, npc := range gsm.world.NPCs {
		if npc.CurrentRoomID == roomID {
			result = append(result, npc)
		}
	}
	return result
}

func (gsm *GameStateManager) GetPlayerRoom(playerID string) *types.Room {
	gsm.mu.RLock()
	defer gsm.mu.RUnlock()
	player, ok := gsm.state.Players[playerID]
	if !ok {
		return nil
	}
	return gsm.mapEngine.GetRoomByID(player.CurrentRoomID)
}

func (gsm *GameStateManager) GetWorld() types.World {
	gsm.mu.RLock()
	defer gsm.mu.RUnlock()
	return gsm.world
}

func (gsm *GameStateManager) GetFullState() types.GameState {
	gsm.mu.RLock()
	defer gsm.mu.RUnlock()
	return gsm.state
}

func (gsm *GameStateManager) GetRecentEvents(count int) []types.GameEvent {
	gsm.mu.RLock()
	defer gsm.mu.RUnlock()
	if len(gsm.eventLog) <= count {
		return gsm.eventLog
	}
	return gsm.eventLog[len(gsm.eventLog)-count:]
}

func (gsm *GameStateManager) GetSemiPublicInfoForPlayer(playerID string) []types.SemiPublicInfo {
	gsm.mu.RLock()
	defer gsm.mu.RUnlock()

	var result []types.SemiPublicInfo
	for _, sp := range gsm.world.Information.SemiPublic {
		for _, targetID := range sp.TargetPlayerIDs {
			if targetID == playerID {
				result = append(result, sp)
				break
			}
		}
	}
	return result
}

func (gsm *GameStateManager) GetMapEngine() *mapengine.MapEngine {
	return gsm.mapEngine
}
