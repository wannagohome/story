package aiface

import (
	"context"

	"github.com/anthropics/story/internal/shared/schemas"
	"github.com/anthropics/story/internal/shared/types"
)

// AILayer is the interface for AI-related operations.
// The real implementation lives in the ai package; this interface
// breaks the dependency cycle between server modules and the AI layer.
type AILayer interface {
	GenerateWorld(ctx context.Context, settings types.GameSettings, playerCount int, themeKeyword string) (*types.World, error)
	EvaluateExamine(ctx context.Context, gameCtx types.GameContext, target string) (*schemas.GameResponse, error)
	EvaluateAction(ctx context.Context, gameCtx types.GameContext, action string) (*schemas.GameResponse, error)
	TalkToNPC(ctx context.Context, gameCtx types.GameContext, npcID string, message string) (*schemas.NPCResponse, error)
	JudgeEndCondition(ctx context.Context, gameCtx types.GameContext, condition types.EndCondition) (bool, error)
	GenerateEndings(ctx context.Context, gameCtx types.GameContext, reason string) (*schemas.Ending, error)
}
