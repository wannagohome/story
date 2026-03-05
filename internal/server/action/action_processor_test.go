package action

import (
	"context"
	"testing"

	"github.com/anthropics/story/internal/server/aiface"
	"github.com/anthropics/story/internal/server/end"
	"github.com/anthropics/story/internal/server/eventbus"
	"github.com/anthropics/story/internal/server/game"
	"github.com/anthropics/story/internal/server/mapengine"
	"github.com/anthropics/story/internal/server/message"
	"github.com/anthropics/story/internal/server/network"
	"github.com/anthropics/story/internal/shared/protocol"
	"github.com/anthropics/story/internal/shared/schemas"
	"github.com/anthropics/story/internal/shared/types"
)

// mockAILayer implements aiface.AILayer for testing.
type mockAILayer struct{}

func (m *mockAILayer) GenerateWorld(_ context.Context, _ types.GameSettings, _ int, _ string) (*types.World, error) {
	return &types.World{}, nil
}

func (m *mockAILayer) EvaluateExamine(_ context.Context, _ types.GameContext, _ string) (*schemas.GameResponse, error) {
	return &schemas.GameResponse{}, nil
}

func (m *mockAILayer) EvaluateAction(_ context.Context, _ types.GameContext, _ string) (*schemas.GameResponse, error) {
	return &schemas.GameResponse{}, nil
}

func (m *mockAILayer) TalkToNPC(_ context.Context, _ types.GameContext, _ string, _ string) (*schemas.NPCResponse, error) {
	return &schemas.NPCResponse{}, nil
}

func (m *mockAILayer) JudgeEndCondition(_ context.Context, _ types.GameContext, _ types.EndCondition) (bool, error) {
	return false, nil
}

func (m *mockAILayer) GenerateEndings(_ context.Context, _ types.GameContext, _ string) (*schemas.Ending, error) {
	return &schemas.Ending{}, nil
}

var _ aiface.AILayer = (*mockAILayer)(nil)

// mockSessionNotifier implements end.SessionNotifier for testing.
type mockSessionNotifier struct {
	startEndingCalled bool
	finishGameCalled  bool
}

func (m *mockSessionNotifier) StartEnding()  { m.startEndingCalled = true }
func (m *mockSessionNotifier) FinishGame()   { m.finishGameCalled = true }

var _ end.SessionNotifier = (*mockSessionNotifier)(nil)

func newTestActionProcessor() (*ActionProcessor, *game.GameStateManager, *eventbus.EventBus) {
	net := network.NewNetworkServer(network.NetworkConfig{Port: 0})
	bus := eventbus.NewEventBus()
	me := mapengine.NewMapEngine()
	gs := game.NewGameStateManager(bus, me)
	ai := &mockAILayer{}
	sn := &mockSessionNotifier{}
	ece := end.NewEndConditionEngine(gs, ai, bus, sn)
	mr := message.NewMessageRouter(net, gs, bus)
	ap := NewActionProcessor(gs, me, ai, ece, bus, net, mr)
	return ap, gs, bus
}

func TestProcessMessageUnknownType(t *testing.T) {
	ap, _, _ := newTestActionProcessor()
	err := ap.ProcessMessage("p1", protocol.ClientMessage{Type: "unknown"})
	if err == nil {
		t.Fatal("expected error for unknown message type")
	}
}

func TestHandleChat(t *testing.T) {
	ap, gs, bus := newTestActionProcessor()

	gs.AddPlayer(&types.Player{
		ID:            "p1",
		Nickname:      "Alice",
		Status:        "connected",
		CurrentRoomID: "room1",
	})

	// Subscribe to chat to verify publishing
	chatCh := bus.SubscribeChat()

	err := ap.ProcessMessage("p1", protocol.ClientMessage{
		Type:    "chat",
		Content: "Hello!",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case chat := <-chatCh:
		if chat.SenderID != "p1" {
			t.Fatalf("expected sender p1, got %s", chat.SenderID)
		}
		if chat.Scope != "room" {
			t.Fatalf("expected scope room, got %s", chat.Scope)
		}
		if chat.Content != "Hello!" {
			t.Fatalf("expected content Hello!, got %s", chat.Content)
		}
	default:
		t.Fatal("expected chat event to be published")
	}
}

func TestHandleShout(t *testing.T) {
	ap, gs, bus := newTestActionProcessor()

	gs.AddPlayer(&types.Player{
		ID:            "p1",
		Nickname:      "Alice",
		Status:        "connected",
		CurrentRoomID: "room1",
	})

	chatCh := bus.SubscribeChat()

	err := ap.ProcessMessage("p1", protocol.ClientMessage{
		Type:    "shout",
		Content: "Hey everyone!",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case chat := <-chatCh:
		if chat.Scope != "global" {
			t.Fatalf("expected scope global, got %s", chat.Scope)
		}
	default:
		t.Fatal("expected chat event to be published")
	}
}

func TestHandleGiveNotSupported(t *testing.T) {
	ap, gs, _ := newTestActionProcessor()

	gs.AddPlayer(&types.Player{
		ID:       "p1",
		Nickname: "Alice",
		Status:   "connected",
	})

	// Should not return error, just sends an error message to client
	err := ap.ProcessMessage("p1", protocol.ClientMessage{Type: "give"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleMoveInvalidRoom(t *testing.T) {
	ap, gs, _ := newTestActionProcessor()

	gs.AddPlayer(&types.Player{
		ID:            "p1",
		Nickname:      "Alice",
		Status:        "connected",
		CurrentRoomID: "room1",
	})

	// Move to nonexistent room - should not return error
	err := ap.ProcessMessage("p1", protocol.ClientMessage{
		Type:         "move",
		TargetRoomID: "nonexistent",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildGameContext(t *testing.T) {
	ap, gs, _ := newTestActionProcessor()

	gs.AddPlayer(&types.Player{
		ID:            "p1",
		Nickname:      "Alice",
		Status:        "connected",
		CurrentRoomID: "room1",
	})

	ctx := ap.buildGameContext("p1")
	if ctx.RequestingPlayer.ID != "p1" {
		t.Fatalf("expected requesting player p1, got %s", ctx.RequestingPlayer.ID)
	}
}

func TestToInterfaceSlice(t *testing.T) {
	result := toInterfaceSlice(nil)
	if len(result) != 0 {
		t.Fatalf("expected empty slice, got %d elements", len(result))
	}
}
