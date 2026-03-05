package schemas

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ItemSchema struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	OwnerID     *string `json:"ownerId"`
	IsKey       bool    `json:"isKey"`
}

type EndConditionSchema struct {
	ID              string                 `json:"id"`
	Description     string                 `json:"description"`
	TriggerType     string                 `json:"triggerType"`
	TriggerCriteria map[string]interface{} `json:"triggerCriteria"`
	IsFallback      bool                   `json:"isFallback"`
}

type RoomSchema struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Type        string       `json:"type"`
	Items       []ItemSchema `json:"items"`
	NPCIDs      []string     `json:"npcIds"`
	ClueIDs     []string     `json:"clueIds"`
}

type PersonalGoalSchema struct {
	ID             string   `json:"id"`
	Description    string   `json:"description"`
	EvaluationHint string   `json:"evaluationHint"`
	EntityRefs     []string `json:"entityRefs"`
}

type RelationshipSchema struct {
	TargetCharacterName string `json:"targetCharacterName"`
	Description         string `json:"description"`
}

type PlayerRoleSchema struct {
	ID            string               `json:"id"`
	CharacterName string               `json:"characterName"`
	Background    string               `json:"background"`
	PersonalGoals []PersonalGoalSchema `json:"personalGoals"`
	Secret        string               `json:"secret"`
	SpecialRole   *string              `json:"specialRole"`
	Relationships []RelationshipSchema `json:"relationships"`
}

func (p *PlayerRoleSchema) Validate() error {
	if len(p.PersonalGoals) < 1 {
		return fmt.Errorf("personalGoals: minimum 1 required")
	}
	return nil
}

type NPCGimmickSchema struct {
	Description      string `json:"description"`
	TriggerCondition string `json:"triggerCondition"`
	Effect           string `json:"effect"`
}

type NPCSchema struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	CurrentRoomID     string            `json:"currentRoomId"`
	Persona           string            `json:"persona"`
	KnownInfo         []string          `json:"knownInfo"`
	HiddenInfo        []string          `json:"hiddenInfo"`
	BehaviorPrinciple string            `json:"behaviorPrinciple"`
	Gimmick           *NPCGimmickSchema `json:"gimmick"`
	InitialTrust      float64           `json:"initialTrust"`
}

type WorldGenerationMeta struct {
	Theme             string `json:"theme"`
	Setting           string `json:"setting"`
	EstimatedDuration int    `json:"estimatedDuration"`
	HasGM             bool   `json:"hasGM"`
	HasNPC            bool   `json:"hasNPC"`
}

func (m *WorldGenerationMeta) Validate() error {
	if m.EstimatedDuration < 10 {
		return fmt.Errorf("estimatedDuration: minimum 10 required")
	}
	if m.EstimatedDuration > 30 {
		return fmt.Errorf("estimatedDuration: maximum 30 allowed")
	}
	return nil
}

type WorldGenerationWorld struct {
	Title      string `json:"title"`
	Synopsis   string `json:"synopsis"`
	Atmosphere string `json:"atmosphere"`
}

type WinConditionSchema struct {
	Description        string `json:"description"`
	EvaluationCriteria string `json:"evaluationCriteria"`
}

type WorldGenerationGameStructure struct {
	Concept          string               `json:"concept"`
	CoreConflict     string               `json:"coreConflict"`
	ProgressionStyle string               `json:"progressionStyle"`
	CommonGoal       *string              `json:"commonGoal"`
	EndConditions    []EndConditionSchema `json:"endConditions"`
	WinConditions    []WinConditionSchema `json:"winConditions"`
	RequiredSystems  []string             `json:"requiredSystems"`
	BriefingText     FlexString           `json:"briefingText"`
}

// FlexString accepts both a JSON string and an array of strings (joined with newlines).
type FlexString string

func (f *FlexString) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = FlexString(s)
		return nil
	}
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*f = FlexString(strings.Join(arr, "\n"))
		return nil
	}
	return fmt.Errorf("briefingText: expected string or []string, got %s", string(data))
}

