package components

import (
	"strings"

	"charm.land/bubbles/v2/viewport"

	"github.com/anthropics/story/internal/client/renderers"
	"github.com/anthropics/story/internal/client/state"
)

// RenderChatLog renders the scrollable chat/event log.
func RenderChatLog(messages []state.DisplayMessage, vp *viewport.Model) string {
	lines := make([]string, 0, len(messages))
	for _, msg := range messages {
		rendered := renderers.RenderMessage(msg)
		if rendered != "" {
			lines = append(lines, rendered)
		}
	}
	content := strings.Join(lines, "\n")
	vp.SetContent(content)
	// Auto-scroll to bottom when new content arrives
	vp.GotoBottom()
	return vp.View()
}
