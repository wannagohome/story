package state

import (
	"github.com/anthropics/story/internal/shared/protocol"
)

// Re-export protocol types used in client state for convenience.
type LobbyPlayer = protocol.LobbyPlayer
type RoomView = protocol.RoomView
type MapView = protocol.MapView
type MapViewRoom = protocol.MapViewRoom
type PlayerRole = protocol.PlayerRole
type Item = protocol.Item
type Clue = protocol.Clue
type PublicInfo = protocol.PublicInfo
type SemiPublicInfo = protocol.SemiPublicInfo
type PlayerEnding = protocol.PlayerEnding
type SecretReveal = protocol.SecretReveal
type PlayerLocationInfo = protocol.PlayerLocationInfo
type CommandInfo = protocol.CommandInfo
type VoteResultEntry = protocol.VoteResultEntry

type ConnectionStatus string

const (
	StatusConnecting   ConnectionStatus = "connecting"
	StatusConnected    ConnectionStatus = "connected"
	StatusReconnecting ConnectionStatus = "reconnecting"
	StatusDisconnected ConnectionStatus = "disconnected"
)

type GamePhase string

const (
	PhaseConnecting GamePhase = "connecting"
	PhaseNickname   GamePhase = "nickname"
	PhaseLobby      GamePhase = "lobby"
	PhaseGenerating GamePhase = "generating"
	PhaseBriefing   GamePhase = "briefing"
	PhasePlaying    GamePhase = "playing"
	PhaseEnding     GamePhase = "ending"
	PhaseFinished   GamePhase = "finished"
)

type ClientState struct {
	// Connection
	ConnectionStatus ConnectionStatus
	PlayerID         string
	Nickname         string
	RoomCode         string
	GamePhase        GamePhase
	IsHost           bool

	// Lobby
	LobbyPlayers []LobbyPlayer
	MaxPlayers   int

	// Briefing
	BriefingPublic     *PublicInfo
	BriefingSecrets    []string
	BriefingSemiPublic []SemiPublicInfo

	// Game
	WorldTitle      string
	MyRole          *PlayerRole
	CurrentRoom     *RoomView
	MapOverview     *MapView
	Inventory       []Item
	DiscoveredClues []Clue

	// Chat/Events
	Messages []DisplayMessage

	// Voting
	ActiveVote        *ActiveVoteState
	LastVoteResult    *VoteResult
	ActiveEndProposal *EndProposalState

	// Consensus
	ActiveSolve *ActiveSolveState

	// World Generation
	GenerationMessage  string
	GenerationProgress float64

	// Ending
	EndingData        *EndingData
	FeedbackRequested bool

	// Error
	LastError *ClientError

	// Window size
	Width  int
	Height int
}

func NewClientState() ClientState {
	return ClientState{
		ConnectionStatus: StatusConnecting,
		GamePhase:        PhaseConnecting,
		MaxPlayers:       8,
		Width:            80,
		Height:           24,
	}
}
