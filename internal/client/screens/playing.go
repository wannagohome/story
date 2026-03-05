package screens

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	"github.com/anthropics/story/internal/client/components"
	"github.com/anthropics/story/internal/client/state"
)

// RenderPlaying renders the main game screen with header, chat log, and input bar.
func RenderPlaying(s state.ClientState, chatVP *viewport.Model, ti textinput.Model, width, height int) string {
	header := components.RenderHeader(s, width)
	headerHeight := strings.Count(header, "\n") + 1

	inputBar := components.RenderInputBar(ti, width)
	inputHeight := strings.Count(inputBar, "\n") + 1

	chatHeight := height - headerHeight - inputHeight
	if chatHeight < 3 {
		chatHeight = 3
	}

	chatVP.SetWidth(width)
	chatVP.SetHeight(chatHeight)

	chatLog := components.RenderChatLog(s.Messages, chatVP)

	return lipgloss.JoinVertical(lipgloss.Left, header, chatLog, inputBar)
}
