package screens

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/anthropics/story/internal/client/state"
)

// BriefingPhase tracks which part of the briefing the player is viewing.
type BriefingPhase int

const (
	BriefingPublic         BriefingPhase = iota // Show public briefing
	BriefingWaitingPrivate                      // Waiting for others to acknowledge
	BriefingPrivate                             // Show private role
	BriefingWaitingReady                        // Waiting for all players to be ready
)

// RenderBriefing renders the briefing screen based on the current phase.
func RenderBriefing(s state.ClientState, phase BriefingPhase, width int) string {
	boxW := width - 4
	if boxW > 70 {
		boxW = 70
	}
	if boxW < 30 {
		boxW = 30
	}

	switch phase {
	case BriefingPublic:
		return renderBriefingPublic(s, boxW)
	case BriefingWaitingPrivate:
		return boxStyle.Width(boxW).Render(
			subtitleStyle.Render("=== Briefing ===") + "\n\n" +
				"  Waiting for all players to read the briefing...",
		)
	case BriefingPrivate:
		return renderBriefingPrivate(s, boxW)
	case BriefingWaitingReady:
		return boxStyle.Width(boxW).Render(
			subtitleStyle.Render("=== Your Role ===") + "\n\n" +
				"  Waiting for all players to be ready...",
		)
	}
	return ""
}

func renderBriefingPublic(s state.ClientState, boxW int) string {
	title := subtitleStyle.Render("=== Briefing ===")

	if s.BriefingPublic == nil {
		return boxStyle.Width(boxW).Render(title + "\n\n  Loading briefing...")
	}

	info := s.BriefingPublic
	var sections []string
	sections = append(sections, titleStyle.Render("["+info.Title+"]"))
	sections = append(sections, "")
	sections = append(sections, info.Synopsis)

	if len(info.CharacterList) > 0 {
		sections = append(sections, "")
		sections = append(sections, lipgloss.NewStyle().Bold(true).Render("Characters:"))
		for _, c := range info.CharacterList {
			sections = append(sections, fmt.Sprintf("  - %s: %s", c.Name, c.PublicDescription))
		}
	}

	if len(info.NPCList) > 0 {
		sections = append(sections, "")
		sections = append(sections, lipgloss.NewStyle().Bold(true).Render("NPCs:"))
		for _, npc := range info.NPCList {
			sections = append(sections, fmt.Sprintf("  - %s (%s)", npc.Name, npc.Location))
		}
	}

	if info.GameRules != "" {
		sections = append(sections, "")
		sections = append(sections, lipgloss.NewStyle().Bold(true).Render("Rules:"))
		sections = append(sections, "  "+info.GameRules)
	}

	sections = append(sections, "")
	sections = append(sections, lipgloss.NewStyle().Faint(true).Render("  Press Enter to continue"))

	body := strings.Join(sections, "\n")
	return boxStyle.Width(boxW).Render(title + "\n\n" + body)
}

func renderBriefingPrivate(s state.ClientState, boxW int) string {
	title := subtitleStyle.Render("=== Your Role (Private) ===")

	if s.MyRole == nil {
		return boxStyle.Width(boxW).Render(title + "\n\n  Loading role info...")
	}

	role := s.MyRole
	var sections []string
	sections = append(sections, fmt.Sprintf("  Name: %s", lipgloss.NewStyle().Bold(true).Render(role.CharacterName)))
	sections = append(sections, fmt.Sprintf("  Background: %s", role.Background))

	if len(role.PersonalGoals) > 0 {
		sections = append(sections, "")
		sections = append(sections, lipgloss.NewStyle().Bold(true).Render("  [Personal Goals]"))
		for _, g := range role.PersonalGoals {
			sections = append(sections, "  - "+g.Description)
		}
	}

	if role.Secret != "" {
		sections = append(sections, "")
		sections = append(sections, lipgloss.NewStyle().Bold(true).Render("  [Secret]"))
		sections = append(sections, "  "+role.Secret)
	}

	if len(s.BriefingSecrets) > 0 {
		sections = append(sections, "")
		for _, secret := range s.BriefingSecrets {
			sections = append(sections, "  "+secret)
		}
	}

	sections = append(sections, "")
	sections = append(sections, lipgloss.NewStyle().Faint(true).Render("  Press Enter when ready"))

	body := strings.Join(sections, "\n")
	return boxStyle.Width(boxW).Render(title + "\n\n" + body)
}
