// Package integration provides end-to-end tests that wire the complete server
// stack with a mock AI provider and real WebSocket connections.
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/anthropics/story/internal/server/action"
	"github.com/anthropics/story/internal/server/aiface"
	"github.com/anthropics/story/internal/server/end"
	"github.com/anthropics/story/internal/server/eventbus"
	"github.com/anthropics/story/internal/server/game"
	"github.com/anthropics/story/internal/server/mapengine"
	"github.com/anthropics/story/internal/server/message"
	"github.com/anthropics/story/internal/server/network"
	"github.com/anthropics/story/internal/server/session"
	"github.com/anthropics/story/internal/shared/protocol"
	"github.com/anthropics/story/internal/shared/schemas"
	"github.com/anthropics/story/internal/shared/types"
)

// ----------------------------------------------------------------------------
// Mock AI provider
// ----------------------------------------------------------------------------

// mockAI implements aiface.AILayer and returns deterministic canned responses
// for all AI calls, enabling fast and reproducible E2E tests.
type mockAI struct {
	world *types.World
}

var _ aiface.AILayer = (*mockAI)(nil)

func newMockAI() *mockAI {
	room1ID := "room-library"
	room2ID := "room-study"
	npcID := "npc-butler"
	clueID := "clue-letter"

	world := &types.World{
		Title:      "The Vanishing Heir",
		Synopsis:   "A mystery set in a grand manor.",
		Atmosphere: "gothic",
		GameStructure: types.GameStructure{
			Concept:           "murder mystery",
			CoreConflict:      "who stole the heir",
			ProgressionStyle:  "investigation",
			EstimatedDuration: 20,
			EndConditions: []types.EndCondition{
				{
					ID:          "end-propose",
					Description: "Players propose to end the game",
					TriggerType: "timeout",
					IsFallback:  true,
				},
			},
			WinConditions: []types.WinCondition{
				{Description: "Identify the culprit", EvaluationCriteria: "vote result"},
			},
		},
		Map: types.GameMap{
			Rooms: []types.Room{
				{
					ID:          room1ID,
					Name:        "Library",
					Description: "A dusty library with tall shelves.",
					Type:        "public",
					Items:       []types.Item{},
					NPCIDs:      []string{npcID},
					ClueIDs:     []string{clueID},
				},
				{
					ID:          room2ID,
					Name:        "Study",
					Description: "A small study with a mahogany desk.",
					Type:        "private",
					Items:       []types.Item{},
					NPCIDs:      []string{},
					ClueIDs:     []string{},
				},
			},
			Connections: []types.Connection{
				{RoomA: room1ID, RoomB: room2ID, Bidirectional: true},
			},
		},
		PlayerRoles: []types.PlayerRole{
			{
				ID:            "role-detective",
				CharacterName: "Detective Harris",
				Background:    "Seasoned investigator.",
				Secret:        "Has a personal grudge against the butler.",
				PersonalGoals: []types.PersonalGoal{
					{
						ID:             "goal-identify",
						Description:    "Identify the culprit",
						EvaluationHint: "Did the player correctly accuse someone?",
						EntityRefs:     []string{},
					},
				},
				Relationships: []types.Relationship{
					{TargetCharacterName: "Lady Pemberton", Description: "old acquaintance"},
				},
			},
			{
				ID:            "role-suspect",
				CharacterName: "Lady Pemberton",
				Background:    "Wealthy socialite.",
				Secret:        "Was in the library when the heir disappeared.",
				PersonalGoals: []types.PersonalGoal{
					{
						ID:             "goal-deflect",
						Description:    "Avoid suspicion",
						EvaluationHint: "Did the player keep their secret?",
						EntityRefs:     []string{},
					},
				},
				Relationships: []types.Relationship{
					{TargetCharacterName: "Detective Harris", Description: "old acquaintance"},
				},
			},
		},
		NPCs: []types.NPC{
			{
				ID:                npcID,
				Name:              "Mr Hobbs",
				CurrentRoomID:     room1ID,
				Persona:           "dutiful but nervous butler",
				KnownInfo:         []string{"the heir was last seen at midnight"},
				HiddenInfo:        []string{"he saw the heir leave with a stranger"},
				BehaviorPrinciple: "protective of the family",
				InitialTrust:      0.5,
			},
		},
		Clues: []types.Clue{
			{
				ID:                clueID,
				Name:              "Torn Letter",
				Description:       "A letter torn into pieces, mentioning a secret meeting.",
				RoomID:            room1ID,
				DiscoverCondition: "examine the fireplace",
				RelatedClueIDs:    []string{},
			},
		},
		Gimmicks: []types.Gimmick{},
		Information: types.InformationLayers{
			Public: types.PublicInfo{
				Title:    "The Vanishing Heir",
				Synopsis: "A mystery set in a grand manor.",
				CharacterList: []types.CharacterListEntry{
					{Name: "Detective Harris", PublicDescription: "A well-known investigator."},
					{Name: "Lady Pemberton", PublicDescription: "A wealthy socialite."},
				},
				Relationships: "Detective Harris and Lady Pemberton are old acquaintances.",
				MapOverview:   "A two-room manor: the Library and the Study.",
				NPCList: []types.NPCListEntry{
					{Name: "Mr Hobbs", Location: "Library"},
				},
				GameRules: "Investigate, talk to NPCs, and propose an end when ready.",
			},
			SemiPublic: []types.SemiPublicInfo{},
			Private:    []types.PrivateInfo{},
		},
	}

	return &mockAI{world: world}
}

