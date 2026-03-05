package types

type World struct {
	Title         string            `json:"title"`
	Synopsis      string            `json:"synopsis"`
	Atmosphere    string            `json:"atmosphere"`
	GameStructure GameStructure     `json:"gameStructure"`
	Map           GameMap           `json:"map"`
	PlayerRoles   []PlayerRole      `json:"playerRoles"`
	NPCs          []NPC             `json:"npcs"`
	Clues         []Clue            `json:"clues"`
	Gimmicks      []Gimmick         `json:"gimmicks"`
	Information   InformationLayers `json:"information"`
}

type GameStructure struct {
	Concept           string           `json:"concept"`
	CoreConflict      string           `json:"coreConflict"`
	ProgressionStyle  string           `json:"progressionStyle"`
	CommonGoal        *string          `json:"commonGoal"`
	EstimatedDuration int              `json:"estimatedDuration"`
	EndConditions     []EndCondition   `json:"endConditions"`
	WinConditions     []WinCondition   `json:"winConditions"`
	RequiredSystems   []RequiredSystem `json:"requiredSystems"`
	BriefingText      string           `json:"briefingText"`
}

type RequiredSystem string

const (
	RequiredSystemVote      RequiredSystem = "vote"
	RequiredSystemConsensus RequiredSystem = "consensus"
	RequiredSystemAIJudge   RequiredSystem = "ai_judge"
)

type EndCondition struct {
	ID              string                 `json:"id"`
	Description     string                 `json:"description"`
	TriggerType     string                 `json:"triggerType"`
	TriggerCriteria map[string]interface{} `json:"triggerCriteria"`
	IsFallback      bool                   `json:"isFallback"`
}

type WinCondition struct {
	Description        string `json:"description"`
	EvaluationCriteria string `json:"evaluationCriteria"`
}
