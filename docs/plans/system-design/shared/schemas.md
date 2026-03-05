# Go Struct Validation (`internal/shared/schemas/`)

AI 출력 검증용 Go struct + Validate() 메서드. 런타임에 AI 응답이 정의된 구조를 따르는지 검증.

---

## 왜 Go struct validation인가

- Go struct + json 태그로 언마샬링 시 기본 타입 검증 자동 수행
- Validate() 메서드로 범위, 최솟값, 최댓값 등 제약 조건을 명시적으로 검증
- 검증 실패 시 `fmt.Errorf`로 어떤 필드가 잘못되었는지 명확히 전달

## WorldGeneration Schema

세계 생성 AI 출력의 전체 스키마.

```go
// internal/shared/schemas/world_generation.go
package schemas

import "fmt"

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
	TriggerType     string                 `json:"triggerType"` // "vote" | "consensus" | "event" | "ai_judgment" | "timeout"
	TriggerCriteria map[string]interface{} `json:"triggerCriteria"`
	IsFallback      bool                   `json:"isFallback"`
}

type RoomSchema struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Type        string       `json:"type"` // "public" | "private"
	Items       []ItemSchema `json:"items"`
	NPCIDs      []string     `json:"npcIds"`
	ClueIDs     []string     `json:"clueIds"`
}

type PersonalGoalSchema struct {
	ID             string   `json:"id"`
	Description    string   `json:"description"`
	EvaluationHint string   `json:"evaluationHint"` // optional: AI가 목표 달성 여부를 판단할 때 참고할 힌트
	EntityRefs     []string `json:"entityRefs"`     // 이 목표가 참조하는 엔티티 ID 목록 (NPC, 방, 단서 등). StoryValidator가 참조 무결성 검증에 사용
}

type RelationshipSchema struct {
	TargetCharacterName string `json:"targetCharacterName"`
	Description         string `json:"description"`
}

type PlayerRoleSchema struct {
	ID            string               `json:"id"`            // 플레이어 역할 식별자
	CharacterName string               `json:"characterName"`
	Background    string               `json:"background"`
	PersonalGoals []PersonalGoalSchema `json:"personalGoals"` // min(1)
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
	InitialTrust      float64           `json:"initialTrust"` // 초기 신뢰도 (0~1, 기본 0.5). types.md NPC.InitialTrust와 일치.
}

type WorldGenerationMeta struct {
	Theme             string `json:"theme"`
	Setting           string `json:"setting"`
	EstimatedDuration int    `json:"estimatedDuration"` // min(10), max(30)
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
	EndConditions    []EndConditionSchema `json:"endConditions"` // min(1)
	WinConditions    []WinConditionSchema `json:"winConditions"`
	RequiredSystems  []string             `json:"requiredSystems"` // "vote" | "consensus" | "ai_judge"
	BriefingText     string               `json:"briefingText"`
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
	Bidirectional bool   `json:"bidirectional"` // 양방향 여부 (PRD: 기본 true). types.md Connection과 일치.
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
// This ensures no isolated rooms exist (PRD FR-010).
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
	// PRD FR-010: 방의 수는 플레이어 수 + 2 이상
	minRooms := len(w.Characters.PlayerRoles) + 2
	if len(w.Map.Rooms) < minRooms {
		return fmt.Errorf("map.rooms: minimum %d rooms required (playerRoles + 2), got %d", minRooms, len(w.Map.Rooms))
	}
	// PRD FR-012: 최소 플레이어 수 x 2개의 단서
	minClues := len(w.Characters.PlayerRoles) * 2
	if len(w.Clues) < minClues {
		return fmt.Errorf("clues: minimum %d clues required (playerRoles * 2), got %d", minClues, len(w.Clues))
	}
	// PRD FR-013: 반공개 정보가 최소 1쌍 이상 (동맹 형성의 씨앗)
	if len(w.Information.SemiPublic) < 1 {
		return fmt.Errorf("information.semiPublic: minimum 1 semi-public info required")
	}
	// NPC 배치 검증: 모든 NPC의 currentRoomId가 유효한 방 ID를 참조하는지 확인
	roomIDs := make(map[string]bool)
	for _, room := range w.Map.Rooms {
		roomIDs[room.ID] = true
	}
	for i, npc := range w.Characters.NPCs {
		if !roomIDs[npc.CurrentRoomID] {
			return fmt.Errorf("characters.npcs[%d]: currentRoomId %q does not reference a valid room", i, npc.CurrentRoomID)
		}
	}
	// 단서 배치 검증 (FR-022)
	for i, clue := range w.Clues {
		if !roomIDs[clue.RoomID] {
			return fmt.Errorf("clues[%d]: roomId %q does not reference a valid room", i, clue.RoomID)
		}
	}
	// 기믹 배치 검증
	for i, gimmick := range w.Gimmicks {
		if !roomIDs[gimmick.RoomID] {
			return fmt.Errorf("gimmicks[%d]: roomId %q does not reference a valid room", i, gimmick.RoomID)
		}
	}
	// Room 참조 무결성 검증
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
	// PRD FR-010: 맵 연결성 검증 — 고립된 방 없음
	if err := validateMapConnectivity(w.Map); err != nil {
		return err
	}
	// SemiPublicInfo.TargetPlayerIDs 참조 무결성 검증
	// 모든 TargetPlayerIDs가 유효한 PlayerRole ID를 참조하는지 확인
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
```

