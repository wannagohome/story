# GameStateManager (`internal/server/game/`)

## 책임

게임 상태의 **유일한 소유자**. 상태 변이(mutation)와 플레이어별 필터링된 뷰 생성.

## 의존하는 모듈

EventBus

## 핵심 원칙

- **직접 네트워크 메시지를 보내지 않는다.** 상태를 변경하고 EventBus에 이벤트를 발행할 뿐.
- **모든 상태 변이는 이벤트를 발행한다.** MessageRouter가 이벤트를 구독해서 적절한 플레이어에게 전달.
- **뷰 생성 시 가시성 필터링을 적용한다.** 클라이언트에게 보이면 안 되는 정보를 제거.

## 인터페이스

```go
type GameStateManager struct {
    state    GameState
    eventBus *eventbus.EventBus
    mu       sync.RWMutex
}

func NewGameStateManager(bus *eventbus.EventBus) *GameStateManager

// ── 초기화 ──
func (gsm *GameStateManager) InitializeWorld(world World, roleAssignments map[string]PlayerRole)

// ── 상태 변이 (모두 이벤트를 발행함) ──
func (gsm *GameStateManager) MovePlayer(playerId string, targetRoomId string)
func (gsm *GameStateManager) AddItemToPlayer(playerId string, item Item)
func (gsm *GameStateManager) RemoveItemFromPlayer(playerId string, itemId string)
func (gsm *GameStateManager) DiscoverClue(playerId string, clueId string)
func (gsm *GameStateManager) UpdateNPCTrust(npcId string, playerId string, delta float64)
func (gsm *GameStateManager) TriggerGimmick(gimmickId string)
// P1: MoveNPC(npcId, targetRoomId) will be added for FR-058. NPC location is immutable in MVP.

// ── 개인 목표 진행 추적 (FR-092) ──
func (gsm *GameStateManager) RecordGoalProgress(playerId string, goalId string, evidence string)
func (gsm *GameStateManager) GetGoalProgress(playerId string) []GoalProgressEntry

// ── 뷰 생성 (가시성 필터링) ──
func (gsm *GameStateManager) GetPlayerView(playerId string) PlayerView
func (gsm *GameStateManager) GetMapView(playerId string) MapView    // myRoomId 설정을 위해 playerId 필요
func (gsm *GameStateManager) GetRoomView(playerId string) RoomView

// ── 브리핑 정보 조회 ──
// 반공개 정보: AI가 지정한 플레이어 그룹에게만 공개되는 정보 (예: 같은 진영 공유 비밀)
func (gsm *GameStateManager) GetSemiPublicInfoForPlayer(playerId string) []SemiPublicInfo

// ── 조회 ──
func (gsm *GameStateManager) GetPlayer(playerId string) *Player
func (gsm *GameStateManager) GetAllPlayerIDs() []string
func (gsm *GameStateManager) GetPlayersInRoom(roomId string) []*Player
func (gsm *GameStateManager) GetNPCsInRoom(roomId string) []*NPC
func (gsm *GameStateManager) GetAdjacentRooms(roomId string) []*Room  // MapEngine에 위임. GameStateManager를 통해 접근 시 MapEngine.GetAdjacentRooms()를 호출한다.
func (gsm *GameStateManager) GetPlayerRoom(playerId string) *Room
func (gsm *GameStateManager) GetWorld() World
func (gsm *GameStateManager) GetRecentEvents(count int) []GameEvent
func (gsm *GameStateManager) GetFullState() GameState  // AI 전용 — 필터링 없는 전체 상태

// ── NPC 대화 기록 ──
// ActionProcessor가 /talk 처리 후 대화 기록을 저장. NPCEngine이 참조.
// NPCState.ConversationHistory에 추가하며, 20턴 초과 시 오래된 기록부터 삭제.
func (gsm *GameStateManager) AddConversation(npcId string, record ConversationRecord)
```

## 뷰 타입 (가시성 필터링 적용)

```go
// 플레이어에게 전달되는 필터링된 뷰
type PlayerView struct {
    MyPlayer    Player
    CurrentRoom RoomView
    MapOverview MapView
}

// RoomView, MapView 및 하위 타입은 protocol.md의 정의를 사용한다.
type RoomView struct {
    ID          string
    Name        string
    Description string
    Type        string           // "public" | "private"
    Items       []RoomViewItem   // 이름만 노출 (설명은 examine으로)
    Players     []RoomViewPlayer
    NPCs        []RoomViewNPC
    // clueIds, clue 배치 정보는 절대 미포함 (examine으로만 발견)
}

type RoomViewItem struct {
    ID   string
    Name string
}

type RoomViewPlayer struct {
    ID       string
    Nickname string
}

type RoomViewNPC struct {
    ID   string
    Name string
}

type MapView struct {
    Rooms       []MapViewRoom
    Connections []Connection
    MyRoomID    string
}

type MapViewRoom struct {
    ID          string
    Name        string
    Type        string  // "public" | "private"
    PlayerCount int
    PlayerNames []string
}
```

