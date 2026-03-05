package session

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
	"unicode"

	"github.com/gorilla/websocket"

	"github.com/anthropics/story/internal/server/aiface"
	"github.com/anthropics/story/internal/server/end"
	"github.com/anthropics/story/internal/server/eventbus"
	"github.com/anthropics/story/internal/server/game"
	"github.com/anthropics/story/internal/server/network"
	"github.com/anthropics/story/internal/shared/protocol"
	"github.com/anthropics/story/internal/shared/types"
)

const (
	maxPlayers     = 6
	minPlayers     = 2
	maxNicknameLen = 20
)

// JoinError represents an error that occurs when a player tries to join.
type JoinError string

const (
	JoinErrorGameStarted       JoinError = "GAME_ALREADY_STARTED"
	JoinErrorRoomFull          JoinError = "ROOM_FULL"
	JoinErrorDuplicateNickname JoinError = "DUPLICATE_NICKNAME"
	JoinErrorInvalidNickname   JoinError = "INVALID_NICKNAME"
)

func (e JoinError) Error() string { return string(e) }

// StartError represents an error that occurs when the host tries to start the game.
type StartError string

const (
	StartErrorNotHost          StartError = "NOT_HOST"
	StartErrorNotEnoughPlayers StartError = "NOT_ENOUGH_PLAYERS"
)

func (e StartError) Error() string { return string(e) }

// SessionManager manages the game session lifecycle.
type SessionManager struct {
	game         *types.Game
	network      *network.NetworkServer
	eventBus     *eventbus.EventBus
	gameState    *game.GameStateManager
	aiLayer      aiface.AILayer
	endCondition *end.EndConditionEngine
	readyPlayers map[string]bool
	mu           sync.Mutex
}

// NewSessionManager creates a new SessionManager.
func NewSessionManager(
	net *network.NetworkServer,
	bus *eventbus.EventBus,
	gs *game.GameStateManager,
	ail aiface.AILayer,
) *SessionManager {
	return &SessionManager{
		network:      net,
		eventBus:     bus,
		gameState:    gs,
		aiLayer:      ail,
		readyPlayers: make(map[string]bool),
	}
}

// SetEndConditionEngine sets the end condition engine reference.
// Called after construction to break the circular dependency.
func (sm *SessionManager) SetEndConditionEngine(ece *end.EndConditionEngine) {
	sm.endCondition = ece
}

// CreateSession creates a new game session and returns the room code.
func (sm *SessionManager) CreateSession() string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	roomCode := GenerateRoomCode()
	now := time.Now().UnixMilli()

	sm.game = &types.Game{
		ID:       roomCode,
		RoomCode: roomCode,
		Status:   types.GameStatusLobby,
		Settings: types.GameSettings{
			MaxPlayers:     maxPlayers,
			TimeoutMinutes: 20,
		},
		Players:   make(map[string]*types.Player),
		EventLog:  []interface{}{},
		CreatedAt: now,
	}

	return roomCode
}

// AddPlayer adds a player to the session.
func (sm *SessionManager) AddPlayer(conn *websocket.Conn, nickname string) (*types.Player, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.game == nil {
		return nil, fmt.Errorf("no active session")
	}

	// Validate game status
	if sm.game.Status != types.GameStatusLobby {
		return nil, JoinErrorGameStarted
	}

	// Validate nickname
	if !isValidNickname(nickname) {
		return nil, JoinErrorInvalidNickname
	}

	// Check duplicate nickname
	for _, p := range sm.game.Players {
		if p.Nickname == nickname {
			return nil, JoinErrorDuplicateNickname
		}
	}

	// Check max players
	if len(sm.game.Players) >= sm.game.Settings.MaxPlayers {
		return nil, JoinErrorRoomFull
	}

	// Create player
	isHost := len(sm.game.Players) == 0
	playerID := fmt.Sprintf("player_%d", time.Now().UnixNano())
	player := &types.Player{
		ID:          playerID,
		Nickname:    nickname,
		IsHost:      isHost,
		Status:      "connected",
		ConnectedAt: time.Now().UnixMilli(),
	}

	if isHost {
		sm.game.HostID = playerID
	}

	sm.game.Players[playerID] = player
	sm.gameState.AddPlayer(player)

	// Bind socket
	sm.network.BindPlayerToSocket(playerID, conn)

	// Send joined message to the player
	sm.network.SendTo(playerID, protocol.JoinedMessage{
		Type:     protocol.SMsgTypeJoined,
		PlayerID: playerID,
		RoomCode: sm.game.RoomCode,
		IsHost:   isHost,
	})

	// Broadcast lobby update to all
	sm.broadcastLobbyUpdate()

	// Publish player connected event
	sm.eventBus.PublishPlayerConnected(eventbus.PlayerConnectedData{
		PlayerID: playerID,
	})

	return player, nil
}

