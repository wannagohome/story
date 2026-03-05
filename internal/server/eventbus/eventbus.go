package eventbus

import (
	"log/slog"
	"sync"

	"github.com/anthropics/story/internal/shared/types"
)

// ChatData represents a chat message event.
type ChatData struct {
	SenderID   string
	SenderName string
	RoomID     string
	Content    string
	Scope      string // "room" | "global"
}

// PlayerConnectedData represents a player connection event.
type PlayerConnectedData struct{ PlayerID string }

// PlayerDisconnectedData represents a player disconnection event.
type PlayerDisconnectedData struct{ PlayerID string }

// StateChangedData represents a game state change event.
type StateChangedData struct {
	ChangeType string
	Data       any
}

// GameStatusChangedData represents a game status transition event.
type GameStatusChangedData struct {
	From types.GameStatus
	To   types.GameStatus
}

const bufferSize = 256

// EventBus provides typed internal pub/sub using Go channels.
type EventBus struct {
	gameEventSubs     []chan types.GameEvent
	chatSubs          []chan ChatData
	playerConnSubs    []chan PlayerConnectedData
	playerDiscSubs    []chan PlayerDisconnectedData
	stateChangedSubs  []chan StateChangedData
	statusChangedSubs []chan GameStatusChangedData
	sendEndingsSubs   []chan types.GameEndData
	feedbackSubs      []chan types.Feedback
	mu                sync.RWMutex
	closed            bool
}

// NewEventBus creates a new EventBus.
func NewEventBus() *EventBus {
	return &EventBus{}
}

// Close marks the event bus as closed and closes all subscriber channels.
func (eb *EventBus) Close() {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.closed = true
	for _, ch := range eb.gameEventSubs {
		close(ch)
	}
	for _, ch := range eb.chatSubs {
		close(ch)
	}
	for _, ch := range eb.playerConnSubs {
		close(ch)
	}
	for _, ch := range eb.playerDiscSubs {
		close(ch)
	}
	for _, ch := range eb.stateChangedSubs {
		close(ch)
	}
	for _, ch := range eb.statusChangedSubs {
		close(ch)
	}
	for _, ch := range eb.sendEndingsSubs {
		close(ch)
	}
	for _, ch := range eb.feedbackSubs {
		close(ch)
	}
}

// SubscribeGameEvent creates and returns a buffered channel for game events.
func (eb *EventBus) SubscribeGameEvent() <-chan types.GameEvent {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	ch := make(chan types.GameEvent, bufferSize)
	eb.gameEventSubs = append(eb.gameEventSubs, ch)
	return ch
}

// PublishGameEvent sends a game event to all subscribers (non-blocking).
func (eb *EventBus) PublishGameEvent(event types.GameEvent) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	for _, ch := range eb.gameEventSubs {
		select {
		case ch <- event:
		default:
			slog.Warn("eventbus: game event subscriber buffer full, dropping event")
		}
	}
}

// SubscribeChat creates and returns a buffered channel for chat events.
func (eb *EventBus) SubscribeChat() <-chan ChatData {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	ch := make(chan ChatData, bufferSize)
	eb.chatSubs = append(eb.chatSubs, ch)
	return ch
}

// PublishChat sends a chat event to all subscribers (non-blocking).
func (eb *EventBus) PublishChat(data ChatData) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	for _, ch := range eb.chatSubs {
		select {
		case ch <- data:
		default:
			slog.Warn("eventbus: chat subscriber buffer full, dropping event")
		}
	}
}

// SubscribePlayerConnected creates and returns a buffered channel for player connected events.
func (eb *EventBus) SubscribePlayerConnected() <-chan PlayerConnectedData {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	ch := make(chan PlayerConnectedData, bufferSize)
	eb.playerConnSubs = append(eb.playerConnSubs, ch)
	return ch
}

// PublishPlayerConnected sends a player connected event to all subscribers (non-blocking).
func (eb *EventBus) PublishPlayerConnected(data PlayerConnectedData) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	for _, ch := range eb.playerConnSubs {
		select {
		case ch <- data:
		default:
			slog.Warn("eventbus: player connected subscriber buffer full, dropping event")
		}
	}
}

// SubscribePlayerDisconnected creates and returns a buffered channel for player disconnected events.
func (eb *EventBus) SubscribePlayerDisconnected() <-chan PlayerDisconnectedData {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	ch := make(chan PlayerDisconnectedData, bufferSize)
	eb.playerDiscSubs = append(eb.playerDiscSubs, ch)
	return ch
}

// PublishPlayerDisconnected sends a player disconnected event to all subscribers (non-blocking).
func (eb *EventBus) PublishPlayerDisconnected(data PlayerDisconnectedData) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	for _, ch := range eb.playerDiscSubs {
		select {
		case ch <- data:
		default:
			slog.Warn("eventbus: player disconnected subscriber buffer full, dropping event")
		}
	}
}

// SubscribeStateChanged creates and returns a buffered channel for state changed events.
func (eb *EventBus) SubscribeStateChanged() <-chan StateChangedData {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	ch := make(chan StateChangedData, bufferSize)
	eb.stateChangedSubs = append(eb.stateChangedSubs, ch)
	return ch
}

// PublishStateChanged sends a state changed event to all subscribers (non-blocking).
func (eb *EventBus) PublishStateChanged(data StateChangedData) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	for _, ch := range eb.stateChangedSubs {
		select {
		case ch <- data:
		default:
			slog.Warn("eventbus: state changed subscriber buffer full, dropping event")
		}
	}
}

// SubscribeGameStatusChanged creates and returns a buffered channel for game status changed events.
func (eb *EventBus) SubscribeGameStatusChanged() <-chan GameStatusChangedData {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	ch := make(chan GameStatusChangedData, bufferSize)
	eb.statusChangedSubs = append(eb.statusChangedSubs, ch)
	return ch
}

// PublishGameStatusChanged sends a game status changed event to all subscribers (non-blocking).
func (eb *EventBus) PublishGameStatusChanged(data GameStatusChangedData) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	for _, ch := range eb.statusChangedSubs {
		select {
		case ch <- data:
		default:
			slog.Warn("eventbus: game status changed subscriber buffer full, dropping event")
		}
	}
}

// SubscribeSendEndings creates and returns a buffered channel for send endings events.
func (eb *EventBus) SubscribeSendEndings() <-chan types.GameEndData {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	ch := make(chan types.GameEndData, bufferSize)
	eb.sendEndingsSubs = append(eb.sendEndingsSubs, ch)
	return ch
}

// PublishSendEndings sends a send endings event to all subscribers (non-blocking).
func (eb *EventBus) PublishSendEndings(data types.GameEndData) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	for _, ch := range eb.sendEndingsSubs {
		select {
		case ch <- data:
		default:
			slog.Warn("eventbus: send endings subscriber buffer full, dropping event")
		}
	}
}

// SubscribeFeedback creates and returns a buffered channel for feedback events.
func (eb *EventBus) SubscribeFeedback() <-chan types.Feedback {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	ch := make(chan types.Feedback, bufferSize)
	eb.feedbackSubs = append(eb.feedbackSubs, ch)
	return ch
}

// PublishFeedback sends a feedback event to all subscribers (non-blocking).
func (eb *EventBus) PublishFeedback(data types.Feedback) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	for _, ch := range eb.feedbackSubs {
		select {
		case ch <- data:
		default:
			slog.Warn("eventbus: feedback subscriber buffer full, dropping event")
		}
	}
}