func (m *mockAI) GenerateWorld(
	_ context.Context,
	_ types.GameSettings,
	_ int,
	_ string,
) (*types.World, error) {
	return m.world, nil
}

func (m *mockAI) EvaluateExamine(
	_ context.Context,
	_ types.GameContext,
	target string,
) (*schemas.GameResponse, error) {
	examineResult, _ := json.Marshal(schemas.AIExamineResultEvent{
		Type: "examine_result",
		Data: schemas.AIExamineResultEventData{
			Target:      target,
			Description: "You find nothing unusual at first glance.",
			ClueFound:   false,
		},
	})
	return &schemas.GameResponse{
		Events:       []json.RawMessage{examineResult},
		StateChanges: []json.RawMessage{},
	}, nil
}

func (m *mockAI) EvaluateAction(
	_ context.Context,
	_ types.GameContext,
	actionStr string,
) (*schemas.GameResponse, error) {
	actionResult, _ := json.Marshal(schemas.AIActionResultEvent{
		Type: "action_result",
		Data: schemas.AIActionResultEventData{
			Action:          actionStr,
			Result:          "Nothing notable happens.",
			TriggeredEvents: []string{},
		},
	})
	return &schemas.GameResponse{
		Events:       []json.RawMessage{actionResult},
		StateChanges: []json.RawMessage{},
	}, nil
}

func (m *mockAI) TalkToNPC(
	_ context.Context,
	_ types.GameContext,
	_ string,
	_ string,
) (*schemas.NPCResponse, error) {
	return &schemas.NPCResponse{
		Dialogue:         "I cannot say anything further.",
		Emotion:          "nervous",
		InternalThought:  "I must protect the family.",
		InfoRevealed:     []string{},
		TrustChange:      0,
		TriggeredGimmick: false,
		Events:           []json.RawMessage{},
	}, nil
}

func (m *mockAI) JudgeEndCondition(
	_ context.Context,
	_ types.GameContext,
	_ types.EndCondition,
) (bool, error) {
	return false, nil
}

func (m *mockAI) GenerateEndings(
	_ context.Context,
	_ types.GameContext,
	_ string,
) (*schemas.Ending, error) {
	return &schemas.Ending{
		CommonResult:  "The mystery remains unsolved. The heir was never found.",
		PlayerEndings: []schemas.PlayerEndingSchema{},
	}, nil
}

// ----------------------------------------------------------------------------
// Test harness
// ----------------------------------------------------------------------------

// serverStack holds all wired components for the test.
type serverStack struct {
	net        *network.NetworkServer
	bus        *eventbus.EventBus
	gameState  *game.GameStateManager
	sessionMgr *session.SessionManager
	mapEng     *mapengine.MapEngine
	endEngine  *end.EndConditionEngine
	msgRouter  *message.MessageRouter
	actionProc *action.ActionProcessor
	ai         *mockAI
	port       int
	roomCode   string

	// connCh delivers server-side *websocket.Conn objects to the test
	// each time a client dials the server, so AddPlayer can be called
	// with the correct server-side connection.
	connCh chan *websocket.Conn
}

