package action

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/story/internal/server/aiface"
	"github.com/anthropics/story/internal/server/end"
	"github.com/anthropics/story/internal/server/eventbus"
	"github.com/anthropics/story/internal/server/game"
	"github.com/anthropics/story/internal/server/mapengine"
	"github.com/anthropics/story/internal/server/message"
	"github.com/anthropics/story/internal/server/network"
	"github.com/anthropics/story/internal/shared/events"
	"github.com/anthropics/story/internal/shared/protocol"
	"github.com/anthropics/story/internal/shared/schemas"
	"github.com/anthropics/story/internal/shared/types"
)

const aiTimeout = 15 * time.Second

// ActionProcessor handles game-time player commands.
type ActionProcessor struct {
	gameState     *game.GameStateManager
	mapEngine     *mapengine.MapEngine
	aiLayer       aiface.AILayer
	endCondition  *end.EndConditionEngine
	eventBus      *eventbus.EventBus
	network       *network.NetworkServer
	messageRouter *message.MessageRouter
}

// NewActionProcessor creates a new ActionProcessor.
func NewActionProcessor(
	gs *game.GameStateManager,
	me *mapengine.MapEngine,
	ail aiface.AILayer,
	ece *end.EndConditionEngine,
	bus *eventbus.EventBus,
	net *network.NetworkServer,
	mr *message.MessageRouter,
) *ActionProcessor {
	return &ActionProcessor{
		gameState:     gs,
		mapEngine:     me,
		aiLayer:       ail,
		endCondition:  ece,
		eventBus:      bus,
		network:       net,
		messageRouter: mr,
	}
}

