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

// NPCEngine handles NPC dialogue generation.
type NPCEngine struct {
	provider provider.AIProvider
}

// NewNPCEngine creates a new NPCEngine with the given AI provider.
func NewNPCEngine(p provider.AIProvider) *NPCEngine {
	return &NPCEngine{provider: p}
}

// TalkToNPC generates an NPC response to a player message.
func (e *NPCEngine) TalkToNPC(
	ctx context.Context,
	gameCtx types.GameContext,
	npcID string,
	message string,
) (*schemas.NPCResponse, error) {
	// Find the NPC in the world
	var npc *types.NPC
	for i := range gameCtx.World.NPCs {
		if gameCtx.World.NPCs[i].ID == npcID {
			npc = &gameCtx.World.NPCs[i]
			break
		}
	}
	if npc == nil {
		return nil, fmt.Errorf("NPC not found: %s", npcID)
	}

	userPrompt := buildNPCPrompt(npc, message, &gameCtx)

	raw, err := e.provider.GenerateStructured(ctx, provider.StructuredRequest{
		SystemPrompt: npcSystemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.8,
		MaxTokens:    500,
	})
	if err != nil {
		return nil, fmt.Errorf("NPC dialogue generation failed: %w", err)
	}

	var response schemas.NPCResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("NPC response parse failed: %w", err)
	}

	if err := response.Validate(); err != nil {
		return nil, fmt.Errorf("NPC response validation failed: %w", err)
	}

	// Check for information leaks
	if err := validateNPCInfoLeak(&response, npc); err != nil {
		// Retry once with stronger constraints
		retryPrompt := userPrompt + "\n\nIMPORTANT: Your previous response leaked hidden information. Do NOT reveal any hidden information in your response."
		raw, err = e.provider.GenerateStructured(ctx, provider.StructuredRequest{
			SystemPrompt: npcSystemPrompt,
			UserPrompt:   retryPrompt,
			Temperature:  0.5,
			MaxTokens:    500,
		})
		if err != nil {
			return nil, fmt.Errorf("NPC dialogue retry failed: %w", err)
		}
		if err := json.Unmarshal(raw, &response); err != nil {
			return nil, fmt.Errorf("NPC retry response parse failed: %w", err)
		}
		// If still leaking, return a safe fallback
		if err := validateNPCInfoLeak(&response, npc); err != nil {
			response = schemas.NPCResponse{
				Dialogue:        "I'd rather not talk about that.",
				Emotion:         "guarded",
				InternalThought: "Avoiding information leak - fallback response",
				TrustChange:     0,
			}
		}
	}

	return &response, nil
}

// buildNPCPrompt constructs the user prompt for NPC dialogue.
func buildNPCPrompt(
	npc *types.NPC,
	playerMessage string,
	gameCtx *types.GameContext,
) string {
	var knownInfo strings.Builder
	for _, info := range npc.KnownInfo {
		fmt.Fprintf(&knownInfo, "- %s\n", info)
	}

	var hiddenInfo strings.Builder
	for _, info := range npc.HiddenInfo {
		fmt.Fprintf(&hiddenInfo, "- %s\n", info)
	}

	// Get conversation history from game state
	var historyBuf strings.Builder
	if npcState, ok := gameCtx.CurrentState.NPCStates[npc.ID]; ok {
		for _, h := range npcState.ConversationHistory {
			fmt.Fprintf(&historyBuf, "Player: %s\n%s: %s\n", h.Message, npc.Name, h.Response)
		}
	}

	// Get trust level for requesting player
	trustLevel := npc.InitialTrust
	if npcState, ok := gameCtx.CurrentState.NPCStates[npc.ID]; ok {
		if tl, ok := npcState.TrustLevels[gameCtx.RequestingPlayer.ID]; ok {
			trustLevel = tl
		}
	}

	characterName := ""
	if gameCtx.RequestingPlayer.Role != nil {
		characterName = gameCtx.RequestingPlayer.Role.CharacterName
	}

	gimmickSection := ""
	if npc.Gimmick != nil {
		triggered := "not triggered"
		if npcState, ok := gameCtx.CurrentState.NPCStates[npc.ID]; ok && npcState.GimmickTriggered {
			triggered = "already triggered"
		}
		gimmickSection = fmt.Sprintf(`
[Gimmick]
%s
Trigger condition: %s
Current status: %s`, npc.Gimmick.Description, npc.Gimmick.TriggerCondition, triggered)
	}

	return fmt.Sprintf(`[Your Identity]
Name: %s
Persona: %s
Behavior Principle: %s

[What You Know]
%s
[What You Are Hiding]
%s
[Current Player]
Name: %s
Trust Level: %.1f / 1.0
%s
[Previous Conversation]
%s
[Current Input]
%s: "%s"
`,
		npc.Name, npc.Persona, npc.BehaviorPrinciple,
		knownInfo.String(),
		hiddenInfo.String(),
		characterName,
		trustLevel,
		gimmickSection,
		historyBuf.String(),
		characterName, playerMessage,
	)
}

// validateNPCInfoLeak checks if the NPC response leaks hidden information.
func validateNPCInfoLeak(response *schemas.NPCResponse, npc *types.NPC) error {
	for _, hidden := range npc.HiddenInfo {
		if containsHiddenKeywords(response.Dialogue, hidden) {
			return fmt.Errorf("NPC response leaked hidden information: %s", hidden)
		}
	}
	return nil
}

// containsHiddenKeywords checks if dialogue contains keywords from hidden info.
// Uses a simple substring check on significant words (4+ chars).
func containsHiddenKeywords(dialogue, hiddenInfo string) bool {
	lowerDialogue := strings.ToLower(dialogue)
	words := strings.Fields(hiddenInfo)
	for _, word := range words {
		// Only check words with 4+ characters to reduce false positives
		cleaned := strings.Trim(strings.ToLower(word), ".,!?;:'\"()-")
		if len(cleaned) >= 4 && strings.Contains(lowerDialogue, cleaned) {
			return true
		}
	}
	return false
}
