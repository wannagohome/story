# ActionProcessor (`internal/server/action/`)

## 책임

**게임 중** 플레이어 명령어를 해석하고, 필요 시 AI에 위임하여 결과를 생성, 상태에 반영.

> 세션 관련 메시지(`join`, `rejoin`, `start_game`, `ready`, `cancel_game`)는 SessionManager가 처리. ActionProcessor는 게임 진행 중 메시지만 담당.

## 의존하는 모듈

GameStateManager, MapEngine, AILayer (런타임 AI 호출용), EndConditionEngine, EventBus

> **참고:** ActionProcessor는 AILayer를 통해 런타임 AI(GM/NPC/행동평가/종료판정)만 사용한다. 세계 생성(Orchestrator 멀티 모델 파이프라인)은 SessionManager가 직접 호출하며, ActionProcessor는 관여하지 않는다.

## 인터페이스

```go
type ActionProcessor struct {
    gameState    *game.GameStateManager
    mapEngine    *mapengine.MapEngine
    aiLayer      *ai.AILayer
    endCondition *end.EndConditionEngine
    eventBus     *eventbus.EventBus
}

func NewActionProcessor(
    gs  *game.GameStateManager,
    me  *mapengine.MapEngine,
    ail *ai.AILayer,
    ece *end.EndConditionEngine,
    bus *eventbus.EventBus,
) *ActionProcessor

// 클라이언트 메시지 처리 (메인 디스패처)
func (ap *ActionProcessor) ProcessMessage(playerId string, message ClientMessage) error
```

## 명령어 분류

| 명령어 | 처리 유형 | AI 호출 | 상태 변이 |
|--------|----------|---------|-----------|
| `chat` | 로컬 | X | X |
| `shout` | 로컬 | X | X |
| `move` | 로컬 | X | O (위치 변경) |
| `examine` | AI 위임 | O | O (단서 발견 가능) |
| `do` | AI 위임 | O | O (이벤트 트리거 가능) |
| `talk` | AI 위임 | O | O (신뢰도 변화 가능) |
| `give` | 로컬 | X | O (아이템 이동) | // P1: MVP 이후 구현 |
| `vote` | → EndConditionEngine | X | O (투표 집계) |
| `solve` | → EndConditionEngine | X | O (합의안 제출) |
| `propose_end` | → EndConditionEngine | X | O (종료 투표 시작) |
| `end_vote` | → EndConditionEngine | X | O (종료 투표 응답) |
| `request_look` | 로컬 조회 | X | X (현재 방 RoomView 재전송) |
| `request_*` | 로컬 조회 | X | X |
| `submit_feedback` | 로컬 | X | O (피드백 저장, FR-071) |
| `skip_feedback` | 로컬 | X | X |

## 디스패치 로직

```go
func (ap *ActionProcessor) ProcessMessage(playerId string, message ClientMessage) error {
    switch message.Type {
    case "chat":            return ap.handleChat(playerId, message)
    case "shout":           return ap.handleShout(playerId, message)
    case "move":            return ap.handleMove(playerId, message)
    case "examine":         return ap.handleExamine(playerId, message)
    case "do":              return ap.handleDo(playerId, message)
    case "talk":            return ap.handleTalk(playerId, message)
    case "give":            return ap.handleGive(playerId, message)  // P1: MVP 이후 구현. MVP에서는 "이 기능은 아직 사용할 수 없습니다" 반환
    case "vote":            return ap.handleVote(playerId, message)
    case "solve":           return ap.handleSolve(playerId, message)
    case "propose_end":     return ap.handleProposeEnd(playerId)
    case "end_vote":        return ap.handleEndVote(playerId, message)
    case "request_look":      return ap.handleLook(playerId)
    case "request_inventory": return ap.handleInventory(playerId)
    case "request_role":      return ap.handleRole(playerId)
    case "request_map":       return ap.handleMap(playerId)
    case "request_who":       return ap.handleWho(playerId)
    case "request_help":      return ap.handleHelp(playerId)
    case "submit_feedback":   return ap.handleSubmitFeedback(playerId, message)  // FR-071
    case "skip_feedback":     return ap.handleSkipFeedback(playerId)             // FR-071
    default:
        return fmt.Errorf("unknown message type: %s", message.Type)
    }
}
```

## 명령어 처리 흐름

### 로컬 처리 (채팅, 이동)

```
Client → { type: 'chat', content }
    │
    ▼
handleChat()
    │
    ▼
eventBus.PublishChat(ChatData{SenderID, RoomID, Content, Scope: "room"})
    │
    ▼
MessageRouter가 같은 방 플레이어에게만 전달
```

### handleMove 흐름