## GameResponse Schema

게임 중 AI 응답 스키마. /examine, /do, NPC 대화 등의 결과.

```go
// internal/shared/schemas/game_response.go
package schemas

import (
	"encoding/json"
	"fmt"
)

// AI가 생성하는 게임 이벤트. type 필드로 구분.
// 참고: BaseEvent의 id, timestamp, visibility는 서버에서 부여. AI 응답에는 포함하지 않음.

// AIGameEvent — AI 응답 이벤트의 공통 인터페이스
type AIGameEvent interface {
	AIEventType() string
}

// 각 이벤트 타입별 데이터 struct
type AINarrationEventData struct {
	Text string `json:"text"`
	Mood string `json:"mood"`
}
type AINarrationEvent struct {
	Type string               `json:"type"` // "narration"
	Data AINarrationEventData `json:"data"`
}
func (e AINarrationEvent) AIEventType() string { return "narration" }

type AINPCDialogueEventData struct {
	NPCID   string `json:"npcId"`
	NPCName string `json:"npcName"`
	Text    string `json:"text"`
	Emotion string `json:"emotion"`
}
type AINPCDialogueEvent struct {
	Type string                 `json:"type"` // "npc_dialogue"
	Data AINPCDialogueEventData `json:"data"`
}
func (e AINPCDialogueEvent) AIEventType() string { return "npc_dialogue" }

type AINPCGiveItemEventData struct {
	NPCID      string     `json:"npcId"`
	NPCName    string     `json:"npcName"`
	PlayerID   string     `json:"playerId"`
	PlayerName string     `json:"playerName"`
	Item       ItemSchema `json:"item"`
}
type AINPCGiveItemEvent struct {
	Type string                 `json:"type"` // "npc_give_item"
	Data AINPCGiveItemEventData `json:"data"`
}
func (e AINPCGiveItemEvent) AIEventType() string { return "npc_give_item" }

type AINPCReceiveItemEventData struct {
	NPCID      string     `json:"npcId"`
	NPCName    string     `json:"npcName"`
	PlayerID   string     `json:"playerId"`
	PlayerName string     `json:"playerName"`
	Item       ItemSchema `json:"item"`
}
type AINPCReceiveItemEvent struct {
	Type string                    `json:"type"` // "npc_receive_item"
	Data AINPCReceiveItemEventData `json:"data"`
}
func (e AINPCReceiveItemEvent) AIEventType() string { return "npc_receive_item" }

type AINPCRevealClue struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
type AINPCRevealEventData struct {
	NPCID      string           `json:"npcId"`
	NPCName    string           `json:"npcName"`
	Revelation string           `json:"revelation"`
	Clue       *AINPCRevealClue `json:"clue"`
}
type AINPCRevealEvent struct {
	Type string               `json:"type"` // "npc_reveal"
	Data AINPCRevealEventData `json:"data"`
}
func (e AINPCRevealEvent) AIEventType() string { return "npc_reveal" }

type AIClueFoundEventData struct {
	PlayerID   string          `json:"playerId"`
	PlayerName string          `json:"playerName"`
	Clue       AINPCRevealClue `json:"clue"`
	Location   string          `json:"location"`
}
type AIClueFoundEvent struct {
	Type string               `json:"type"` // "clue_found"
	Data AIClueFoundEventData `json:"data"`
}
func (e AIClueFoundEvent) AIEventType() string { return "clue_found" }

type AIStoryEventData struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Consequences []string `json:"consequences"`
}
type AIStoryEvent struct {
	Type string           `json:"type"` // "story_event"
	Data AIStoryEventData `json:"data"`
}
func (e AIStoryEvent) AIEventType() string { return "story_event" }

type AIExamineResultEventData struct {
	PlayerID    string `json:"playerId"`
	PlayerName  string `json:"playerName"`
	Target      string `json:"target"`
	Description string `json:"description"`
	ClueFound   bool   `json:"clueFound"`
}
type AIExamineResultEvent struct {
	Type string                   `json:"type"` // "examine_result"
	Data AIExamineResultEventData `json:"data"`
}
func (e AIExamineResultEvent) AIEventType() string { return "examine_result" }

type AIActionResultEventData struct {
	PlayerID        string   `json:"playerId"`
	PlayerName      string   `json:"playerName"`
	Action          string   `json:"action"`
	Result          string   `json:"result"`
	TriggeredEvents []string `json:"triggeredEvents"`
}
type AIActionResultEvent struct {
	Type string                  `json:"type"` // "action_result"
	Data AIActionResultEventData `json:"data"`
}
func (e AIActionResultEvent) AIEventType() string { return "action_result" }

type AIPlayerMoveEventData struct {
	PlayerID   string `json:"playerId"`
	PlayerName string `json:"playerName"`
	From       string `json:"from"`
	To         string `json:"to"`
}
type AIPlayerMoveEvent struct {
	Type string                `json:"type"` // "player_move"
	Data AIPlayerMoveEventData `json:"data"`
}
func (e AIPlayerMoveEvent) AIEventType() string { return "player_move" }

type AIGameEndEventData struct {
	Reason       string `json:"reason"`
	CommonResult string `json:"commonResult"`
}
type AIGameEndEvent struct {
	Type string             `json:"type"` // "game_end"
	Data AIGameEndEventData `json:"data"`
}
func (e AIGameEndEvent) AIEventType() string { return "game_end" }

type AITimeWarningEventData struct {
	RemainingMinutes int `json:"remainingMinutes"`
}
type AITimeWarningEvent struct {
	Type string                 `json:"type"` // "time_warning"
	Data AITimeWarningEventData `json:"data"`
}
func (e AITimeWarningEvent) AIEventType() string { return "time_warning" }

// ParseAIGameEvent — type 필드로 구분한 뒤 구체 타입으로 역직렬화
func ParseAIGameEvent(data json.RawMessage) (AIGameEvent, error) {
	var raw struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	switch raw.Type {
	case "narration":
		var e AINarrationEvent
		return e, json.Unmarshal(data, &e)
	case "npc_dialogue":
		var e AINPCDialogueEvent
		return e, json.Unmarshal(data, &e)
	case "npc_give_item":
		var e AINPCGiveItemEvent
		return e, json.Unmarshal(data, &e)
	case "npc_receive_item":
		var e AINPCReceiveItemEvent
		return e, json.Unmarshal(data, &e)
	case "npc_reveal":
		var e AINPCRevealEvent
		return e, json.Unmarshal(data, &e)
	case "clue_found":
		var e AIClueFoundEvent
		return e, json.Unmarshal(data, &e)
	case "story_event":
		var e AIStoryEvent
		return e, json.Unmarshal(data, &e)
	case "examine_result":
		var e AIExamineResultEvent
		return e, json.Unmarshal(data, &e)
	case "action_result":
		var e AIActionResultEvent
		return e, json.Unmarshal(data, &e)
	case "player_move":
		var e AIPlayerMoveEvent
		return e, json.Unmarshal(data, &e)
	case "game_end":
		var e AIGameEndEvent
		return e, json.Unmarshal(data, &e)
	case "time_warning":
		var e AITimeWarningEvent
		return e, json.Unmarshal(data, &e)
	default:
		return nil, fmt.Errorf("unknown event type: %s", raw.Type)
	}
}

// StateChange — AI가 요청하는 게임 상태 변경
type StateChange interface {
	StateChangeType() string
}

type StateChangeDiscoverClue struct {
	Type     string `json:"type"`     // "discover_clue"
	PlayerID string `json:"playerId"`
	ClueID   string `json:"clueId"`
}
func (s StateChangeDiscoverClue) StateChangeType() string { return "discover_clue" }

type StateChangeAddItem struct {
	Type     string     `json:"type"`     // "add_item"
	PlayerID string     `json:"playerId"`
	Item     ItemSchema `json:"item"`
}
func (s StateChangeAddItem) StateChangeType() string { return "add_item" }

type StateChangeRemoveItem struct {
	Type     string `json:"type"`     // "remove_item"
	PlayerID string `json:"playerId"`
	ItemID   string `json:"itemId"`
}
func (s StateChangeRemoveItem) StateChangeType() string { return "remove_item" }

type StateChangeTriggerGimmick struct {
	Type      string `json:"type"`      // "trigger_gimmick"
	GimmickID string `json:"gimmickId"`
}
func (s StateChangeTriggerGimmick) StateChangeType() string { return "trigger_gimmick" }

type StateChangeTriggerEvent struct {
	Type             string `json:"type"`             // "trigger_event"
	EventDescription string `json:"eventDescription"`
}
func (s StateChangeTriggerEvent) StateChangeType() string { return "trigger_event" }

type StateChangeUpdateNPCTrust struct {
	Type  string  `json:"type"`  // "update_npc_trust"
	NPCID string  `json:"npcId"`
	Delta float64 `json:"delta"` // 신뢰도 변화량 (-1.0 ~ 1.0)
}
func (s StateChangeUpdateNPCTrust) StateChangeType() string { return "update_npc_trust" }

type GameResponse struct {
	Events       []json.RawMessage `json:"events"`
	StateChanges []json.RawMessage `json:"stateChanges"`
}

func (r *GameResponse) ParsedEvents() ([]AIGameEvent, error) {
	events := make([]AIGameEvent, 0, len(r.Events))
	for _, raw := range r.Events {
		e, err := ParseAIGameEvent(raw)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}
```

