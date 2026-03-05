package message

import (
	"testing"

	"github.com/anthropics/story/internal/server/eventbus"
	"github.com/anthropics/story/internal/server/game"
	"github.com/anthropics/story/internal/server/mapengine"
	"github.com/anthropics/story/internal/server/network"
	"github.com/anthropics/story/internal/shared/types"
)

func newTestMessageRouter() (*MessageRouter, *network.NetworkServer, *game.GameStateManager, *eventbus.EventBus) {
	net := network.NewNetworkServer(network.NetworkConfig{Port: 0})
	bus := eventbus.NewEventBus()
	me := mapengine.NewMapEngine()
	gs := game.NewGameStateManager(bus, me)
	mr := NewMessageRouter(net, gs, bus)
	return mr, net, gs, bus
}

func TestResolveRecipientsAll(t *testing.T) {
	mr, _, gs, _ := newTestMessageRouter()

	gs.AddPlayer(&types.Player{ID: "p1", Nickname: "Alice", Status: "connected"})
	gs.AddPlayer(&types.Player{ID: "p2", Nickname: "Bob", Status: "connected"})

	vis := types.EventVisibility{Scope: "all"}
	recipients := mr.resolveRecipients(vis)
	if len(recipients) != 2 {
		t.Fatalf("expected 2 recipients, got %d", len(recipients))
	}
}

func TestResolveRecipientsRoom(t *testing.T) {
	mr, _, gs, _ := newTestMessageRouter()

	gs.AddPlayer(&types.Player{ID: "p1", Nickname: "Alice", Status: "connected", CurrentRoomID: "room1"})
	gs.AddPlayer(&types.Player{ID: "p2", Nickname: "Bob", Status: "connected", CurrentRoomID: "room2"})

	vis := types.EventVisibility{Scope: "room", RoomID: "room1"}
	recipients := mr.resolveRecipients(vis)
	if len(recipients) != 1 {
		t.Fatalf("expected 1 recipient, got %d", len(recipients))
	}
	if recipients[0] != "p1" {
		t.Fatalf("expected p1, got %s", recipients[0])
	}
}

func TestResolveRecipientsPlayers(t *testing.T) {
	mr, _, _, _ := newTestMessageRouter()

	vis := types.EventVisibility{Scope: "players", PlayerIDs: []string{"p1", "p3"}}
	recipients := mr.resolveRecipients(vis)
	if len(recipients) != 2 {
		t.Fatalf("expected 2 recipients, got %d", len(recipients))
	}
}

func TestResolveRecipientsUnknownScope(t *testing.T) {
	mr, _, _, _ := newTestMessageRouter()

	vis := types.EventVisibility{Scope: "unknown"}
	recipients := mr.resolveRecipients(vis)
	if recipients != nil {
		t.Fatalf("expected nil recipients, got %v", recipients)
	}
}
