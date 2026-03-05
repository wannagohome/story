package renderers

import "charm.land/lipgloss/v2"

// Color palette (Claude Code style - dark theme with subtle colors)
var (
	cyanColor    = lipgloss.Color("6")  // system messages
	greenColor   = lipgloss.Color("2")  // NPC names, actions
	yellowColor  = lipgloss.Color("3")  // global chat, warnings
	redColor     = lipgloss.Color("1")  // story events, errors
	magentaColor = lipgloss.Color("5")  // narration (GM)
	dimColor     = lipgloss.Color("8")  // timestamps, metadata
)

var (
	systemStyle = lipgloss.NewStyle().Foreground(cyanColor).Faint(true)
	dimStyle    = lipgloss.NewStyle().Faint(true)
	boldStyle   = lipgloss.NewStyle().Bold(true)

	narrationStyle = lipgloss.NewStyle().
			Foreground(magentaColor).
			Italic(true)

	npcNameStyle = lipgloss.NewStyle().
			Foreground(greenColor).
			Bold(true)

	globalScopeStyle = lipgloss.NewStyle().Foreground(yellowColor)
	globalNameStyle  = lipgloss.NewStyle().Foreground(yellowColor).Bold(true)

	clueBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(cyanColor).
			Foreground(cyanColor).
			PaddingLeft(1).
			PaddingRight(1)

	storyEventBoxStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.DoubleBorder()).
				BorderForeground(redColor).
				Foreground(redColor).
				PaddingLeft(1).
				PaddingRight(1)

	storyEventTitleStyle = lipgloss.NewStyle().
				Foreground(redColor).
				Bold(true)

	revealBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(greenColor).
			Foreground(greenColor).
			PaddingLeft(1).
			PaddingRight(1)

	timeWarningBoxStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(yellowColor).
				Foreground(yellowColor).
				Bold(true).
				PaddingLeft(1).
				PaddingRight(1)
)
