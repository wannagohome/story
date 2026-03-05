package protocol

import (
	"encoding/json"
	"testing"

	"github.com/anthropics/story/internal/shared/types"
)

// --- Client Messages ---

func TestClientMessage_JoinParsing(t *testing.T) {
	raw := `{"type":"join","nickname":"Alice"}`
	var msg ClientMessage
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg.Type != "join" {
		t.Errorf("Type: got %q", msg.Type)
	}
	if msg.Nickname != "Alice" {
		t.Errorf("Nickname: got %q", msg.Nickname)
	}
}

func TestClientMessage_ChatParsing(t *testing.T) {
	raw := `{"type":"chat","content":"Hello everyone"}`
	var msg ClientMessage
	json.Unmarshal([]byte(raw), &msg)
	if msg.Type != "chat" || msg.Content != "Hello everyone" {
		t.Errorf("unexpected: %+v", msg)
	}
}

func TestClientMessage_MoveParsing(t *testing.T) {
	raw := `{"type":"move","targetRoomId":"Bridge"}`
	var msg ClientMessage
	json.Unmarshal([]byte(raw), &msg)
	if msg.TargetRoomID != "Bridge" {
		t.Errorf("TargetRoomID: got %q", msg.TargetRoomID)
	}
}

func TestClientMessage_ExamineWithTarget(t *testing.T) {
	target := "desk"
	raw := `{"type":"examine","target":"desk"}`
	var msg ClientMessage
	json.Unmarshal([]byte(raw), &msg)
	if msg.Target == nil || *msg.Target != target {
		t.Errorf("Target: got %v", msg.Target)
	}
}

func TestClientMessage_ExamineWithoutTarget(t *testing.T) {
	raw := `{"type":"examine"}`
	var msg ClientMessage
	json.Unmarshal([]byte(raw), &msg)
	if msg.Target != nil {
		t.Errorf("expected nil target, got %v", msg.Target)
	}
}

func TestClientMessage_ReadyPhases(t *testing.T) {
	raw := `{"type":"ready","phase":"briefing_read"}`
	var msg ClientMessage
	json.Unmarshal([]byte(raw), &msg)
	if msg.Phase != "briefing_read" {
		t.Errorf("Phase: got %q", msg.Phase)
	}
}

func TestClientMessage_VoteParsing(t *testing.T) {
	raw := `{"type":"vote","targetId":"p2"}`
	var msg ClientMessage
	json.Unmarshal([]byte(raw), &msg)
	if msg.TargetID != "p2" {
		t.Errorf("TargetID: got %q", msg.TargetID)
	}
}

func TestClientMessage_EndVoteParsing(t *testing.T) {
	raw := `{"type":"end_vote","agree":true}`
	var msg ClientMessage
	json.Unmarshal([]byte(raw), &msg)
	if !msg.Agree {
		t.Errorf("expected agree=true")
	}
}

func TestClientMessage_FeedbackParsing(t *testing.T) {
	comment := "Great!"
	raw := `{"type":"submit_feedback","funRating":5,"immersionRating":4,"comment":"Great!"}`
	var msg ClientMessage
	json.Unmarshal([]byte(raw), &msg)
	if msg.FunRating != 5 {
		t.Errorf("FunRating: got %d", msg.FunRating)
	}
	if msg.Comment == nil || *msg.Comment != comment {
		t.Errorf("Comment mismatch")
	}
}

func TestClientMessage_JSONRoundTrip(t *testing.T) {
	msg := ClientMessage{
		Type:    "do",
		Action:  "pull lever",
	}
	data, _ := json.Marshal(msg)
	var got ClientMessage
	json.Unmarshal(data, &got)
	if got.Action != "pull lever" {
		t.Errorf("Action: got %q", got.Action)
	}
}

func TestJoinMessage_Standalone(t *testing.T) {
	msg := JoinMessage{Type: "join", Nickname: "Bob"}
	data, _ := json.Marshal(msg)
	var got JoinMessage
	json.Unmarshal(data, &got)
	if got.Nickname != "Bob" {
		t.Errorf("Nickname: got %q", got.Nickname)
	}
}

