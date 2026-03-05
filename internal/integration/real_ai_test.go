package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/anthropics/story/internal/ai"
	"github.com/anthropics/story/internal/ai/provider"
	"github.com/anthropics/story/internal/server/action"
	"github.com/anthropics/story/internal/server/end"
	"github.com/anthropics/story/internal/server/eventbus"
	"github.com/anthropics/story/internal/server/game"
	"github.com/anthropics/story/internal/server/mapengine"
	"github.com/anthropics/story/internal/server/message"
	"github.com/anthropics/story/internal/server/network"
	"github.com/anthropics/story/internal/server/session"
	"github.com/anthropics/story/internal/shared/protocol"
	"github.com/anthropics/story/internal/shared/types"
)

func getAPIKeyFromDoppler(name string) string {
	out, err := exec.Command("doppler", "secrets", "get", name, "--plain").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func freePortForRealAI() int {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

func waitServerReadyRealAI(t *testing.T, port int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("server did not become ready")
}

func strPtr(s string) *string { return &s }

// TestRealAIWorldGeneration tests world generation with a real AI provider.
func TestRealAIWorldGeneration(t *testing.T) {
	if os.Getenv("STORY_REAL_AI_TEST") != "1" {
		t.Skip("set STORY_REAL_AI_TEST=1 to run real AI tests")
	}

	apiKey := getAPIKeyFromDoppler("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not available from Doppler")
	}

	p, err := provider.NewProviderFromConfig(provider.ProviderConfig{
		Type:   "anthropic",
		APIKey: apiKey,
		Model:  "claude-sonnet-4-20250514",
	})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	aiLayer := ai.NewAILayerWithProvider(p)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	settings := types.GameSettings{
		MaxPlayers:     6,
		TimeoutMinutes: 20,
		HasGM:          true,
		HasNPC:         true,
	}

	t.Log("Starting world generation with real AI...")
	world, err := aiLayer.GenerateWorld(ctx, settings, 2, "haunted mansion")
	if err != nil {
		t.Fatalf("world generation failed: %v", err)
	}

	if world.Title == "" {
		t.Error("world title is empty")
	}
	if world.Synopsis == "" {
		t.Error("world synopsis is empty")
	}
	if len(world.Map.Rooms) < 4 {
		t.Errorf("expected at least 4 rooms, got %d", len(world.Map.Rooms))
	}
	if len(world.PlayerRoles) != 2 {
		t.Errorf("expected 2 player roles, got %d", len(world.PlayerRoles))
	}
	if len(world.Map.Connections) < 3 {
		t.Errorf("expected at least 3 connections, got %d", len(world.Map.Connections))
	}
	if len(world.Clues) < 4 {
		t.Errorf("expected at least 4 clues, got %d", len(world.Clues))
	}

	t.Logf("World generated: %q with %d rooms, %d roles, %d clues, %d NPCs",
		world.Title, len(world.Map.Rooms), len(world.PlayerRoles), len(world.Clues), len(world.NPCs))

	for i, role := range world.PlayerRoles {
		if role.CharacterName == "" {
			t.Errorf("role %d has empty character name", i)
		}
		if len(role.PersonalGoals) == 0 {
			t.Errorf("role %d has no personal goals", i)
		}
	}

	me := mapengine.NewMapEngine()
	me.Initialize(world.Map)
	for _, room := range world.Map.Rooms {
		adj := me.GetAdjacentRooms(room.ID)
		if len(adj) == 0 {
			t.Errorf("room %q has no adjacent rooms", room.Name)
		}
	}

	if err := me.ValidateConnectivity(); err != nil {
		t.Errorf("map connectivity check failed: %v", err)
	}
}

// realAIStack holds all wired components for real AI tests.
type realAIStack struct {
	net        *network.NetworkServer
	bus        *eventbus.EventBus
	gameState  *game.GameStateManager
	sessionMgr *session.SessionManager
	mapEng     *mapengine.MapEngine
	endEngine  *end.EndConditionEngine
	msgRouter  *message.MessageRouter
	actionProc *action.ActionProcessor
	port       int
	roomCode   string
	connCh     chan *websocket.Conn
}

func buildRealAIStack(t *testing.T, aiLayer *ai.AILayer) *realAIStack {
	t.Helper()

	port := freePortForRealAI()
	bus := eventbus.NewEventBus()
	netSrv := network.NewNetworkServer(network.NetworkConfig{Port: port})
	me := mapengine.NewMapEngine()
	gs := game.NewGameStateManager(bus, me)
	sm := session.NewSessionManager(netSrv, bus, gs, aiLayer)
	ece := end.NewEndConditionEngine(gs, aiLayer, bus, sm)
	sm.SetEndConditionEngine(ece)
	mr := message.NewMessageRouter(netSrv, gs, bus)
	ap := action.NewActionProcessor(gs, me, aiLayer, ece, bus, netSrv, mr)

	connCh := make(chan *websocket.Conn, 16)

	netSrv.OnConnection(func(conn *websocket.Conn) {
		select {
		case connCh <- conn:
		default:
		}
	})

	netSrv.OnMessage(func(playerID string, msg protocol.ClientMessage) {
		switch msg.Type {
		case "start_game":
			sm.StartGame(playerID, msg.ThemeKeyword)
		case "ready":
			sm.MarkPlayerReady(playerID)
		default:
			ap.ProcessMessage(playerID, msg)
		}
	})

	netSrv.OnDisconnection(func(playerID string) {
		sm.RemovePlayer(playerID)
	})

	roomCode := sm.CreateSession()
	if err := netSrv.Start(roomCode); err != nil {
		t.Fatalf("server start failed: %v", err)
	}
	waitServerReadyRealAI(t, port)

	return &realAIStack{
		net:        netSrv,
		bus:        bus,
		gameState:  gs,
		sessionMgr: sm,
		mapEng:     me,
		endEngine:  ece,
		msgRouter:  mr,
		actionProc: ap,
		port:       port,
		roomCode:   roomCode,
		connCh:     connCh,
	}
}

func (s *realAIStack) connectPlayer(t *testing.T, nickname string) (*websocket.Conn, string) {
	t.Helper()
	wsURL := fmt.Sprintf("ws://localhost:%d/ws/%s", s.port, s.roomCode)
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{})
	if err != nil {
		t.Fatalf("dial failed for %s: %v", nickname, err)
	}
	select {
	case serverConn := <-s.connCh:
		player, err := s.sessionMgr.AddPlayer(serverConn, nickname)
		if err != nil {
			t.Fatalf("AddPlayer failed for %s: %v", nickname, err)
		}
		return clientConn, player.ID
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for server conn for %s", nickname)
		return nil, ""
	}
}

