// storyserver starts a headless game server for testing.
// It prints the server info as JSON to stdout, then waits for the game to finish.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
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
)

func main() {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY not set")
	}

	p, err := provider.NewProviderFromConfig(provider.ProviderConfig{
		Type:   "anthropic",
		APIKey: apiKey,
		Model:  "claude-sonnet-4-20250514",
	})
	if err != nil {
		log.Fatalf("provider: %v", err)
	}
	aiLayer := ai.NewAILayerWithProvider(p)

	bus := eventbus.NewEventBus()
	netSrv := network.NewNetworkServer(network.NetworkConfig{Port: 0})
	me := mapengine.NewMapEngine()
	gs := game.NewGameStateManager(bus, me)
	sm := session.NewSessionManager(netSrv, bus, gs, aiLayer)
	ece := end.NewEndConditionEngine(gs, aiLayer, bus, sm)
	sm.SetEndConditionEngine(ece)
	mr := message.NewMessageRouter(netSrv, gs, bus)
	ap := action.NewActionProcessor(gs, me, aiLayer, ece, bus, netSrv, mr)

	netSrv.OnConnection(func(conn *websocket.Conn) {})

	// Handle messages from connections not yet bound to a player (e.g. "join").
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
			time.Sleep(5 * time.Second)
			status := sm.GetGameStatus()
			if status == "finished" {
				slog.Info("game finished, shutting down in 5s")
				time.Sleep(5 * time.Second)
				os.Exit(0)
			}
		}
	}()

	<-interrupt
	slog.Info("shutting down")
	netSrv.Stop()
}
