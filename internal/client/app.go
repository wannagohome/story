package client

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	tea "charm.land/bubbletea/v2"

	"github.com/anthropics/story/internal/client/components"
	"github.com/anthropics/story/internal/client/input"
	"github.com/anthropics/story/internal/client/network"
	"github.com/anthropics/story/internal/client/screens"
	"github.com/anthropics/story/internal/client/state"
	"github.com/anthropics/story/internal/shared/protocol"
)

// Screen represents the current screen being displayed.
type Screen int

const (
	ScreenConnecting Screen = iota
	ScreenNickname
	ScreenLobby
	ScreenGenerating
	ScreenBriefing
	ScreenGame
	ScreenEnding
	ScreenFinished
)

// AppModel is the root Bubble Tea model for the client TUI.
type AppModel struct {
	screen  Screen
	state   state.ClientState
	network *network.NetworkClient

	// Screen sub-models
	connectSpinner spinner.Model
	nickInput      textinput.Model
	nickError      string
	lobbyInput     textinput.Model
	progressBar    progress.Model
	briefingPhase  screens.BriefingPhase
	chatViewport   viewport.Model
	gameInput      textinput.Model
	endingPhase    screens.EndingPhase
	funRating      int
	immersion      int
	focusField     int
	commentInput   textinput.Model
}

// ClientConfig holds configuration for creating a new client.
type ClientConfig struct {
	ServerURL string
	RoomCode  string
	Nickname  string
	IsHost    bool
}

// NewAppModel creates a new AppModel with the given configuration.
func NewAppModel(cfg ClientConfig) AppModel {
	nc := network.NewNetworkClient(cfg.ServerURL)

	s := state.NewClientState()
	s.RoomCode = cfg.RoomCode
	s.Nickname = cfg.Nickname
	s.IsHost = cfg.IsHost

	sp := spinner.New()

	ni := textinput.New()
	ni.Placeholder = "Enter your nickname"
	ni.Prompt = "> "
	ni.Focus()

	li := textinput.New()
	li.Prompt = "> "

	pb := progress.New()

	gi := components.NewTextInput()

	ci := textinput.New()
	ci.Placeholder = "Your thoughts..."
	ci.Prompt = "> "

	return AppModel{
		screen:       ScreenConnecting,
		state:        s,
		network:      nc,
		connectSpinner: sp,
		nickInput:    ni,
		lobbyInput:   li,
		progressBar:  pb,
		chatViewport: viewport.New(viewport.WithWidth(80), viewport.WithHeight(20)),
		gameInput:    gi,
		funRating:    3,
		immersion:    3,
		commentInput: ci,
	}
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.connectSpinner.Tick,
		m.network.ConnectCmd(),
	)
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.state.Width = msg.Width
		m.state.Height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			m.network.Disconnect()
			return m, tea.Quit
		}

	case network.ConnectSuccess:
		if m.state.PlayerID != "" {
			// Reconnect: rejoin
			return m, tea.Batch(
				m.network.ListenCmd(),
				m.network.SendCmd(protocol.ClientMessage{Type: "rejoin", PlayerID: m.state.PlayerID}),
			)
		}
		if m.state.Nickname == "" {
			m.state.GamePhase = state.PhaseNickname
			m.screen = ScreenNickname
			return m, m.network.ListenCmd()
		}
		// Nickname provided via CLI flag; send join immediately
		return m, tea.Batch(
			m.network.ListenCmd(),
			m.network.SendCmd(protocol.ClientMessage{
				Type:     "join",
				Nickname: m.state.Nickname,
				// RoomCode is sent via URL path in the server, not in the message
			}),
		)

	case network.ConnectError:
		m.state.LastError = &state.ClientError{
			Code:    "CONNECTION_FAILED",
			Message: fmt.Sprintf("Failed to connect: %v", msg.Err),
		}
		return m, nil

	case network.ServerMsgReceived:
		m.state = state.ApplyServerMessage(m.state, msg.Msg)
		m = m.syncScreenFromPhase()

		// Handle screen-specific server message logic
		var screenCmd tea.Cmd
		m, screenCmd = m.handleScreenServerMsg(msg)
		if screenCmd != nil {
			cmds = append(cmds, screenCmd)
		}

		cmds = append(cmds, m.network.ListenCmd())
		return m, tea.Batch(cmds...)

	case network.Disconnected:
		m.state.ConnectionStatus = state.StatusReconnecting
		return m, network.ReconnectCmd(1)

	case network.ParseError:
		// Skip bad messages, keep listening
		return m, m.network.ListenCmd()

	case network.ReconnectFailed:
		m.state.LastError = &state.ClientError{
			Code:    "CONNECTION_LOST",
			Message: "Connection lost. Please rejoin the game.",
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.connectSpinner, cmd = m.connectSpinner.Update(msg)
		return m, cmd
	}

	// Delegate to current screen
	return m.updateCurrentScreen(msg)
}