```
Client → { type: 'move', target: '부엌' }
    │
    ▼
handleMove()
    │
    ├── mapEngine.GetRoomByName(target) → targetRoomId
    │     └── 실패(nil) → 에러 메시지 반환
    │
    ├── mapEngine.MovePlayer(playerId, targetRoomId)
    │     ├── 인접 검증 실패 → ErrNotAdjacent (에러 메시지 반환)
    │     └── 성공 → MoveResult{FromRoom, ToRoom, PlayersInNewRoom, NPCsInNewRoom}
    │
    ├── eventBus.PublishGameEvent(player_move, scope: "all")
    │     └── MessageRouter → 전체 플레이어에게 이동 사실 전달
    │
    ├── network.SendTo(playerId, room_changed + RoomView)
    │     └── 이동한 플레이어에게 새 방의 RoomView 전송
    │
    ├── eventBus.PublishGameEvent(system_message, scope: "room", roomId: FromRoom.ID)
    │     └── "[닉네임]이(가) 나갔습니다" → 출발 방 플레이어들에게 전달 (FR-040)
    │
    ├── eventBus.PublishGameEvent(system_message, scope: "room", roomId: ToRoom.ID)
    │     └── "[닉네임]이(가) 들어왔습니다" → 도착 방 플레이어들에게 전달 (FR-040)
    │
    └── messageRouter.BroadcastMapUpdate()
          └── 전체 플레이어에게 개인화된 map_info push (HeaderBar 실시간 갱신)
```

### AI 위임 (examine, do, talk)

모든 AI 호출에는 `context.WithTimeout(ctx, 5*time.Second)` 하드 타임아웃을 적용한다.

```
Client → { type: 'examine', target: '책상' }
    │
    ▼
handleExamine()
    │
    ├── 현재 방, 게임 컨텍스트 조회
    │
    ├── network.SendTo(playerId, { type: 'thinking' })  ← AI 처리 중 표시기 전송
    │
    ▼
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
aiLayer.EvaluateExamine(ctx, room, target)
    │
    ├── 타임아웃(5초) 초과 시:
    │     ├── slog.Warn("AI 응답 타임아웃", ...)
    │     └── network.SendTo(playerId, { type: 'error', message: '응답 시간이 초과되었습니다. 다시 시도해 주세요.' })
    │
    ▼
AI 응답 수신 (GameResponse: Events + StateChanges)
    │
    ├── 정보 누출 검사: AI 응답에 다른 플레이어의 개인 비밀이 포함되어 있는지 확인
    │     └── 누출 감지 시 해당 내용을 제거하고 slog.Warn("AI 응답 정보 누출 감지", ...)
    │
    ├── StateChanges 적용 → gameState.*
    │     (예: DiscoverClue)
    │
    └── Events 발행 → eventBus.PublishGameEvent(...)
          (예: examine_result, clue_found)
                │
                ▼
          MessageRouter가 같은 방 플레이어에게 전달
```

### 정보 조회 (inventory, role, map, who, help)

```
Client → { type: 'request_inventory' }
    │
    ▼
handleInventory()
    │
    ▼
GameStateManager에서 해당 플레이어의 인벤토리 조회
    │
    ▼
network.SendTo(playerId, ServerMessage{Type: "inventory", Items: items, Clues: clues})
(요청한 플레이어에게만 직접 응답)
```

## GameContext 구성

`GameContext` 타입은 `internal/types/context.go`에 정의. 여기서는 구성 로직만 기술.

```go
func (ap *ActionProcessor) buildGameContext(playerId string) GameContext {
    player := ap.gameState.GetPlayer(playerId)
    currentRoom := ap.gameState.GetPlayerRoom(playerId)
    return GameContext{
        World:           ap.gameState.GetWorld(),
        CurrentState:    ap.gameState.GetFullState(),
        RecentEvents:    ap.gameState.GetRecentEvents(20),
        RequestingPlayer: player,
        CurrentRoom:     currentRoom,
        PlayersInRoom:   ap.gameState.GetPlayersInRoom(currentRoom.ID),
    }
}
```

## AI 응답 후처리 원칙

1. **정보 누출 검사:** AI 응답을 EventBus에 발행하기 전, ActionProcessor는 응답 텍스트에 다른 플레이어의 개인 비밀(playerSecret), 미공개 역할 정보, 다른 방의 비공개 정보가 포함되어 있는지 확인한다. 누출이 감지되면 해당 내용을 제거하고 `slog.Warn`으로 기록한다.
2. **5초 하드 타임아웃:** 모든 AI 호출에 `context.WithTimeout(ctx, 5*time.Second)`를 적용. 타임아웃 시 플레이어에게 에러 메시지를 반환하고 경고를 로깅한다.
3. **thinking 표시기:** AI 호출 전 플레이어에게 `{ type: 'thinking' }` 메시지를 전송하여 처리 중임을 알린다.