func (g *WorldGenerationGameStructure) Validate() error {
	if len(g.EndConditions) < 1 {
		return fmt.Errorf("endConditions: minimum 1 required")
	}
	hasFallback := false
	for _, ec := range g.EndConditions {
		if ec.IsFallback {
			hasFallback = true
			break
		}
	}
	if !hasFallback {
		return fmt.Errorf("endConditions: at least one condition must have isFallback == true (PRD FR-019, FR-027)")
	}
	return nil
}

type ConnectionSchema struct {
	RoomA         string `json:"roomA"`
	RoomB         string `json:"roomB"`
	Bidirectional bool   `json:"bidirectional"`
}

type WorldGenerationMap struct {
	Rooms       []RoomSchema       `json:"rooms"`
	Connections []ConnectionSchema `json:"connections"`
}

type WorldGenerationCharacters struct {
	PlayerRoles []PlayerRoleSchema `json:"playerRoles"`
	NPCs        []NPCSchema        `json:"npcs"`
}

type CharacterListEntrySchema struct {
	Name              string `json:"name"`
	PublicDescription string `json:"publicDescription"`
}

type NPCListEntrySchema struct {
	Name     string `json:"name"`
	Location string `json:"location"`
}

type WorldGenerationPublicInfo struct {
	Title         string                     `json:"title"`
	Synopsis      string                     `json:"synopsis"`
	CharacterList []CharacterListEntrySchema `json:"characterList"`
	Relationships string                     `json:"relationships"`
	MapOverview   string                     `json:"mapOverview"`
	NPCList       []NPCListEntrySchema       `json:"npcList"`
	GameRules     string                     `json:"gameRules"`
}

type SemiPublicInfoSchema struct {
	ID              string   `json:"id"`
	TargetPlayerIDs []string `json:"targetPlayerIds"`
	Content         string   `json:"content"`
}

type PrivateInfoSchema struct {
	PlayerID          string   `json:"playerId"`
	AdditionalSecrets []string `json:"additionalSecrets"`
}

type WorldGenerationInformation struct {
	Public     WorldGenerationPublicInfo `json:"public"`
	SemiPublic []SemiPublicInfoSchema    `json:"semiPublic"`
	Private    []PrivateInfoSchema       `json:"private"`
}

type ClueSchema struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	RoomID            string   `json:"roomId"`
	DiscoverCondition string   `json:"discoverCondition"`
	RelatedClueIDs    []string `json:"relatedClueIds"`
}

type GimmickSchema struct {
	ID               string `json:"id"`
	Description      string `json:"description"`
	RoomID           string `json:"roomId"`
	TriggerCondition string `json:"triggerCondition"`
	Effect           string `json:"effect"`
}

type WorldGeneration struct {
	Meta          WorldGenerationMeta          `json:"meta"`
	World         WorldGenerationWorld         `json:"world"`
	GameStructure WorldGenerationGameStructure `json:"gameStructure"`
	Map           WorldGenerationMap           `json:"map"`
	Characters    WorldGenerationCharacters    `json:"characters"`
	Information   WorldGenerationInformation   `json:"information"`
	Clues         []ClueSchema                 `json:"clues"`
	Gimmicks      []GimmickSchema              `json:"gimmicks"`
}

