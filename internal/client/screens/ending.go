package screens

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"

	"github.com/anthropics/story/internal/client/state"
)

// EndingPhase tracks which part of the ending the player is viewing.
type EndingPhase int

const (
	EndingResult   EndingPhase = iota // Common result + personal ending
	EndingReveal                      // Secret reveal
	EndingFeedback                    // Feedback input
)

// RenderEnding renders the game ending screen based on the current phase.
func RenderEnding(s state.ClientState, phase EndingPhase, funRating, immersion, focusField int, commentInput textinput.Model, width int) string {
	boxW := width - 4
	if boxW > 70 {
		boxW = 70
	}
	if boxW < 30 {
		boxW = 30
	}

	switch phase {
	case EndingResult:
		return renderEndingResult(s, boxW)
	case EndingReveal:
		return renderEndingReveal(s, boxW)
	case EndingFeedback:
		return renderEndingFeedback(funRating, immersion, focusField, commentInput, boxW)
	}
	return ""
}

func renderEndingResult(s state.ClientState, boxW int) string {
	title := subtitleStyle.Render("=== Game Over ===")

	if s.EndingData == nil {
		return boxStyle.Width(boxW).Render(title + "\n\n  Loading results...")
	}

	var sections []string
	sections = append(sections, s.EndingData.CommonResult)

	pe := s.EndingData.PersonalEnding
	if pe.Summary != "" {
		sections = append(sections, "")
		sections = append(sections, lipgloss.NewStyle().Bold(true).Render("-- Your Story --"))
		sections = append(sections, pe.Summary)
	}

	if pe.Narrative != "" {
		sections = append(sections, "")
		sections = append(sections, pe.Narrative)
	}

	if len(pe.GoalResults) > 0 {
		sections = append(sections, "")
		sections = append(sections, lipgloss.NewStyle().Bold(true).Render("[Goal Results]"))
		for _, g := range pe.GoalResults {
			mark := successStyle.Render("V")
			if !g.Achieved {
				mark = errorStyle.Render("X")
			}
			sections = append(sections, fmt.Sprintf("  %s %s", mark, g.Description))
		}
	}

	sections = append(sections, "")
	sections = append(sections, lipgloss.NewStyle().Faint(true).Render("  Press Enter to view secret reveal"))

	body := strings.Join(sections, "\n")
	return boxStyle.Width(boxW).Render(title + "\n\n" + body)
}

func renderEndingReveal(s state.ClientState, boxW int) string {
	title := subtitleStyle.Render("=== Secret Reveal ===")

	if s.EndingData == nil {
		return boxStyle.Width(boxW).Render(title + "\n\n  Loading...")
	}

	sr := s.EndingData.SecretReveal
	var sections []string

	if len(sr.PlayerSecrets) > 0 {
		sections = append(sections, lipgloss.NewStyle().Bold(true).Render("[Player Secrets]"))
		for _, ps := range sr.PlayerSecrets {
			role := ""
			if ps.SpecialRole != nil {
				role = " (" + *ps.SpecialRole + ")"
			}
			sections = append(sections, fmt.Sprintf("  %s%s: %s", ps.CharacterName, role, ps.Secret))
		}
	}

	if len(sr.UndiscoveredClues) > 0 {
		sections = append(sections, "")
		sections = append(sections, lipgloss.NewStyle().Bold(true).Render("[Undiscovered Clues]"))
		for _, uc := range sr.UndiscoveredClues {
			sections = append(sections, fmt.Sprintf("  - %s (%s): %s", uc.Clue.Name, uc.RoomName, uc.Clue.Description))
		}
	}

	sections = append(sections, "")
	sections = append(sections, lipgloss.NewStyle().Faint(true).Render("  Press Enter to continue"))

	body := strings.Join(sections, "\n")
	return boxStyle.Width(boxW).Render(title + "\n\n" + body)
}

func renderEndingFeedback(funRating, immersion, focusField int, commentInput textinput.Model, boxW int) string {
	title := subtitleStyle.Render("=== Feedback ===")

	funLine := fmt.Sprintf("  Story Rating: %s (%d/5)", renderStars(funRating), funRating)
	immLine := fmt.Sprintf("  Immersion:    %s (%d/5)", renderStars(immersion), immersion)

	arrowHint := "  Use left/right keys to adjust"
	if focusField == 0 {
		funLine = lipgloss.NewStyle().Bold(true).Render(funLine) + "\n" + arrowHint
	} else if focusField == 1 {
		immLine = lipgloss.NewStyle().Bold(true).Render(immLine) + "\n" + arrowHint
	}

	commentLine := fmt.Sprintf("  Comment (optional): %s", commentInput.View())

	body := lipgloss.JoinVertical(lipgloss.Left,
		funLine,
		"",
		immLine,
		"",
		commentLine,
		"",
		"  Enter: Submit   Esc: Skip",
	)

	return boxStyle.Width(boxW).Render(title + "\n\n" + body)
}

func renderStars(rating int) string {
	stars := ""
	for i := 1; i <= 5; i++ {
		if i <= rating {
			stars += "*"
		} else {
			stars += "."
		}
	}
	return stars
}
