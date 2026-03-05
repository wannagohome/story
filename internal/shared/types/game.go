package types

type GameStatus string

const (
	GameStatusLobby      GameStatus = "lobby"
	GameStatusGenerating GameStatus = "generating"
	GameStatusBriefing   GameStatus = "briefing"
	GameStatusPlaying    GameStatus = "playing"
	GameStatusEnding     GameStatus = "ending"
	GameStatusFinished   GameStatus = "finished"
)

type Game struct {
	ID        string             `json:"id"`
	RoomCode  string             `json:"roomCode"`
	HostID    string             `json:"hostId"`
	Status    GameStatus         `json:"status"`
	Settings  GameSettings       `json:"settings"`
	World     *World             `json:"world"`
	Players   map[string]*Player `json:"players"`
	EventLog  []interface{}      `json:"eventLog"`
	CreatedAt int64              `json:"createdAt"`
	StartedAt *int64             `json:"startedAt"`
	EndedAt   *int64             `json:"endedAt"`
}

type GameSettings struct {
	MaxPlayers     int  `json:"maxPlayers"`
	TimeoutMinutes int  `json:"timeoutMinutes"`
	HasGM          bool `json:"hasGM"`
	HasNPC         bool `json:"hasNPC"`
}
