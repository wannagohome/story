package state

type ActiveVoteState struct {
	Reason         string
	Candidates     []string
	TimeoutSeconds int
	VotedCount     int
	TotalVoters    int
}

type EndProposalState struct {
	ProposerID     string
	ProposerName   string
	TimeoutSeconds int
}

type VoteResult struct {
	Results []VoteResultEntry
	Outcome string
}

type ActiveSolveState struct {
	Prompt         string
	TimeoutSeconds int
	SubmittedCount int
	TotalPlayers   int
}

type EndingData struct {
	CommonResult   string
	PersonalEnding PlayerEnding
	SecretReveal   SecretReveal
}

type ClientError struct {
	Code    string
	Message string
}

type GameEvent struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Timestamp int64          `json:"timestamp"`
	Data      map[string]any `json:"data"`
}

type DisplayMessage struct {
	ID             string
	Kind           string // "chat" | "event" | "system"
	SenderID       string
	SenderName     string
	Content        string
	Scope          string  // "room" | "global"
	SenderLocation *string // only for global scope
	Event          GameEvent
	Timestamp      int64
}