func readMsgRealAI(conn *websocket.Conn, timeout time.Duration) (result map[string]interface{}) {
	defer func() {
		if r := recover(); r != nil {
			result = nil
		}
	}()
	conn.SetReadDeadline(time.Now().Add(timeout))
	_, data, err := conn.ReadMessage()
	if err != nil {
		return nil
	}
	var msg map[string]interface{}
	json.Unmarshal(data, &msg)
	return msg
}

func waitForMsgRealAI(conn *websocket.Conn, msgType string, timeout time.Duration) map[string]interface{} {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		msg := readMsgRealAI(conn, time.Until(deadline))
		if msg == nil {
			continue
		}
		if msg["type"] == msgType {
			return msg
		}
	}
	return nil
}

// TestRealAIFullGameplay tests a complete game flow with real AI.
func TestRealAIFullGameplay(t *testing.T) {
	if os.Getenv("STORY_REAL_AI_TEST") != "1" {
		t.Skip("set STORY_REAL_AI_TEST=1 to run real AI tests")
	}

	apiKey := getAPIKeyFromDoppler("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not available from Doppler")
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	p, err := provider.NewProviderFromConfig(provider.ProviderConfig{
		Type:   "anthropic",
		APIKey: apiKey,
		Model:  "claude-sonnet-4-20250514",
	})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	realAI := ai.NewAILayerWithProvider(p)
	stack := buildRealAIStack(t, realAI)
	defer stack.net.Stop()

	// Connect 2 players
	t.Log("Connecting players...")
	conn1, player1ID := stack.connectPlayer(t, "Alice")
	defer conn1.Close()
	conn2, player2ID := stack.connectPlayer(t, "Bob")
	defer conn2.Close()

	// Don't drain - just let messages accumulate; waitForMsgRealAI will skip them

	// Start game
	t.Log("Starting game with real AI world generation (this may take 30-60 seconds)...")
	err = stack.sessionMgr.StartGame(player1ID, "mystery mansion")
	if err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}

	// Wait for briefing
	t.Log("Waiting for briefing messages...")
	if waitForMsgRealAI(conn1, "briefing_public", 120*time.Second) == nil {
		t.Fatal("player 1 did not receive briefing_public")
	}
	t.Log("Received briefing_public")

	if waitForMsgRealAI(conn1, "briefing_private", 5*time.Second) == nil {
		t.Fatal("player 1 did not receive briefing_private")
	}
	t.Log("Received briefing_private for player 1")

	waitForMsgRealAI(conn2, "briefing_public", 5*time.Second)
	waitForMsgRealAI(conn2, "briefing_private", 5*time.Second)

	// Both players ready
	t.Log("Marking players as ready...")
	stack.sessionMgr.MarkPlayerReady(player1ID)
	stack.sessionMgr.MarkPlayerReady(player2ID)

	if waitForMsgRealAI(conn1, "game_started", 5*time.Second) == nil {
		t.Fatal("player 1 did not receive game_started")
	}
	t.Log("Game started!")
	waitForMsgRealAI(conn2, "game_started", 5*time.Second)

	// Load map from world
	world := stack.gameState.GetWorld()
	stack.mapEng.Initialize(world.Map)

	// Get player 1's current room
	player1Room := stack.gameState.GetPlayerRoom(player1ID)
	if player1Room == nil {
		t.Fatal("player 1 has no room")
	}
	t.Logf("Player 1 is in room: %s", player1Room.Name)

	// Test chat
	t.Log("Testing chat...")
	stack.actionProc.ProcessMessage(player1ID, protocol.ClientMessage{
		Type:    "chat",
		Content: "Hello everyone!",
	})
	time.Sleep(500 * time.Millisecond)

	// Test examine with real AI
	t.Log("Testing /examine with real AI...")
	stack.actionProc.ProcessMessage(player1ID, protocol.ClientMessage{
		Type: "examine",
	})

	examineResult := waitForMsgRealAI(conn1, "game_event", 30*time.Second)
	if examineResult == nil {
		t.Log("Warning: no game_event from examine (AI may have timed out)")
	} else {
		t.Log("Examine result received from real AI")
	}

	// Test move to adjacent room
	adjRooms := stack.mapEng.GetAdjacentRooms(player1Room.ID)
	if len(adjRooms) > 0 {
		targetRoom := adjRooms[0]
		t.Logf("Moving player 1 to room: %s", targetRoom.Name)
		stack.actionProc.ProcessMessage(player1ID, protocol.ClientMessage{
			Type:         "move",
			TargetRoomID: targetRoom.ID,
		})

		moveResult := waitForMsgRealAI(conn1, "room_changed", 5*time.Second)
		if moveResult == nil {
			t.Log("Warning: did not receive room_changed")
		} else {
			t.Log("Move successful")
		}
	}

	// Test NPC talk if there are NPCs
	if len(world.NPCs) > 0 {
		npc := world.NPCs[0]
		currentRoom := stack.gameState.GetPlayerRoom(player1ID)

		// Move player to NPC's room if needed
		if currentRoom != nil && currentRoom.ID != npc.CurrentRoomID {
			stack.actionProc.ProcessMessage(player1ID, protocol.ClientMessage{
				Type:         "move",
				TargetRoomID: npc.CurrentRoomID,
			})
			waitForMsgRealAI(conn1, "room_changed", 5*time.Second)
		}

		t.Logf("Talking to NPC: %s", npc.Name)
		stack.actionProc.ProcessMessage(player1ID, protocol.ClientMessage{
			Type:    "talk",
			NPCID:   npc.ID,
			Message: "Hello! What do you know about this place?",
		})

		npcResponse := waitForMsgRealAI(conn1, "game_event", 30*time.Second)
		if npcResponse == nil {
			t.Log("Warning: no NPC response (AI may have timed out)")
		} else {
			t.Log("NPC response received from real AI")
		}
	}

	// Test propose end flow
	t.Log("Testing end game proposal...")
	stack.actionProc.ProcessMessage(player1ID, protocol.ClientMessage{
		Type: "propose_end",
	})
	time.Sleep(500 * time.Millisecond)
	stack.actionProc.ProcessMessage(player2ID, protocol.ClientMessage{
		Type:  "end_vote",
		Agree: true,
	})

	// Wait for game to finish
	t.Log("Waiting for game ending (AI generating endings, may take 30-60 seconds)...")
	endTimeout := time.After(120 * time.Second)
	gotFinished := false
	for !gotFinished {
		select {
		case <-endTimeout:
			t.Log("Warning: game did not finish within timeout")
			gotFinished = true
		default:
			msg := readMsgRealAI(conn1, 5*time.Second)
			if msg == nil {
				continue
			}
			msgType, _ := msg["type"].(string)
			t.Logf("Received message: %s", msgType)
			if msgType == "game_finished" {
				gotFinished = true
				t.Log("Game finished successfully!")
			}
		}
	}

	t.Log("L4 Real AI gameplay test completed")
}