// buildStack wires all server components together with the mock AI provider.
func buildStack(t *testing.T) *serverStack {
	t.Helper()

	ai := newMockAI()

	// 1. EventBus
	bus := eventbus.NewEventBus()

	// 2. NetworkServer — use port 0 so the OS picks a free port atomically.
	netSrv := network.NewNetworkServer(network.NetworkConfig{Port: 0})

	// 3. MapEngine
	me := mapengine.NewMapEngine()

	// 4. GameStateManager
	gs := game.NewGameStateManager(bus, me)

	// 5. SessionManager
	sm := session.NewSessionManager(netSrv, bus, gs, ai)

	// 6. EndConditionEngine
	ece := end.NewEndConditionEngine(gs, ai, bus, sm)

	// 7. Wire EndConditionEngine back into SessionManager
	sm.SetEndConditionEngine(ece)

	// 8. MessageRouter (subscribes to EventBus; must be created before any
	//    events are published so its goroutines are ready)
	mr := message.NewMessageRouter(netSrv, gs, bus)

	// 9. ActionProcessor
	ap := action.NewActionProcessor(gs, me, ai, ece, bus, netSrv, mr)

	// connCh delivers the server-side websocket.Conn to the test so it can
	// call AddPlayer with the correct conn (server side, not client side).
	connCh := make(chan *websocket.Conn, 16)

	// 10. Register connection handlers.
	//
	//     Design rationale: the NetworkServer's readLoop only dispatches
	//     OnMessage for players already bound via BindPlayerToSocket.
	//     The test calls AddPlayer directly (passing the server-side conn)
	//     after the dial completes, which binds the player.  Subsequent
	//     messages from that player flow through the readLoop to OnMessage.
	netSrv.OnConnection(func(conn *websocket.Conn) {
		// Forward the server-side conn to the test goroutine so it can call
		// AddPlayer with the correct object.
		select {
		case connCh <- conn:
		default:
			t.Logf("connCh: buffer full, dropping conn")
		}
	})

	netSrv.OnMessage(func(playerID string, msg protocol.ClientMessage) {
		// Route session-level messages to SessionManager; game messages to
		// ActionProcessor.
		switch msg.Type {
		case "start_game":
			if err := sm.StartGame(playerID, msg.ThemeKeyword); err != nil {
				t.Logf("StartGame error (playerID=%s): %v", playerID, err)
			}
		case "ready":
			sm.MarkPlayerReady(playerID)
		default:
			if err := ap.ProcessMessage(playerID, msg); err != nil {
				t.Logf("ProcessMessage error (playerID=%s, type=%s): %v", playerID, msg.Type, err)
			}
		}
	})

	netSrv.OnDisconnection(func(playerID string) {
		sm.RemovePlayer(playerID)
	})

	// 11. Create the game session.
	roomCode := sm.CreateSession()

	// 12. Start the HTTP server.
	if err := netSrv.Start(roomCode); err != nil {
		t.Fatalf("NetworkServer.Start: %v", err)
	}

	port := netSrv.Port()

	return &serverStack{
		net:        netSrv,
		bus:        bus,
		gameState:  gs,
		sessionMgr: sm,
		mapEng:     me,
		endEngine:  ece,
		msgRouter:  mr,
		actionProc: ap,
		ai:         ai,
		port:       port,
		roomCode:   roomCode,
		connCh:     connCh,
	}
}

// teardown stops the HTTP server and closes the EventBus.
func (s *serverStack) teardown(t *testing.T) {
	t.Helper()
	s.endEngine.StopMonitoring()
	s.bus.Close()
	if err := s.net.Stop(); err != nil {
		t.Logf("NetworkServer.Stop: %v", err)
	}
}

// ----------------------------------------------------------------------------
// WebSocket test client helpers
// ----------------------------------------------------------------------------

// testClient wraps the client-side gorilla WebSocket connection for test use.
type testClient struct {
	clientConn *websocket.Conn // client-side conn (used to send messages)
	playerID   string
	msgs       chan map[string]interface{}
	closeOnce  sync.Once
}

// connectWS dials the test server and starts a receive goroutine that forwards
// all incoming JSON messages to a buffered channel.  It returns the client-side
// testClient.  The caller must separately call addPlayer to register the player
// using the server-side conn (obtained from serverStack.connCh).
func connectWS(t *testing.T, s *serverStack) *testClient {
	t.Helper()

	url := fmt.Sprintf("ws://127.0.0.1:%d/ws/%s", s.port, s.roomCode)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("websocket.Dial(%s): %v", url, err)
	}

	tc := &testClient{
		clientConn: conn,
		msgs:       make(chan map[string]interface{}, 128),
	}

	go func() {
		defer close(tc.msgs)
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var m map[string]interface{}
			if err := json.Unmarshal(raw, &m); err != nil {
				t.Logf("testClient: unmarshal error: %v (raw: %s)", err, raw)
				continue
			}
			tc.msgs <- m
		}
	}()

	return tc
}

// addPlayer registers the player with the session manager using the server-side
// conn that was captured in buildStack's OnConnection handler.  It blocks until
// the server-side conn is available (with a 2-second timeout).
func addPlayer(t *testing.T, s *serverStack, tc *testClient, nickname string) *types.Player {
	t.Helper()

	// Wait for the server-side conn to be delivered.
	var serverConn *websocket.Conn
	select {
	case serverConn = <-s.connCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("addPlayer(%s): timed out waiting for server-side conn", nickname)
	}

	player, err := s.sessionMgr.AddPlayer(serverConn, nickname)
	if err != nil {
		t.Fatalf("AddPlayer(%s): %v", nickname, err)
	}
	tc.playerID = player.ID
	return player
}