// validateMapConnectivity checks that all rooms are reachable from the first room via BFS.
func validateMapConnectivity(m WorldGenerationMap) error {
	if len(m.Rooms) == 0 {
		return nil
	}
	adj := make(map[string][]string)
	for _, c := range m.Connections {
		adj[c.RoomA] = append(adj[c.RoomA], c.RoomB)
		adj[c.RoomB] = append(adj[c.RoomB], c.RoomA)
	}
	visited := make(map[string]bool)
	queue := []string{m.Rooms[0].ID}
	visited[m.Rooms[0].ID] = true
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, neighbor := range adj[cur] {
			if !visited[neighbor] {
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}
	for _, room := range m.Rooms {
		if !visited[room.ID] {
			return fmt.Errorf("map: room %q is isolated (not reachable from %q)", room.ID, m.Rooms[0].ID)
		}
	}
	return nil
}

func (w *WorldGeneration) Validate() error {
	if err := w.Meta.Validate(); err != nil {
		return fmt.Errorf("meta: %w", err)
	}
	if err := w.GameStructure.Validate(); err != nil {
		return fmt.Errorf("gameStructure: %w", err)
	}
	for i, role := range w.Characters.PlayerRoles {
		if err := role.Validate(); err != nil {
			return fmt.Errorf("characters.playerRoles[%d]: %w", i, err)
		}
	}
	// PRD FR-010: rooms >= playerRoles + 2
	minRooms := len(w.Characters.PlayerRoles) + 2
	if len(w.Map.Rooms) < minRooms {
		return fmt.Errorf("map.rooms: minimum %d rooms required (playerRoles + 2), got %d", minRooms, len(w.Map.Rooms))
	}
	// PRD FR-012: clues >= playerRoles * 2
	minClues := len(w.Characters.PlayerRoles) * 2
	if len(w.Clues) < minClues {
		return fmt.Errorf("clues: minimum %d clues required (playerRoles * 2), got %d", minClues, len(w.Clues))
	}
	// PRD FR-013: at least 1 semi-public info
	if len(w.Information.SemiPublic) < 1 {
		return fmt.Errorf("information.semiPublic: minimum 1 semi-public info required")
	}
	// NPC placement: all NPC currentRoomIds must reference valid rooms
	roomIDs := make(map[string]bool)
	for _, room := range w.Map.Rooms {
		roomIDs[room.ID] = true
	}
	for i, npc := range w.Characters.NPCs {
		if !roomIDs[npc.CurrentRoomID] {
			return fmt.Errorf("characters.npcs[%d]: currentRoomId %q does not reference a valid room", i, npc.CurrentRoomID)
		}
	}
	// Clue placement validation
	for i, clue := range w.Clues {
		if !roomIDs[clue.RoomID] {
			return fmt.Errorf("clues[%d]: roomId %q does not reference a valid room", i, clue.RoomID)
		}
	}
	// Gimmick placement validation
	for i, gimmick := range w.Gimmicks {
		if !roomIDs[gimmick.RoomID] {
			return fmt.Errorf("gimmicks[%d]: roomId %q does not reference a valid room", i, gimmick.RoomID)
		}
	}
	// Room reference integrity
	npcIDs := make(map[string]bool)
	for _, npc := range w.Characters.NPCs {
		npcIDs[npc.ID] = true
	}
	clueIDs := make(map[string]bool)
	for _, clue := range w.Clues {
		clueIDs[clue.ID] = true
	}
	for i, room := range w.Map.Rooms {
		for _, npcID := range room.NPCIDs {
			if !npcIDs[npcID] {
				return fmt.Errorf("map.rooms[%d]: npcId %q does not reference a valid NPC", i, npcID)
			}
		}
		for _, clueID := range room.ClueIDs {
			if !clueIDs[clueID] {
				return fmt.Errorf("map.rooms[%d]: clueId %q does not reference a valid clue", i, clueID)
			}
		}
	}
	// Map connectivity: no isolated rooms
	if err := validateMapConnectivity(w.Map); err != nil {
		return err
	}
	// SemiPublicInfo.TargetPlayerIDs reference integrity
	roleIDs := make(map[string]bool)
	for _, role := range w.Characters.PlayerRoles {
		roleIDs[role.ID] = true
	}
	for i, sp := range w.Information.SemiPublic {
		for j, pid := range sp.TargetPlayerIDs {
			if !roleIDs[pid] {
				return fmt.Errorf("information.semiPublic[%d].targetPlayerIds[%d]: %q does not reference a valid PlayerRole ID", i, j, pid)
			}
		}
	}
	return nil
}