## NPCResponse Schema

NPC 대화 AI 응답 스키마.

```go
// internal/shared/schemas/npc_response.go
package schemas

import "fmt"

type NPCResponse struct {
	Dialogue         string            `json:"dialogue"`
	Emotion          string            `json:"emotion"`          // events.md NPCDialogueData.Emotion과 매핑
	InternalThought  string            `json:"internalThought"` // 디버깅용, 플레이어에게 비공개
	InfoRevealed     []string          `json:"infoRevealed"`
	TrustChange      float64           `json:"trustChange"`      // min(-1), max(1)
	TriggeredGimmick bool              `json:"triggeredGimmick"`
	Events           []json.RawMessage `json:"events"`
}

func (n *NPCResponse) Validate() error {
	if n.TrustChange < -1 {
		return fmt.Errorf("trustChange: minimum -1 required")
	}
	if n.TrustChange > 1 {
		return fmt.Errorf("trustChange: maximum 1 allowed")
	}
	return nil
}

func (n *NPCResponse) ParsedEvents() ([]AIGameEvent, error) {
	events := make([]AIGameEvent, 0, len(n.Events))
	for _, raw := range n.Events {
		e, err := ParseAIGameEvent(raw)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}
```

## Ending Schema

엔딩 생성 AI 응답 스키마.

```go
// internal/shared/schemas/ending.go
package schemas

type GoalResultSchema struct {
	GoalID      string `json:"goalId"`
	Description string `json:"description"`
	Achieved    bool   `json:"achieved"`
	Evaluation  string `json:"evaluation"`
}

type PlayerEndingSchema struct {
	PlayerID    string             `json:"playerId"`
	Summary     string             `json:"summary"`
	GoalResults []GoalResultSchema `json:"goalResults"`
	Narrative   string             `json:"narrative"`
}

type Ending struct {
	CommonResult  string               `json:"commonResult"`
	PlayerEndings []PlayerEndingSchema `json:"playerEndings"`
	// SecretReveal은 여기에 포함하지 않음.
	// AI가 아닌 규칙 기반으로 서버에서 직접 구성 (EndingGenerator 참조).
	// - PlayerSecrets: 각 PlayerRole.Secret
	// - SemiPublicReveal: world.Information.SemiPublic
	// - UndiscoveredClues: IsDiscovered == false인 단서
	// - NPCSecrets: 각 NPC.HiddenInfo
}
```
