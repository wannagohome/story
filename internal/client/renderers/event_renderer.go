package renderers

import (
	"encoding/json"
	"fmt"

	"charm.land/lipgloss/v2"

	"github.com/anthropics/story/internal/client/state"
)

// EventRendererFn renders a game event to a styled string.
type EventRendererFn func(event state.GameEvent) string

var rendererRegistry = map[string]EventRendererFn{
	"narration":        renderNarration,
	"npc_dialogue":     renderNPCDialogue,
	"clue_found":       renderClueFound,
	"story_event":      renderStoryEvent,
	"game_end":         renderGameEnd,
	"examine_result":   renderExamineResult,
	"action_result":    renderActionResult,
	"player_move":      renderPlayerMove,
	"time_warning":     renderTimeWarning,
	"npc_moved":        renderNPCMoved,
	"npc_give_item":    renderNPCGiveItem,
	"npc_receive_item": renderNPCReceiveItem,
	"npc_reveal":       renderNPCReveal,
}

// RenderMessage renders a DisplayMessage to a styled string for the chat log.
func RenderMessage(msg state.DisplayMessage) string {
	switch msg.Kind {
	case "chat":
		return renderChatMessage(msg)
	case "event":
		fn, ok := rendererRegistry[msg.Event.Type]
		if ok {
			return fn(msg.Event)
		}
		return renderFallback(msg.Event)
	case "system":
		return renderSystemMessage(msg.Content)
	}
	return ""
}

func renderChatMessage(msg state.DisplayMessage) string {
	if msg.Scope == "global" {
		location := ""
		if msg.SenderLocation != nil {
			location = " (" + *msg.SenderLocation + ")"
		}
		return fmt.Sprintf("%s %s: %s",
			globalScopeStyle.Render("[Global]"),
			globalNameStyle.Render(msg.SenderName+location),
			msg.Content,
		)
	}
	return fmt.Sprintf("%s: %s",
		boldStyle.Render(msg.SenderName),
		msg.Content,
	)
}

func renderSystemMessage(content string) string {
	return systemStyle.Render("--- " + content + " ---")
}

func renderNarration(event state.GameEvent) string {
	return narrationStyle.Render("[GM] " + getString(event.Data, "text"))
}

func renderNPCDialogue(event state.GameEvent) string {
	npcName := getString(event.Data, "npcName")
	text := getString(event.Data, "text")
	return fmt.Sprintf("%s %s",
		npcNameStyle.Render("["+npcName+"]"),
		text,
	)
}

func renderClueFound(event state.GameEvent) string {
	playerName := getString(event.Data, "playerName")
	clueMap := getMap(event.Data, "clue")
	clueName := getString(clueMap, "name")
	text := fmt.Sprintf("* %s found [%s]!", playerName, clueName)
	return clueBoxStyle.Render(text)
}

func renderStoryEvent(event state.GameEvent) string {
	title := getString(event.Data, "title")
	desc := getString(event.Data, "description")
	body := lipgloss.JoinVertical(lipgloss.Left,
		storyEventTitleStyle.Render("[Event] "+title),
		desc,
	)
	return storyEventBoxStyle.Render(body)
}

func renderGameEnd(event state.GameEvent) string {
	reason := getString(event.Data, "reason")
	result := getString(event.Data, "commonResult")
	body := lipgloss.JoinVertical(lipgloss.Left,
		storyEventTitleStyle.Render("[Game Over] "+reason),
		result,
	)
	return storyEventBoxStyle.Render(body)
}

func renderExamineResult(event state.GameEvent) string {
	playerName := getString(event.Data, "playerName")
	target := getString(event.Data, "target")
	desc := getString(event.Data, "description")
	return narrationStyle.Render(
		fmt.Sprintf("[Examine] %s examined %s.\n%s", playerName, target, desc),
	)
}

func renderActionResult(event state.GameEvent) string {
	playerName := getString(event.Data, "playerName")
	action := getString(event.Data, "action")
	result := getString(event.Data, "result")
	return narrationStyle.Render(
		fmt.Sprintf("[Action] %s: %s\n%s", playerName, action, result),
	)
}

func renderPlayerMove(event state.GameEvent) string {
	playerName := getString(event.Data, "playerName")
	from := getString(event.Data, "from")
	to := getString(event.Data, "to")
	return systemStyle.Render(
		fmt.Sprintf("--- %s moved from %s to %s ---", playerName, from, to),
	)
}

func renderTimeWarning(event state.GameEvent) string {
	remaining := int(getFloat64(event.Data, "remainingMinutes"))
	return timeWarningBoxStyle.Render(
		fmt.Sprintf("Time remaining: %d minutes", remaining),
	)
}

func renderNPCMoved(event state.GameEvent) string {
	npcName := getString(event.Data, "npcName")
	from := getString(event.Data, "from")
	to := getString(event.Data, "to")
	return systemStyle.Render(
		fmt.Sprintf("--- %s moved from %s to %s ---", npcName, from, to),
	)
}

func renderNPCGiveItem(event state.GameEvent) string {
	npcName := getString(event.Data, "npcName")
	playerName := getString(event.Data, "playerName")
	item := getMap(event.Data, "item")
	itemName := getString(item, "name")
	return fmt.Sprintf("%s gave [%s] to %s.",
		npcNameStyle.Render("["+npcName+"]"), itemName, playerName)
}

func renderNPCReceiveItem(event state.GameEvent) string {
	npcName := getString(event.Data, "npcName")
	playerName := getString(event.Data, "playerName")
	item := getMap(event.Data, "item")
	itemName := getString(item, "name")
	return fmt.Sprintf("%s gave [%s] to %s.",
		playerName, itemName, npcNameStyle.Render("["+npcName+"]"))
}

func renderNPCReveal(event state.GameEvent) string {
	npcName := getString(event.Data, "npcName")
	revelation := getString(event.Data, "revelation")
	return revealBoxStyle.Render(
		fmt.Sprintf("%s %s", npcNameStyle.Render("["+npcName+"]"), revelation),
	)
}

func renderFallback(event state.GameEvent) string {
	b, _ := json.Marshal(event)
	return systemStyle.Render("[unknown event] " + string(b))
}