// send serialises v as JSON and writes it to the client-side WebSocket.
func (tc *testClient) send(t *testing.T, v interface{}) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("testClient.send: marshal: %v", err)
	}
	if err := tc.clientConn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("testClient.send: write: %v", err)
	}
}

// expectMsg blocks until a message of the given type arrives or the timeout
// fires.  It returns the decoded message map on success.
func (tc *testClient) expectMsg(t *testing.T, msgType string, timeout time.Duration) map[string]interface{} {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case m, ok := <-tc.msgs:
			if !ok {
				t.Fatalf("expectMsg(%s): channel closed", msgType)
			}
			if got, _ := m["type"].(string); got == msgType {
				return m
			}
			// Skip unrelated messages and keep draining.
		case <-deadline:
			t.Fatalf("expectMsg(%s): timed out after %s", msgType, timeout)
		}
	}
}

// drainUntil drains the channel until fn returns true or the timeout expires.
// Returns the matching message, or nil on timeout (does NOT call t.Fatal).
func (tc *testClient) drainUntil(
	_ *testing.T,
	fn func(map[string]interface{}) bool,
	timeout time.Duration,
) map[string]interface{} {
	deadline := time.After(timeout)
	for {
		select {
		case m, ok := <-tc.msgs:
			if !ok {
				return nil
			}
			if fn(m) {
				return m
			}
		case <-deadline:
			return nil
		}
	}
}

// close cleanly closes the client-side WebSocket connection.
func (tc *testClient) close(t *testing.T) {
	t.Helper()
	tc.closeOnce.Do(func() {
		_ = tc.clientConn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		tc.clientConn.Close()
	})
}

// msgTypeOf extracts the "type" field from a decoded message map.
func msgTypeOf(m map[string]interface{}) string {
	t, _ := m["type"].(string)
	return t
}

// ----------------------------------------------------------------------------
// Full lifecycle E2E test
// ----------------------------------------------------------------------------

