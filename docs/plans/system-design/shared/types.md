# Domain Types (`internal/shared/`)

PRD Section 7의 데이터 모델을 Go 타입으로 정의.

---

## Game (`internal/shared/game.go`)

게임 세션의 최상위 엔티티.

```go
// internal/shared/game.go

type GameStatus string

const (
	GameStatusLobby      GameStatus = "lobby"
	GameStatusGenerating GameStatus = "generating"
	GameStatusBriefing   GameStatus = "briefing"
	GameStatusPlaying    GameStatus = "playing"
	GameStatusEnding     GameStatus = "ending"
	GameStatusFinished   GameStatus = "finished"
)

// PRD 7.1의 feedback 필드는 P1(FR-071) 구현 시 추가 예정
type Game struct {
	ID        string             `json:"id"`
	RoomCode  string             `json:"roomCode"`
	HostID    string             `json:"hostId"`
	Status    GameStatus         `json:"status"`
	Settings  GameSettings       `json:"settings"`
	World     *World             `json:"world"`      // generating 이후 채워짐
	Players   map[string]*Player `json:"players"`
	EventLog  []GameEvent        `json:"eventLog"`
	CreatedAt int64              `json:"createdAt"`
	StartedAt *int64             `json:"startedAt"`
	EndedAt   *int64             `json:"endedAt"`
}

type GameSettings struct {
	MaxPlayers     int  `json:"maxPlayers"`     // 2~8, 기본 8
	TimeoutMinutes int  `json:"timeoutMinutes"` // AI가 결정, 범위 10~30, 기본 20 (concept: "10~30분 내 완결, 최대 30분")
	HasGM          bool `json:"hasGM"`          // AI가 결정
	HasNPC         bool `json:"hasNPC"`         // AI가 결정
}
```

## World (`internal/shared/world.go`)

AI가 생성하는 세계 전체. 게임 시작 후 불변.

```go
// internal/shared/world.go

// PRD 7.1의 timeline 필드는 미사용. 시간선 정보는 synopsis에 통합.
type World struct {
	Title         string            `json:"title"`
	Synopsis      string            `json:"synopsis"`
	Atmosphere    string            `json:"atmosphere"`
	GameStructure GameStructure     `json:"gameStructure"`
	Map           GameMap           `json:"map"`
	PlayerRoles   []PlayerRole      `json:"playerRoles"` // 플레이어 수만큼
	NPCs          []NPC             `json:"npcs"`
	Clues         []Clue            `json:"clues"`
	Gimmicks      []Gimmick         `json:"gimmicks"`
	Information   InformationLayers `json:"information"`
}

type GameStructure struct {
	Concept           string           `json:"concept"`           // "6인의 우주 비행사 중 배신자 색출"
	CoreConflict      string           `json:"coreConflict"`
	ProgressionStyle  string           `json:"progressionStyle"`
	CommonGoal        *string          `json:"commonGoal"`
	EstimatedDuration int              `json:"estimatedDuration"` // 분 단위 (10~30). AI 출력의 Meta.EstimatedDuration에서 복사됨. 원본은 WorldGenerationMeta에 위치.
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
	TriggerType     string                 `json:"triggerType"` // "vote" | "consensus" | "event" | "ai_judgment" | "timeout"
	TriggerCriteria map[string]interface{} `json:"triggerCriteria"`
	IsFallback      bool                   `json:"isFallback"`
}

type WinCondition struct {
	Description        string `json:"description"`
	EvaluationCriteria string `json:"evaluationCriteria"`
}
```

## Map (`internal/shared/map.go`)

방들의 네트워크. 무방향 그래프.

```go
// internal/shared/map.go

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
	RoomA         string `json:"roomA"`         // Room ID
	RoomB         string `json:"roomB"`         // Room ID
	Bidirectional bool   `json:"bidirectional"` // 양방향 여부 (PRD: 기본 true)
}
```

## Player (`internal/shared/player.go`)

플레이어 상태. 런타임에 변경됨.

