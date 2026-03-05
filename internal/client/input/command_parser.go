package input

import "strings"

type InputKind int

const (
	InputKindChat InputKind = iota
	InputKindCommand
)

type ParsedInput struct {
	Kind    InputKind
	Command string // when Kind == InputKindCommand
	Args    string // command arguments (empty if none)
	Content string // when Kind == InputKindChat
}

// ParseInput parses raw user input into a structured ParsedInput.
func ParseInput(raw string) ParsedInput {
	trimmed := strings.TrimSpace(raw)

	if strings.HasPrefix(trimmed, "/") {
		spaceIdx := strings.Index(trimmed, " ")
		if spaceIdx == -1 {
			return ParsedInput{
				Kind:    InputKindCommand,
				Command: trimmed[1:],
			}
		}
		return ParsedInput{
			Kind:    InputKindCommand,
			Command: trimmed[1:spaceIdx],
			Args:    strings.TrimSpace(trimmed[spaceIdx+1:]),
		}
	}

	return ParsedInput{
		Kind:    InputKindChat,
		Content: trimmed,
	}
}