// TestE2EFullGameLifecycle exercises the complete game flow:
//
//	lobby → generating → briefing → playing → ending → finished
//
// Two players connect over real WebSocket connections.  The AI layer is mocked
// to return deterministic responses so the test is fast and hermetic.
func TestE2EFullGameLifecycle(t *testing.T) {
	s := buildStack(t)
	defer s.teardown(t)

	msgTimeout := 5 * time.Second

	// -------------------------------------------------------------------------
	// Phase 1: Lobby – Player 1 connects and joins
	// -------------------------------------------------------------------------

	// Dial from the client side.  The server's OnConnection fires synchronously
	// inside HandleConnection and delivers the server-side conn to connCh.
	p1 := connectWS(t, s)
	defer p1.close(t)

	// Register with the session manager using the server-side conn.
	player1 := addPlayer(t, s, p1, "Alice")

	// Player 1 should receive "joined" with isHost=true.
	joinedMsg := p1.expectMsg(t, protocol.SMsgTypeJoined, msgTimeout)
	if isHost, _ := joinedMsg["isHost"].(bool); !isHost {
		t.Errorf("Player 1: expected isHost=true, got %v", joinedMsg["isHost"])
	}
	if pid, _ := joinedMsg["playerId"].(string); pid != player1.ID {
		t.Errorf("Player 1: joined.playerId mismatch: want %s, got %s", player1.ID, pid)
	}

	// Player 1 also receives lobby_update (broadcast inside AddPlayer).
	lobbyMsg := p1.expectMsg(t, protocol.SMsgTypeLobbyUpdate, msgTimeout)
	players1, _ := lobbyMsg["players"].([]interface{})
	if len(players1) != 1 {
		t.Errorf("Lobby after P1 join: expected 1 player, got %d", len(players1))
	}

	// -------------------------------------------------------------------------
	// Phase 2: Lobby – Player 2 connects
	// -------------------------------------------------------------------------

	p2 := connectWS(t, s)
	defer p2.close(t)

	player2 := addPlayer(t, s, p2, "Bob")

	// Player 2 receives "joined" with isHost=false.
	joined2 := p2.expectMsg(t, protocol.SMsgTypeJoined, msgTimeout)
	if isHost, _ := joined2["isHost"].(bool); isHost {
		t.Errorf("Player 2: expected isHost=false")
	}
	if pid, _ := joined2["playerId"].(string); pid != player2.ID {
		t.Errorf("Player 2: joined.playerId mismatch: want %s, got %s", player2.ID, pid)
	}

	// Both players receive a lobby_update with 2 players.
	lobbyP2 := p2.expectMsg(t, protocol.SMsgTypeLobbyUpdate, msgTimeout)
	lobbyPlayers2, _ := lobbyP2["players"].([]interface{})
	if len(lobbyPlayers2) != 2 {
		t.Errorf("Lobby after P2 join (P2 view): expected 2 players, got %d", len(lobbyPlayers2))
	}

	lobbyP1After := p1.expectMsg(t, protocol.SMsgTypeLobbyUpdate, msgTimeout)
	lobbyPlayers1After, _ := lobbyP1After["players"].([]interface{})
	if len(lobbyPlayers1After) != 2 {
		t.Errorf("Lobby after P2 join (P1 view): expected 2 players, got %d", len(lobbyPlayers1After))
	}

	// -------------------------------------------------------------------------
	// Phase 3: Host starts the game – world generation (mock AI, synchronous)
	// -------------------------------------------------------------------------

	if err := s.sessionMgr.StartGame(player1.ID, "mystery"); err != nil {
		t.Fatalf("StartGame: %v", err)
	}

	// Both players should receive briefing_public.
	p1.expectMsg(t, protocol.SMsgTypeBriefingPublic, msgTimeout)
	p2.expectMsg(t, protocol.SMsgTypeBriefingPublic, msgTimeout)

	// Each player receives their private briefing.
	p1.expectMsg(t, protocol.SMsgTypeBriefingPrivate, msgTimeout)
	p2.expectMsg(t, protocol.SMsgTypeBriefingPrivate, msgTimeout)

	// -------------------------------------------------------------------------
	// Phase 4: Both players signal ready → game_started
	// -------------------------------------------------------------------------

	s.sessionMgr.MarkPlayerReady(player1.ID)
	s.sessionMgr.MarkPlayerReady(player2.ID)

	// Both players receive game_started with an initial room view.
	gs1 := p1.expectMsg(t, protocol.SMsgTypeGameStarted, msgTimeout)
	gs2 := p2.expectMsg(t, protocol.SMsgTypeGameStarted, msgTimeout)

	for _, m := range []map[string]interface{}{gs1, gs2} {
		ir, _ := m["initialRoom"].(map[string]interface{})
		if ir == nil {
			t.Errorf("game_started missing initialRoom field: %v", m)
			continue
		}
		if roomName, _ := ir["name"].(string); roomName == "" {
			t.Errorf("game_started.initialRoom.name is empty")
		}
	}

	// Verify game status transitioned to playing.
	if status := s.sessionMgr.GetGameStatus(); status != types.GameStatusPlaying {
		t.Errorf("expected game status 'playing', got %q", status)
	}

	// -------------------------------------------------------------------------
	// Phase 5: Chat – Player 1 sends a chat; both players receive it
	// -------------------------------------------------------------------------

	p1.send(t, protocol.ClientMessage{Type: "chat", Content: "Hello, suspects!"})

	// Both players are in the same room (Library) so both receive the chat.
	p1Chat := p1.expectMsg(t, protocol.SMsgTypeChat, msgTimeout)
	if content, _ := p1Chat["content"].(string); content != "Hello, suspects!" {
		t.Errorf("chat content (P1): want 'Hello, suspects!', got %q", content)
	}
	if scope, _ := p1Chat["scope"].(string); scope != "room" {
		t.Errorf("chat scope (P1): want 'room', got %q", scope)
	}

	p2Chat := p2.expectMsg(t, protocol.SMsgTypeChat, msgTimeout)
	if content, _ := p2Chat["content"].(string); content != "Hello, suspects!" {
		t.Errorf("chat content (P2): want 'Hello, suspects!', got %q", content)
	}

	// -------------------------------------------------------------------------
	// Phase 6: Move – Player 1 moves to the adjacent room (Study)
	// -------------------------------------------------------------------------

	p1.send(t, protocol.ClientMessage{Type: "move", TargetRoomID: "Study"})

	// Player 1 receives room_changed.
	roomChanged := p1.expectMsg(t, protocol.SMsgTypeRoomChanged, msgTimeout)
	room, _ := roomChanged["room"].(map[string]interface{})
	if room == nil {
		t.Fatal("room_changed: missing 'room' field")
	}
	if roomName, _ := room["name"].(string); roomName != "Study" {
		t.Errorf("room_changed: expected 'Study', got %q", roomName)
	}

	// Both players receive a map_info broadcast after the move.
	p1.expectMsg(t, protocol.SMsgTypeMapInfo, msgTimeout)
	p2.expectMsg(t, protocol.SMsgTypeMapInfo, msgTimeout)

	// -------------------------------------------------------------------------
	// Phase 7: Examine – Player 1 examines something in the Study
	// -------------------------------------------------------------------------

	// Player 1 is now in the Study.
	target := "desk"
	p1.send(t, protocol.ClientMessage{Type: "examine", Target: &target})

	// Player 1 receives a "thinking" indicator first.
	p1.expectMsg(t, protocol.SMsgTypeThinking, msgTimeout)

	// Player 1 (and others in the same room) receives a game_event for the
	// examine_result published by the mock AI.
	p1.expectMsg(t, protocol.SMsgTypeGameEvent, msgTimeout)

	// -------------------------------------------------------------------------
	// Phase 8: End proposal flow
	// -------------------------------------------------------------------------

	// Move Player 1 back to Library so the propose_end message routes correctly.
	p1.send(t, protocol.ClientMessage{Type: "move", TargetRoomID: "Library"})
	p1.expectMsg(t, protocol.SMsgTypeRoomChanged, msgTimeout)
	p1.expectMsg(t, protocol.SMsgTypeMapInfo, msgTimeout)
	p2.expectMsg(t, protocol.SMsgTypeMapInfo, msgTimeout)

	// Player 1 proposes to end the game (routes via OnMessage → ActionProcessor
	// → EndConditionEngine.ProposeEnd).
	p1.send(t, protocol.ClientMessage{Type: "propose_end"})

	// isGameEvent returns true if a message is a game_event whose inner event
	// matches the given type string.
	isGameEvent := func(wantInnerType string) func(map[string]interface{}) bool {
		return func(m map[string]interface{}) bool {
			if msgTypeOf(m) != protocol.SMsgTypeGameEvent {
				return false
			}
			event, _ := m["event"].(map[string]interface{})
			if event == nil {
				return false
			}
			evType, _ := event["type"].(string)
			return evType == wantInnerType
		}
	}

	// Both players receive a game_event of type "end_proposed".
	// Drain any stale narration events that arrived from the move-back.
	endProposedP1 := p1.drainUntil(t, isGameEvent("end_proposed"), msgTimeout)
	endProposedP2 := p2.drainUntil(t, isGameEvent("end_proposed"), msgTimeout)
	if endProposedP1 == nil {
		t.Fatal("P1: did not receive game_event(end_proposed)")
	}
	if endProposedP2 == nil {
		t.Fatal("P2: did not receive game_event(end_proposed)")
	}

	// Player 2 agrees to end the game.
	p2.send(t, protocol.ClientMessage{Type: "end_vote", Agree: true})

	// With proposer (auto-agree) and P2 agreeing, the end proposal passes.
	// After the vote, each player receives (in unpredictable order):
	//   - game_event("end_vote_result")  – published synchronously before triggerEnding
	//   - game_finished                  – sent by FinishGame() which runs in
	//                                      triggerEnding's goroutine BEFORE the
	//                                      MessageRouter delivers game_ending
	//   - game_ending                    – delivered asynchronously via listenSendEndings
	//
	// Collect all three in a single time-bounded pass so that none are discarded.
	collectEndPhase := func(tc *testClient, label string) (endVoteResult, ending, finished map[string]interface{}) {
		deadline := time.After(msgTimeout)
		for {
			allFound := endVoteResult != nil && ending != nil && finished != nil
			if allFound {
				return
			}
			select {
			case m, ok := <-tc.msgs:
				if !ok {
					return
				}
				switch msgTypeOf(m) {
				case protocol.SMsgTypeGameEvent:
					event, _ := m["event"].(map[string]interface{})
					if event != nil {
						if et, _ := event["type"].(string); et == "end_vote_result" {
							endVoteResult = m
						}
					}
				case protocol.SMsgTypeGameEnding:
					ending = m
				case protocol.SMsgTypeGameFinished:
					finished = m
				}
			case <-deadline:
				t.Errorf("%s: end-phase timeout; endVoteResult=%v ending=%v finished=%v",
					label, endVoteResult != nil, ending != nil, finished != nil)
				return
			}
		}
	}

	endVoteP1, gameEnding1, gameFinished1 := collectEndPhase(p1, "P1")
	endVoteP2, gameEnding2, gameFinished2 := collectEndPhase(p2, "P2")

	if endVoteP1 == nil {
		t.Error("P1: did not receive game_event(end_vote_result)")
	}
	if endVoteP2 == nil {
		t.Error("P2: did not receive game_event(end_vote_result)")
	}
	if gameEnding1 == nil {
		t.Error("P1: did not receive game_ending")
	} else if cr, _ := gameEnding1["commonResult"].(string); cr == "" {
		t.Errorf("P1 game_ending: commonResult is empty: %v", gameEnding1)
	}
	if gameEnding2 == nil {
		t.Error("P2: did not receive game_ending")
	} else if cr, _ := gameEnding2["commonResult"].(string); cr == "" {
		t.Errorf("P2 game_ending: commonResult is empty: %v", gameEnding2)
	}
	if gameFinished1 == nil {
		t.Error("P1: did not receive game_finished")
	}
	if gameFinished2 == nil {
		t.Error("P2: did not receive game_finished")
	}

	// Verify final game status.
	if status := s.sessionMgr.GetGameStatus(); status != types.GameStatusFinished {
		t.Errorf("expected game status 'finished', got %q", status)
	}
}

