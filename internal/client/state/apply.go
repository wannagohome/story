package state

import (
	"encoding/json"
	"strings"
)

// ServerMessage is a generic envelope for all server-to-client messages.
// We unmarshal into this union struct and dispatch on Type.
type ServerMessage struct {
	Type string `json:"type"`

	// joined
	PlayerID string `json:"playerId,omitempty"`
	RoomCode string `json:"roomCode,omitempty"`
	IsHost   bool   `json:"isHost,omitempty"`

	// lobby_update
	Players    []LobbyPlayer `json:"players,omitempty"`
	MaxPlayers int           `json:"maxPlayers,omitempty"`

	// error
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`

	// player_disconnected / player_reconnected / player_joined_room
	Nickname string `json:"nickname,omitempty"`

	// player_left_room
	Destination string `json:"destination,omitempty"`

	// generation_progress
	Step     string  `json:"step,omitempty"`
	Progress float64 `json:"progress,omitempty"`

	// briefing_public
	Info *PublicInfo `json:"info,omitempty"`

	// briefing_private
	Role           *PlayerRole    `json:"role,omitempty"`
	Secrets        []string       `json:"secrets,omitempty"`
	SemiPublicInfo []SemiPublicInfo `json:"semiPublicInfo,omitempty"`

	// game_started
	InitialRoom *RoomView `json:"initialRoom,omitempty"`

	// chat_message
	SenderID       string  `json:"senderId,omitempty"`
	SenderName     string  `json:"senderName,omitempty"`
	Content        string  `json:"content,omitempty"`
	Scope          string  `json:"scope,omitempty"`
	SenderLocation *string `json:"senderLocation,omitempty"`
	Timestamp      int64   `json:"timestamp,omitempty"`

	// game_event
	Event json.RawMessage `json:"event,omitempty"`

	// room_changed
	Room *RoomView `json:"room,omitempty"`

	// inventory
	Items []Item `json:"items,omitempty"`
	Clues []Clue `json:"clues,omitempty"`

	// map_info
	Map *MapView `json:"map,omitempty"`

	// who_info
	WhoPlayers []PlayerLocationInfo `json:"whoPlayers,omitempty"`

	// help_info
	Commands []CommandInfo `json:"commands,omitempty"`

	// vote_started
	Reason         string   `json:"reason,omitempty"`
	Candidates     []string `json:"candidates,omitempty"`
	TimeoutSeconds int      `json:"timeoutSeconds,omitempty"`

	// vote_progress
	VotedCount  int `json:"votedCount,omitempty"`
	TotalVoters int `json:"totalVoters,omitempty"`

	// vote_ended
	Results []VoteResultEntry `json:"results,omitempty"`
	Outcome string            `json:"outcome,omitempty"`

	// end_proposed
	ProposerID   string `json:"proposerId,omitempty"`
	ProposerName string `json:"proposerName,omitempty"`

	// end_vote_result
	Agreed    int  `json:"agreed,omitempty"`
	Disagreed int  `json:"disagreed,omitempty"`
	Passed    bool `json:"passed,omitempty"`

	// solve_started
	Prompt string `json:"prompt,omitempty"`

	// solve_progress
	SubmittedCount int `json:"submittedCount,omitempty"`
	TotalPlayers   int `json:"totalPlayers,omitempty"`

	// game_ending
	CommonResult   string          `json:"commonResult,omitempty"`
	PersonalEnding *PlayerEnding   `json:"personalEnding,omitempty"`
	SecretReveal   *SecretReveal   `json:"secretReveal,omitempty"`

	// game_cancelled
	CancelReason string `json:"cancelReason,omitempty"`
}

// ApplyServerMessage applies a server message to the client state, returning the new state.
func ApplyServerMessage(s ClientState, msg ServerMessage) ClientState {
	switch msg.Type {
	case "joined":
		s.ConnectionStatus = StatusConnected
		s.PlayerID = msg.PlayerID
		s.RoomCode = msg.RoomCode
		s.IsHost = msg.IsHost
		s.GamePhase = PhaseLobby

	case "lobby_update":
		s.LobbyPlayers = msg.Players
		if msg.MaxPlayers > 0 {
			s.MaxPlayers = msg.MaxPlayers
		}

	case "error":
		s.LastError = &ClientError{Code: msg.Code, Message: msg.Message}

	case "player_disconnected":
		s = AddSystemMessage(s, msg.Nickname+" disconnected")

	case "player_reconnected":
		s = AddSystemMessage(s, msg.Nickname+" reconnected")

	case "generation_progress":
		s.GamePhase = PhaseGenerating
		s.GenerationMessage = msg.Message
		s.GenerationProgress = msg.Progress

	case "briefing_public":
		s.GamePhase = PhaseBriefing
		s.BriefingPublic = msg.Info
		if msg.Info != nil {
			s.WorldTitle = msg.Info.Title
		}

	case "briefing_private":
		s.MyRole = msg.Role
		s.BriefingSecrets = msg.Secrets
		s.BriefingSemiPublic = msg.SemiPublicInfo

	case "game_started":
		s.GamePhase = PhasePlaying
		s.CurrentRoom = msg.InitialRoom

	case "room_changed":
		s.CurrentRoom = msg.Room

	case "chat_message":
		s = AddMessage(s, DisplayMessage{
			ID:             generateID(),
			Kind:           "chat",
			SenderID:       msg.SenderID,
			SenderName:     msg.SenderName,
			Content:        msg.Content,
			Scope:          msg.Scope,
			SenderLocation: msg.SenderLocation,
			Timestamp:      msg.Timestamp,
		})

	case "game_event":
		if msg.Event != nil {
			var evt GameEvent
			if err := json.Unmarshal(msg.Event, &evt); err == nil {
				s = AddMessage(s, DisplayMessage{
					ID:        evt.ID,
					Kind:      "event",
					Event:     evt,
					Timestamp: evt.Timestamp,
				})
			}
		}

	case "system_message", "system":
		s = AddSystemMessage(s, msg.Content)

	case "player_joined_room":
		s = AddSystemMessage(s, msg.Nickname+" entered the room")

	case "player_left_room":
		s = AddSystemMessage(s, msg.Nickname+" left to "+msg.Destination)

	case "inventory":
		s.Inventory = msg.Items
		s.DiscoveredClues = msg.Clues

	case "role_info":
		s.MyRole = msg.Role

	case "map_info":
		s.MapOverview = msg.Map

	case "who_info":
		if msg.WhoPlayers != nil {
			lines := make([]string, 0, len(msg.WhoPlayers))
			for _, p := range msg.WhoPlayers {
				line := p.Nickname + " -> " + p.RoomName
				if p.Status == "disconnected" {
					line += " (disconnected)"
				}
				lines = append(lines, line)
			}
			s = AddSystemMessage(s, strings.Join(lines, "\n"))
		}

	case "help", "help_info":
		if msg.Commands != nil {
			lines := make([]string, 0, len(msg.Commands))
			for _, c := range msg.Commands {
				lines = append(lines, c.Command+" - "+c.Description)
			}
			s = AddSystemMessage(s, strings.Join(lines, "\n"))
		}

	case "vote_started":
		s.ActiveVote = &ActiveVoteState{
			Reason:         msg.Reason,
			Candidates:     msg.Candidates,
			TimeoutSeconds: msg.TimeoutSeconds,
		}

	case "vote_progress":
		if s.ActiveVote != nil {
			s.ActiveVote.VotedCount = msg.VotedCount
			s.ActiveVote.TotalVoters = msg.TotalVoters
		}

	case "vote_ended":
		s.ActiveVote = nil
		s.LastVoteResult = &VoteResult{Results: msg.Results, Outcome: msg.Outcome}

	case "solve_started":
		s.ActiveSolve = &ActiveSolveState{
			Prompt:         msg.Prompt,
			TimeoutSeconds: msg.TimeoutSeconds,
		}

	case "solve_progress":
		if s.ActiveSolve != nil {
			s.ActiveSolve.SubmittedCount = msg.SubmittedCount
			s.ActiveSolve.TotalPlayers = msg.TotalPlayers
		}

	case "solve_result":
		s.ActiveSolve = nil
		s = AddSystemMessage(s, "Consensus result: "+msg.Outcome)

	case "end_proposed":
		s.ActiveEndProposal = &EndProposalState{
			ProposerID:     msg.ProposerID,
			ProposerName:   msg.ProposerName,
			TimeoutSeconds: msg.TimeoutSeconds,
		}

	case "end_vote_result":
		s.ActiveEndProposal = nil
		outcome := "End game proposal was rejected."
		if msg.Passed {
			outcome = "End game has been decided."
		}
		s = AddSystemMessage(s, outcome)

	case "feedback_request":
		s.FeedbackRequested = true

	case "feedback_ack":
		s = AddSystemMessage(s, "Feedback submitted. Thank you!")

	case "game_ending":
		s.GamePhase = PhaseEnding
		ending := &EndingData{
			CommonResult: msg.CommonResult,
		}
		if msg.PersonalEnding != nil {
			ending.PersonalEnding = *msg.PersonalEnding
		}
		if msg.SecretReveal != nil {
			ending.SecretReveal = *msg.SecretReveal
		}
		s.EndingData = ending

	case "game_cancelled":
		s.GamePhase = PhaseFinished
		reason := msg.Reason
		if reason == "" {
			reason = msg.CancelReason
		}
		s.LastError = &ClientError{Code: "GAME_CANCELLED", Message: reason}
		s = AddSystemMessage(s, "Game cancelled: "+reason)

	case "game_finished":
		s.GamePhase = PhaseFinished
	}

	return s
}
