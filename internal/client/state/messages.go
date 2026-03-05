package state

import (
	"crypto/rand"
	"fmt"
	"time"
)

const maxMessages = 200

func AddMessage(s ClientState, msg DisplayMessage) ClientState {
	s.Messages = append(s.Messages, msg)
	if len(s.Messages) > maxMessages {
		s.Messages = s.Messages[len(s.Messages)-maxMessages:]
	}
	return s
}

func AddSystemMessage(s ClientState, content string) ClientState {
	return AddMessage(s, DisplayMessage{
		ID:        generateID(),
		Kind:      "system",
		Content:   content,
		Timestamp: time.Now().UnixMilli(),
	})
}

func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
