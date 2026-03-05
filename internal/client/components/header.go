package components

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/anthropics/story/internal/client/state"
)

var headerStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderBottom(true).
	BorderForeground(lipgloss.Color("8")).
	PaddingLeft(1).
	PaddingRight(1)

var titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
var headerDimStyle = lipgloss.NewStyle().Faint(true)

// RenderHeader renders the header bar with room info and player locations.
func RenderHeader(s state.ClientState, width int) string {
	roomName := ""
	if s.CurrentRoom != nil {
		roomName = s.CurrentRoom.Name
	}

	title := fmt.Sprintf("[%s] %s - %s",
		s.RoomCode,
		s.WorldTitle,
		titleStyle.Render(roomName),
	)

	if s.MapOverview == nil {
		return headerStyle.Width(width - 2).Render(title)
	}

	// Build overview of rooms with players
	rooms := make([]string, 0, len(s.MapOverview.Rooms))
	for _, room := range s.MapOverview.Rooms {
		if s.CurrentRoom != nil && room.ID == s.CurrentRoom.ID {
			names := strings.Join(room.PlayerNames, ", ")
			if names == "" {
				names = "you"
			}
			rooms = append(rooms, "Here: "+names)
		} else if room.PlayerCount > 0 {
			rooms = append(rooms, room.Name+": "+strings.Join(room.PlayerNames, ", "))
		}
	}

	if len(rooms) == 0 {
		return headerStyle.Width(width - 2).Render(title)
	}

	overview := headerDimStyle.Render(strings.Join(rooms, "  /  "))
	content := lipgloss.JoinVertical(lipgloss.Left, title, overview)
	return headerStyle.Width(width - 2).Render(content)
}
