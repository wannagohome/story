package protocol

import (
	"encoding/json"

	"github.com/anthropics/story/internal/shared/types"
)

// Re-export types used in server messages for convenience.
type PublicInfo = types.PublicInfo
type PlayerRole = types.PlayerRole
type SemiPublicInfo = types.SemiPublicInfo
type PlayerEnding = types.PlayerEnding
type SecretReveal = types.SecretReveal
type Item = types.Item
type Clue = types.Clue
type Connection = types.Connection

// --- Session Management ---

type JoinedMessage struct {
	Type     string `json:"type"`
	PlayerID string `json:"playerId"`
	RoomCode string `json:"roomCode"`
	IsHost   bool   `json:"isHost"`
}

type LobbyUpdateMessage struct {
	Type       string        `json:"type"`
	Players    []LobbyPlayer `json:"players"`
	MaxPlayers int           `json:"maxPlayers"`
}

type GenerationProgressMessage struct {
	Type     string  `json:"type"`
	Step     string  `json:"step"`
	Message  string  `json:"message"`
	Progress float64 `json:"progress"`
}

type ErrorMessage struct {
	Type    string    `json:"type"`
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

type PlayerDisconnectedMessage struct {
	Type     string `json:"type"`
	PlayerID string `json:"playerId"`
	Nickname string `json:"nickname"`
}

type PlayerReconnectedMessage struct {
	Type     string `json:"type"`
	PlayerID string `json:"playerId"`
	Nickname string `json:"nickname"`
}

// --- Game Start (Briefing) ---

type BriefingPublicMessage struct {
	Type string     `json:"type"`
	Info PublicInfo `json:"info"`
}

type BriefingPrivateMessage struct {
	Type           string           `json:"type"`
	Role           PlayerRole       `json:"role"`
	Secrets        []string         `json:"secrets"`
	SemiPublicInfo []SemiPublicInfo `json:"semiPublicInfo"`
}

type GameStartedMessage struct {
	Type        string   `json:"type"`
	InitialRoom RoomView `json:"initialRoom"`
}

// --- Game Progress ---

type ChatServerMessage struct {
	Type           string  `json:"type"`
	SenderID       string  `json:"senderId"`
	SenderName     string  `json:"senderName"`
	Content        string  `json:"content"`
	Scope          string  `json:"scope"`
	SenderLocation *string `json:"senderLocation,omitempty"`
	Timestamp      int64   `json:"timestamp"`
}

type GameEventMessage struct {
	Type  string          `json:"type"`
	Event json.RawMessage `json:"event"`
}

type RoomChangedMessage struct {
	Type string   `json:"type"`
	Room RoomView `json:"room"`
}

type PlayerJoinedRoomMessage struct {
	Type     string `json:"type"`
	PlayerID string `json:"playerId"`
	Nickname string `json:"nickname"`
}

type PlayerLeftRoomMessage struct {
	Type        string `json:"type"`
	PlayerID    string `json:"playerId"`
	Nickname    string `json:"nickname"`
	Destination string `json:"destination"`
}

type SystemMessage struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// --- Info Query Responses ---

type InventoryMessage struct {
	Type  string `json:"type"`
	Items []Item `json:"items"`
	Clues []Clue `json:"clues"`
}

type RoleInfoMessage struct {
	Type string     `json:"type"`
	Role PlayerRole `json:"role"`
}

type MapInfoMessage struct {
	Type string  `json:"type"`
	Map  MapView `json:"map"`
}

type WhoInfoMessage struct {
	Type    string               `json:"type"`
	Players []PlayerLocationInfo `json:"players"`
}

type HelpInfoMessage struct {
	Type     string        `json:"type"`
	Commands []CommandInfo `json:"commands"`
}

// --- Voting ---

type VoteStartedMessage struct {
	Type           string   `json:"type"`
	Reason         string   `json:"reason"`
	Candidates     []string `json:"candidates"`
	TimeoutSeconds int      `json:"timeoutSeconds"`
}

type VoteProgressMessage struct {
	Type        string `json:"type"`
	VotedCount  int    `json:"votedCount"`
	TotalVoters int    `json:"totalVoters"`
}

type VoteEndedMessage struct {
	Type    string            `json:"type"`
	Results []VoteResultEntry `json:"results"`
	Outcome string            `json:"outcome"`
}

type EndProposedMessage struct {
	Type           string `json:"type"`
	ProposerID     string `json:"proposerId"`
	ProposerName   string `json:"proposerName"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
}

type EndVoteResultMessage struct {
	Type      string `json:"type"`
	Agreed    int    `json:"agreed"`
	Disagreed int    `json:"disagreed"`
	Passed    bool   `json:"passed"`
}

// --- Consensus (solve system) ---

type SolveStartedMessage struct {
	Type           string `json:"type"`
	Prompt         string `json:"prompt"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
}

type SolveProgressMessage struct {
	Type           string `json:"type"`
	SubmittedCount int    `json:"submittedCount"`
	TotalPlayers   int    `json:"totalPlayers"`
}

type SolveAnswerEntry struct {
	PlayerID   string `json:"playerId"`
	PlayerName string `json:"playerName"`
	Answer     string `json:"answer"`
}

type SolveResultMessage struct {
	Type    string             `json:"type"`
	Answers []SolveAnswerEntry `json:"answers"`
	Outcome string             `json:"outcome"`
}

// --- Game End ---

type GameEndingMessage struct {
	Type           string       `json:"type"`
	CommonResult   string       `json:"commonResult"`
	PersonalEnding PlayerEnding `json:"personalEnding"`
	SecretReveal   SecretReveal `json:"secretReveal"`
}

type FeedbackRequestMessage struct {
	Type string `json:"type"`
}

type FeedbackAckMessage struct {
	Type string `json:"type"`
}

type GameCancelledMessage struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

type GameFinishedMessage struct {
	Type string `json:"type"`
}

// --- AI Processing Indicator ---

type ThinkingMessage struct {
	Type string `json:"type"`
}

// RawServerMessage is used for deferred parsing of server messages.
type RawServerMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"-"`
}
