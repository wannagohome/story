package protocol

import "github.com/anthropics/story/internal/shared/types"

type LobbyPlayer struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	IsHost   bool   `json:"isHost"`
}

type RoomViewItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RoomViewPlayer struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
}

type RoomViewNPC struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RoomView struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Type        string           `json:"type"`
	Items       []RoomViewItem   `json:"items"`
	Players     []RoomViewPlayer `json:"players"`
	NPCs        []RoomViewNPC    `json:"npcs"`
}

type MapViewRoom struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	PlayerCount int      `json:"playerCount"`
	PlayerNames []string `json:"playerNames"`
}

type MapView struct {
	Rooms       []MapViewRoom      `json:"rooms"`
	Connections []types.Connection `json:"connections"`
	MyRoomID    string             `json:"myRoomId"`
}

type PlayerLocationInfo struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	RoomID   string `json:"roomId"`
	RoomName string `json:"roomName"`
	Status   string `json:"status"`
}

type CommandInfo struct {
	Command     string `json:"command"`
	Description string `json:"description"`
	Usage       string `json:"usage"`
}

type VoteResultEntry struct {
	CandidateID   string `json:"candidateId"`
	CandidateName string `json:"candidateName"`
	Votes         int    `json:"votes"`
}

type ErrorCode string

const (
	ErrorCodeInvalidRoomCode       ErrorCode = "INVALID_ROOM_CODE"
	ErrorCodeGameAlreadyStarted    ErrorCode = "GAME_ALREADY_STARTED"
	ErrorCodeRoomFull              ErrorCode = "ROOM_FULL"
	ErrorCodeDuplicateNickname     ErrorCode = "DUPLICATE_NICKNAME"
	ErrorCodeNotHost               ErrorCode = "NOT_HOST"
	ErrorCodeNotEnoughPlayers      ErrorCode = "NOT_ENOUGH_PLAYERS"
	ErrorCodeInvalidMove           ErrorCode = "INVALID_MOVE"
	ErrorCodeNPCNotInRoom          ErrorCode = "NPC_NOT_IN_ROOM"
	ErrorCodeItemNotFound          ErrorCode = "ITEM_NOT_FOUND"
	ErrorCodeUnknownCommand        ErrorCode = "UNKNOWN_COMMAND"
	ErrorCodeNotSupported          ErrorCode = "NOT_SUPPORTED"
	ErrorCodeVoteNotActive         ErrorCode = "VOTE_NOT_ACTIVE"
	ErrorCodeVotingDisabled        ErrorCode = "VOTING_DISABLED"
	ErrorCodeEndVoteAlreadyOpen    ErrorCode = "END_VOTE_ALREADY_OPEN"
	ErrorCodeEmptyMessage          ErrorCode = "EMPTY_MESSAGE"
	ErrorCodeMessageTooLong        ErrorCode = "MESSAGE_TOO_LONG"
	ErrorCodeInvalidNickname       ErrorCode = "INVALID_NICKNAME"
	ErrorCodeInvalidAPIKey         ErrorCode = "INVALID_API_KEY"
	ErrorCodeSaveFailed            ErrorCode = "SAVE_FAILED"
	ErrorCodeConnectionLost        ErrorCode = "CONNECTION_LOST"
	ErrorCodeWSConnectionFailed    ErrorCode = "WS_CONNECTION_FAILED"
	ErrorCodeBriefingNotComplete   ErrorCode = "BRIEFING_NOT_COMPLETE"
)
