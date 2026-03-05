package ai

import (
	"context"
	"fmt"

	"github.com/anthropics/story/internal/ai/provider"
	"github.com/anthropics/story/internal/shared/schemas"
	"github.com/anthropics/story/internal/shared/types"
)

// AILayer wraps all AI sub-modules and provides the unified AI interface.
type AILayer struct {
	worldGen *WorldGenerator
	gm       *GMEngine
	npc      *NPCEngine
	judge    *EndJudge
}

// NewAILayer creates a new AILayer using the given provider registry.
// It uses the "runtime" provider for all runtime AI operations and
// the "worldgen" provider for world generation. If a named provider
// is not found, it falls back to the first available provider.
func NewAILayer(registry *provider.ProviderRegistry) (*AILayer, error) {
	runtimeProvider, err := getProviderWithFallback(registry, "runtime")
	if err != nil {
		return nil, fmt.Errorf("no AI provider available: %w", err)
	}

	worldgenProvider, err := getProviderWithFallback(registry, "worldgen")
	if err != nil {
		// Fall back to runtime provider for world generation
		worldgenProvider = runtimeProvider
	}

	return &AILayer{
		worldGen: NewWorldGenerator(worldgenProvider),
		gm:       NewGMEngine(runtimeProvider),
		npc:      NewNPCEngine(runtimeProvider),
		judge:    NewEndJudge(runtimeProvider),
	}, nil
}

// NewAILayerWithProvider creates a new AILayer using a single AI provider
// for all sub-modules.
func NewAILayerWithProvider(p provider.AIProvider) *AILayer {
	return &AILayer{
		worldGen: NewWorldGenerator(p),
		gm:       NewGMEngine(p),
		npc:      NewNPCEngine(p),
		judge:    NewEndJudge(p),
	}
}

// GenerateWorld creates a new game world using AI.
func (l *AILayer) GenerateWorld(
	ctx context.Context,
	settings types.GameSettings,
	playerCount int,
	themeKeyword string,
) (*types.World, error) {
	return l.worldGen.Generate(ctx, settings, playerCount, themeKeyword)
}

// EvaluateExamine evaluates an /examine action.
func (l *AILayer) EvaluateExamine(
	ctx context.Context,
	gameCtx types.GameContext,
	target string,
) (*schemas.GameResponse, error) {
	return l.gm.EvaluateExamine(ctx, gameCtx, target)
}

// EvaluateAction evaluates a /do action.
func (l *AILayer) EvaluateAction(
	ctx context.Context,
	gameCtx types.GameContext,
	action string,
) (*schemas.GameResponse, error) {
	return l.gm.EvaluateAction(ctx, gameCtx, action)
}

// TalkToNPC generates an NPC response to a player message.
func (l *AILayer) TalkToNPC(
	ctx context.Context,
	gameCtx types.GameContext,
	npcID string,
	message string,
) (*schemas.NPCResponse, error) {
	return l.npc.TalkToNPC(ctx, gameCtx, npcID, message)
}

// JudgeEndCondition evaluates whether an end condition has been met.
func (l *AILayer) JudgeEndCondition(
	ctx context.Context,
	gameCtx types.GameContext,
	condition types.EndCondition,
) (bool, error) {
	return l.judge.JudgeEndCondition(ctx, gameCtx, condition)
}

// GenerateEndings generates personalized endings for all players.
func (l *AILayer) GenerateEndings(
	ctx context.Context,
	gameCtx types.GameContext,
	reason string,
) (*schemas.Ending, error) {
	return l.judge.GenerateEndings(ctx, gameCtx, reason)
}

// GenerateNarration generates a GM narration for a given trigger.
func (l *AILayer) GenerateNarration(
	ctx context.Context,
	gameCtx types.GameContext,
	trigger string,
) (string, error) {
	return l.gm.GenerateNarration(ctx, gameCtx, trigger)
}

// getProviderWithFallback tries to get a named provider, falling back to the
// first available provider if not found.
func getProviderWithFallback(registry *provider.ProviderRegistry, name string) (provider.AIProvider, error) {
	p, err := registry.Get(name)
	if err == nil {
		return p, nil
	}
	// Fall back to first available
	available := registry.Available()
	if len(available) == 0 {
		return nil, fmt.Errorf("no providers registered")
	}
	return registry.Get(available[0])
}