// ----------------------------------------------------------------------------
// Focused sub-tests
// ----------------------------------------------------------------------------

// TestE2ELobbyJoinRejectsAfterGameStart verifies that a third player cannot
// join once the game has moved past the lobby phase.
func TestE2ELobbyJoinRejectsAfterGameStart(t *testing.T) {
	s := buildStack(t)
	defer s.teardown(t)

	msgTimeout := 3 * time.Second

	p1 := connectWS(t, s)
	defer p1.close(t)
	player1 := addPlayer(t, s, p1, "Alice")

	p2 := connectWS(t, s)
	defer p2.close(t)
	addPlayer(t, s, p2, "Bob")

	// Drain joined + lobby_update messages for both players.
	p1.expectMsg(t, protocol.SMsgTypeJoined, msgTimeout)
	p1.expectMsg(t, protocol.SMsgTypeLobbyUpdate, msgTimeout)
	p2.expectMsg(t, protocol.SMsgTypeJoined, msgTimeout)
	p2.expectMsg(t, protocol.SMsgTypeLobbyUpdate, msgTimeout)
	p1.expectMsg(t, protocol.SMsgTypeLobbyUpdate, msgTimeout)

	// Start the game.
	if err := s.sessionMgr.StartGame(player1.ID, ""); err != nil {
		t.Fatalf("StartGame: %v", err)
	}

	// Drain briefing messages.
	p1.expectMsg(t, protocol.SMsgTypeBriefingPublic, msgTimeout)
	p2.expectMsg(t, protocol.SMsgTypeBriefingPublic, msgTimeout)
	p1.expectMsg(t, protocol.SMsgTypeBriefingPrivate, msgTimeout)
	p2.expectMsg(t, protocol.SMsgTypeBriefingPrivate, msgTimeout)

	// A late player dials and tries to join after the game started.
	p3 := connectWS(t, s)
	defer p3.close(t)

	// Obtain the server-side conn without going through addPlayer.
	var p3ServerConn *websocket.Conn
	select {
	case p3ServerConn = <-s.connCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for P3 server conn")
	}

	_, joinErr := s.sessionMgr.AddPlayer(p3ServerConn, "Charlie")
	if joinErr == nil {
		t.Error("expected AddPlayer to fail when game is not in lobby; got nil error")
	}
}

