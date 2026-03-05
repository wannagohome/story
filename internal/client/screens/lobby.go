package screens

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/anthropics/story/internal/client/state"
)

// RenderLobby renders the lobby waiting screen.
func RenderLobby(s state.ClientState, width int) string {
	title := titleStyle.Render("Story - Lobby")

	playerLines := make([]string, 0, len(s.LobbyPlayers))
	for _, p := range s.LobbyPlayers {
		label := p.Nickname
		if p.IsHost {
			label += " (host)"
		}
		dot := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("*")
		playerLines = append(playerLines, "    "+dot+" "+label)
	}

	hint := ""
	if s.IsHost {
		hint = "\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render("[Host] Press Enter to start the game")
	}

	body := fmt.Sprintf(
		"  Room Code: %s\n  Share this code with friends!\n\n  Connected (%d/%d):\n%s%s",
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3")).Render(s.RoomCode),
		len(s.LobbyPlayers),
		s.MaxPlayers,
		strings.Join(playerLines, "\n"),
		hint,
	)

	boxW := width - 4
	if boxW > 60 {
		boxW = 60
	}
	if boxW < 30 {
		boxW = 30
	}

	return boxStyle.Width(boxW).Render(title + "\n\n" + body)
}
