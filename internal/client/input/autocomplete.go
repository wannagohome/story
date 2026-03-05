package input

import "strings"

// CompletionContext provides game state information for tab completion.
type CompletionContext struct {
	Commands  []string
	NPCNames  []string
	RoomNames []string
	Players   []string
}

// AvailableCommands lists all supported slash commands.
var AvailableCommands = []string{
	"move", "go", "examine", "do", "talk", "chat", "shout",
	"inventory", "inv", "role", "map", "who", "help",
	"vote", "solve", "end", "endvote", "look", "ready",
}

// Complete returns tab completion candidates for the given input.
func Complete(input string, ctx CompletionContext) []string {
	trimmed := strings.TrimSpace(input)

	if !strings.HasPrefix(trimmed, "/") {
		return nil
	}

	spaceIdx := strings.Index(trimmed, " ")
	if spaceIdx == -1 {
		prefix := trimmed[1:]
		return filterPrefix(ctx.Commands, prefix)
	}

	command := trimmed[1:spaceIdx]
	argPrefix := strings.TrimSpace(trimmed[spaceIdx+1:])

	switch command {
	case "move", "go":
		return filterPrefix(ctx.RoomNames, argPrefix)
	case "talk", "give":
		return filterPrefix(ctx.NPCNames, argPrefix)
	case "vote":
		return filterPrefix(ctx.Players, argPrefix)
	}
	return nil
}

func filterPrefix(candidates []string, prefix string) []string {
	var matches []string
	lowerPrefix := strings.ToLower(prefix)
	for _, c := range candidates {
		if strings.HasPrefix(strings.ToLower(c), lowerPrefix) {
			matches = append(matches, c)
		}
	}
	return matches
}
