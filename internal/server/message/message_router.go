package message

import (
	"time"

	"github.com/anthropics/story/internal/server/eventbus"
	"github.com/anthropics/story/internal/server/game"
	"github.com/anthropics/story/internal/server/network"
	"github.com/anthropics/story/internal/shared/protocol"
	"github.com/anthropics/story/internal/shared/types"
)

// MessageRouter routes game events to the appropriate players based on visibility.
type MessageRouter struct {
	network   *network.NetworkServer
	gameState *game.GameStateManager
	eventBus  *eventbus.EventBus
}

// NewMessageRouter creates a new MessageRouter and subscribes to EventBus channels.
func NewMessageRouter(
	net *network.NetworkServer,
	gs *game.GameStateManager,
	bus *eventbus.EventBus,
) *MessageRouter {
	mr := &MessageRouter{
		network:   net,
		gameState: gs,
		eventBus:  bus,
	}
	go mr.listenGameEvents(bus.SubscribeGameEvent())
	go mr.listenChat(bus.SubscribeChat())
	go mr.listenSendEndings(bus.SubscribeSendEndings())
	go mr.listenFeedback(bus.SubscribeFeedback())
	return mr
}

// BroadcastMapUpdate sends a personalized map_info to each player.
func (mr *MessageRouter) BroadcastMapUpdate() {
	for _, playerID := range mr.gameState.GetAllPlayerIDs() {
		mapView := mr.gameState.GetMapView(playerID)
		mr.network.SendTo(playerID, protocol.MapInfoMessage{
			Type: protocol.SMsgTypeMapInfo,
			Map:  mapView,
		})
	}
}

// listenGameEvents receives game events and routes them based on visibility.
func (mr *MessageRouter) listenGameEvents(ch <-chan types.GameEvent) {
	for event := range ch {
		mr.routeEvent(event)
	}
}

// listenChat receives chat events and routes them by scope.
func (mr *MessageRouter) listenChat(ch <-chan eventbus.ChatData) {
	for chat := range ch {
		mr.routeChat(chat)
	}
}

// listenSendEndings receives ending data and routes personalized endings to each player.
func (mr *MessageRouter) listenSendEndings(ch <-chan types.GameEndData) {
	for endData := range ch {
		mr.routeEndings(endData)
	}
}

// listenFeedback receives feedback events and sends acknowledgements.
func (mr *MessageRouter) listenFeedback(ch <-chan types.Feedback) {
	for fb := range ch {
		mr.routeFeedback(fb)
	}
}

// routeEvent sends a game event to the appropriate recipients based on visibility.
func (mr *MessageRouter) routeEvent(event types.GameEvent) {
	recipients := mr.resolveRecipients(event.GetBaseEvent().Visibility)
	msg := protocol.GameEventMessage{
		Type: protocol.SMsgTypeGameEvent,
	}
	// The event is serialized as part of the message
	mr.network.SendToMany(recipients, struct {
		Type  string          `json:"type"`
		Event types.GameEvent `json:"event"`
	}{
		Type:  protocol.SMsgTypeGameEvent,
		Event: event,
	})
	_ = msg // suppress unused
}

// routeChat routes chat messages based on scope.
func (mr *MessageRouter) routeChat(chat eventbus.ChatData) {
	var senderLocation *string
	if chat.Scope == "global" {
		room := mr.gameState.GetPlayerRoom(chat.SenderID)
		if room != nil {
			name := room.Name
			senderLocation = &name
		}
	}

	msg := protocol.ChatServerMessage{
		Type:           protocol.SMsgTypeChat,
		SenderID:       chat.SenderID,
		SenderName:     chat.SenderName,
		Content:        chat.Content,
		Scope:          chat.Scope,
		SenderLocation: senderLocation,
		Timestamp:      time.Now().UnixMilli(),
	}

	if chat.Scope == "global" {
		mr.network.SendToAll(msg)
	} else {
		roomPlayers := mr.gameState.GetPlayersInRoom(chat.RoomID)
		recipientIDs := make([]string, len(roomPlayers))
		for i, p := range roomPlayers {
			recipientIDs[i] = p.ID
		}
		mr.network.SendToMany(recipientIDs, msg)
	}
}

// routeEndings sends personalized endings to each player.
func (mr *MessageRouter) routeEndings(endData types.GameEndData) {
	for _, playerID := range mr.gameState.GetAllPlayerIDs() {
		var personalEnding types.PlayerEnding
		for _, pe := range endData.PlayerEndings {
			if pe.PlayerID == playerID {
				personalEnding = pe
				break
			}
		}
		mr.network.SendTo(playerID, protocol.GameEndingMessage{
			Type:           protocol.SMsgTypeGameEnding,
			CommonResult:   endData.CommonResult,
			PersonalEnding: personalEnding,
			SecretReveal:   endData.SecretReveal,
		})
	}
}

// routeFeedback sends feedback acknowledgement to the player.
func (mr *MessageRouter) routeFeedback(fb types.Feedback) {
	mr.network.SendTo(fb.PlayerID, protocol.FeedbackAckMessage{
		Type: protocol.SMsgTypeFeedbackAck,
	})
}

// resolveRecipients converts an EventVisibility to a list of player IDs.
func (mr *MessageRouter) resolveRecipients(visibility types.EventVisibility) []string {
	switch visibility.Scope {
	case "all":
		return mr.gameState.GetAllPlayerIDs()
	case "room":
		players := mr.gameState.GetPlayersInRoom(visibility.RoomID)
		ids := make([]string, len(players))
		for i, p := range players {
			ids[i] = p.ID
		}
		return ids
	case "player", "players":
		return visibility.PlayerIDs
	default:
		return nil
	}
}
