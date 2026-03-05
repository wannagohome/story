package network

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"

	"github.com/anthropics/story/internal/shared/protocol"
)

const maxMessageSize = 64 * 1024 // 64KB

// NetworkConfig holds configuration for the network server.
type NetworkConfig struct {
	Port int
}

// NetworkServer manages WebSocket connections and message routing.
type NetworkServer struct {
	upgrader      websocket.Upgrader
	clients       map[string]*websocket.Conn
	mu            sync.RWMutex
	config        NetworkConfig
	server        *http.Server
	onConnHandler func(conn *websocket.Conn)
	onDiscHandler func(playerId string)
	onMsgHandler  func(playerId string, msg protocol.ClientMessage)
}

// NewNetworkServer creates a new NetworkServer with the given config.
func NewNetworkServer(config NetworkConfig) *NetworkServer {
	return &NetworkServer{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		clients: make(map[string]*websocket.Conn),
		config:  config,
	}
}

// Start begins the HTTP server in a background goroutine, serving WebSocket
// connections at /ws/{roomCode}.
func (ns *NetworkServer) Start(roomCode string) error {
	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/ws/%s", roomCode), ns.HandleConnection)

	ns.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", ns.config.Port),
		Handler: mux,
	}

	go func() {
		if err := ns.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("network server error", "error", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the HTTP server.
func (ns *NetworkServer) Stop() error {
	if ns.server == nil {
		return nil
	}
	return ns.server.Shutdown(context.Background())
}

// OnConnection registers a handler called when a new WebSocket connection is established.
func (ns *NetworkServer) OnConnection(handler func(conn *websocket.Conn)) {
	ns.onConnHandler = handler
}

// OnDisconnection registers a handler called when a player disconnects.
func (ns *NetworkServer) OnDisconnection(handler func(playerId string)) {
	ns.onDiscHandler = handler
}

// OnMessage registers a handler called when a message is received from a player.
func (ns *NetworkServer) OnMessage(handler func(playerId string, msg protocol.ClientMessage)) {
	ns.onMsgHandler = handler
}

// SendTo sends a JSON message to a specific player.
func (ns *NetworkServer) SendTo(playerId string, msg interface{}) {
	ns.mu.RLock()
	conn, ok := ns.clients[playerId]
	ns.mu.RUnlock()
	if !ok {
		slog.Warn("network: player not found for SendTo", "playerId", playerId)
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("network: failed to marshal message", "error", err)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		slog.Error("network: failed to send message", "playerId", playerId, "error", err)
	}
}

// SendToMany sends a JSON message to multiple players.
func (ns *NetworkServer) SendToMany(playerIds []string, msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("network: failed to marshal message", "error", err)
		return
	}

	ns.mu.RLock()
	defer ns.mu.RUnlock()

	for _, id := range playerIds {
		conn, ok := ns.clients[id]
		if !ok {
			continue
		}
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			slog.Error("network: failed to send message", "playerId", id, "error", err)
		}
	}
}

// SendToAll sends a JSON message to all connected players.
func (ns *NetworkServer) SendToAll(msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("network: failed to marshal message", "error", err)
		return
	}

	ns.mu.RLock()
	defer ns.mu.RUnlock()

	for id, conn := range ns.clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			slog.Error("network: failed to send message", "playerId", id, "error", err)
		}
	}
}

// BindPlayerToSocket associates a player ID with a WebSocket connection.
func (ns *NetworkServer) BindPlayerToSocket(playerId string, conn *websocket.Conn) {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	ns.clients[playerId] = conn
}

// UnbindPlayer removes the association between a player ID and its WebSocket connection.
func (ns *NetworkServer) UnbindPlayer(playerId string) {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	delete(ns.clients, playerId)
}

// HandleConnection upgrades an HTTP request to a WebSocket connection,
// calls the connection handler, and starts a read loop.
func (ns *NetworkServer) HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := ns.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("network: failed to upgrade connection", "error", err)
		return
	}

	conn.SetReadLimit(maxMessageSize)

	if ns.onConnHandler != nil {
		ns.onConnHandler(conn)
	}

	go ns.readLoop(conn)
}

// readLoop reads messages from a WebSocket connection until it closes.
func (ns *NetworkServer) readLoop(conn *websocket.Conn) {
	defer conn.Close()

	// Find the player ID for this connection (for disconnect handling).
	findPlayerID := func() string {
		ns.mu.RLock()
		defer ns.mu.RUnlock()
		for id, c := range ns.clients {
			if c == conn {
				return id
			}
		}
		return ""
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			playerID := findPlayerID()
			if playerID != "" && ns.onDiscHandler != nil {
				ns.onDiscHandler(playerID)
			}
			return
		}

		var msg protocol.ClientMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			slog.Warn("network: failed to unmarshal client message", "error", err)
			continue
		}

		playerID := findPlayerID()
		if playerID != "" && ns.onMsgHandler != nil {
			ns.onMsgHandler(playerID, msg)
		}
	}
}
