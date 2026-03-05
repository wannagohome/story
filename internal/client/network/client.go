package network

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"

	"github.com/anthropics/story/internal/client/state"
	"github.com/anthropics/story/internal/shared/protocol"

	tea "charm.land/bubbletea/v2"
)

// ServerMsgReceived is sent when a server message is received.
type ServerMsgReceived struct {
	Msg state.ServerMessage
}

// ConnectSuccess is sent when the WebSocket connection is established.
type ConnectSuccess struct{}

// ConnectError is sent when connection fails.
type ConnectError struct {
	Err error
}

// Disconnected is sent when the connection is lost.
type Disconnected struct{}

// ParseError is sent when a server message cannot be parsed.
type ParseError struct {
	Err error
}

// SendError is sent when a message cannot be serialized.
type SendError struct {
	Err error
}

// ReconnectFailed is sent when all reconnect attempts are exhausted.
type ReconnectFailed struct{}

type reconnectTick struct {
	Attempt int
}

// NetworkClient manages the WebSocket connection to the game server.
type NetworkClient struct {
	conn      *websocket.Conn
	ServerURL string
	send      chan []byte
}

// NewNetworkClient creates a new network client targeting the given server URL.
func NewNetworkClient(serverURL string) *NetworkClient {
	return &NetworkClient{
		ServerURL: serverURL,
		send:      make(chan []byte, 64),
	}
}

// ConnectCmd returns a tea.Cmd that establishes the WebSocket connection.
func (nc *NetworkClient) ConnectCmd() tea.Cmd {
	return func() tea.Msg {
		conn, _, err := websocket.DefaultDialer.Dial(nc.ServerURL, nil)
		if err != nil {
			return ConnectError{Err: err}
		}
		nc.conn = conn
		go nc.writeLoop()
		return ConnectSuccess{}
	}
}

// SendCmd returns a tea.Cmd that serializes and sends a ClientMessage.
func (nc *NetworkClient) SendCmd(msg protocol.ClientMessage) tea.Cmd {
	return func() tea.Msg {
		data, err := json.Marshal(msg)
		if err != nil {
			return SendError{Err: err}
		}
		select {
		case nc.send <- data:
		default:
			return SendError{Err: err}
		}
		return nil
	}
}

// ListenCmd returns a tea.Cmd that reads the next message from the server.
// After processing, the caller must issue ListenCmd again.
func (nc *NetworkClient) ListenCmd() tea.Cmd {
	return func() tea.Msg {
		if nc.conn == nil {
			return Disconnected{}
		}
		_, data, err := nc.conn.ReadMessage()
		if err != nil {
			return Disconnected{}
		}
		var msg state.ServerMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return ParseError{Err: err}
		}
		return ServerMsgReceived{Msg: msg}
	}
}

// Disconnect closes the connection.
func (nc *NetworkClient) Disconnect() {
	close(nc.send)
	if nc.conn != nil {
		nc.conn.Close()
	}
}

func (nc *NetworkClient) writeLoop() {
	for data := range nc.send {
		if nc.conn == nil {
			return
		}
		if err := nc.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			return
		}
	}
}

// BackoffDelay calculates the reconnect delay for the given attempt (1-indexed).
func BackoffDelay(attempt int) time.Duration {
	return time.Duration(1<<(attempt-1)) * time.Second
}

// ReconnectCmd returns a tea.Cmd that waits and then attempts to reconnect.
func ReconnectCmd(attempt int) tea.Cmd {
	if attempt > 3 {
		return func() tea.Msg {
			return ReconnectFailed{}
		}
	}
	delay := BackoffDelay(attempt)
	return tea.Tick(delay, func(_ time.Time) tea.Msg {
		return reconnectTick{Attempt: attempt}
	})
}
