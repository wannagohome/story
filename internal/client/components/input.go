package components

import (
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
)

var inputBarStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderTop(true).
	BorderForeground(lipgloss.Color("8")).
	PaddingLeft(1).
	PaddingRight(1)

// RenderInputBar renders the command input bar at the bottom.
func RenderInputBar(ti textinput.Model, width int) string {
	hint := lipgloss.NewStyle().Faint(true).Render("[/help] [/map]")
	row := lipgloss.JoinHorizontal(lipgloss.Top, ti.View(), "  ", hint)
	return inputBarStyle.Width(width - 2).Render(row)
}

// NewTextInput creates a configured textinput model for command entry.
func NewTextInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "Type a command (/help)"
	ti.Prompt = "> "
	ti.Focus()
	return ti
}