// TestE2EStartGameRejectsNonHost ensures only the host can start the game.
func TestE2EStartGameRejectsNonHost(t *testing.T) {
	s := buildStack(t)
	defer s.teardown(t)

	p1 := connectWS(t, s)
	defer p1.close(t)
	addPlayer(t, s, p1, "Alice")

	p2 := connectWS(t, s)
	defer p2.close(t)
	player2 := addPlayer(t, s, p2, "Bob")

	// Non-host tries to start the game.
	if err := s.sessionMgr.StartGame(player2.ID, ""); err == nil {
		t.Error("expected StartGame to fail for non-host; got nil error")
	}
}

// TestE2EChatIsRoomScoped verifies that a chat message in one room is not
// delivered to a player who has moved to a different room.
func TestE2EChatIsRoomScoped(t *testing.T) {
	s := buildStack(t)
	defer s.teardown(t)

	msgTimeout := 5 * time.Second

	p1 := connectWS(t, s)
	defer p1.close(t)
	player1 := addPlayer(t, s, p1, "Alice")

	p2 := connectWS(t, s)
	defer p2.close(t)
	player2 := addPlayer(t, s, p2, "Bob")

	// Drain lobby messages.
	p1.expectMsg(t, protocol.SMsgTypeJoined, msgTimeout)
	p1.expectMsg(t, protocol.SMsgTypeLobbyUpdate, msgTimeout)
	p2.expectMsg(t, protocol.SMsgTypeJoined, msgTimeout)
	p2.expectMsg(t, protocol.SMsgTypeLobbyUpdate, msgTimeout)
	p1.expectMsg(t, protocol.SMsgTypeLobbyUpdate, msgTimeout)

	// Advance to playing.
	if err := s.sessionMgr.StartGame(player1.ID, "mystery"); err != nil {
		t.Fatalf("StartGame: %v", err)
	}
	p1.expectMsg(t, protocol.SMsgTypeBriefingPublic, msgTimeout)
	p2.expectMsg(t, protocol.SMsgTypeBriefingPublic, msgTimeout)
	p1.expectMsg(t, protocol.SMsgTypeBriefingPrivate, msgTimeout)
	p2.expectMsg(t, protocol.SMsgTypeBriefingPrivate, msgTimeout)

	s.sessionMgr.MarkPlayerReady(player1.ID)
	s.sessionMgr.MarkPlayerReady(player2.ID)

	p1.expectMsg(t, protocol.SMsgTypeGameStarted, msgTimeout)
	p2.expectMsg(t, protocol.SMsgTypeGameStarted, msgTimeout)

	// Move Player 2 to Study (different room from Player 1 who is in Library).
	p2.send(t, protocol.ClientMessage{Type: "move", TargetRoomID: "Study"})
	p2.expectMsg(t, protocol.SMsgTypeRoomChanged, msgTimeout)
	p1.expectMsg(t, protocol.SMsgTypeMapInfo, msgTimeout)
	p2.expectMsg(t, protocol.SMsgTypeMapInfo, msgTimeout)

	// Player 1 sends a room-scoped chat from Library.
	p1.send(t, protocol.ClientMessage{Type: "chat", Content: "Library secret"})

	// Player 1 should receive the chat.
	chatP1 := p1.expectMsg(t, protocol.SMsgTypeChat, msgTimeout)
	if content, _ := chatP1["content"].(string); content != "Library secret" {
		t.Errorf("P1 chat content: want 'Library secret', got %q", content)
	}

	// Player 2 is in Study and must NOT receive the Library chat.
	unexpected := p2.drainUntil(nil, func(m map[string]interface{}) bool {
		return msgTypeOf(m) == protocol.SMsgTypeChat
	}, 300*time.Millisecond)
	if unexpected != nil {
		t.Errorf("P2 received room-scoped chat from Library unexpectedly: %v", unexpected)
	}
}