// RemovePlayer removes a player from the session.
func (sm *SessionManager) RemovePlayer(playerID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.game == nil {
		return
	}

	if p, ok := sm.game.Players[playerID]; ok {
		p.Status = "disconnected"
	}

	sm.gameState.RemovePlayer(playerID)
	sm.network.UnbindPlayer(playerID)

	// Publish disconnection
	sm.eventBus.PublishPlayerDisconnected(eventbus.PlayerDisconnectedData{
		PlayerID: playerID,
	})

	// Broadcast lobby update if still in lobby
	if sm.game.Status == types.GameStatusLobby {
		sm.broadcastLobbyUpdate()
	}
}

// GetPlayers returns all players in the session.
func (sm *SessionManager) GetPlayers() []*types.Player {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.game == nil {
		return nil
	}

	players := make([]*types.Player, 0, len(sm.game.Players))
	for _, p := range sm.game.Players {
		players = append(players, p)
	}
	return players
}

// StartGame transitions from lobby to generating and starts world generation.
func (sm *SessionManager) StartGame(hostID string, themeKeyword string) error {
	sm.mu.Lock()

	if sm.game == nil {
		sm.mu.Unlock()
		return fmt.Errorf("no active session")
	}

	if sm.game.HostID != hostID {
		sm.mu.Unlock()
		return StartErrorNotHost
	}

	// Count connected players
	connectedCount := 0
	for _, p := range sm.game.Players {
		if p.Status == "connected" {
			connectedCount++
		}
	}
	if connectedCount < minPlayers {
		sm.mu.Unlock()
		return StartErrorNotEnoughPlayers
	}

	// Transition to generating
	sm.game.Status = types.GameStatusGenerating
	now := time.Now().UnixMilli()
	sm.game.StartedAt = &now
	playerCount := connectedCount
	settings := sm.game.Settings

	sm.mu.Unlock()

	sm.eventBus.PublishGameStatusChanged(eventbus.GameStatusChangedData{
		From: types.GameStatusLobby,
		To:   types.GameStatusGenerating,
	})

	// Start world generation in goroutine
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		world, err := sm.aiLayer.GenerateWorld(ctx, settings, playerCount, themeKeyword)
		if err != nil {
			slog.Error("world generation failed", "error", err)
			sm.network.SendToAll(protocol.ErrorMessage{
				Type:    protocol.SMsgTypeError,
				Code:    "GENERATION_FAILED",
				Message: "World generation failed. Please try again.",
			})
			// Revert to lobby
			sm.mu.Lock()
			sm.game.Status = types.GameStatusLobby
			sm.game.StartedAt = nil
			sm.mu.Unlock()
			return
		}
		sm.OnWorldGenerated(world)
	}()

	return nil
}

// OnWorldGenerated handles world generation completion: initializes game state,
// assigns roles, sends briefings, and transitions to briefing status.
func (sm *SessionManager) OnWorldGenerated(world *types.World) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.game.World = world
	sm.game.Status = types.GameStatusBriefing

	// Assign roles to players
	roleAssignments := make(map[string]types.PlayerRole)
	playerIDs := make([]string, 0)
	for id, p := range sm.game.Players {
		if p.Status == "connected" {
			playerIDs = append(playerIDs, id)
		}
	}

	for i, playerID := range playerIDs {
		if i < len(world.PlayerRoles) {
			roleAssignments[playerID] = world.PlayerRoles[i]
		}
	}

	// Initialize game state with world and role assignments
	sm.gameState.InitializeWorld(*world, roleAssignments)

	sm.eventBus.PublishGameStatusChanged(eventbus.GameStatusChangedData{
		From: types.GameStatusGenerating,
		To:   types.GameStatusBriefing,
	})

	// Send public briefing to all players
	sm.network.SendToAll(protocol.BriefingPublicMessage{
		Type: protocol.SMsgTypeBriefingPublic,
		Info: world.Information.Public,
	})

	// Send private briefing to each player
	for playerID, role := range roleAssignments {
		semiPublicInfo := sm.gameState.GetSemiPublicInfoForPlayer(playerID)
		secrets := []string{role.Secret}

		// Include additional secrets from private info
		for _, pi := range world.Information.Private {
			if pi.PlayerID == role.ID {
				secrets = append(secrets, pi.AdditionalSecrets...)
			}
		}

		sm.network.SendTo(playerID, protocol.BriefingPrivateMessage{
			Type:           protocol.SMsgTypeBriefingPrivate,
			Role:           role,
			Secrets:        secrets,
			SemiPublicInfo: semiPublicInfo,
		})
	}
}

