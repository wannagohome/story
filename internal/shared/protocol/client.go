package protocol

// --- Standalone client message types (for type-safe construction) ---

type JoinMessage struct {
	Type     string `json:"type"`     // "join"
	Nickname string `json:"nickname"`
}

type RejoinMessage struct {
	Type     string `json:"type"`     // "rejoin"
	PlayerID string `json:"playerId"`
}

type StartGameMessage struct {
	Type         string `json:"type"`                   // "start_game"
	ThemeKeyword string `json:"themeKeyword,omitempty"`
}

type ReadyMessage struct {
	Type  string `json:"type"`  // "ready"
	Phase string `json:"phase"` // "briefing_read" | "game_ready"
}

type CancelGameMessage struct {
	Type string `json:"type"` // "cancel_game"
}

type ChatClientMessage struct {
	Type    string `json:"type"`    // "chat"
	Content string `json:"content"`
}

type ShoutMessage struct {
	Type    string `json:"type"`    // "shout"
	Content string `json:"content"`
}

type MoveMessage struct {
	Type         string `json:"type"`         // "move"
	TargetRoomID string `json:"targetRoomId"`
}

type ExamineMessage struct {
	Type   string  `json:"type"`             // "examine"
	Target *string `json:"target,omitempty"`
}

type DoMessage struct {
	Type   string `json:"type"`   // "do"
	Action string `json:"action"`
}

type TalkMessage struct {
	Type    string `json:"type"`    // "talk"
	NPCID   string `json:"npcId"`
	Message string `json:"message"`
}

type GiveMessage struct {
	Type   string `json:"type"`   // "give"
	NPCID  string `json:"npcId"`
	ItemID string `json:"itemId"`
}

type VoteMessage struct {
	Type     string `json:"type"`     // "vote"
	TargetID string `json:"targetId"`
}

type SolveMessage struct {
	Type   string `json:"type"`   // "solve"
	Answer string `json:"answer"`
}

type ProposeEndMessage struct {
	Type string `json:"type"` // "propose_end"
}

type EndVoteMessage struct {
	Type  string `json:"type"`  // "end_vote"
	Agree bool   `json:"agree"`
}

type RequestLookMessage struct {
	Type string `json:"type"` // "request_look"
}

type RequestInventoryMessage struct {
	Type string `json:"type"` // "request_inventory"
}

type RequestRoleMessage struct {
	Type string `json:"type"` // "request_role"
}

type RequestMapMessage struct {
	Type string `json:"type"` // "request_map"
}

type RequestWhoMessage struct {
	Type string `json:"type"` // "request_who"
}

type RequestHelpMessage struct {
	Type string `json:"type"` // "request_help"
}

type SubmitFeedbackMessage struct {
	Type            string  `json:"type"`            // "submit_feedback"
	FunRating       int     `json:"funRating"`
	ImmersionRating int     `json:"immersionRating"`
	Comment         *string `json:"comment"`
}

type SkipFeedbackMessage struct {
	Type string `json:"type"` // "skip_feedback"
}

// ClientMessage is the union struct for all client-to-server messages.
// The Type field determines which other fields are relevant.
type ClientMessage struct {
	Type            string  `json:"type"`
	Nickname        string  `json:"nickname,omitempty"`
	Content         string  `json:"content,omitempty"`
	Scope           string  `json:"scope,omitempty"`
	TargetRoomID    string  `json:"targetRoomId,omitempty"`
	Target          *string `json:"target,omitempty"`
	Action          string  `json:"action,omitempty"`
	NPCID           string  `json:"npcId,omitempty"`
	Message         string  `json:"message,omitempty"`
	TargetID        string  `json:"targetId,omitempty"`
	Agree           bool    `json:"agree,omitempty"`
	Answer          string  `json:"answer,omitempty"`
	ItemName        string  `json:"itemName,omitempty"`
	Phase           string  `json:"phase,omitempty"`
	FunRating       int     `json:"funRating,omitempty"`
	ImmersionRating int     `json:"immersionRating,omitempty"`
	Comment         *string `json:"comment,omitempty"`
	PlayerID        string  `json:"playerId,omitempty"`
	ThemeKeyword    string  `json:"themeKeyword,omitempty"`
}