func TestStartGameMessage_WithTheme(t *testing.T) {
	msg := StartGameMessage{Type: "start_game", ThemeKeyword: "mystery"}
	data, _ := json.Marshal(msg)
	var got StartGameMessage
	json.Unmarshal(data, &got)
	if got.ThemeKeyword != "mystery" {
		t.Errorf("ThemeKeyword: got %q", got.ThemeKeyword)
	}
}

func TestStartGameMessage_WithoutTheme(t *testing.T) {
	msg := StartGameMessage{Type: "start_game"}
	data, _ := json.Marshal(msg)
	s := string(data)
	// themeKeyword should be omitted when empty
	var m map[string]interface{}
	json.Unmarshal([]byte(s), &m)
	if _, ok := m["themeKeyword"]; ok {
		t.Errorf("expected themeKeyword to be omitted")
	}
}

// --- Server Messages ---

func TestJoinedMessage_JSONRoundTrip(t *testing.T) {
	msg := JoinedMessage{
		Type:     "joined",
		PlayerID: "p1",
		RoomCode: "ABCD",
		IsHost:   true,
	}
	data, _ := json.Marshal(msg)
	var got JoinedMessage
	json.Unmarshal(data, &got)
	if got.PlayerID != "p1" || !got.IsHost {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestLobbyUpdateMessage_JSONRoundTrip(t *testing.T) {
	msg := LobbyUpdateMessage{
		Type: "lobby_update",
		Players: []LobbyPlayer{
			{ID: "p1", Nickname: "Alice", IsHost: true},
			{ID: "p2", Nickname: "Bob", IsHost: false},
		},
		MaxPlayers: 8,
	}
	data, _ := json.Marshal(msg)
	var got LobbyUpdateMessage
	json.Unmarshal(data, &got)
	if len(got.Players) != 2 {
		t.Errorf("Players length: got %d", len(got.Players))
	}
	if got.MaxPlayers != 8 {
		t.Errorf("MaxPlayers: got %d", got.MaxPlayers)
	}
}

func TestGenerationProgressMessage_JSONRoundTrip(t *testing.T) {
	msg := GenerationProgressMessage{
		Type:     "generation_progress",
		Step:     "world",
		Message:  "Generating world...",
		Progress: 0.5,
	}
	data, _ := json.Marshal(msg)
	var got GenerationProgressMessage
	json.Unmarshal(data, &got)
	if got.Progress != 0.5 {
		t.Errorf("Progress: got %f", got.Progress)
	}
}

func TestErrorMessage_JSONRoundTrip(t *testing.T) {
	msg := ErrorMessage{
		Type:    "error",
		Code:    ErrorCodeInvalidMove,
		Message: "Cannot move there",
	}
	data, _ := json.Marshal(msg)
	var got ErrorMessage
	json.Unmarshal(data, &got)
	if got.Code != ErrorCodeInvalidMove {
		t.Errorf("Code: got %q", got.Code)
	}
}

func TestErrorCode_Constants(t *testing.T) {
	codes := []ErrorCode{
		ErrorCodeInvalidRoomCode,
		ErrorCodeGameAlreadyStarted,
		ErrorCodeRoomFull,
		ErrorCodeDuplicateNickname,
		ErrorCodeNotHost,
		ErrorCodeNotEnoughPlayers,
		ErrorCodeInvalidMove,
		ErrorCodeNPCNotInRoom,
		ErrorCodeItemNotFound,
		ErrorCodeUnknownCommand,
		ErrorCodeNotSupported,
		ErrorCodeVoteNotActive,
		ErrorCodeVotingDisabled,
		ErrorCodeEndVoteAlreadyOpen,
		ErrorCodeEmptyMessage,
		ErrorCodeMessageTooLong,
		ErrorCodeInvalidNickname,
		ErrorCodeInvalidAPIKey,
		ErrorCodeSaveFailed,
		ErrorCodeConnectionLost,
		ErrorCodeWSConnectionFailed,
		ErrorCodeBriefingNotComplete,
	}
	// Just verify they all have non-empty string values
	for _, c := range codes {
		if c == "" {
			t.Errorf("empty error code found")
		}
	}
}

func TestBriefingPublicMessage_JSONRoundTrip(t *testing.T) {
	msg := BriefingPublicMessage{
		Type: "briefing_public",
		Info: types.PublicInfo{
			Title:    "Mystery Manor",
			Synopsis: "A dark tale",
		},
	}
	data, _ := json.Marshal(msg)
	var got BriefingPublicMessage
	json.Unmarshal(data, &got)
	if got.Info.Title != "Mystery Manor" {
		t.Errorf("Title: got %q", got.Info.Title)
	}
}

func TestBriefingPrivateMessage_JSONRoundTrip(t *testing.T) {
	msg := BriefingPrivateMessage{
		Type: "briefing_private",
		Role: types.PlayerRole{
			ID:            "role-1",
			CharacterName: "Detective",
			Secret:        "hidden secret",
		},
		Secrets: []string{"secret1"},
		SemiPublicInfo: []types.SemiPublicInfo{
			{ID: "sp1", TargetPlayerIDs: []string{"p1"}, Content: "shared info"},
		},
	}
	data, _ := json.Marshal(msg)
	var got BriefingPrivateMessage
	json.Unmarshal(data, &got)
	if got.Role.CharacterName != "Detective" {
		t.Errorf("CharacterName: got %q", got.Role.CharacterName)
	}
}

func TestGameStartedMessage_JSONRoundTrip(t *testing.T) {
	msg := GameStartedMessage{
		Type: "game_started",
		InitialRoom: RoomView{
			ID:   "r1",
			Name: "Lobby",
			Players: []RoomViewPlayer{
				{ID: "p1", Nickname: "Alice"},
			},
		},
	}
	data, _ := json.Marshal(msg)
	var got GameStartedMessage
	json.Unmarshal(data, &got)
	if got.InitialRoom.Name != "Lobby" {
		t.Errorf("InitialRoom.Name: got %q", got.InitialRoom.Name)
	}
}

func TestChatServerMessage_JSONRoundTrip(t *testing.T) {
	location := "Bridge"
	msg := ChatServerMessage{
		Type:           "chat_message",
		SenderID:       "p1",
		SenderName:     "Alice",
		Content:        "Hello",
		Scope:          "global",
		SenderLocation: &location,
		Timestamp:      12345,
	}
	data, _ := json.Marshal(msg)
	var got ChatServerMessage
	json.Unmarshal(data, &got)
	if got.SenderLocation == nil || *got.SenderLocation != "Bridge" {
		t.Errorf("SenderLocation mismatch")
	}
}

func TestChatServerMessage_RoomScope_NoLocation(t *testing.T) {
	msg := ChatServerMessage{
		Type:      "chat_message",
		SenderID:  "p1",
		Scope:     "room",
		Timestamp: 100,
	}
	data, _ := json.Marshal(msg)
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	if _, ok := m["senderLocation"]; ok {
		t.Errorf("expected senderLocation to be omitted for room scope")
	}
}

func TestGameEventMessage_JSONRoundTrip(t *testing.T) {
	eventData := json.RawMessage(`{"type":"narration","data":{"text":"Dark clouds gather","mood":"tense"}}`)
	msg := GameEventMessage{
		Type:  "game_event",
		Event: eventData,
	}
	data, _ := json.Marshal(msg)
	var got GameEventMessage
	json.Unmarshal(data, &got)
	if got.Event == nil {
		t.Errorf("expected non-nil event")
	}
}

func TestRoomChangedMessage_JSONRoundTrip(t *testing.T) {
	msg := RoomChangedMessage{
		Type: "room_changed",
		Room: RoomView{
			ID:          "r2",
			Name:        "Bridge",
			Description: "The command center",
			Type:        "public",
			Items:       []RoomViewItem{{ID: "i1", Name: "Console"}},
			Players:     []RoomViewPlayer{{ID: "p1", Nickname: "Alice"}},
			NPCs:        []RoomViewNPC{{ID: "npc1", Name: "Captain"}},
		},
	}
	data, _ := json.Marshal(msg)
	var got RoomChangedMessage
	json.Unmarshal(data, &got)
	if got.Room.Name != "Bridge" {
		t.Errorf("Room.Name: got %q", got.Room.Name)
	}
	if len(got.Room.NPCs) != 1 {
		t.Errorf("NPCs length: got %d", len(got.Room.NPCs))
	}
}

func TestMapInfoMessage_JSONRoundTrip(t *testing.T) {
	msg := MapInfoMessage{
		Type: "map_info",
		Map: MapView{
			Rooms: []MapViewRoom{
				{ID: "r1", Name: "Lobby", Type: "public", PlayerCount: 2, PlayerNames: []string{"Alice", "Bob"}},
			},
			Connections: []types.Connection{
				{RoomA: "r1", RoomB: "r2", Bidirectional: true},
			},
			MyRoomID: "r1",
		},
	}
	data, _ := json.Marshal(msg)
	var got MapInfoMessage
	json.Unmarshal(data, &got)
	if got.Map.MyRoomID != "r1" {
		t.Errorf("MyRoomID: got %q", got.Map.MyRoomID)
	}
	if len(got.Map.Rooms) != 1 {
		t.Errorf("Rooms length: got %d", len(got.Map.Rooms))
	}
}

func TestWhoInfoMessage_JSONRoundTrip(t *testing.T) {
	msg := WhoInfoMessage{
		Type: "who_info",
		Players: []PlayerLocationInfo{
			{ID: "p1", Nickname: "Alice", RoomID: "r1", RoomName: "Lobby", Status: "connected"},
		},
	}
	data, _ := json.Marshal(msg)
	var got WhoInfoMessage
	json.Unmarshal(data, &got)
	if got.Players[0].Status != "connected" {
		t.Errorf("Status: got %q", got.Players[0].Status)
	}
}

func TestHelpInfoMessage_JSONRoundTrip(t *testing.T) {
	msg := HelpInfoMessage{
		Type: "help_info",
		Commands: []CommandInfo{
			{Command: "/look", Description: "Look around", Usage: "/look"},
		},
	}
	data, _ := json.Marshal(msg)
	var got HelpInfoMessage
	json.Unmarshal(data, &got)
	if got.Commands[0].Command != "/look" {
		t.Errorf("Command: got %q", got.Commands[0].Command)
	}
}

func TestVoteStartedMessage_JSONRoundTrip(t *testing.T) {
	msg := VoteStartedMessage{
		Type:           "vote_started",
		Reason:         "Identify the traitor",
		Candidates:     []string{"p1", "p2", "p3"},
		TimeoutSeconds: 60,
	}
	data, _ := json.Marshal(msg)
	var got VoteStartedMessage
	json.Unmarshal(data, &got)
	if len(got.Candidates) != 3 {
		t.Errorf("Candidates length: got %d", len(got.Candidates))
	}
}

func TestVoteEndedMessage_JSONRoundTrip(t *testing.T) {
	msg := VoteEndedMessage{
		Type: "vote_ended",
		Results: []VoteResultEntry{
			{CandidateID: "p1", CandidateName: "Alice", Votes: 3},
		},
		Outcome: "Alice is the traitor",
	}
	data, _ := json.Marshal(msg)
	var got VoteEndedMessage
	json.Unmarshal(data, &got)
	if got.Results[0].Votes != 3 {
		t.Errorf("Votes: got %d", got.Results[0].Votes)
	}
}

func TestSolveResultMessage_JSONRoundTrip(t *testing.T) {
	msg := SolveResultMessage{
		Type: "solve_result",
		Answers: []SolveAnswerEntry{
			{PlayerID: "p1", PlayerName: "Alice", Answer: "The butler did it"},
		},
		Outcome: "Partially correct",
	}
	data, _ := json.Marshal(msg)
	var got SolveResultMessage
	json.Unmarshal(data, &got)
	if got.Answers[0].Answer != "The butler did it" {
		t.Errorf("Answer: got %q", got.Answers[0].Answer)
	}
}

func TestGameEndingMessage_JSONRoundTrip(t *testing.T) {
	msg := GameEndingMessage{
		Type:         "game_ending",
		CommonResult: "The mystery was solved",
		PersonalEnding: types.PlayerEnding{
			PlayerID:  "p1",
			Summary:   "Heroic detective",
			Narrative: "You saved the day",
		},
		SecretReveal: types.SecretReveal{
			PlayerSecrets: []types.PlayerSecretEntry{
				{PlayerID: "p2", CharacterName: "Villain", Secret: "Was the mastermind"},
			},
		},
	}
	data, _ := json.Marshal(msg)
	var got GameEndingMessage
	json.Unmarshal(data, &got)
	if got.CommonResult != "The mystery was solved" {
		t.Errorf("CommonResult: got %q", got.CommonResult)
	}
	if got.PersonalEnding.PlayerID != "p1" {
		t.Errorf("PersonalEnding.PlayerID: got %q", got.PersonalEnding.PlayerID)
	}
}

func TestInventoryMessage_JSONRoundTrip(t *testing.T) {
	msg := InventoryMessage{
		Type:  "inventory",
		Items: []types.Item{{ID: "i1", Name: "Flashlight"}},
		Clues: []types.Clue{{ID: "c1", Name: "Letter"}},
	}
	data, _ := json.Marshal(msg)
	var got InventoryMessage
	json.Unmarshal(data, &got)
	if len(got.Items) != 1 || len(got.Clues) != 1 {
		t.Errorf("unexpected inventory: items=%d, clues=%d", len(got.Items), len(got.Clues))
	}
}

func TestRawServerMessage_TypeParsing(t *testing.T) {
	raw := `{"type":"joined","playerId":"p1","roomCode":"ABCD","isHost":true}`
	var msg RawServerMessage
	json.Unmarshal([]byte(raw), &msg)
	if msg.Type != "joined" {
		t.Errorf("Type: got %q", msg.Type)
	}
}

func TestPlayerDisconnectedMessage_JSONRoundTrip(t *testing.T) {
	msg := PlayerDisconnectedMessage{
		Type:     "player_disconnected",
		PlayerID: "p1",
		Nickname: "Alice",
	}
	data, _ := json.Marshal(msg)
	var got PlayerDisconnectedMessage
	json.Unmarshal(data, &got)
	if got.Nickname != "Alice" {
		t.Errorf("Nickname: got %q", got.Nickname)
	}
}

func TestEndProposedMessage_JSONRoundTrip(t *testing.T) {
	msg := EndProposedMessage{
		Type:           "end_proposed",
		ProposerID:     "p1",
		ProposerName:   "Alice",
		TimeoutSeconds: 30,
	}
	data, _ := json.Marshal(msg)
	var got EndProposedMessage
	json.Unmarshal(data, &got)
	if got.TimeoutSeconds != 30 {
		t.Errorf("TimeoutSeconds: got %d", got.TimeoutSeconds)
	}
}

// --- Types ---

func TestRoomView_JSONRoundTrip(t *testing.T) {
	rv := RoomView{
		ID:          "r1",
		Name:        "Library",
		Description: "Dusty shelves",
		Type:        "public",
		Items:       []RoomViewItem{{ID: "i1", Name: "Book"}},
		Players:     []RoomViewPlayer{{ID: "p1", Nickname: "Alice"}},
		NPCs:        []RoomViewNPC{{ID: "npc1", Name: "Librarian"}},
	}
	data, _ := json.Marshal(rv)
	var got RoomView
	json.Unmarshal(data, &got)
	if got.Name != "Library" {
		t.Errorf("Name: got %q", got.Name)
	}
}
