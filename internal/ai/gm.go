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

// GMEngine handles GM narration and action/examine evaluation.
type GMEngine struct {
	provider provider.AIProvider
}

// NewGMEngine creates a new GMEngine with the given AI provider.
func NewGMEngine(p provider.AIProvider) *GMEngine {
	return &GMEngine{provider: p}
}

// EvaluateExamine evaluates an /examine action and returns the AI response.
func (g *GMEngine) EvaluateExamine(
	ctx context.Context,
	gameCtx types.GameContext,
	target string,
) (*schemas.GameResponse, error) {
	room := gameCtx.CurrentRoom

	// Find undiscovered clues in this room
	var undiscovered []string
	for _, c := range gameCtx.World.Clues {
		if c.RoomID == room.ID {
			if cs, ok := gameCtx.CurrentState.ClueStates[c.ID]; !ok || !cs.IsDiscovered {
				undiscovered = append(undiscovered,
					fmt.Sprintf("- %s: %s (discover condition: %s)", c.Name, c.Description, c.DiscoverCondition))
			}
		}
	}

	var items []string
	for _, item := range room.Items {
		items = append(items, item.Name)
	}

	targetLine := "Examining the entire room."
	if target != "" {
		targetLine = fmt.Sprintf("Examining: %s", target)
	}

	characterName := ""
	var personalGoals []string
	if gameCtx.RequestingPlayer.Role != nil {
		characterName = gameCtx.RequestingPlayer.Role.CharacterName
		for _, g := range gameCtx.RequestingPlayer.Role.PersonalGoals {
			personalGoals = append(personalGoals, g.Description)
		}
	}

	prompt := fmt.Sprintf(`[Situation]
%s examines %s.
%s

[Room Info]
%s
Items: %s

[Undiscovered Clues in This Room]
%s

[Requesting Player]
Role: %s
Personal Goals: %s

[Current Game State]
Elapsed: %d min / %d min
Discovered Clues: %d / %d

[Instructions]
1. Describe the examination result vividly (2-3 sentences)
2. If the target matches a clue's discover condition, trigger clue discovery
3. Even without clues, describe the room's atmosphere and details
4. Return {"events": [...], "stateChanges": [...]} JSON`,
		characterName, room.Name, targetLine,
		room.Description,
		strings.Join(items, ", "),
		strings.Join(undiscovered, "\n"),
		characterName,
		strings.Join(personalGoals, ", "),
		gameCtx.CurrentState.ElapsedTime/60, gameCtx.World.GameStructure.EstimatedDuration,
		countDiscoveredClues(gameCtx), len(gameCtx.World.Clues),
	)

	raw, err := g.provider.GenerateStructured(ctx, provider.StructuredRequest{
		SystemPrompt: evaluatorSystemPrompt,
		UserPrompt:   prompt,
		Temperature:  0.7,
		MaxTokens:    500,
	})
	if err != nil {
		return nil, fmt.Errorf("examine evaluation failed: %w", err)
	}

	var result schemas.GameResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("examine response parse failed: %w", err)
	}
	return &result, nil
}

// EvaluateAction evaluates a /do action and returns the AI response.
func (g *GMEngine) EvaluateAction(
	ctx context.Context,
	gameCtx types.GameContext,
	action string,
) (*schemas.GameResponse, error) {
	var playersInRoom []string
	for _, p := range gameCtx.PlayersInRoom {
		playersInRoom = append(playersInRoom, p.Nickname)
	}

	characterName := ""
	var personalGoals []string
	if gameCtx.RequestingPlayer.Role != nil {
		characterName = gameCtx.RequestingPlayer.Role.CharacterName
		for _, g := range gameCtx.RequestingPlayer.Role.PersonalGoals {
			personalGoals = append(personalGoals, g.Description)
		}
	}

	prompt := fmt.Sprintf(`[Situation]
%s performs an action: "%s"
Location: %s
Others in room: %s

[World Setting]
%s

[Requesting Player]
Role: %s
Personal Goals: %s

[Current Game State]
Elapsed: %d min / %d min
Discovered Clues: %d / %d

[Instructions]
1. Judge and describe the result of this action (2-3 sentences)
2. Trigger events if the action impacts the story
3. If the action is impossible, describe why
4. Results must be consistent with the game world
5. Return {"events": [...], "stateChanges": [...]} JSON`,
		characterName, action,
		gameCtx.CurrentRoom.Name,
		strings.Join(playersInRoom, ", "),
		gameCtx.World.Synopsis,
		characterName,
		strings.Join(personalGoals, ", "),
		gameCtx.CurrentState.ElapsedTime/60, gameCtx.World.GameStructure.EstimatedDuration,
		countDiscoveredClues(gameCtx), len(gameCtx.World.Clues),
	)

	raw, err := g.provider.GenerateStructured(ctx, provider.StructuredRequest{
		SystemPrompt: evaluatorSystemPrompt,
		UserPrompt:   prompt,
		Temperature:  0.8,
		MaxTokens:    500,
	})
	if err != nil {
		return nil, fmt.Errorf("action evaluation failed: %w", err)
	}

	var result schemas.GameResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("action response parse failed: %w", err)
	}
	return &result, nil
}

// GenerateNarration generates a GM narration for a given trigger.
func (g *GMEngine) GenerateNarration(
	ctx context.Context,
	gameCtx types.GameContext,
	trigger string,
) (string, error) {
	prompt := fmt.Sprintf(`Generate a brief GM narration for the following trigger.

World: %s
Atmosphere: %s
Trigger: %s

Write 1-3 sentences of atmospheric narration. Return JSON: {"text": "..."}`,
		gameCtx.World.Title, gameCtx.World.Atmosphere, trigger)

	raw, err := g.provider.GenerateStructured(ctx, provider.StructuredRequest{
		SystemPrompt: gmSystemPrompt,
		UserPrompt:   prompt,
		Temperature:  0.9,
		MaxTokens:    300,
	})
	if err != nil {
		return "", fmt.Errorf("narration generation failed: %w", err)
	}

	var response struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return "", fmt.Errorf("narration parse failed: %w", err)
	}
	return response.Text, nil
}

// countDiscoveredClues counts the number of discovered clues in the game state.
func countDiscoveredClues(gameCtx types.GameContext) int {
	count := 0
	for _, cs := range gameCtx.CurrentState.ClueStates {
		if cs.IsDiscovered {
			count++
		}
	}
	return count
}
