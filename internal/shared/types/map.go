package types

type GameMap struct {
	Rooms       []Room       `json:"rooms"`
	Connections []Connection `json:"connections"`
}

type Room struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        string   `json:"type"` // "public" | "private"
	Items       []Item   `json:"items"`
	NPCIDs      []string `json:"npcIds"`
	ClueIDs     []string `json:"clueIds"`
}

type Connection struct {
	RoomA         string `json:"roomA"`
	RoomB         string `json:"roomB"`
	Bidirectional bool   `json:"bidirectional"`
}
