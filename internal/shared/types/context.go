package types

type GameContext struct {
	World            World       `json:"world"`
	CurrentState     GameState   `json:"currentState"`
	RecentEvents     []interface{} `json:"recentEvents"`
	ActionLog        []interface{} `json:"actionLog"`
	RequestingPlayer Player      `json:"requestingPlayer"`
	CurrentRoom      Room        `json:"currentRoom"`
	PlayersInRoom    []Player    `json:"playersInRoom"`
}
