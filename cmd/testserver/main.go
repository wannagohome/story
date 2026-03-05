// testserver starts a headless game server with a mock AI provider.
// It prints server info as JSON to stdout. Players join via "join" messages.
// Used for L4 subagent testing without real AI API calls.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
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

// mockAI implements aiface.AILayer with deterministic responses.
type mockAI struct{}

var _ aiface.AILayer = (*mockAI)(nil)

func (m *mockAI) GenerateWorld(_ context.Context, _ types.GameSettings, playerCount int, _ string) (*types.World, error) {
	rooms := []types.Room{
		{ID: "room-foyer", Name: "Foyer", Description: "A grand entrance hall.", Type: "public", Items: []types.Item{}, NPCIDs: []string{"npc-butler"}, ClueIDs: []string{"clue-letter"}},
		{ID: "room-library", Name: "Library", Description: "Dusty library with tall shelves.", Type: "public", Items: []types.Item{}, NPCIDs: []string{}, ClueIDs: []string{}},
		{ID: "room-study", Name: "Study", Description: "A small study with a desk.", Type: "private", Items: []types.Item{}, NPCIDs: []string{}, ClueIDs: []string{}},
		{ID: "room-garden", Name: "Garden", Description: "An overgrown garden.", Type: "public", Items: []types.Item{}, NPCIDs: []string{}, ClueIDs: []string{}},
	}
	connections := []types.Connection{
		{RoomA: "room-foyer", RoomB: "room-library", Bidirectional: true},
		{RoomA: "room-foyer", RoomB: "room-study", Bidirectional: true},
		{RoomA: "room-foyer", RoomB: "room-garden", Bidirectional: true},
		{RoomA: "room-library", RoomB: "room-study", Bidirectional: true},
	}

	roles := make([]types.PlayerRole, playerCount)
	names := []string{"Detective Harris", "Lady Pemberton", "Professor Oak", "Captain Nemo", "Dr. Watson", "Miss Scarlet"}
	for i := 0; i < playerCount; i++ {
		name := names[i%len(names)]
		roles[i] = types.PlayerRole{
			ID:            fmt.Sprintf("role-%d", i),
			CharacterName: name,
			Background:    fmt.Sprintf("A character in the mystery (%s).", name),
			Secret:        fmt.Sprintf("Secret of %s.", name),
			PersonalGoals: []types.PersonalGoal{{
				ID:             fmt.Sprintf("goal-%d", i),
				Description:    "Solve the mystery",
				EvaluationHint: "Did the player solve it?",
				EntityRefs:     []string{},
			}},
			Relationships: []types.Relationship{},
		}
	}

	world := &types.World{
		Title:      "The Vanishing Heir",
		Synopsis:   "A mystery set in a grand manor.",
		Atmosphere: "gothic",
		GameStructure: types.GameStructure{
			Concept:           "murder mystery",
			CoreConflict:      "who stole the heir",
			ProgressionStyle:  "investigation",
			EstimatedDuration: 20,
			EndConditions: []types.EndCondition{{
				ID: "end-propose", Description: "Players propose end", TriggerType: "timeout", IsFallback: true,
			}},
			WinConditions: []types.WinCondition{{Description: "Identify the culprit", EvaluationCriteria: "vote result"}},
		},
		Map:         types.GameMap{Rooms: rooms, Connections: connections},
		PlayerRoles: roles,
		NPCs: []types.NPC{{
			ID: "npc-butler", Name: "Mr Hobbs", CurrentRoomID: "room-foyer",
			Persona: "nervous butler", KnownInfo: []string{"the heir was last seen at midnight"},
			HiddenInfo: []string{"he saw the heir leave"}, BehaviorPrinciple: "protective",
			InitialTrust: 0.5,
		}},
		Clues: []types.Clue{{
			ID: "clue-letter", Name: "Torn Letter", Description: "A letter torn into pieces.",
			RoomID: "room-foyer", DiscoverCondition: "examine the fireplace", RelatedClueIDs: []string{},
		}},
		Gimmicks: []types.Gimmick{},
		Information: types.InformationLayers{
			Public: types.PublicInfo{
				Title: "The Vanishing Heir", Synopsis: "A mystery set in a grand manor.",
				CharacterList: []types.CharacterListEntry{},
				Relationships: "Complex relationships.", MapOverview: "A manor with 4 rooms.",
				NPCList:   []types.NPCListEntry{{Name: "Mr Hobbs", Location: "Foyer"}},
				GameRules: "Investigate, talk, and propose an end.",
			},
			SemiPublic: []types.SemiPublicInfo{},
			Private:    []types.PrivateInfo{},
		},
	}
	return world, nil
}

