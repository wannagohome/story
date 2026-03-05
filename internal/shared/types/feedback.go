package types

type Feedback struct {
	PlayerID        string  `json:"playerId"`
	FunRating       int     `json:"funRating"`
	ImmersionRating int     `json:"immersionRating"`
	Comment         *string `json:"comment"`
	SubmittedAt     int64   `json:"submittedAt"`
}
