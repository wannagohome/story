package types

// GameEvent interface - all events implement this
type GameEvent interface {
	EventType() string
	GetBaseEvent() BaseEvent
}

type BaseEvent struct {
	ID         string          `json:"id"`
	Timestamp  int64           `json:"timestamp"`
	Visibility EventVisibility `json:"visibility"`
}

type EventVisibility struct {
	Scope     string   `json:"scope"`               // "all" | "room" | "player"
	RoomID    string   `json:"roomId,omitempty"`     // scope == "room"
	PlayerIDs []string `json:"playerIds,omitempty"`  // scope == "player"
}

type Result[T any, E any] struct {
	Ok    bool `json:"ok"`
	Value T    `json:"value,omitempty"`
	Error E    `json:"error,omitempty"`
}
