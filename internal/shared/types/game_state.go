package types

type GameState struct {
	Players       map[string]*Player      `json:"players"`
	ClueStates    map[string]ClueState    `json:"clueStates"`
	NPCStates     map[string]NPCState     `json:"npcStates"`
	GimmickStates map[string]GimmickState `json:"gimmickStates"`
	ElapsedTime   int64                   `json:"elapsedTime"`
}

type ClueState struct {
	IsDiscovered bool     `json:"isDiscovered"`
	DiscoveredBy []string `json:"discoveredBy"`
}

type NPCState struct {
	TrustLevels         map[string]float64   `json:"trustLevels"`
	ConversationHistory []ConversationRecord `json:"conversationHistory"`
	GimmickTriggered    bool                 `json:"gimmickTriggered"`
}

type ConversationRecord struct {
	PlayerID  string `json:"playerId"`
	Message   string `json:"message"`
	Response  string `json:"response"`
	Timestamp int64  `json:"timestamp"`
}

type GimmickState struct {
	IsTriggered bool   `json:"isTriggered"`
	TriggeredAt *int64 `json:"triggeredAt,omitempty"`
}

type GoalProgressEntry struct {
	GoalID   string   `json:"goalId"`
	Evidence []string `json:"evidence"`
}
