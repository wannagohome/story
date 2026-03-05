package types

type Player struct {
	ID                string       `json:"id"`
	Nickname          string       `json:"nickname"`
	IsHost            bool         `json:"isHost"`
	Status            string       `json:"status"` // "connected" | "disconnected" | "inactive"
	CurrentRoomID     string       `json:"currentRoomId"`
	Role              *PlayerRole  `json:"role"`
	Inventory         []Item       `json:"inventory"`
	DiscoveredClueIDs []string     `json:"discoveredClueIds"`
	MoveHistory       []MoveRecord `json:"moveHistory"`
	ConnectedAt       int64        `json:"connectedAt"`
}

type MoveRecord struct {
	RoomID    string `json:"roomId"`
	RoomName  string `json:"roomName"`
	EnteredAt int64  `json:"enteredAt"`
	LeftAt    *int64 `json:"leftAt,omitempty"`
}

type PlayerRole struct {
	ID            string         `json:"id"`
	CharacterName string         `json:"characterName"`
	Background    string         `json:"background"`
	PersonalGoals []PersonalGoal `json:"personalGoals"`
	Secret        string         `json:"secret"`
	SpecialRole   *string        `json:"specialRole"`
	Relationships []Relationship `json:"relationships"`
}

type PersonalGoal struct {
	ID             string   `json:"id"`
	Description    string   `json:"description"`
	EvaluationHint string   `json:"evaluationHint"`
	EntityRefs     []string `json:"entityRefs"`
	IsAchieved     *bool    `json:"isAchieved"`
}

type Relationship struct {
	TargetCharacterName string `json:"targetCharacterName"`
	Description         string `json:"description"`
}
