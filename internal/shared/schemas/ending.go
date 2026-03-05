package schemas

// GoalResultSchema represents the result of a personal goal evaluation.
type GoalResultSchema struct {
	GoalID      string `json:"goalId"`
	Description string `json:"description"`
	Achieved    bool   `json:"achieved"`
	Evaluation  string `json:"evaluation"`
}

// PlayerEndingSchema represents a single player's ending.
type PlayerEndingSchema struct {
	PlayerID    string             `json:"playerId"`
	Summary     string             `json:"summary"`
	GoalResults []GoalResultSchema `json:"goalResults"`
	Narrative   string             `json:"narrative"`
}

// Ending is the AI response schema for game ending generation.
// SecretReveal is not included here; it is constructed server-side
// from rule-based logic (PlayerRole.Secret, world.Information.SemiPublic,
// undiscovered clues, NPC.HiddenInfo).
type Ending struct {
	CommonResult  string               `json:"commonResult"`
	PlayerEndings []PlayerEndingSchema `json:"playerEndings"`
}
