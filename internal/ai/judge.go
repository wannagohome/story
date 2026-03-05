package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/story/internal/ai/provider"
	"github.com/anthropics/story/internal/shared/schemas"
	"github.com/anthropics/story/internal/shared/types"
)

// EndJudge handles AI-based end condition judgment and ending generation.
type EndJudge struct {
	provider provider.AIProvider
}

// NewEndJudge creates a new EndJudge with the given AI provider.
func NewEndJudge(p provider.AIProvider) *EndJudge {
	return &EndJudge{provider: p}
}

// endJudgment is the AI response for end condition evaluation.
type endJudgment struct {
	ShouldEnd bool   `json:"shouldEnd"`
	Reason    string `json:"reason"`
}

// JudgeEndCondition evaluates whether an end condition has been met.
func (j *EndJudge) JudgeEndCondition(
	ctx context.Context,
	gameCtx types.GameContext,
	condition types.EndCondition,
) (bool, error) {
	criteriaJSON, _ := json.Marshal(condition.TriggerCriteria)

	discoveredClues := countDiscoveredClues(gameCtx)

	prompt := fmt.Sprintf(`[End Condition]
%s
Trigger criteria: %s

[Current Game State]
Elapsed: %d min / %d min
Discovered clues: %d / %d

[Judgment]
Determine if this end condition has been met.
Provide shouldEnd: true/false and your reasoning.`,
		condition.Description, string(criteriaJSON),
		gameCtx.CurrentState.ElapsedTime/60, gameCtx.World.GameStructure.EstimatedDuration,
		discoveredClues, len(gameCtx.World.Clues),
	)

	raw, err := j.provider.GenerateStructured(ctx, provider.StructuredRequest{
		SystemPrompt: endJudgeSystemPrompt,
		UserPrompt:   prompt,
		Temperature:  0.3,
		MaxTokens:    300,
	})
	if err != nil {
		return false, fmt.Errorf("end condition judgment failed: %w", err)
	}

	var judgment endJudgment
	if err := json.Unmarshal(raw, &judgment); err != nil {
		return false, fmt.Errorf("end judgment parse failed: %w", err)
	}
	return judgment.ShouldEnd, nil
}

// GenerateEndings generates personalized endings for all players.
func (j *EndJudge) GenerateEndings(
	ctx context.Context,
	gameCtx types.GameContext,
	reason string,
) (*schemas.Ending, error) {
	prompt := buildEndingPrompt(gameCtx, reason)

	raw, err := j.provider.GenerateStructured(ctx, provider.StructuredRequest{
		SystemPrompt: endingSystemPrompt,
		UserPrompt:   prompt,
		Temperature:  0.9,
		MaxTokens:    3000,
	})
	if err != nil {
		return nil, fmt.Errorf("ending generation failed: %w", err)
	}

	var ending schemas.Ending
	if err := json.Unmarshal(raw, &ending); err != nil {
		return nil, fmt.Errorf("ending parse failed: %w", err)
	}
	return &ending, nil
}

// buildEndingPrompt constructs the prompt for ending generation.
func buildEndingPrompt(gameCtx types.GameContext, endReason string) string {
	var playerInfo strings.Builder
	for _, role := range gameCtx.World.PlayerRoles {
		fmt.Fprintf(&playerInfo, "\nPlayer: %s (ID: %s)\n", role.CharacterName, role.ID)
		fmt.Fprintf(&playerInfo, "Background: %s\n", role.Background)
		fmt.Fprintf(&playerInfo, "Secret: %s\n", role.Secret)
		fmt.Fprintf(&playerInfo, "Goals:\n")
		for _, g := range role.PersonalGoals {
			fmt.Fprintf(&playerInfo, "  - %s (ID: %s)\n", g.Description, g.ID)
		}
	}

	timeoutNote := ""
	if endReason == "timeout" {
		timeoutNote = "\n[NOTE] Game ended by timeout. Provide closure even if mysteries remain unsolved. Make the ending feel satisfying despite unresolved elements.\n"
	}

	return fmt.Sprintf(`[World]
%s - %s

[Game Result]
End reason: %s
%s
[Player Roles and Goals]
%s
[Instructions]
1. Write a dramatic common result (3-5 sentences)
2. For each player:
   - Summarize their journey (2-3 sentences)
   - Evaluate each personal goal (achieved/failed with evidence)
   - Write a personalized ending narrative (3-5 sentences)
3. Create catharsis — players should think "so THAT'S why..."`,
		gameCtx.World.Title, gameCtx.World.Synopsis,
		endReason,
		timeoutNote,
		playerInfo.String(),
	)
}