// ProcessMessage dispatches a client message to the appropriate handler.
func (ap *ActionProcessor) ProcessMessage(playerID string, msg protocol.ClientMessage) error {
	switch msg.Type {
	case "chat":
		return ap.handleChat(playerID, msg)
	case "shout":
		return ap.handleShout(playerID, msg)
	case "move":
		return ap.handleMove(playerID, msg)
	case "examine":
		return ap.handleExamine(playerID, msg)
	case "do":
		return ap.handleDo(playerID, msg)
	case "talk":
		return ap.handleTalk(playerID, msg)
	case "give":
		return ap.handleGive(playerID)
	case "vote":
		return ap.handleVote(playerID, msg)
	case "solve":
		return ap.handleSolve(playerID, msg)
	case "propose_end":
		return ap.handleProposeEnd(playerID)
	case "end_vote":
		return ap.handleEndVote(playerID, msg)
	case "request_look":
		return ap.handleLook(playerID)
	case "request_inventory":
		return ap.handleInventory(playerID)
	case "request_role":
		return ap.handleRole(playerID)
	case "request_map":
		return ap.handleMap(playerID)
	case "request_who":
		return ap.handleWho(playerID)
	case "request_help":
		return ap.handleHelp(playerID)
	case "submit_feedback":
		return ap.handleSubmitFeedback(playerID, msg)
	case "skip_feedback":
		return ap.handleSkipFeedback(playerID)
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

// handleChat publishes a room-scoped chat message.
func (ap *ActionProcessor) handleChat(playerID string, msg protocol.ClientMessage) error {
	player := ap.gameState.GetPlayer(playerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", playerID)
	}

	ap.eventBus.PublishChat(eventbus.ChatData{
		SenderID:   playerID,
		SenderName: player.Nickname,
		RoomID:     player.CurrentRoomID,
		Content:    msg.Content,
		Scope:      "room",
	})

	return nil
}

// handleShout publishes a global chat message.
func (ap *ActionProcessor) handleShout(playerID string, msg protocol.ClientMessage) error {
	player := ap.gameState.GetPlayer(playerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", playerID)
	}

	ap.eventBus.PublishChat(eventbus.ChatData{
		SenderID:   playerID,
		SenderName: player.Nickname,
		RoomID:     player.CurrentRoomID,
		Content:    msg.Content,
		Scope:      "global",
	})

	return nil
}

// handleMove validates adjacency, moves the player, and publishes events.
func (ap *ActionProcessor) handleMove(playerID string, msg protocol.ClientMessage) error {
	player := ap.gameState.GetPlayer(playerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", playerID)
	}

	// Resolve target room
	targetRoom := ap.mapEngine.GetRoomByName(msg.TargetRoomID)
	if targetRoom == nil {
		// Try by ID
		targetRoom = ap.mapEngine.GetRoomByID(msg.TargetRoomID)
	}
	if targetRoom == nil {
		ap.sendError(playerID, protocol.ErrorCodeInvalidMove, "Room not found")
		return nil
	}

	// Check adjacency
	if !ap.mapEngine.IsAdjacent(player.CurrentRoomID, targetRoom.ID) {
		ap.sendError(playerID, protocol.ErrorCodeInvalidMove, "That room is not adjacent to your current location")
		return nil
	}

	// Check same room
	if player.CurrentRoomID == targetRoom.ID {
		ap.sendError(playerID, protocol.ErrorCodeInvalidMove, "You are already in that room")
		return nil
	}

	fromRoom := ap.mapEngine.GetRoomByID(player.CurrentRoomID)
	fromRoomName := ""
	if fromRoom != nil {
		fromRoomName = fromRoom.Name
	}

	// Move the player
	ap.gameState.MovePlayer(playerID, targetRoom.ID)

	// Publish player_move event (all)
	ap.eventBus.PublishGameEvent(events.PlayerMoveEvent{
		BaseEvent: newBaseEvent("all", "", nil),
		Type:      "player_move",
		Data: events.PlayerMoveData{
			PlayerID:   playerID,
			PlayerName: player.Nickname,
			From:       fromRoomName,
			To:         targetRoom.Name,
		},
	})

	// Send room_changed to the moving player
	roomView := ap.gameState.GetRoomView(playerID)
	ap.network.SendTo(playerID, protocol.RoomChangedMessage{
		Type: protocol.SMsgTypeRoomChanged,
		Room: roomView,
	})

	// System message: player left the origin room
	if fromRoom != nil {
		ap.eventBus.PublishGameEvent(events.NarrationEvent{
			BaseEvent: newBaseEvent("room", fromRoom.ID, nil),
			Type:      "narration",
			Data: events.NarrationData{
				Text: fmt.Sprintf("%s left the room.", player.Nickname),
				Mood: "neutral",
			},
		})
	}

	// System message: player entered the destination room
	ap.eventBus.PublishGameEvent(events.NarrationEvent{
		BaseEvent: newBaseEvent("room", targetRoom.ID, nil),
		Type:      "narration",
		Data: events.NarrationData{
			Text: fmt.Sprintf("%s entered the room.", player.Nickname),
			Mood: "neutral",
		},
	})

	// Broadcast map update to all
	ap.messageRouter.BroadcastMapUpdate()

	return nil
}

// handleExamine calls AI to evaluate an examine action.
func (ap *ActionProcessor) handleExamine(playerID string, msg protocol.ClientMessage) error {
	player := ap.gameState.GetPlayer(playerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", playerID)
	}

	target := ""
	if msg.Target != nil {
		target = *msg.Target
	}

	// Send thinking indicator
	ap.network.SendTo(playerID, protocol.ThinkingMessage{Type: protocol.SMsgTypeThinking})

	ctx, cancel := context.WithTimeout(context.Background(), aiTimeout)
	defer cancel()

	gameCtx := ap.buildGameContext(playerID)
	resp, err := ap.aiLayer.EvaluateExamine(ctx, gameCtx, target)
	if err != nil {
		slog.Warn("AI examine timeout/error", "error", err)
		ap.sendError(playerID, "AI_ERROR", "Response timed out. Please try again.")
		return nil
	}

	ap.applyAIResponse(playerID, resp)
	return nil
}

// handleDo calls AI to evaluate a free-form action.
func (ap *ActionProcessor) handleDo(playerID string, msg protocol.ClientMessage) error {
	player := ap.gameState.GetPlayer(playerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", playerID)
	}

	// Send thinking indicator
	ap.network.SendTo(playerID, protocol.ThinkingMessage{Type: protocol.SMsgTypeThinking})

	ctx, cancel := context.WithTimeout(context.Background(), aiTimeout)
	defer cancel()

	gameCtx := ap.buildGameContext(playerID)
	resp, err := ap.aiLayer.EvaluateAction(ctx, gameCtx, msg.Action)
	if err != nil {
		slog.Warn("AI action timeout/error", "error", err)
		ap.sendError(playerID, "AI_ERROR", "Response timed out. Please try again.")
		return nil
	}

	ap.applyAIResponse(playerID, resp)
	return nil
}

// handleTalk calls AI for NPC conversation.
func (ap *ActionProcessor) handleTalk(playerID string, msg protocol.ClientMessage) error {
	player := ap.gameState.GetPlayer(playerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", playerID)
	}

	// Verify NPC is in the same room
	npcsInRoom := ap.gameState.GetNPCsInRoom(player.CurrentRoomID)
	var targetNPC *types.NPC
	for i := range npcsInRoom {
		if npcsInRoom[i].ID == msg.NPCID {
			targetNPC = &npcsInRoom[i]
			break
		}
	}
	if targetNPC == nil {
		ap.sendError(playerID, protocol.ErrorCodeNPCNotInRoom, "That NPC is not in this room")
		return nil
	}

	// Send thinking indicator
	ap.network.SendTo(playerID, protocol.ThinkingMessage{Type: protocol.SMsgTypeThinking})

	ctx, cancel := context.WithTimeout(context.Background(), aiTimeout)
	defer cancel()

	gameCtx := ap.buildGameContext(playerID)
	npcResp, err := ap.aiLayer.TalkToNPC(ctx, gameCtx, msg.NPCID, msg.Message)
	if err != nil {
		slog.Warn("AI NPC talk timeout/error", "error", err)
		ap.sendError(playerID, "AI_ERROR", "Response timed out. Please try again.")
		return nil
	}

	// Apply trust change
	if npcResp.TrustChange != 0 {
		ap.gameState.UpdateNPCTrust(msg.NPCID, playerID, npcResp.TrustChange)
	}

	// Record conversation
	ap.gameState.AddConversation(msg.NPCID, types.ConversationRecord{
		PlayerID:  playerID,
		Message:   msg.Message,
		Response:  npcResp.Dialogue,
		Timestamp: time.Now().UnixMilli(),
	})

	// Publish NPC dialogue event (room scope)
	ap.eventBus.PublishGameEvent(events.NPCDialogueEvent{
		BaseEvent: newBaseEvent("room", player.CurrentRoomID, nil),
		Type:      "npc_dialogue",
		Data: events.NPCDialogueData{
			NPCID:      msg.NPCID,
			NPCName:    targetNPC.Name,
			PlayerID:   playerID,
			PlayerName: player.Nickname,
			Text:       npcResp.Dialogue,
			Emotion:    npcResp.Emotion,
		},
	})

	// Process any additional events from NPC response
	if len(npcResp.Events) > 0 {
		parsedEvents, err := npcResp.ParsedEvents()
		if err != nil {
			slog.Warn("failed to parse NPC response events", "error", err)
		} else {
			ap.publishAIEvents(playerID, parsedEvents)
		}
	}

	// Handle gimmick trigger
	if npcResp.TriggeredGimmick && targetNPC.Gimmick != nil {
		ap.gameState.TriggerGimmick(targetNPC.ID)
	}

	return nil
}

// handleGive is not yet implemented (P1).
func (ap *ActionProcessor) handleGive(playerID string) error {
	ap.sendError(playerID, protocol.ErrorCodeNotSupported, "This feature is not yet available")
	return nil
}

// handleVote delegates vote to EndConditionEngine.
func (ap *ActionProcessor) handleVote(playerID string, msg protocol.ClientMessage) error {
	return ap.endCondition.CastVote(playerID, msg.TargetID)
}

// handleSolve delegates solution submission to EndConditionEngine.
func (ap *ActionProcessor) handleSolve(playerID string, msg protocol.ClientMessage) error {
	return ap.endCondition.SubmitSolution(playerID, msg.Answer)
}

// handleProposeEnd delegates end proposal to EndConditionEngine.
func (ap *ActionProcessor) handleProposeEnd(playerID string) error {
	return ap.endCondition.ProposeEnd(playerID)
}

// handleEndVote delegates end vote response to EndConditionEngine.
func (ap *ActionProcessor) handleEndVote(playerID string, msg protocol.ClientMessage) error {
	return ap.endCondition.RespondToEndProposal(playerID, msg.Agree)
}

// handleLook re-sends the current room view to the player.
func (ap *ActionProcessor) handleLook(playerID string) error {
	roomView := ap.gameState.GetRoomView(playerID)
	ap.network.SendTo(playerID, protocol.RoomChangedMessage{
		Type: protocol.SMsgTypeRoomChanged,
		Room: roomView,
	})
	return nil
}

// handleInventory sends inventory data to the player.
func (ap *ActionProcessor) handleInventory(playerID string) error {
	player := ap.gameState.GetPlayer(playerID)
	if player == nil {
		return fmt.Errorf("player not found: %s", playerID)
	}

	// Get discovered clues
	world := ap.gameState.GetWorld()
	var clues []types.Clue
	for _, clueID := range player.DiscoveredClueIDs {
		for _, c := range world.Clues {
			if c.ID == clueID {
				clues = append(clues, c)
				break
			}
		}
	}

	ap.network.SendTo(playerID, protocol.InventoryMessage{
		Type:  protocol.SMsgTypeInventory,
		Items: player.Inventory,
		Clues: clues,
	})
	return nil
}

// handleRole sends role information to the player.
func (ap *ActionProcessor) handleRole(playerID string) error {
	player := ap.gameState.GetPlayer(playerID)
	if player == nil || player.Role == nil {
		return fmt.Errorf("player or role not found: %s", playerID)
	}

	ap.network.SendTo(playerID, protocol.RoleInfoMessage{
		Type: protocol.SMsgTypeRoleInfo,
		Role: *player.Role,
	})
	return nil
}

// handleMap sends map information to the player.
func (ap *ActionProcessor) handleMap(playerID string) error {
	mapView := ap.gameState.GetMapView(playerID)
	ap.network.SendTo(playerID, protocol.MapInfoMessage{
		Type: protocol.SMsgTypeMapInfo,
		Map:  mapView,
	})
	return nil
}

// handleWho sends player location information.
func (ap *ActionProcessor) handleWho(playerID string) error {
	allIDs := ap.gameState.GetAllPlayerIDs()
	var players []protocol.PlayerLocationInfo

	for _, id := range allIDs {
		p := ap.gameState.GetPlayer(id)
		if p == nil {
			continue
		}
		room := ap.gameState.GetPlayerRoom(id)
		roomID := ""
		roomName := ""
		if room != nil {
			roomID = room.ID
			roomName = room.Name
		}
		players = append(players, protocol.PlayerLocationInfo{
			ID:       p.ID,
			Nickname: p.Nickname,
			RoomID:   roomID,
			RoomName: roomName,
			Status:   p.Status,
		})
	}

	ap.network.SendTo(playerID, protocol.WhoInfoMessage{
		Type:    protocol.SMsgTypeWhoInfo,
		Players: players,
	})
	return nil
}

// handleHelp sends available commands to the player.
func (ap *ActionProcessor) handleHelp(playerID string) error {
	commands := []protocol.CommandInfo{
		{Command: "/chat", Description: "Send a message to players in the same room", Usage: "/chat <message>"},
		{Command: "/shout", Description: "Send a message to all players", Usage: "/shout <message>"},
		{Command: "/move", Description: "Move to an adjacent room", Usage: "/move <room name>"},
		{Command: "/examine", Description: "Examine something in the room", Usage: "/examine [target]"},
		{Command: "/do", Description: "Perform a free-form action", Usage: "/do <action>"},
		{Command: "/talk", Description: "Talk to an NPC", Usage: "/talk <npc> <message>"},
		{Command: "/inventory", Description: "View your inventory", Usage: "/inventory"},
		{Command: "/role", Description: "View your role information", Usage: "/role"},
		{Command: "/map", Description: "View the map", Usage: "/map"},
		{Command: "/who", Description: "See where all players are", Usage: "/who"},
		{Command: "/vote", Description: "Cast a vote", Usage: "/vote <target>"},
		{Command: "/solve", Description: "Submit a solution", Usage: "/solve <answer>"},
		{Command: "/end", Description: "Propose ending the game", Usage: "/end"},
	}

	ap.network.SendTo(playerID, protocol.HelpInfoMessage{
		Type:     protocol.SMsgTypeHelp,
		Commands: commands,
	})
	return nil
}

// handleSubmitFeedback saves player feedback.
func (ap *ActionProcessor) handleSubmitFeedback(playerID string, msg protocol.ClientMessage) error {
	feedback := types.Feedback{
		PlayerID:        playerID,
		FunRating:       msg.FunRating,
		ImmersionRating: msg.ImmersionRating,
		Comment:         msg.Comment,
		SubmittedAt:     time.Now().UnixMilli(),
	}
	ap.eventBus.PublishFeedback(feedback)
	return nil
}

// handleSkipFeedback acknowledges feedback skip.
func (ap *ActionProcessor) handleSkipFeedback(playerID string) error {
	ap.network.SendTo(playerID, protocol.FeedbackAckMessage{
		Type: protocol.SMsgTypeFeedbackAck,
	})
	return nil
}

// buildGameContext constructs the GameContext for a player's AI request.
func (ap *ActionProcessor) buildGameContext(playerID string) types.GameContext {
	player := ap.gameState.GetPlayer(playerID)
	currentRoom := ap.gameState.GetPlayerRoom(playerID)

	var playerValue types.Player
	if player != nil {
		playerValue = *player
	}

	var roomValue types.Room
	if currentRoom != nil {
		roomValue = *currentRoom
	}

	playersInRoom := ap.gameState.GetPlayersInRoom(roomValue.ID)
	playersInRoomValues := make([]types.Player, len(playersInRoom))
	for i, p := range playersInRoom {
		playersInRoomValues[i] = *p
	}

	return types.GameContext{
		World:            ap.gameState.GetWorld(),
		CurrentState:     ap.gameState.GetFullState(),
		RecentEvents:     toInterfaceSlice(ap.gameState.GetRecentEvents(20)),
		RequestingPlayer: playerValue,
		CurrentRoom:      roomValue,
		PlayersInRoom:    playersInRoomValues,
	}
}

// applyAIResponse applies state changes and publishes events from an AI GameResponse.
func (ap *ActionProcessor) applyAIResponse(playerID string, resp *schemas.GameResponse) {
	// Apply state changes
	ap.applyStateChanges(playerID, resp.StateChanges)

	// Parse and publish events
	parsedEvents, err := resp.ParsedEvents()
	if err != nil {
		slog.Warn("failed to parse AI response events", "error", err)
		return
	}

	ap.publishAIEvents(playerID, parsedEvents)
}

// applyStateChanges applies AI-requested state mutations.
func (ap *ActionProcessor) applyStateChanges(playerID string, changes []json.RawMessage) {
	for _, raw := range changes {
		var base struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &base); err != nil {
			slog.Warn("failed to parse state change type", "error", err)
			continue
		}

		switch base.Type {
		case "discover_clue":
			var sc schemas.StateChangeDiscoverClue
			if err := json.Unmarshal(raw, &sc); err == nil {
				ap.gameState.DiscoverClue(sc.PlayerID, sc.ClueID)
			}
		case "add_item":
			var sc schemas.StateChangeAddItem
			if err := json.Unmarshal(raw, &sc); err == nil {
				ap.gameState.AddItemToPlayer(sc.PlayerID, types.Item{
					ID:          sc.Item.ID,
					Name:        sc.Item.Name,
					Description: sc.Item.Description,
					OwnerID:     sc.Item.OwnerID,
					IsKey:       sc.Item.IsKey,
				})
			}
		case "remove_item":
			var sc schemas.StateChangeRemoveItem
			if err := json.Unmarshal(raw, &sc); err == nil {
				ap.gameState.RemoveItemFromPlayer(sc.PlayerID, sc.ItemID)
			}
		case "trigger_gimmick":
			var sc schemas.StateChangeTriggerGimmick
			if err := json.Unmarshal(raw, &sc); err == nil {
				ap.gameState.TriggerGimmick(sc.GimmickID)
			}
		case "update_npc_trust":
			var sc schemas.StateChangeUpdateNPCTrust
			if err := json.Unmarshal(raw, &sc); err == nil {
				ap.gameState.UpdateNPCTrust(sc.NPCID, playerID, sc.Delta)
			}
		default:
			slog.Warn("unknown state change type", "type", base.Type)
		}
	}
}

// publishAIEvents converts AI events to game events and publishes them.
func (ap *ActionProcessor) publishAIEvents(playerID string, aiEvents []schemas.AIGameEvent) {
	player := ap.gameState.GetPlayer(playerID)
	if player == nil {
		return
	}

	for _, aiEvent := range aiEvents {
		switch e := aiEvent.(type) {
		case schemas.AINarrationEvent:
			ap.eventBus.PublishGameEvent(events.NarrationEvent{
				BaseEvent: newBaseEvent("room", player.CurrentRoomID, nil),
				Type:      "narration",
				Data: events.NarrationData{
					Text: e.Data.Text,
					Mood: e.Data.Mood,
				},
			})
		case schemas.AIExamineResultEvent:
			ap.eventBus.PublishGameEvent(events.ExamineResultEvent{
				BaseEvent: newBaseEvent("room", player.CurrentRoomID, nil),
				Type:      "examine_result",
				Data: events.ExamineResultData{
					PlayerID:    playerID,
					PlayerName:  player.Nickname,
					Target:      e.Data.Target,
					Description: e.Data.Description,
					ClueFound:   e.Data.ClueFound,
				},
			})
		case schemas.AIActionResultEvent:
			ap.eventBus.PublishGameEvent(events.ActionResultEvent{
				BaseEvent: newBaseEvent("room", player.CurrentRoomID, nil),
				Type:      "action_result",
				Data: events.ActionResultData{
					PlayerID:        playerID,
					PlayerName:      player.Nickname,
					Action:          e.Data.Action,
					Result:          e.Data.Result,
					TriggeredEvents: e.Data.TriggeredEvents,
				},
			})
		case schemas.AIClueFoundEvent:
			ap.eventBus.PublishGameEvent(events.ClueFoundEvent{
				BaseEvent: newBaseEvent("room", player.CurrentRoomID, nil),
				Type:      "clue_found",
				Data: events.ClueFoundData{
					PlayerID:   playerID,
					PlayerName: player.Nickname,
					Clue: types.Clue{
						ID:          e.Data.Clue.ID,
						Name:        e.Data.Clue.Name,
						Description: e.Data.Clue.Description,
					},
					Location: e.Data.Location,
				},
			})
		case schemas.AIStoryEvent:
			ap.eventBus.PublishGameEvent(events.StoryEventEvent{
				BaseEvent: newBaseEvent("room", player.CurrentRoomID, nil),
				Type:      "story_event",
				Data: events.StoryEventData{
					Title:        e.Data.Title,
					Description:  e.Data.Description,
					Consequences: e.Data.Consequences,
				},
			})
		case schemas.AINPCDialogueEvent:
			ap.eventBus.PublishGameEvent(events.NPCDialogueEvent{
				BaseEvent: newBaseEvent("room", player.CurrentRoomID, nil),
				Type:      "npc_dialogue",
				Data: events.NPCDialogueData{
					NPCID:      e.Data.NPCID,
					NPCName:    e.Data.NPCName,
					PlayerID:   playerID,
					PlayerName: player.Nickname,
					Text:       e.Data.Text,
					Emotion:    e.Data.Emotion,
				},
			})
		default:
			slog.Warn("unhandled AI event type", "type", aiEvent.AIEventType())
		}
	}
}

// sendError sends an error message to a player.
func (ap *ActionProcessor) sendError(playerID string, code protocol.ErrorCode, message string) {
	ap.network.SendTo(playerID, protocol.ErrorMessage{
		Type:    protocol.SMsgTypeError,
		Code:    code,
		Message: message,
	})
}

// newBaseEvent creates a BaseEvent with visibility.
func newBaseEvent(scope string, roomID string, playerIDs []string) types.BaseEvent {
	return types.BaseEvent{
		ID:        time.Now().Format("20060102150405.000000000"),
		Timestamp: time.Now().UnixMilli(),
		Visibility: types.EventVisibility{
			Scope:     scope,
			RoomID:    roomID,
			PlayerIDs: playerIDs,
		},
	}
}

// toInterfaceSlice converts a slice of GameEvent to []interface{}.
func toInterfaceSlice(events []types.GameEvent) []interface{} {
	result := make([]interface{}, len(events))
	for i, e := range events {
		result[i] = e
	}
	return result
}
