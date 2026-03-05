# EventBus (`internal/server/eventbus/`)

## 책임

서버 내부의 이벤트 발행/구독. 모듈 간 결합도를 낮추는 핵심 인프라.

## 의존하는 모듈

없음 (인프라)

## 의존받는 모듈

모든 서버 모듈이 사용

## 인터페이스

```go
// 이벤트 데이터 타입
type PlayerConnectedData struct{ PlayerID string }
type PlayerDisconnectedData struct{ PlayerID string }
type StateChangedData struct {
    ChangeType string
    Data       any
}
type GameStatusChangedData struct {
    From GameStatus
    To   GameStatus
}

type EventBus struct {
    gameEventSubs     []chan GameEvent
    chatSubs          []chan ChatData
    playerConnSubs    []chan PlayerConnectedData
    playerDiscSubs    []chan PlayerDisconnectedData
    stateChangedSubs  []chan StateChangedData
    statusChangedSubs []chan GameStatusChangedData
    sendEndingsSubs   []chan GameEndData
    feedbackSubs      []chan Feedback
    mu                sync.RWMutex
}

func NewEventBus() *EventBus

// Close는 모든 구독 채널을 닫고 리소스를 정리.
func (eb *EventBus) Close()

// 이벤트별 Subscribe/Publish 쌍
func (eb *EventBus) SubscribeGameEvent() <-chan GameEvent
func (eb *EventBus) PublishGameEvent(event GameEvent)

func (eb *EventBus) SubscribeChat() <-chan ChatData
func (eb *EventBus) PublishChat(data ChatData)

func (eb *EventBus) SubscribePlayerConnected() <-chan PlayerConnectedData
func (eb *EventBus) PublishPlayerConnected(data PlayerConnectedData)

func (eb *EventBus) SubscribePlayerDisconnected() <-chan PlayerDisconnectedData
func (eb *EventBus) PublishPlayerDisconnected(data PlayerDisconnectedData)

func (eb *EventBus) SubscribeStateChanged() <-chan StateChangedData
func (eb *EventBus) PublishStateChanged(data StateChangedData)

func (eb *EventBus) SubscribeGameStatusChanged() <-chan GameStatusChangedData
func (eb *EventBus) PublishGameStatusChanged(data GameStatusChangedData)

func (eb *EventBus) SubscribeSendEndings() <-chan GameEndData
func (eb *EventBus) PublishSendEndings(data GameEndData)

func (eb *EventBus) SubscribeFeedback() <-chan Feedback
func (eb *EventBus) PublishFeedback(data Feedback)
```

## 보조 타입

```go
type ChatData struct {
    SenderID   string
    SenderName string
    RoomID     string
    Content    string
    Scope      string  // "room" | "global"
}
```

## 이벤트 흐름 예시 — 플레이어가 방을 조사할 때

```
 1. Client → NetworkServer: { type: 'examine', target: '책상' }
 2. NetworkServer → ActionProcessor.ProcessMessage()
 3. ActionProcessor → aiLayer.EvaluateExamine()
 4. AILayer → AI Provider API 호출
 5. AI 응답 파싱 → ExamineResultEvent + (optional) ClueFoundEvent
 6. ActionProcessor → gameState.DiscoverClue()   (단서 발견 시)
 7. GameStateManager → eventBus.PublishStateChanged(StateChangedData{ChangeType: "clue_discovered", Data: ...})
 8. ActionProcessor → eventBus.PublishGameEvent(examineResultEvent)
 9. EventBus → (채널을 통해) → MessageRouter.listenGameEvents() goroutine
10. MessageRouter → network.SendToMany(같은 방 플레이어들)
11. NetworkServer → 각 Client에 WebSocket 메시지 전달
```

## 설계 결정

- **Go 채널 기반.** 단일 프로세스이므로 외부 메시지 큐 불필요. 채널의 FIFO 보장으로 이벤트 순서 유지.
- **이벤트 유형별 타입 안전.** 별도 메서드 쌍(Subscribe/Publish)으로 컴파일 타임 타입 체크.
- **버퍼 채널 (크기: 256).** 각 구독 채널은 버퍼 크기 256으로 생성되어 발행자가 블로킹되지 않음. 구독자 goroutine이 순서대로 처리.
- **오버플로우 처리.** 채널 버퍼가 가득 찬 경우 non-blocking send(select default)를 사용하여 이벤트를 드랍하고 `slog.Warn`으로 경고를 기록한다. 발행자는 블로킹되지 않는다.
- **goroutine당 하나의 채널.** 각 구독자는 전용 goroutine에서 채널을 range로 읽어 처리. 이벤트 순서는 채널 내에서 보장됨.
