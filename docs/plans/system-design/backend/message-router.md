# MessageRouter (`internal/server/message/`)

## 책임

이벤트의 **가시성(visibility)**에 따라 올바른 플레이어에게만 메시지를 전달. 정보 비대칭 시스템의 핵심 인프라.

## 의존하는 모듈

NetworkServer, GameStateManager, EventBus

## 인터페이스

```go
type MessageRouter struct {
    network   *network.NetworkServer
    gameState *game.GameStateManager
    eventBus  *eventbus.EventBus
}

func NewMessageRouter(
    net *network.NetworkServer,
    gs  *game.GameStateManager,
    bus *eventbus.EventBus,
) *MessageRouter {
    mr := &MessageRouter{
        network:   net,
        gameState: gs,
        eventBus:  bus,
    }
    // EventBus 채널을 구독: 별도 goroutine에서 수신
    go mr.listenGameEvents(bus.SubscribeGameEvent())
    go mr.listenChat(bus.SubscribeChat())
    go mr.listenSendEndings(bus.SubscribeSendEndings())
    go mr.listenFeedback(bus.SubscribeFeedback())
    return mr
}

// ── 이벤트 수신 루프 ──
func (mr *MessageRouter) listenGameEvents(ch <-chan GameEvent)
func (mr *MessageRouter) listenChat(ch <-chan ChatData)
func (mr *MessageRouter) listenSendEndings(ch <-chan GameEndData)
func (mr *MessageRouter) listenFeedback(ch <-chan Feedback)  // FR-071: 피드백 수집/스킵 처리. types.md Feedback 타입 사용.

// ── 이벤트 라우팅 ──
func (mr *MessageRouter) routeEvent(event GameEvent)

// ── 채팅 라우팅 ──
func (mr *MessageRouter) routeChat(chat ChatData)

// ── 엔딩 라우팅 (per-player) ──
func (mr *MessageRouter) routeEndings(endData GameEndData)

// ── 맵 갱신 브로드캐스트 ──
// 플레이어 이동 시 전체 플레이어에게 map_info push (HeaderBar 실시간 갱신용)
func (mr *MessageRouter) BroadcastMapUpdate()

// ── 가시성 → 수신자 목록 변환 ──
func (mr *MessageRouter) resolveRecipients(visibility EventVisibility) []string
```

## 라우팅 로직

### 이벤트 라우팅

```go
func (mr *MessageRouter) routeEvent(event GameEvent) {
    recipients := mr.resolveRecipients(event.GetBaseEvent().Visibility)
    message := ServerMessage{Type: "game_event", Event: event}
    mr.network.SendToMany(recipients, message)
}

func (mr *MessageRouter) resolveRecipients(visibility EventVisibility) []string {
    switch visibility.Scope {
    case "all":
        return mr.gameState.GetAllPlayerIDs()
    case "room":
        players := mr.gameState.GetPlayersInRoom(visibility.RoomID)
        ids := make([]string, len(players))
        for i, p := range players {
            ids[i] = p.ID
        }
        return ids
    case "players":
        return visibility.PlayerIDs
    default:
        return nil
    }
}
```

### 채팅 라우팅

```go
func (mr *MessageRouter) routeChat(chat ChatData) {
    // ChatData → protocol.md ChatServerMessage 변환
    var senderLocation *string
    if chat.Scope == "global" {
        // 글로벌 채팅 시 발신자 위치 포함 (FR-037 AC3)
        name := mr.gameState.GetPlayerRoom(chat.SenderID).Name
        senderLocation = &name
    }
    msg := ServerMessage{
        Type:           "chat_message",
        SenderID:       chat.SenderID,
        SenderName:     chat.SenderName,
        Content:        chat.Content,
        Scope:          chat.Scope,
        SenderLocation: senderLocation,
        Timestamp:      time.Now().UnixMilli(),
    }
    if chat.Scope == "global" {
        mr.network.SendToAll(msg)
    } else {
        roomPlayers := mr.gameState.GetPlayersInRoom(chat.RoomID)
        recipientIds := make([]string, len(roomPlayers))
        for i, p := range roomPlayers {
            recipientIds[i] = p.ID
        }
        mr.network.SendToMany(recipientIds, msg)
    }
}
```

### 맵 갱신 브로드캐스트

플레이어 이동 시 ActionProcessor가 호출. 모든 플레이어에게 개인화된 `map_info`를 전송.
(MapView.MyRoomID가 각 플레이어마다 다르므로 개별 전송 필요)

```go
func (mr *MessageRouter) BroadcastMapUpdate() {
    for _, playerId := range mr.gameState.GetAllPlayerIDs() {
        mapView := mr.gameState.GetMapView(playerId)
        mr.network.SendTo(playerId, ServerMessage{Type: "map_info", Map: mapView})
    }
}
```

### 엔딩 라우팅 (per-player)

각 플레이어에게 개인화된 엔딩을 전달. `PersonalEnding`이 플레이어마다 다르므로 개별 전송 필수.

```go
func (mr *MessageRouter) routeEndings(endData GameEndData) {
    for _, playerId := range mr.gameState.GetAllPlayerIDs() {
        var personalEnding PlayerEnding
        for _, pe := range endData.PlayerEndings {
            if pe.PlayerID == playerId {
                personalEnding = pe
                break
            }
        }
        // protocol.md GameEndingMessage: PersonalEnding은 PlayerEnding struct
        mr.network.SendTo(playerId, ServerMessage{
            Type:           "game_ending",
            CommonResult:   endData.CommonResult,
            PersonalEnding: personalEnding,
            SecretReveal:   endData.SecretReveal,
        })
    }
}
```

## 라우팅 규칙 매트릭스

| 데이터 유형 | scope | 대상 결정 방법 |
|------------|-------|---------------|
| 같은 방 채팅 | `room` | 발신자의 currentRoomId |
| 글로벌 채팅 | `all` | 발신자 위치(방 이름) 포함 (FR-037 AC3) |
| /examine 결과 | `room` | 실행자의 currentRoomId |
| /do 결과 | `room` | 실행자의 currentRoomId |
| NPC 대화 | `room` | NPC의 currentRoomId |
| GM 서술 | `all` 또는 `room` | AI가 지정 |
| 플레이어 이동 | `all` | - |
| 맵 갱신 (이동 후) | `all` (개별) | 전체 (MyRoomID가 다르므로 개별 전송) |
| 투표 시작/결과 | `all` | - |
| 엔딩 (game_ending) | `all` (개별) | 전체 (PersonalEnding이 다르므로 개별 전송) |
| 역할/인벤토리 조회 | `players` | 요청한 플레이어만 |
| 반공개 정보 (briefing) | `players` | AI가 지정한 플레이어 그룹 |
| 피드백 요청 (feedback_request) | `all` (개별) | 엔딩 후 전체 (FR-071). EndConditionEngine이 엔딩 전달 완료 후 발행 |
| 피드백 응답 (feedback_ack) | `players` | 제출/스킵한 플레이어에게만 |

## 핵심 보안 원칙

**클라이언트를 신뢰하지 않는다.** 모든 가시성 필터링은 이 모듈에서 수행.
클라이언트가 어떤 요청을 보내든, 서버가 가시성 규칙에 따라 정보를 필터링하여 전송한다.
