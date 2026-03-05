# MapEngine (`internal/server/map/`)

## 책임

맵 그래프 관리, 인접 검증, 이동 처리, ASCII 맵 생성.

## 의존하는 모듈

GameStateManager

## 인터페이스

```go
type MapEngine struct {
    adjacencyMap map[string]map[string]struct{}  // roomId → Set<인접 roomId>
    rooms        []*Room
    gameState    *game.GameStateManager
}

func NewMapEngine(gs *game.GameStateManager) *MapEngine

// ── 초기화 (세계 생성 후) ──
func (me *MapEngine) Initialize(gameMap GameMap)

// ── 이동 검증 및 실행 ──
func (me *MapEngine) MovePlayer(playerId string, targetRoomId string) (*MoveResult, error)

// ── 조회 ──
func (me *MapEngine) GetAdjacentRooms(roomId string) []*Room
func (me *MapEngine) IsAdjacent(roomA string, roomB string) bool
func (me *MapEngine) GetRoomByName(name string) *Room  // nil이면 찾지 못함

// ── 맵 렌더링 ──
// GenerateAsciiMap: highlightRoomId에 해당하는 방을 강조 표시한 ASCII 맵 문자열 반환.
// 출력 형식: 방 이름을 대괄호로 표시 ([방이름]), 연결은 '--'로 표현.
// 현재 플레이어 위치의 방은 '[*방이름*]'으로 강조. GameStateManager에서 플레이어 위치 정보를 조회하여 반영.
func (me *MapEngine) GenerateAsciiMap(highlightRoomId string) string

// MoveError
var (
    ErrNotAdjacent   = errors.New("not_adjacent")
    ErrRoomNotFound  = errors.New("room_not_found")
    ErrGameNotPlaying = errors.New("game_not_playing")
)

type MoveResult struct {
    FromRoom       *Room
    ToRoom         *Room
    PlayersInNewRoom []*Player
    NPCsInNewRoom  []*NPC
}
```

## 내부 자료구조

맵은 **무방향 그래프**. `[]Connection`으로부터 adjacency map을 빌드.

```go
func (me *MapEngine) Initialize(gameMap GameMap) {
    me.rooms = gameMap.Rooms
    me.adjacencyMap = make(map[string]map[string]struct{})

    for _, room := range gameMap.Rooms {
        me.adjacencyMap[room.ID] = make(map[string]struct{})
    }

    for _, conn := range gameMap.Connections {
        me.adjacencyMap[conn.RoomA][conn.RoomB] = struct{}{}
        if conn.Bidirectional {
            me.adjacencyMap[conn.RoomB][conn.RoomA] = struct{}{}
        }
    }
}
```

## 이동 흐름

```
MovePlayer(playerId, targetRoomId)
    │
    ├── 현재 방 조회
    ├── targetRoomId가 유효한 방인지 확인
    ├── 현재 방 ↔ 대상 방 인접 여부 확인
    │     └── 실패 → (nil, ErrNotAdjacent)
    │
    ├── gameState.MovePlayer(playerId, targetRoomId)
    │
    └── (&MoveResult{
          FromRoom, ToRoom,
          PlayersInNewRoom,
          NPCsInNewRoom,
        }, nil)
```

## 방 이름으로 이동

플레이어는 방 ID가 아니라 이름으로 이동 (`/move 부엌`). `GetRoomByName()`으로 변환.

```go
func (me *MapEngine) GetRoomByName(name string) *Room {
    for _, r := range me.rooms {
        if r.Name == name || strings.Contains(r.Name, name) {
            return r
        }
    }
    return nil
}
```

부분 일치도 허용하여 사용성 향상 (`/move 부` → `부엌`).

> **제약:** 부분 일치는 첫 번째로 매칭되는 방을 반환한다. 동일한 부분 문자열을 포함하는 방이 여러 개 있을 경우 예측 불가능한 결과가 발생할 수 있다. 정확히 일치하는 방 이름이 있으면 항상 우선적으로 반환한다.

## 맵 연결성 검증

세계 생성 후 `Initialize()` 시 BFS로 모든 방이 연결되어 있는지 검증.
Concept: "별도의 밀담 기능을 두지 않는다. 밀담이 필요하면 조용한 방으로 이동하면 된다." → 모든 방이 접근 가능해야 함.

```go
// ValidateConnectivity는 모든 방이 연결 그래프에서 도달 가능한지 BFS로 검증.
func (me *MapEngine) ValidateConnectivity() error {
    if len(me.rooms) == 0 {
        return errors.New("no rooms in map")
    }

    visited := make(map[string]bool)
    queue := []string{me.rooms[0].ID}
    visited[me.rooms[0].ID] = true

    for len(queue) > 0 {
        current := queue[0]
        queue = queue[1:]
        for neighbor := range me.adjacencyMap[current] {
            if !visited[neighbor] {
                visited[neighbor] = true
                queue = append(queue, neighbor)
            }
        }
    }

    if len(visited) != len(me.rooms) {
        return fmt.Errorf("isolated rooms detected: %d/%d reachable", len(visited), len(me.rooms))
    }
    return nil
}
```

## 방 수 최소 제약 (FR-010, FR-035)

PRD: "최소 방 수 = `플레이어 수 + 2`". 밀담용 비공개 방을 확보하기 위한 최소 요구.

> **concept.md와의 차이:** concept.md는 6인 기준 5~7개 방을 예시했으나, PRD FR-010/FR-035는 `playerCount + 2`로 공식화. 6인 게임 = 8개 방. PRD 기준을 따름.

```go
func (me *MapEngine) ValidateRoomCount(playerCount int) error {
    minRooms := playerCount + 2
    if len(me.rooms) < minRooms {
        return fmt.Errorf("insufficient rooms: got %d, need >= %d (playerCount+2)", len(me.rooms), minRooms)
    }
    return nil
}
```