```go
// internal/shared/player.go

type Player struct {
	ID                string       `json:"id"`
	Nickname          string       `json:"nickname"`
	IsHost            bool         `json:"isHost"`
	Status            string       `json:"status"` // "connected" | "disconnected" | "inactive"
	CurrentRoomID     string       `json:"currentRoomId"`
	Role              *PlayerRole  `json:"role"`              // briefing 이후 채워짐
	Inventory         []Item       `json:"inventory"`
	DiscoveredClueIDs []string     `json:"discoveredClueIds"`
	MoveHistory       []MoveRecord `json:"moveHistory"`       // 이동 이력 (FR-087)
	ConnectedAt       int64        `json:"connectedAt"`       // 최초 접속 시각
}

type MoveRecord struct {
	RoomID    string `json:"roomId"`
	RoomName  string `json:"roomName"`
	EnteredAt int64  `json:"enteredAt"`
	LeftAt    *int64 `json:"leftAt,omitempty"`
}

type PlayerRole struct {
	ID            string         `json:"id"`            // 플레이어 역할 식별자 (예: "role-001"). StoryValidator에서 참조 무결성 검증에 사용
	CharacterName string         `json:"characterName"`
	Background    string         `json:"background"`
	PersonalGoals []PersonalGoal `json:"personalGoals"`
	Secret        string         `json:"secret"`
	SpecialRole   *string        `json:"specialRole"` // "culprit", "traitor", "guardian" 등
	Relationships []Relationship `json:"relationships"`
}

type PersonalGoal struct {
	ID             string   `json:"id"`
	Description    string   `json:"description"`
	EvaluationHint string   `json:"evaluationHint"` // AI가 목표 달성 여부를 판단할 때 참고할 힌트 (schemas.md PersonalGoalSchema와 일치)
	EntityRefs     []string `json:"entityRefs"`     // 이 목표가 참조하는 엔티티 ID 목록 (NPC, 방, 단서 등). StoryValidator가 참조 무결성 검증에 사용
	IsAchieved     *bool    `json:"isAchieved"`     // 엔딩에서 판정
}

type Relationship struct {
	TargetCharacterName string `json:"targetCharacterName"`
	Description         string `json:"description"`
}
```

## NPC (`internal/shared/npc.go`)

AI가 운영하는 비플레이어 캐릭터. World에 포함되는 **불변** 템플릿.
런타임 상태(신뢰도, 대화 이력, 기믹 발동 여부)는 `GameState.npcStates`에서 관리. → game-state-manager.md 참조.

```go
// internal/shared/npc.go
// PRD 7.1과의 차이: NPC 런타임 상태(conversationHistory, trustLevel)는 GameState.NPCStates에서 별도 관리 (불변 템플릿 분리 원칙)

type NPC struct {
	ID                string      `json:"id"`
	Name              string      `json:"name"`
	CurrentRoomID     string      `json:"currentRoomId"`
	Persona           string      `json:"persona"`           // 성격, 말투, 태도
	KnownInfo         []string    `json:"knownInfo"`
	HiddenInfo        []string    `json:"hiddenInfo"`
	BehaviorPrinciple string      `json:"behaviorPrinciple"` // "직접 묻지 않으면 먼저 말하지 않는다"
	Gimmick           *NPCGimmick `json:"gimmick"`
	InitialTrust      float64     `json:"initialTrust"` // 초기 신뢰도 (0~1, 기본 0.5)
}

type NPCGimmick struct {
	Description      string `json:"description"`
	TriggerCondition string `json:"triggerCondition"`
	Effect           string `json:"effect"`
	// IsTriggered는 런타임 상태 → GameState.npcStates에서 추적
}
```

## Item & Clue (`internal/shared/items.go`)

```go
// internal/shared/items.go

type Item struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	OwnerID     *string `json:"ownerId"` // Player/NPC ID
	IsKey       bool    `json:"isKey"`
}

// World에 포함되는 단서 **불변** 템플릿.
// 발견 여부 등 런타임 상태는 GameState.clueStates에서 관리. → game-state-manager.md 참조.
type Clue struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	RoomID           string   `json:"roomId"`           // 배치된 방
	DiscoverCondition string  `json:"discoverCondition"`
	RelatedClueIDs   []string `json:"relatedClueIds"`
	// IsDiscovered, DiscoveredBy는 런타임 상태 → GameState.clueStates에서 추적
}

// World에 포함되는 기믹 **불변** 템플릿.
// 발동 여부는 런타임에 GameState에서 추적.
type Gimmick struct {
	ID               string `json:"id"`
	Description      string `json:"description"`
	RoomID           string `json:"roomId"`
	TriggerCondition string `json:"triggerCondition"`
	Effect           string `json:"effect"`
	// IsTriggered는 런타임 상태 → GameState에서 추적
}
```

## Information Layers (`internal/shared/information.go`)

정보 비대칭의 핵심 구조. 공개/반공개/비공개 3계층.

```go
// internal/shared/information.go

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
	Title         string               `json:"title"`     // World.title (게임 제목)
	Synopsis      string               `json:"synopsis"`
	CharacterList []CharacterListEntry `json:"characterList"`
	Relationships string               `json:"relationships"`
	MapOverview   string               `json:"mapOverview"`
	NPCList       []NPCListEntry       `json:"npcList"`
	GameRules     string               `json:"gameRules"`
}

type SemiPublicInfo struct {
	ID              string   `json:"id"`
	TargetPlayerIDs []string `json:"targetPlayerIds"` // 이 정보를 받는 플레이어들
	Content         string   `json:"content"`
}

type PrivateInfo struct {
	PlayerID          string   `json:"playerId"`
	AdditionalSecrets []string `json:"additionalSecrets"`
	// Role은 여기에 포함하지 않음.
	// PlayerRole은 World.playerRoles에서 관리되며, briefing 시 개별 전송.
}
```

## Game Ending (`internal/shared/ending.go`)

게임 종료 시 생성되는 결과 데이터. EndingGenerator(AI)와 서버(규칙 기반)가 협력하여 구성.