func (m AppModel) View() tea.View {
	w := m.state.Width
	h := m.state.Height

	var content string

	switch m.screen {
	case ScreenConnecting:
		content = m.viewConnecting(w)
	case ScreenNickname:
		content = m.viewNickname(w)
	case ScreenLobby:
		content = screens.RenderLobby(m.state, w)
	case ScreenGenerating:
		content = screens.RenderGenerating(m.state, m.progressBar, w)
	case ScreenBriefing:
		content = screens.RenderBriefing(m.state, m.briefingPhase, w)
	case ScreenGame:
		content = screens.RenderPlaying(m.state, &m.chatViewport, m.gameInput, w, h)
	case ScreenEnding:
		content = screens.RenderEnding(m.state, m.endingPhase, m.funRating, m.immersion, m.focusField, m.commentInput, w)
	case ScreenFinished:
		content = m.viewFinished(w)
	}

	// Show error overlay if present (except on game screen where it's in the log)
	if m.state.LastError != nil && m.screen != ScreenGame {
		errMsg := lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).
			Bold(true).
			Render("Error: " + m.state.LastError.Message)
		content = content + "\n" + errMsg
	}

	v := tea.NewView(content)
	if m.screen == ScreenGame {
		v.AltScreen = true
	}
	return v
}

// syncScreenFromPhase synchronizes the screen with the current game phase.
func (m AppModel) syncScreenFromPhase() AppModel {
	switch m.state.GamePhase {
	case state.PhaseConnecting:
		m.screen = ScreenConnecting
	case state.PhaseNickname:
		m.screen = ScreenNickname
	case state.PhaseLobby:
		m.screen = ScreenLobby
	case state.PhaseGenerating:
		m.screen = ScreenGenerating
	case state.PhaseBriefing:
		m.screen = ScreenBriefing
	case state.PhasePlaying:
		m.screen = ScreenGame
	case state.PhaseEnding:
		m.screen = ScreenEnding
	case state.PhaseFinished:
		m.screen = ScreenFinished
	}
	return m
}

// handleScreenServerMsg handles screen-specific server message processing.
func (m AppModel) handleScreenServerMsg(msg network.ServerMsgReceived) (AppModel, tea.Cmd) {
	switch m.screen {
	case ScreenNickname:
		if msg.Msg.Type == "error" && msg.Msg.Code == string(protocol.ErrorCodeDuplicateNickname) {
			m.nickError = "Nickname already taken. Please choose another."
			m.nickInput.Reset()
		}
	case ScreenBriefing:
		if msg.Msg.Type == "briefing_private" {
			m.briefingPhase = screens.BriefingPrivate
		}
	case ScreenEnding:
		if msg.Msg.Type == "feedback_request" {
			m.endingPhase = screens.EndingFeedback
			m.funRating = 3
			m.immersion = 3
			m.focusField = 0
		}
	}
	return m, nil
}

// updateCurrentScreen delegates input handling to the active screen.
func (m AppModel) updateCurrentScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case ScreenNickname:
		return m.updateNickname(msg)
	case ScreenLobby:
		return m.updateLobby(msg)
	case ScreenBriefing:
		return m.updateBriefing(msg)
	case ScreenGame:
		return m.updateGame(msg)
	case ScreenEnding:
		return m.updateEnding(msg)
	}
	return m, nil
}

// --- Screen-specific update handlers ---

func (m AppModel) updateNickname(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "enter" {
			nick := strings.TrimSpace(m.nickInput.Value())
			if len([]rune(nick)) < 1 || len([]rune(nick)) > 20 {
				m.nickError = "Nickname must be 1-20 characters."
				return m, nil
			}
			for _, r := range nick {
				if r < 0x20 {
					m.nickError = "Nickname cannot contain control characters."
					return m, nil
				}
			}
			m.nickError = ""
			m.state.Nickname = nick
			return m, m.network.SendCmd(protocol.ClientMessage{Type: "join", Nickname: nick})
		}
		var cmd tea.Cmd
		m.nickInput, cmd = m.nickInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m AppModel) updateLobby(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "enter" {
			raw := strings.TrimSpace(m.lobbyInput.Value())
			if raw == "" && m.state.IsHost {
				return m, m.network.SendCmd(protocol.ClientMessage{Type: "start_game"})
			}
			if raw != "" {
				m.lobbyInput.Reset()
				return m, m.network.SendCmd(protocol.ClientMessage{Type: "chat", Content: raw})
			}
		}
		var cmd tea.Cmd
		m.lobbyInput, cmd = m.lobbyInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m AppModel) updateBriefing(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "enter" {
			switch m.briefingPhase {
			case screens.BriefingPublic:
				m.briefingPhase = screens.BriefingWaitingPrivate
				return m, m.network.SendCmd(protocol.ClientMessage{Type: "ready", Phase: "briefing_read"})
			case screens.BriefingPrivate:
				m.briefingPhase = screens.BriefingWaitingReady
				return m, m.network.SendCmd(protocol.ClientMessage{Type: "ready", Phase: "game_ready"})
			}
		}
	}
	return m, nil
}