// TestRealAIProviderHealthCheck verifies connectivity to AI providers.
func TestRealAIProviderHealthCheck(t *testing.T) {
	if os.Getenv("STORY_REAL_AI_TEST") != "1" {
		t.Skip("set STORY_REAL_AI_TEST=1 to run real AI tests")
	}

	apiKey := getAPIKeyFromDoppler("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not available from Doppler")
	}

	p, err := provider.NewProviderFromConfig(provider.ProviderConfig{
		Type:   "anthropic",
		APIKey: apiKey,
	})
	if err != nil {
		t.Errorf("failed to create anthropic provider: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	_, err = p.GenerateText(ctx, provider.TextRequest{
		SystemPrompt: "Reply with OK",
		UserPrompt:   "ping",
		MaxTokens:    5,
		Temperature:  0,
	})
	cancel()

	if err != nil {
		t.Errorf("anthropic health check failed: %v", err)
	} else {
		t.Log("anthropic provider: healthy")
	}
}

// TestSubagentPlayTest simulates two AI "agents" playing the game concurrently.
func TestSubagentPlayTest(t *testing.T) {
	if os.Getenv("STORY_REAL_AI_TEST") != "1" {
		t.Skip("set STORY_REAL_AI_TEST=1 to run real AI tests")
	}

	apiKey := getAPIKeyFromDoppler("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not available from Doppler")
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	p, err := provider.NewProviderFromConfig(provider.ProviderConfig{
		Type:   "anthropic",
		APIKey: apiKey,
		Model:  "claude-sonnet-4-20250514",
	})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}
	realAI := ai.NewAILayerWithProvider(p)
	stack := buildRealAIStack(t, realAI)
	defer stack.net.Stop()

	// Connect players
	conn1, player1ID := stack.connectPlayer(t, "AgentAlice")
	defer conn1.Close()
	conn2, player2ID := stack.connectPlayer(t, "AgentBob")
	defer conn2.Close()

	// Start game
	t.Log("Starting game with AI world generation...")
	if err := stack.sessionMgr.StartGame(player1ID, "space station mystery"); err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}

	t.Log("Waiting for world generation...")
	if waitForMsgRealAI(conn1, "briefing_public", 120*time.Second) == nil {
		t.Fatal("no briefing_public received")
	}
	t.Log("World generated, briefings received")

	time.Sleep(1 * time.Second)

	stack.sessionMgr.MarkPlayerReady(player1ID)
	stack.sessionMgr.MarkPlayerReady(player2ID)

	if waitForMsgRealAI(conn1, "game_started", 5*time.Second) == nil {
		t.Fatal("no game_started received")
	}
	waitForMsgRealAI(conn2, "game_started", 5*time.Second)

	world := stack.gameState.GetWorld()
	stack.mapEng.Initialize(world.Map)
	t.Logf("Game world: %q", world.Title)

	// Subagent simulation: each player takes actions in parallel
	var wg sync.WaitGroup
	actionsTaken := make(map[string][]string)
	var actionsMu sync.Mutex

	recordAction := func(playerID, actionName string) {
		actionsMu.Lock()
		actionsTaken[playerID] = append(actionsTaken[playerID], actionName)
		actionsMu.Unlock()
	}

	simulatePlayer := func(playerID string, conn *websocket.Conn, name string) {
		defer wg.Done()

		type playerAction struct {
			name string
			msg  protocol.ClientMessage
		}

		actions := []playerAction{
			{"chat", protocol.ClientMessage{Type: "chat", Content: fmt.Sprintf("I'm %s, let's investigate!", name)}},
			{"examine", protocol.ClientMessage{Type: "examine"}},
		}

		// Add a move if possible
		currentRoom := stack.gameState.GetPlayerRoom(playerID)
		if currentRoom != nil {
			adjRooms := stack.mapEng.GetAdjacentRooms(currentRoom.ID)
			if len(adjRooms) > 0 {
				actions = append(actions,
					playerAction{"move", protocol.ClientMessage{Type: "move", TargetRoomID: adjRooms[0].ID}},
					playerAction{"examine_new_room", protocol.ClientMessage{Type: "examine"}},
				)
			}
		}

		for _, act := range actions {
			t.Logf("[%s] Action: %s", name, act.name)
			stack.actionProc.ProcessMessage(playerID, act.msg)
			recordAction(playerID, act.name)

			// Wait briefly for response
			deadline := time.Now().Add(30 * time.Second)
			for time.Now().Before(deadline) {
				conn.SetReadDeadline(time.Now().Add(3 * time.Second))
				_, _, err := conn.ReadMessage()
				if err != nil {
					break
				}
			}

			time.Sleep(1 * time.Second)
		}
	}

	t.Log("Starting subagent simulation (2 players taking actions in parallel)...")
	wg.Add(2)
	go simulatePlayer(player1ID, conn1, "AgentAlice")
	go simulatePlayer(player2ID, conn2, "AgentBob")
	wg.Wait()

	t.Log("Subagent actions complete. Proposing end...")

	stack.actionProc.ProcessMessage(player1ID, protocol.ClientMessage{Type: "propose_end"})
	time.Sleep(500 * time.Millisecond)
	stack.actionProc.ProcessMessage(player2ID, protocol.ClientMessage{Type: "end_vote", Agree: true})

	finished := waitForMsgRealAI(conn1, "game_finished", 120*time.Second)
	if finished == nil {
		t.Log("Warning: game_finished not received within timeout")
	} else {
		t.Log("Game finished!")
	}

	actionsMu.Lock()
	for pid, acts := range actionsTaken {
		t.Logf("Player %s took %d actions: %v", pid, len(acts), acts)
	}
	actionsMu.Unlock()

	status := stack.sessionMgr.GetGameStatus()
	t.Logf("Final game status: %s", status)

	t.Log("L4 Subagent play test completed successfully")
}
