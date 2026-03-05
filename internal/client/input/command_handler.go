package input

import (
	"strings"

	"github.com/anthropics/story/internal/shared/protocol"
)

// CommandResult holds the result of processing a parsed input.
type CommandResult struct {
	Message  *protocol.ClientMessage
	ErrorMsg string
}

// CommandToClientMessage converts a ParsedInput to a ClientMessage.
func CommandToClientMessage(parsed ParsedInput) CommandResult {
	if parsed.Kind == InputKindChat {
		return CommandResult{Message: &protocol.ClientMessage{Type: "chat", Content: parsed.Content}}
	}

	switch parsed.Command {
	case "shout":
		if parsed.Args == "" {
			return CommandResult{ErrorMsg: "Usage: /shout <message>"}
		}
		return CommandResult{Message: &protocol.ClientMessage{Type: "shout", Content: parsed.Args}}

	case "move", "go":
		if parsed.Args == "" {
			return CommandResult{ErrorMsg: "Usage: /move <room name>"}
		}
		return CommandResult{Message: &protocol.ClientMessage{Type: "move", TargetRoomID: parsed.Args}}

	case "examine":
		var target *string
		if parsed.Args != "" {
			target = &parsed.Args
		}
		return CommandResult{Message: &protocol.ClientMessage{Type: "examine", Target: target}}

	case "do":
		if parsed.Args == "" {
			return CommandResult{ErrorMsg: "Usage: /do <action description>"}
		}
		return CommandResult{Message: &protocol.ClientMessage{Type: "do", Action: parsed.Args}}

	case "talk":
		if parsed.Args == "" {
			return CommandResult{ErrorMsg: "Usage: /talk <NPC name> [message]"}
		}
		spaceIdx := strings.Index(parsed.Args, " ")
		if spaceIdx == -1 {
			return CommandResult{Message: &protocol.ClientMessage{Type: "talk", NPCID: parsed.Args, Message: ""}}
		}
		return CommandResult{Message: &protocol.ClientMessage{
			Type:    "talk",
			NPCID:   parsed.Args[:spaceIdx],
			Message: parsed.Args[spaceIdx+1:],
		}}

	case "chat":
		if parsed.Args == "" {
			return CommandResult{ErrorMsg: "Usage: /chat <message>"}
		}
		return CommandResult{Message: &protocol.ClientMessage{Type: "chat", Content: parsed.Args}}

	case "give":
		return CommandResult{ErrorMsg: "/give is not yet supported."}

	case "vote":
		if parsed.Args == "" {
			return CommandResult{ErrorMsg: "Usage: /vote <target name>"}
		}
		return CommandResult{Message: &protocol.ClientMessage{Type: "vote", TargetID: parsed.Args}}

	case "solve":
		if parsed.Args == "" {
			return CommandResult{ErrorMsg: "Usage: /solve <solution>"}
		}
		return CommandResult{Message: &protocol.ClientMessage{Type: "solve", Answer: parsed.Args}}

	case "end":
		return CommandResult{Message: &protocol.ClientMessage{Type: "propose_end"}}

	case "endvote":
		if parsed.Args == "" {
			return CommandResult{ErrorMsg: "Usage: /endvote yes or /endvote no"}
		}
		switch strings.ToLower(parsed.Args) {
		case "yes", "y":
			return CommandResult{Message: &protocol.ClientMessage{Type: "end_vote", Agree: true}}
		case "no", "n":
			return CommandResult{Message: &protocol.ClientMessage{Type: "end_vote", Agree: false}}
		default:
			return CommandResult{ErrorMsg: "Usage: /endvote yes or /endvote no"}
		}

	case "look":
		return CommandResult{Message: &protocol.ClientMessage{Type: "request_look"}}
	case "map":
		return CommandResult{Message: &protocol.ClientMessage{Type: "request_map"}}
	case "inventory", "inv":
		return CommandResult{Message: &protocol.ClientMessage{Type: "request_inventory"}}
	case "role":
		return CommandResult{Message: &protocol.ClientMessage{Type: "request_role"}}
	case "who":
		return CommandResult{Message: &protocol.ClientMessage{Type: "request_who"}}
	case "help":
		return CommandResult{Message: &protocol.ClientMessage{Type: "request_help"}}
	case "ready":
		return CommandResult{Message: &protocol.ClientMessage{Type: "ready"}}

	default:
		return CommandResult{ErrorMsg: "Unknown command: /" + parsed.Command + ". Type /help for a list."}
	}
}