func (m AppModel) updateGame(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			raw := m.gameInput.Value()
			m.gameInput.Reset()
			raw = strings.TrimSpace(raw)
			if raw == "" {
				return m, nil
			}
			parsed := input.ParseInput(raw)
			result := input.CommandToClientMessage(parsed)
			if result.Message != nil {
				return m, m.network.SendCmd(*result.Message)
			}
			m.state = state.AddSystemMessage(m.state, result.ErrorMsg)
			return m, nil

		case "tab":
			raw := m.gameInput.Value()
			ctx := input.CompletionContext{
				Commands: input.AvailableCommands,
			}
			// Populate context from current state
			if m.state.CurrentRoom != nil {
				for _, npc := range m.state.CurrentRoom.NPCs {
					ctx.NPCNames = append(ctx.NPCNames, npc.Name)
				}
			}
			if m.state.MapOverview != nil {
				for _, room := range m.state.MapOverview.Rooms {
					ctx.RoomNames = append(ctx.RoomNames, room.Name)
				}
			}
			for _, p := range m.state.LobbyPlayers {
				ctx.Players = append(ctx.Players, p.Nickname)
			}

			candidates := input.Complete(raw, ctx)
			if len(candidates) == 1 {
				m.gameInput.SetValue(applyCompletion(raw, candidates[0]))
			} else if len(candidates) > 1 {
				m.state = state.AddSystemMessage(m.state, "Candidates: "+strings.Join(candidates, ", "))
			}
			return m, nil

		case "pgup", "pgdown":
			var cmd tea.Cmd
			m.chatViewport, cmd = m.chatViewport.Update(msg)
			return m, cmd
		}

		// Forward to text input
		var cmd tea.Cmd
		m.gameInput, cmd = m.gameInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m AppModel) updateEnding(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "enter" {
			switch m.endingPhase {
			case screens.EndingResult:
				m.endingPhase = screens.EndingReveal
			case screens.EndingReveal:
				// Wait for server's feedback_request
			case screens.EndingFeedback:
				var comment *string
				if v := m.commentInput.Value(); v != "" {
					comment = &v
				}
				return m, m.network.SendCmd(protocol.ClientMessage{
					Type:            "submit_feedback",
					FunRating:       m.funRating,
					ImmersionRating: m.immersion,
					Comment:         comment,
				})
			}
			return m, nil
		}
		if m.endingPhase == screens.EndingFeedback {
			switch msg.String() {
			case "esc":
				return m, m.network.SendCmd(protocol.ClientMessage{Type: "skip_feedback"})
			case "tab":
				m.focusField = (m.focusField + 1) % 3
				return m, nil
			case "left":
				switch m.focusField {
				case 0:
					if m.funRating > 1 {
						m.funRating--
					}
				case 1:
					if m.immersion > 1 {
						m.immersion--
					}
				}
				return m, nil
			case "right":
				switch m.focusField {
				case 0:
					if m.funRating < 5 {
						m.funRating++
					}
				case 1:
					if m.immersion < 5 {
						m.immersion++
					}
				}
				return m, nil
			}
			if m.focusField == 2 {
				var cmd tea.Cmd
				m.commentInput, cmd = m.commentInput.Update(msg)
				return m, cmd
			}
		}
	}
	return m, nil
}

// --- View helpers ---

func (m AppModel) viewConnecting(width int) string {
	boxW := width - 4
	if boxW > 50 {
		boxW = 50
	}
	if boxW < 20 {
		boxW = 20
	}

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Padding(2, 4)

	return style.Width(boxW).Render(
		fmt.Sprintf("%s Connecting to server...", m.connectSpinner.View()),
	)
}

func (m AppModel) viewNickname(width int) string {
	boxW := width - 4
	if boxW > 50 {
		boxW = 50
	}
	if boxW < 20 {
		boxW = 20
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("Story - Enter Nickname")

	errLine := ""
	if m.nickError != "" {
		errLine = "\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(m.nickError)
	}

	body := fmt.Sprintf(
		"  Enter a nickname (1-20 chars):\n  %s%s\n\n  Press Enter to join",
		m.nickInput.View(),
		errLine,
	)

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Padding(1, 2)

	return style.Width(boxW).Render(title + "\n\n" + body)
}

func (m AppModel) viewFinished(width int) string {
	boxW := width - 4
	if boxW > 60 {
		boxW = 60
	}
	if boxW < 20 {
		boxW = 20
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("=== Game Over ===")
	body := lipgloss.JoinVertical(lipgloss.Left,
		"  The game has ended.",
		"",
		"  To play again, start a new game.",
		"  Press Ctrl+C to exit.",
	)

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Padding(1, 2)

	return style.Width(boxW).Render(title + "\n\n" + body)
}

// applyCompletion replaces the current incomplete argument with the completed value.
func applyCompletion(raw, completed string) string {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "/") {
		return raw
	}

	spaceIdx := strings.Index(trimmed, " ")
	if spaceIdx == -1 {
		// Complete the command itself
		return "/" + completed + " "
	}

	// Complete the argument
	return trimmed[:spaceIdx+1] + completed + " "
}