## 가시성 필터링 규칙

PRD 7.3 기반. 서버에서 필터링하여 클라이언트에는 볼 수 있는 정보만 전송.

| 데이터 | 가시성 규칙 |
|--------|-------------|
| 맵 구조 (방 이름, 연결) | 전체 공개 |
| 각 방 플레이어 위치 | 전체 공개 |
| 각 방 NPC 위치 | 전체 공개 |
| 방 상세 설명 | 해당 방에 있는 플레이어만 |
| 같은 방 채팅 | 같은 방 플레이어만 |
| NPC 대화 내용 | 같은 방 플레이어만 |
| /examine 결과 | 같은 방 플레이어만 |
| /do 결과 | 같은 방 플레이어만 |
| 플레이어 역할 (본인) | 본인만 |
| 플레이어 역할 (타인) | 비공개 (종료 후 공개) |
| 개인 목표 | 본인만 |
| 인벤토리 | 본인만 |
| 이동 사실 | 전체 공개 |
| 반공개 정보 (브리핑) | AI가 지정한 플레이어 그룹만 (예: 같은 진영 공유 비밀) |

## 브리핑 단계 처리 흐름

`generating → briefing` 전이 후 SessionManager가 아래 순서로 GameStateManager를 호출하여 정보를 전달.

```
OnWorldGenerated() 호출 (SessionManager)
    │
    ├── 1. 전체 공개 정보 브로드캐스트
    │       GameStateManager가 world/map 정보를 EventBus에 발행
    │       → MessageRouter가 모든 플레이어에게 전달 (scope: "all")
    │
    ├── 2. 개인 역할/비밀/비공개 정보 개별 전송
    │       각 플레이어에 대해 GetPlayerView(playerId) 조회
    │       → network.SendTo(playerId, role/inventory/privateInfo)
    │
    ├── 3. 반공개 정보를 해당 그룹에 전송
    │       각 플레이어에 대해 GetSemiPublicInfoForPlayer(playerId) 조회
    │       → 해당 그룹 플레이어들에게만 전달 (scope: "players")
    │
    └── 4. 모든 플레이어의 ready 수신 대기
            전원 ready → OnAllPlayersReady() → briefing → playing
```

## 내부 상태 구조

```go
// GameStateManager는 types.md의 공유 GameState를 내장하고,
// 추가로 World, EventLog를 별도 필드로 보유한다.
// GameState(Players, ClueStates, NPCStates, GimmickStates, ElapsedTime)는
// types.md의 정의를 그대로 사용한다.
type GameStateManager struct {
    // ... (기존 필드)
    state    GameState    // types.md GameState (Players, ClueStates, NPCStates, GimmickStates, ElapsedTime)
    world    World        // 세계 데이터 (별도 보유, types.md GameState에 미포함)
    eventLog []GameEvent  // 이벤트 로그 (별도 보유, types.md GameState에 미포함)
}

// ClueState, NPCState, GimmickState, ConversationRecord는 types.md의 정의를 사용한다:
// type ClueState struct {
//     IsDiscovered bool     `json:"isDiscovered"`
//     DiscoveredBy []string `json:"discoveredBy"` // 발견한 플레이어 ID 목록
// }
// type NPCState struct {
//     TrustLevels         map[string]float64    `json:"trustLevels"`
//     ConversationHistory []ConversationRecord  `json:"conversationHistory"`
//     GimmickTriggered    bool                  `json:"gimmickTriggered"`
// }
// type GimmickState struct {
//     IsTriggered bool   `json:"isTriggered"`
//     TriggeredAt *int64 `json:"triggeredAt,omitempty"`
// }
// type ConversationRecord struct {
//     PlayerID  string `json:"playerId"`
//     Message   string `json:"message"`
//     Response  string `json:"response"`
//     Timestamp int64  `json:"timestamp"`
// }

// 개인 목표 진행 상태 (FR-092). 게임 중 플레이어 행동에서 목표 관련 증거를 축적.
// 엔딩 시 AI가 이 데이터를 참조하여 달성 여부를 판정.
type GoalProgressEntry struct {
    GoalID    string
    Evidence  []string  // 목표와 관련된 행동/이벤트 기록
}
```
