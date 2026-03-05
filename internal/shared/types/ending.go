package types

type GameEndData struct {
	CommonResult  string         `json:"commonResult"`
	PlayerEndings []PlayerEnding `json:"playerEndings"`
	SecretReveal  SecretReveal   `json:"secretReveal"`
}

type PlayerEnding struct {
	PlayerID    string       `json:"playerId"`
	Summary     string       `json:"summary"`
	GoalResults []GoalResult `json:"goalResults"`
	Narrative   string       `json:"narrative"`
}

type GoalResult struct {
	GoalID      string `json:"goalId"`
	Description string `json:"description"`
	Achieved    bool   `json:"achieved"`
	Evaluation  string `json:"evaluation"`
}

type PlayerSecretEntry struct {
	PlayerID      string  `json:"playerId"`
	CharacterName string  `json:"characterName"`
	Secret        string  `json:"secret"`
	SpecialRole   *string `json:"specialRole"`
}

type SemiPublicRevealEntry struct {
	Info          string   `json:"info"`
	SharedBetween []string `json:"sharedBetween"`
}

type UndiscoveredClueEntry struct {
	Clue     Clue   `json:"clue"`
	RoomName string `json:"roomName"`
}

type NPCSecretEntry struct {
	NPCName    string   `json:"npcName"`
	HiddenInfo []string `json:"hiddenInfo"`
}

type GimmickReveal struct {
	GimmickID   string `json:"gimmickId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	RoomID      string `json:"roomId"`
	Condition   string `json:"condition"`
}

type SecretReveal struct {
	PlayerSecrets       []PlayerSecretEntry     `json:"playerSecrets"`
	SemiPublicReveal    []SemiPublicRevealEntry  `json:"semiPublicReveal"`
	UndiscoveredClues   []UndiscoveredClueEntry  `json:"undiscoveredClues"`
	NPCSecrets          []NPCSecretEntry         `json:"npcSecrets"`
	UntriggeredGimmicks []GimmickReveal          `json:"untriggeredGimmicks"`
}