```go
// internal/shared/ending.go

type GameEndData struct {
	CommonResult  string         `json:"commonResult"`  // AI가 생성 — 전체 게임 결과 서술
	PlayerEndings []PlayerEnding `json:"playerEndings"` // AI가 생성 — 개인별 엔딩
	SecretReveal  SecretReveal   `json:"secretReveal"`  // 서버가 규칙 기반으로 구성
}

type PlayerEnding struct {
	PlayerID    string       `json:"playerId"`
	Summary     string       `json:"summary"`     // 이 플레이어의 행동 요약
	GoalResults []GoalResult `json:"goalResults"` // 개인 목표 달성 여부
	Narrative   string       `json:"narrative"`   // 개인화된 엔딩 서술
}

type GoalResult struct {
	GoalID      string `json:"goalId"`
	Description string `json:"description"`
	Achieved    bool   `json:"achieved"`
	Evaluation  string `json:"evaluation"` // AI의 판정 근거
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
	Condition   string `json:"condition"`   // 트리거 조건
}

type SecretReveal struct {
	PlayerSecrets        []PlayerSecretEntry     `json:"playerSecrets"`
	SemiPublicReveal     []SemiPublicRevealEntry `json:"semiPublicReveal"`
	UndiscoveredClues    []UndiscoveredClueEntry `json:"undiscoveredClues"`
	NPCSecrets           []NPCSecretEntry        `json:"npcSecrets"`
	UntriggeredGimmicks  []GimmickReveal         `json:"untriggeredGimmicks"` // 트리거되지 않은 기믹 목록
}
```

## Game State (`internal/shared/game_state.go`)

GameContext.CurrentState에서 참조하는 현재 게임 런타임 상태. GameStateManager가 관리.

```go
// internal/shared/game_state.go

type GameState struct {
	Players    map[string]*Player   `json:"players"`
	ClueStates map[string]ClueState `json:"clueStates"` // clueID → 발견 여부 등
	NPCStates  map[string]NPCState  `json:"npcStates"`  // npcID → 신뢰도, 대화 이력, 기믹 발동 등
	GimmickStates map[string]GimmickState `json:"gimmickStates"` // gimmickID → 발동 여부
	ElapsedTime int64                `json:"elapsedTime"` // 게임 시작 후 경과 시간 (초)
}

type ClueState struct {
	IsDiscovered bool     `json:"isDiscovered"`
	DiscoveredBy []string `json:"discoveredBy"` // 발견한 플레이어 ID 목록
}

type NPCState struct {
	TrustLevels         map[string]float64    `json:"trustLevels"`         // playerID → 신뢰도
	ConversationHistory []ConversationRecord  `json:"conversationHistory"` // 대화 이력 (최대 20턴)
	GimmickTriggered    bool                  `json:"gimmickTriggered"`
}

type ConversationRecord struct {
	PlayerID  string `json:"playerId"`
	Message   string `json:"message"`
	Response  string `json:"response"`
	Timestamp int64  `json:"timestamp"`
}

type GimmickState struct {
	IsTriggered bool  `json:"isTriggered"`
	TriggeredAt *int64 `json:"triggeredAt,omitempty"`
}
```

## Game Context (`internal/shared/context.go`)

AI 모듈에 전달하는 게임 컨텍스트. ActionProcessor, ActionEvaluator, NPCEngine, GMEngine, EndConditionEngine 등 여러 모듈에서 공유.

```go
// internal/shared/context.go

type GameContext struct {
	World            World       `json:"world"`            // 세계 설정 전체
	CurrentState     GameState   `json:"currentState"`     // 현재 게임 상태
	RecentEvents     []GameEvent `json:"recentEvents"`     // 최근 이벤트 (최대 20개)
	ActionLog        []GameEvent `json:"actionLog"`        // 게임 시작부터 현재까지의 전체 행동 로그. 엔딩 생성 시 사용.
	RequestingPlayer Player      `json:"requestingPlayer"` // 요청한 플레이어
	CurrentRoom      Room        `json:"currentRoom"`      // 요청 플레이어가 있는 방
	PlayersInRoom    []Player    `json:"playersInRoom"`    // 같은 방 플레이어들
}
```

## Common Utilities (`internal/shared/result.go`)

```go
// internal/shared/result.go

type Result[T any, E any] struct {
	Ok    bool `json:"ok"`
	Value T    `json:"value,omitempty"`
	Error E    `json:"error,omitempty"`
}
```

## Feedback (`internal/shared/feedback.go`)

게임 종료 후 플레이어가 제출하는 피드백. FR-071.

```go
// internal/shared/feedback.go

type Feedback struct {
	PlayerID        string  `json:"playerId"`
	FunRating       int     `json:"funRating"`       // 1~5 (스토리 재미도)
	ImmersionRating int     `json:"immersionRating"` // 1~5 (몰입도)
	Comment         *string `json:"comment"`         // 자유 텍스트 (선택)
	SubmittedAt     int64   `json:"submittedAt"`
}
```
