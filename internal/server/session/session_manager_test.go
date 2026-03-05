package session

import (
	"context"
	"testing"

	"github.com/anthropics/story/internal/server/aiface"
	"github.com/anthropics/story/internal/server/eventbus"
	"github.com/anthropics/story/internal/server/game"
	"github.com/anthropics/story/internal/server/mapengine"
	"github.com/anthropics/story/internal/server/network"
	"github.com/anthropics/story/internal/shared/schemas"
	"github.com/anthropics/story/internal/shared/types"
)

// mockAILayer implements aiface.AILayer for testing.
type mockAILayer struct {
	generateWorldFn func(ctx context.Context, settings types.GameSettings, playerCount int, themeKeyword string) (*types.World, error)
}

func (m *mockAILayer) GenerateWorld(ctx context.Context, settings types.GameSettings, playerCount int, themeKeyword string) (*types.World, error) {
	if m.generateWorldFn != nil {
		return m.generateWorldFn(ctx, settings, playerCount, themeKeyword)
	}
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

func newTestSessionManager() *SessionManager {
	net := network.NewNetworkServer(network.NetworkConfig{Port: 0})
	bus := eventbus.NewEventBus()
	me := mapengine.NewMapEngine()
	gs := game.NewGameStateManager(bus, me)
	ai := &mockAILayer{}
	return NewSessionManager(net, bus, gs, ai)
}

func TestCreateSession(t *testing.T) {
	sm := newTestSessionManager()
	code := sm.CreateSession()
	if code == "" {
		t.Fatal("expected non-empty room code")
	}
	if sm.GetGameStatus() != types.GameStatusLobby {
		t.Fatalf("expected lobby status, got %s", sm.GetGameStatus())
	}
	if sm.GetRoomCode() != code {
		t.Fatalf("expected room code %s, got %s", code, sm.GetRoomCode())
	}
}

func TestIsValidNickname(t *testing.T) {
	tests := []struct {
		name     string
		nickname string
		valid    bool
	}{
		{"valid", "Alice", true},
		{"empty", "", false},
		{"too long", "abcdefghijklmnopqrstuvwxyz", false},
		{"control char", "test\x00name", false},
		{"single char", "A", true},
		{"max length", "12345678901234567890", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidNickname(tt.nickname)
			if got != tt.valid {
				t.Fatalf("isValidNickname(%q) = %v, want %v", tt.nickname, got, tt.valid)
			}
		})
	}
}

func TestStartGameNotHost(t *testing.T) {
	sm := newTestSessionManager()
	sm.CreateSession()

	err := sm.StartGame("nonexistent", "")
	if err == nil {
		t.Fatal("expected error for non-host start")
	}
	if err != StartErrorNotHost {
		t.Fatalf("expected NOT_HOST error, got %v", err)
	}
}

func TestStartGameNotEnoughPlayers(t *testing.T) {
	sm := newTestSessionManager()
	sm.CreateSession()

	// Manually add a single player to be the host
	sm.mu.Lock()
	sm.game.HostID = "host1"
	sm.game.Players["host1"] = &types.Player{
		ID:       "host1",
		Nickname: "Host",
		IsHost:   true,
		Status:   "connected",
	}
	sm.mu.Unlock()

	err := sm.StartGame("host1", "")
	if err == nil {
		t.Fatal("expected error for not enough players")
	}
	if err != StartErrorNotEnoughPlayers {
		t.Fatalf("expected NOT_ENOUGH_PLAYERS error, got %v", err)
	}
}

func TestGetGameStatusNoSession(t *testing.T) {
	sm := newTestSessionManager()
	if sm.GetGameStatus() != "" {
		t.Fatal("expected empty status with no session")
	}
}

func TestStartEndingAndFinishGame(t *testing.T) {
	sm := newTestSessionManager()
	sm.CreateSession()

	sm.mu.Lock()
	sm.game.Status = types.GameStatusPlaying
	sm.mu.Unlock()

	sm.StartEnding()
	if sm.GetGameStatus() != types.GameStatusEnding {
		t.Fatalf("expected ending, got %s", sm.GetGameStatus())
	}

	sm.FinishGame()
	if sm.GetGameStatus() != types.GameStatusFinished {
		t.Fatalf("expected finished, got %s", sm.GetGameStatus())
	}
}

func TestIsHost(t *testing.T) {
	sm := newTestSessionManager()
	sm.CreateSession()

	sm.mu.Lock()
	sm.game.HostID = "host1"
	sm.mu.Unlock()

	if !sm.IsHost("host1") {
		t.Fatal("expected host1 to be host")
	}
	if sm.IsHost("other") {
		t.Fatal("expected other to not be host")
	}
}
