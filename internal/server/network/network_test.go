package network

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/anthropics/story/internal/shared/protocol"
)

func getFreePort() int {
	// Use port 0 to let OS assign a free port, but for simplicity
	// we'll use a high port range with increments.
	// In practice, we start the server and parse the listener address.
	return 0
}

func TestStartAndStop(t *testing.T) {
	ns := NewNetworkServer(NetworkConfig{Port: 18901})
	if err := ns.Start("TEST-1234"); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Verify server is running by making an HTTP request
	resp, err := http.Get("http://localhost:18901/ws/TEST-1234")
	if err != nil {
		t.Fatalf("server not reachable: %v", err)
	}
	resp.Body.Close()

	if err := ns.Stop(); err != nil {
		t.Fatalf("failed to stop: %v", err)
	}
}

func TestWebSocketConnectivity(t *testing.T) {
	port := 18902
	ns := NewNetworkServer(NetworkConfig{Port: port})

	var connMu sync.Mutex
	connected := false

	ns.OnConnection(func(conn *websocket.Conn) {
		connMu.Lock()
		connected = true
		connMu.Unlock()
	})

	if err := ns.Start("ROOM-001"); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer ns.Stop()
	time.Sleep(50 * time.Millisecond)

	url := fmt.Sprintf("ws://localhost:%d/ws/ROOM-001", port)
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer ws.Close()

	time.Sleep(50 * time.Millisecond)

	connMu.Lock()
	if !connected {
		t.Fatal("OnConnection handler was not called")
	}
	connMu.Unlock()
}

func TestSendAndReceiveMessage(t *testing.T) {
	port := 18903
	ns := NewNetworkServer(NetworkConfig{Port: port})

	receivedCh := make(chan protocol.ClientMessage, 1)

	ns.OnConnection(func(conn *websocket.Conn) {
		ns.BindPlayerToSocket("player-1", conn)
	})

	ns.OnMessage(func(playerId string, msg protocol.ClientMessage) {
		if playerId == "player-1" {
			receivedCh <- msg
		}
	})

	if err := ns.Start("ROOM-002"); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer ns.Stop()
	time.Sleep(50 * time.Millisecond)

	url := fmt.Sprintf("ws://localhost:%d/ws/ROOM-002", port)
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer ws.Close()

	time.Sleep(50 * time.Millisecond)

	// Send a message from client
	clientMsg := protocol.ClientMessage{
		Type:    "chat",
		Content: "hello",
	}
	data, _ := json.Marshal(clientMsg)
	if err := ws.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("failed to send message: %v", err)
	}

	select {
	case received := <-receivedCh:
		if received.Type != "chat" {
			t.Fatalf("expected type chat, got %s", received.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestSendToPlayer(t *testing.T) {
	port := 18904
	ns := NewNetworkServer(NetworkConfig{Port: port})

	ns.OnConnection(func(conn *websocket.Conn) {
		ns.BindPlayerToSocket("player-1", conn)
	})

	if err := ns.Start("ROOM-003"); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer ns.Stop()
	time.Sleep(50 * time.Millisecond)

	url := fmt.Sprintf("ws://localhost:%d/ws/ROOM-003", port)
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer ws.Close()
	time.Sleep(50 * time.Millisecond)

	// Server sends a message to the player
	serverMsg := protocol.SystemMessage{Type: "system", Content: "welcome"}
	ns.SendTo("player-1", serverMsg)

	// Read the message on the client side
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msgData, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read message: %v", err)
	}

	var received protocol.SystemMessage
	if err := json.Unmarshal(msgData, &received); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if received.Type != "system" {
		t.Fatalf("expected type system, got %s", received.Type)
	}
}

func TestDisconnection(t *testing.T) {
	port := 18905
	ns := NewNetworkServer(NetworkConfig{Port: port})

	discCh := make(chan string, 1)

	ns.OnConnection(func(conn *websocket.Conn) {
		ns.BindPlayerToSocket("player-1", conn)
	})

	ns.OnDisconnection(func(playerId string) {
		discCh <- playerId
	})

	if err := ns.Start("ROOM-004"); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer ns.Stop()
	time.Sleep(50 * time.Millisecond)

	url := fmt.Sprintf("ws://localhost:%d/ws/ROOM-004", port)
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Close the client connection
	ws.Close()

	select {
	case playerID := <-discCh:
		if playerID != "player-1" {
			t.Fatalf("expected player-1, got %s", playerID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for disconnection")
	}
}

func TestBindAndUnbindPlayer(t *testing.T) {
	ns := NewNetworkServer(NetworkConfig{Port: 0})

	// Bind a nil connection just to test the map operations
	ns.BindPlayerToSocket("p1", nil)

	ns.mu.RLock()
	_, exists := ns.clients["p1"]
	ns.mu.RUnlock()
	if !exists {
		t.Fatal("expected player p1 to be bound")
	}

	ns.UnbindPlayer("p1")

	ns.mu.RLock()
	_, exists = ns.clients["p1"]
	ns.mu.RUnlock()
	if exists {
		t.Fatal("expected player p1 to be unbound")
	}
}