func (m *mockAI) EvaluateExamine(_ context.Context, _ types.GameContext, target string) (*schemas.GameResponse, error) {
	evt, _ := json.Marshal(schemas.AIExamineResultEvent{
		Type: "examine_result",
		Data: schemas.AIExamineResultEventData{Target: target, Description: "You find nothing unusual.", ClueFound: false},
	})
	return &schemas.GameResponse{Events: []json.RawMessage{evt}, StateChanges: []json.RawMessage{}}, nil
}

func (m *mockAI) EvaluateAction(_ context.Context, _ types.GameContext, act string) (*schemas.GameResponse, error) {
	evt, _ := json.Marshal(schemas.AIActionResultEvent{
		Type: "action_result",
		Data: schemas.AIActionResultEventData{Action: act, Result: "Nothing notable happens.", TriggeredEvents: []string{}},
	})
	return &schemas.GameResponse{Events: []json.RawMessage{evt}, StateChanges: []json.RawMessage{}}, nil
}

func (m *mockAI) TalkToNPC(_ context.Context, _ types.GameContext, _ string, _ string) (*schemas.NPCResponse, error) {
	return &schemas.NPCResponse{
		Dialogue: "I cannot say anything further.", Emotion: "nervous",
		InternalThought: "I must protect the family.", InfoRevealed: []string{},
		TrustChange: 0, TriggeredGimmick: false, Events: []json.RawMessage{},
	}, nil
}

func (m *mockAI) JudgeEndCondition(_ context.Context, _ types.GameContext, _ types.EndCondition) (bool, error) {
	return false, nil
}

func (m *mockAI) GenerateEndings(_ context.Context, _ types.GameContext, _ string) (*schemas.Ending, error) {
	return &schemas.Ending{
		CommonResult:  "The mystery remains unsolved.",
		PlayerEndings: []schemas.PlayerEndingSchema{},
	}, nil
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	ai := &mockAI{}
	bus := eventbus.NewEventBus()
	netSrv := network.NewNetworkServer(network.NetworkConfig{Port: 0})
	me := mapengine.NewMapEngine()
	gs := game.NewGameStateManager(bus, me)
	sm := session.NewSessionManager(netSrv, bus, gs, ai)
	ece := end.NewEndConditionEngine(gs, ai, bus, sm)
	sm.SetEndConditionEngine(ece)
	mr := message.NewMessageRouter(netSrv, gs, bus)
	ap := action.NewActionProcessor(gs, me, ai, ece, bus, netSrv, mr)

	netSrv.OnConnection(func(conn *websocket.Conn) {})

	netSrv.OnUnboundMessage(func(conn *websocket.Conn, msg protocol.ClientMessage) {
		if msg.Type == "join" && msg.Nickname != "" {
			player, err := sm.AddPlayer(conn, msg.Nickname)
			if err != nil {
				slog.Error("AddPlayer error", "nickname", msg.Nickname, "error", err)
				return
			}
			slog.Info("player joined", "id", player.ID, "nickname", player.Nickname, "isHost", player.IsHost)
		}
	})

	netSrv.OnMessage(func(playerID string, msg protocol.ClientMessage) {
		switch msg.Type {
		case "start_game":
			if err := sm.StartGame(playerID, msg.ThemeKeyword); err != nil {
				slog.Error("StartGame error", "error", err)
			}
		case "ready":
			sm.MarkPlayerReady(playerID)
		default:
			if err := ap.ProcessMessage(playerID, msg); err != nil {
				slog.Error("ProcessMessage error", "type", msg.Type, "error", err)
			}
		}
	})

	netSrv.OnDisconnection(func(playerID string) {
		sm.RemovePlayer(playerID)
	})

	roomCode := sm.CreateSession()
	if err := netSrv.Start(roomCode); err != nil {
		log.Fatalf("server start failed: %v", err)
	}
	port := netSrv.Port()

	info := map[string]interface{}{
		"port":     port,
		"roomCode": roomCode,
		"wsURL":    fmt.Sprintf("ws://localhost:%d/ws/%s", port, roomCode),
	}
	infoJSON, _ := json.Marshal(info)
	fmt.Println(string(infoJSON))

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	go func() {
		for {
			time.Sleep(2 * time.Second)
			status := sm.GetGameStatus()
			if status == "finished" {
				slog.Info("game finished, shutting down in 2s")
				time.Sleep(2 * time.Second)
				os.Exit(0)
			}
		}
	}()

	<-interrupt
	slog.Info("shutting down")
	netSrv.Stop()
}
