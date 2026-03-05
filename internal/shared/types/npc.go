package types

type NPC struct {
	ID                string      `json:"id"`
	Name              string      `json:"name"`
	CurrentRoomID     string      `json:"currentRoomId"`
	Persona           string      `json:"persona"`
	KnownInfo         []string    `json:"knownInfo"`
	HiddenInfo        []string    `json:"hiddenInfo"`
	BehaviorPrinciple string      `json:"behaviorPrinciple"`
	Gimmick           *NPCGimmick `json:"gimmick"`
	InitialTrust      float64     `json:"initialTrust"`
}

type NPCGimmick struct {
	Description      string `json:"description"`
	TriggerCondition string `json:"triggerCondition"`
	Effect           string `json:"effect"`
}
