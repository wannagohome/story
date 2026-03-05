package types

type InformationLayers struct {
	Public     PublicInfo       `json:"public"`
	SemiPublic []SemiPublicInfo `json:"semiPublic"`
	Private    []PrivateInfo    `json:"private"`
}

type CharacterListEntry struct {
	Name              string `json:"name"`
	PublicDescription string `json:"publicDescription"`
}

type NPCListEntry struct {
	Name     string `json:"name"`
	Location string `json:"location"`
}

type PublicInfo struct {
	Title         string               `json:"title"`
	Synopsis      string               `json:"synopsis"`
	CharacterList []CharacterListEntry `json:"characterList"`
	Relationships string               `json:"relationships"`
	MapOverview   string               `json:"mapOverview"`
	NPCList       []NPCListEntry       `json:"npcList"`
	GameRules     string               `json:"gameRules"`
}

type SemiPublicInfo struct {
	ID              string   `json:"id"`
	TargetPlayerIDs []string `json:"targetPlayerIds"`
	Content         string   `json:"content"`
}

type PrivateInfo struct {
	PlayerID          string   `json:"playerId"`
	AdditionalSecrets []string `json:"additionalSecrets"`
}
