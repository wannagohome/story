package screens

import "charm.land/lipgloss/v2"

var boxStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("8")).
	Padding(1, 2)

var titleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("6"))

var subtitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("5"))

var errorStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("1"))

var successStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("2"))
