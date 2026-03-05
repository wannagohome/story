// Package store provides client-side state management for the Story game.
// It re-exports the state package types for backward compatibility with the
// directory structure specified in the design docs.
package store

import "github.com/anthropics/story/internal/client/state"

// Re-export core types from the state package.
type ClientState = state.ClientState
type GamePhase = state.GamePhase
type ConnectionStatus = state.ConnectionStatus
type DisplayMessage = state.DisplayMessage
type GameEvent = state.GameEvent
type ServerMessage = state.ServerMessage

// Re-export constructor and apply function.
var NewClientState = state.NewClientState
var ApplyServerMessage = state.ApplyServerMessage
var AddSystemMessage = state.AddSystemMessage
var AddMessage = state.AddMessage

// Re-export constants.
const (
	StatusConnecting   = state.StatusConnecting
	StatusConnected    = state.StatusConnected
	StatusReconnecting = state.StatusReconnecting
	StatusDisconnected = state.StatusDisconnected

	PhaseConnecting = state.PhaseConnecting
	PhaseNickname   = state.PhaseNickname
	PhaseLobby      = state.PhaseLobby
	PhaseGenerating = state.PhaseGenerating
	PhaseBriefing   = state.PhaseBriefing
	PhasePlaying    = state.PhasePlaying
	PhaseEnding     = state.PhaseEnding
	PhaseFinished   = state.PhaseFinished
)