// MarkPlayerReady marks a player as ready and checks if all players are ready.
func (sm *SessionManager) MarkPlayerReady(playerID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.game == nil || sm.game.Status != types.GameStatusBriefing {
		return
	}

	sm.readyPlayers[playerID] = true

	// Check if all connected players are ready
	allReady := true
	for _, p := range sm.game.Players {
		if p.Status == "connected" && !sm.readyPlayers[p.ID] {
			allReady = false
			break
		}
	}

	if allReady {
		go sm.OnAllPlayersReady()
	}
}

// OnAllPlayersReady transitions from briefing to playing.
func (sm *SessionManager) OnAllPlayersReady() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.game.Status != types.GameStatusBriefing {
		return
	}

	sm.game.Status = types.GameStatusPlaying

	sm.eventBus.PublishGameStatusChanged(eventbus.GameStatusChangedData{
		From: types.GameStatusBriefing,
		To:   types.GameStatusPlaying,
	})

	// Send game_started with initial room to each player
	for _, p := range sm.game.Players {
		if p.Status == "connected" {
			roomView := sm.gameState.GetRoomView(p.ID)
			sm.network.SendTo(p.ID, protocol.GameStartedMessage{
				Type:        protocol.SMsgTypeGameStarted,
				InitialRoom: roomView,
			})
		}
	}

	// Start end condition monitoring
	if sm.endCondition != nil && sm.game.World != nil {
		sm.endCondition.StartMonitoring(
			sm.game.World.GameStructure.EndConditions,
			sm.game.World.GameStructure.EstimatedDuration,
		)
	}
}

// StartEnding transitions from playing to ending.
func (sm *SessionManager) StartEnding() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.game == nil {
		return
	}
	sm.game.Status = types.GameStatusEnding

	sm.eventBus.PublishGameStatusChanged(eventbus.GameStatusChangedData{
		From: types.GameStatusPlaying,
		To:   types.GameStatusEnding,
	})
}

// FinishGame transitions from ending to finished.
func (sm *SessionManager) FinishGame() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.game == nil {
		return
	}
	sm.game.Status = types.GameStatusFinished
	now := time.Now().UnixMilli()
	sm.game.EndedAt = &now

	sm.eventBus.PublishGameStatusChanged(eventbus.GameStatusChangedData{
		From: types.GameStatusEnding,
		To:   types.GameStatusFinished,
	})

	sm.network.SendToAll(protocol.GameFinishedMessage{
		Type: protocol.SMsgTypeGameFinished,
	})
}

// CancelSession cancels the session (host only, lobby state).
func (sm *SessionManager) CancelSession(hostID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.game == nil {
		return fmt.Errorf("no active session")
	}

	if sm.game.HostID != hostID {
		return StartErrorNotHost
	}

	if sm.game.Status != types.GameStatusLobby {
		return fmt.Errorf("can only cancel in lobby state")
	}

	sm.network.SendToAll(protocol.GameCancelledMessage{
		Type:   protocol.SMsgTypeGameCancelled,
		Reason: "Host cancelled the game",
	})

	return nil
}

// Shutdown gracefully shuts down the session.
func (sm *SessionManager) Shutdown(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.game == nil {
		return nil
	}

	// Notify all players
	sm.network.SendToAll(protocol.SystemMessage{
		Type:    protocol.SMsgTypeSystem,
		Content: "Server is shutting down",
	})

	// Stop end condition monitoring if active
	if sm.endCondition != nil {
		sm.endCondition.StopMonitoring()
	}

	return sm.network.Stop()
}

// GetGameStatus returns the current game status.
func (sm *SessionManager) GetGameStatus() types.GameStatus {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.game == nil {
		return ""
	}
	return sm.game.Status
}

// GetRoomCode returns the current room code.
func (sm *SessionManager) GetRoomCode() string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.game == nil {
		return ""
	}
	return sm.game.RoomCode
}

// IsHost returns true if the given player ID is the host.
func (sm *SessionManager) IsHost(playerID string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.game == nil {
		return false
	}
	return sm.game.HostID == playerID
}

// broadcastLobbyUpdate sends the current lobby state to all connected players.
func (sm *SessionManager) broadcastLobbyUpdate() {
	var lobbyPlayers []protocol.LobbyPlayer
	for _, p := range sm.game.Players {
		if p.Status == "connected" {
			lobbyPlayers = append(lobbyPlayers, protocol.LobbyPlayer{
				ID:       p.ID,
				Nickname: p.Nickname,
				IsHost:   p.IsHost,
			})
		}
	}

	sm.network.SendToAll(protocol.LobbyUpdateMessage{
		Type:       protocol.SMsgTypeLobbyUpdate,
		Players:    lobbyPlayers,
		MaxPlayers: sm.game.Settings.MaxPlayers,
	})
}

// isValidNickname checks nickname length and characters.
func isValidNickname(nickname string) bool {
	if len(nickname) == 0 || len(nickname) > maxNicknameLen {
		return false
	}
	for _, r := range nickname {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
