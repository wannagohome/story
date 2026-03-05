package types

type Item struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	OwnerID     *string `json:"ownerId"`
	IsKey       bool    `json:"isKey"`
}

type Clue struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	RoomID            string   `json:"roomId"`
	DiscoverCondition string   `json:"discoverCondition"`
	RelatedClueIDs    []string `json:"relatedClueIds"`
}

type Gimmick struct {
	ID               string `json:"id"`
	Description      string `json:"description"`
	RoomID           string `json:"roomId"`
	TriggerCondition string `json:"triggerCondition"`
	Effect           string `json:"effect"`
}
