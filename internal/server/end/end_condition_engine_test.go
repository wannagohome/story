package end

import (
	"context"
	"testing"

	"github.com/anthropics/story/internal/server/aiface"
	"github.com/anthropics/story/internal/server/eventbus"
	"github.com/anthropics/story/internal/server/game"
	"github.com/anthropics/story/internal/server/mapengine"
	"github.com/anthropics/story/internal/shared/schemas"
	"github.com/anthropics/story/internal/shared/types"
)

// mockAILayer implements aiface.AILayer for testing.
type mockAILayer struct {
	judgeResult bool
}

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
	return m.judgeResult, nil
}

func (m *mockAILayer) GenerateEndings(_ context.Context, _ types.GameContext, _ string) (*schemas.Ending, error) {
	return &schemas.Ending{
		CommonResult:  "Test ending",
		PlayerEndings: []schemas.PlayerEndingSchema{},
	}, nil
}

var _ aiface.AILayer = (*mockAILayer)(nil)

// mockSessionNotifier implements SessionNotifier for testing.
type mockSessionNotifier struct {
	startEndingCalled bool
	finishGameCalled  bool
}

func (m *mockSessionNotifier) StartEnding()  { m.startEndingCalled = true }
func (m *mockSessionNotifier) FinishGame()   { m.finishGameCalled = true }

var _ SessionNotifier = (*mockSessionNotifier)(nil)

func newTestEngine() (*EndConditionEngine, *game.GameStateManager, *eventbus.EventBus, *mockSessionNotifier) {
	bus := eventbus.NewEventBus()
	me := mapengine.NewMapEngine()
	gs := game.NewGameStateManager(bus, me)
	ai := &mockAILayer{}
	sn := &mockSessionNotifier{}
	ece := NewEndConditionEngine(gs, ai, bus, sn)
	return ece, gs, bus, sn
}

func TestNewEndConditionEngine(t *testing.T) {
	ece, _, _, _ := newTestEngine()
	if ece == nil {
		t.Fatal("expected non-nil engine")
	}
}

func TestCastVoteNoActiveVote(t *testing.T) {
	ece, _, _, _ := newTestEngine()
	err := ece.CastVote("p1", "target1")
	if err != ErrNoActiveVote {
		t.Fatalf("expected ErrNoActiveVote, got %v", err)
	}
}

func TestSubmitSolutionNoActiveConsensus(t *testing.T) {
	ece, _, _, _ := newTestEngine()
	err := ece.SubmitSolution("p1", "answer")
	if err != ErrNoActiveConsensus {
		t.Fatalf("expected ErrNoActiveConsensus, got %v", err)
	}
}

func TestProposeEndAndDuplicate(t *testing.T) {
	ece, gs, _, _ := newTestEngine()

	gs.AddPlayer(&types.Player{ID: "p1", Nickname: "Alice", Status: "connected"})
	gs.AddPlayer(&types.Player{ID: "p2", Nickname: "Bob", Status: "connected"})

	err := ece.ProposeEnd("p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second proposal should fail
	err = ece.ProposeEnd("p2")
	if err != ErrEndVoteAlreadyOpen {
		t.Fatalf("expected ErrEndVoteAlreadyOpen, got %v", err)
	}
}

func TestRespondToEndProposalNoProposal(t *testing.T) {
	ece, _, _, _ := newTestEngine()
	err := ece.RespondToEndProposal("p1", true)
	if err != ErrNoEndProposal {
		t.Fatalf("expected ErrNoEndProposal, got %v", err)
	}
}

func TestStopMonitoring(t *testing.T) {
	ece, _, _, _ := newTestEngine()

	ece.StartMonitoring([]types.EndCondition{
		{ID: "ec1", TriggerType: "timeout", IsFallback: true},
	}, 20)

	ece.StopMonitoring()

	ece.mu.Lock()
	defer ece.mu.Unlock()
	if ece.monitoring {
		t.Fatal("expected monitoring to be false after stop")
	}
}

func TestBuildFallbackEnding(t *testing.T) {
	ece, gs, _, _ := newTestEngine()

	gs.AddPlayer(&types.Player{ID: "p1", Nickname: "Alice", Status: "connected"})
	gs.AddPlayer(&types.Player{ID: "p2", Nickname: "Bob", Status: "connected"})

	endData := ece.buildFallbackEnding("timeout")
	if endData.CommonResult == "" {
		t.Fatal("expected non-empty common result")
	}
	if len(endData.PlayerEndings) != 2 {
		t.Fatalf("expected 2 player endings, got %d", len(endData.PlayerEndings))
	}
}

func TestMatchEventCondition(t *testing.T) {
	ece, _, _, _ := newTestEngine()

	condition := types.EndCondition{
		TriggerType: "event",
		TriggerCriteria: map[string]interface{}{
			"eventType": "story_event",
		},
	}

	// Create a mock event that matches
	mockEvent := &testGameEvent{eventType: "story_event"}
	if !ece.matchEventCondition(mockEvent, condition) {
		t.Fatal("expected condition to match")
	}

	// Non-matching event
	mockEvent2 := &testGameEvent{eventType: "player_move"}
	if ece.matchEventCondition(mockEvent2, condition) {
		t.Fatal("expected condition to not match")
	}
}

func TestMatchEventConditionNilCriteria(t *testing.T) {
	ece, _, _, _ := newTestEngine()

	condition := types.EndCondition{
		TriggerType:     "event",
		TriggerCriteria: nil,
	}

	mockEvent := &testGameEvent{eventType: "story_event"}
	if ece.matchEventCondition(mockEvent, condition) {
		t.Fatal("expected condition with nil criteria to not match")
	}
}

func TestEndConditionError(t *testing.T) {
	err := ErrNoActiveVote
	if err.Error() != "no active vote" {
		t.Fatalf("expected 'no active vote', got %q", err.Error())
	}
}

// testGameEvent is a test helper that implements types.GameEvent.
type testGameEvent struct {
	eventType string
}

func (e *testGameEvent) EventType() string       { return e.eventType }
func (e *testGameEvent) GetBaseEvent() types.BaseEvent { return types.BaseEvent{} }
