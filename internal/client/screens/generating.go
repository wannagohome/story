package screens

import (
	"charm.land/bubbles/v2/progress"
	"charm.land/lipgloss/v2"

	"github.com/anthropics/story/internal/client/state"
)

// RenderGenerating renders the world generation progress screen.
func RenderGenerating(s state.ClientState, prog progress.Model, width int) string {
	msg := s.GenerationMessage
	if msg == "" {
		msg = "Generating world..."
	}

	bar := prog.ViewAs(s.GenerationProgress)

	body := lipgloss.NewStyle().Padding(1, 2).Render(
		titleStyle.Render("AI is building the world...") + "\n\n" + bar + "\n" + msg,
	)

	boxW := width - 4
	if boxW > 60 {
		boxW = 60
	}
	if boxW < 30 {
		boxW = 30
	}

	return boxStyle.Width(boxW).Render(body)
}
