package schemas

import (
	"encoding/json"
	"fmt"
)

// NPCResponse is the AI response schema for NPC dialogue interactions.
type NPCResponse struct {
	Dialogue         string            `json:"dialogue"`
	Emotion          string            `json:"emotion"`
	InternalThought  string            `json:"internalThought"`
	InfoRevealed     []string          `json:"infoRevealed"`
	TrustChange      float64           `json:"trustChange"`
	TriggeredGimmick bool              `json:"triggeredGimmick"`
	Events           []json.RawMessage `json:"events"`
}

func (n *NPCResponse) Validate() error {
	if n.TrustChange < -1 {
		return fmt.Errorf("trustChange: minimum -1 required")
	}
	if n.TrustChange > 1 {
		return fmt.Errorf("trustChange: maximum 1 allowed")
	}
	return nil
}

// ParsedEvents deserializes all events in the NPC response.
func (n *NPCResponse) ParsedEvents() ([]AIGameEvent, error) {
	events := make([]AIGameEvent, 0, len(n.Events))
	for _, raw := range n.Events {
		e, err := ParseAIGameEvent(raw)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}