// TestE2EMoveRejectsNonAdjacentRoom ensures the server returns an error when a
// player attempts to move to a room that is not adjacent to their current location.
func TestE2EMoveRejectsNonAdjacentRoom(t *testing.T) {
	s := buildStack(t)
	defer s.teardown(t)

	msgTimeout := 5 * time.Second

	p1 := connectWS(t, s)
	defer p1.close(t)
	player1 := addPlayer(t, s, p1, "Alice")

	p2 := connectWS(t, s)
	defer p2.close(t)
	player2 := addPlayer(t, s, p2, "Bob")

	// Drain lobby messages.
	p1.expectMsg(t, protocol.SMsgTypeJoined, msgTimeout)
	p1.expectMsg(t, protocol.SMsgTypeLobbyUpdate, msgTimeout)
	p2.expectMsg(t, protocol.SMsgTypeJoined, msgTimeout)
	p2.expectMsg(t, protocol.SMsgTypeLobbyUpdate, msgTimeout)
	p1.expectMsg(t, protocol.SMsgTypeLobbyUpdate, msgTimeout)

	if err := s.sessionMgr.StartGame(player1.ID, "mystery"); err != nil {
		t.Fatalf("StartGame: %v", err)
	}
	p1.expectMsg(t, protocol.SMsgTypeBriefingPublic, msgTimeout)
	p2.expectMsg(t, protocol.SMsgTypeBriefingPublic, msgTimeout)
	p1.expectMsg(t, protocol.SMsgTypeBriefingPrivate, msgTimeout)
	p2.expectMsg(t, protocol.SMsgTypeBriefingPrivate, msgTimeout)

	s.sessionMgr.MarkPlayerReady(player1.ID)
	s.sessionMgr.MarkPlayerReady(player2.ID)

	p1.expectMsg(t, protocol.SMsgTypeGameStarted, msgTimeout)
	p2.expectMsg(t, protocol.SMsgTypeGameStarted, msgTimeout)

	// Attempt to move to a non-existent room.
	p1.send(t, protocol.ClientMessage{Type: "move", TargetRoomID: "NonExistentRoom"})

	// Server should reply with an error message.
	errMsg := p1.expectMsg(t, protocol.SMsgTypeError, msgTimeout)
	if code, _ := errMsg["code"].(string); code != string(protocol.ErrorCodeInvalidMove) {
		t.Errorf("error code: want %s, got %q", protocol.ErrorCodeInvalidMove, code)
	}
}
