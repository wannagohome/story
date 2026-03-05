package components

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/anthropics/story/internal/client/state"
)

var sidebarStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderLeft(true).
	BorderForeground(lipgloss.Color("8")).
	PaddingLeft(1).
	PaddingRight(1).
	Width(24)

var sidebarTitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("6"))

var sidebarDimStyle = lipgloss.NewStyle().Faint(true)

// RenderSidebar renders a compact sidebar with room info.
func RenderSidebar(s state.ClientState, height int) string {
	var sections []string

	// Current room info
	if s.CurrentRoom != nil {
		sections = append(sections,
			sidebarTitleStyle.Render("Room"),
			s.CurrentRoom.Name,
		)

		if len(s.CurrentRoom.Players) > 0 {
			sections = append(sections, "")
			sections = append(sections, sidebarTitleStyle.Render("Players"))
			for _, p := range s.CurrentRoom.Players {
				sections = append(sections, "  "+p.Nickname)
			}
		}

		if len(s.CurrentRoom.NPCs) > 0 {
			sections = append(sections, "")
			sections = append(sections, sidebarTitleStyle.Render("NPCs"))
			for _, npc := range s.CurrentRoom.NPCs {
				sections = append(sections, "  "+npc.Name)
			}
		}

		if len(s.CurrentRoom.Items) > 0 {
			sections = append(sections, "")
			sections = append(sections, sidebarTitleStyle.Render("Items"))
			for _, item := range s.CurrentRoom.Items {
				sections = append(sections, "  "+item.Name)
			}
		}
	}

	// Map overview
	if s.MapOverview != nil {
		sections = append(sections, "")
		sections = append(sections, sidebarTitleStyle.Render("Map"))
		for _, room := range s.MapOverview.Rooms {
			marker := "  "
			if s.CurrentRoom != nil && room.ID == s.CurrentRoom.ID {
				marker = "> "
			}
			info := room.Name
			if room.PlayerCount > 0 {
				info += fmt.Sprintf(" (%d)", room.PlayerCount)
			}
			sections = append(sections, marker+info)
		}
	}

	content := strings.Join(sections, "\n")

	// Pad to fill height
	lines := strings.Split(content, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}

	return sidebarStyle.Height(height).Render(strings.Join(lines[:height], "\n"))
}
