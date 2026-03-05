# Backend - Server (`internal/server/`)

호스트 머신에서 실행되는 게임 서버. **게임 상태의 유일한 관리자.**

## 모듈 목록

| 모듈 | 패키지 | 핵심 책임 |
|------|--------|-----------|
| [NetworkServer](./network-server.md) | `internal/server/network/` | WebSocket 연결 관리, 메시지 송수신 |
| [SessionManager](./session-manager.md) | `internal/server/session/` | 세션 생명주기, 로비, 상태 전이 |
| [GameStateManager](./game-state-manager.md) | `internal/server/game/` | 게임 상태 소유, 변이, 필터링된 뷰 |
| [MessageRouter](./message-router.md) | `internal/server/message/` | 가시성 기반 메시지 라우팅 |
| [MapEngine](./map-engine.md) | `internal/server/map/` | 맵 그래프, 인접 검증, 이동 |
| [ActionProcessor](./action-processor.md) | `internal/server/action/` | 명령어 디스패치, AI 위임, 상태 반영 |
| [EndConditionEngine](./end-condition-engine.md) | `internal/server/end/` | 종료 조건 평가, 투표, 타임아웃 |
| [EventBus](./event-bus.md) | `internal/server/eventbus/` | 내부 이벤트 발행/구독 |

### AI 레이어 모듈 (ai/ 디렉터리)

SessionManager는 게임 시작 시 AI 레이어 모듈들을 오케스트레이션한다.

| 모듈 | 위치 | 핵심 책임 |
|------|------|-----------|
| Orchestrator | `ai/orchestrator.md` | 멀티 모델 세계 생성 파이프라인 |
| WorldGenerator | `ai/world-generator.md` | 세계 생성 (Orchestrator 래퍼) |
| StoryValidator | `ai/story-validator.md` | 생성된 시나리오 검증 (규칙 기반) |
| StoryBible | `ai/story-bible.md` | concept/PRD 캐시 압축 |

**세계 생성 오케스트레이션 흐름:**
```
SessionManager.StartGame()
    → AILayer.GenerateWorld()
        → Orchestrator.GenerateWorld() (멀티 모델 파이프라인)
            → Phase 1: Seeds (병렬, 멀티 모델)
            → Phase 2: Seed Score & Select (규칙 기반)
            → Phase 3: Design (모드별 분기)
            → Phase 4: Integration (Showrunner)
            → Phase 5: Validate + Repair
    → 검증 통과 시: GameStateManager.InitializeWorld() → OnWorldGenerated() → briefing 전이
```

> **StoryValidator 통합 포인트:** `SessionManager.StartGame()`이 `AILayer.GenerateWorld()`를 호출하면, `GenerateWorld()` 내부에서 `StoryValidator`가 검증 파이프라인의 일부로 실행된다. `MapEngine.ValidateConnectivity()`와 `ValidateRoomCount()`는 `StoryValidator`가 Phase 5(Validate + Repair) 중에 호출한다.

**GM 서술 처리 (GM이 있는 게임):**
GM 서술은 ai/ 디렉터리의 GMEngine이 담당한다. ActionProcessor는 EventBus를 통해 플레이어 액션을 구독하고, 주기적으로 GMEngine에 개입 필요 여부를 확인한다.
```
플레이어 액션 발생
    → ActionProcessor가 EventBus에서 수신
    → GMEngine.ShouldIntervene(context) 확인
    → 개입 필요 시: GMEngine.GenerateNarration(context)
    → EventBus.PublishGameEvent(narrationEvent, scope: "all" 또는 "room")
    → MessageRouter가 AI 지정 수신자에게 전달
```

## 모듈 의존성 그래프

```
                    ┌──────────────────────┐
                    │  cmd/story/main.go   │ (진입점)
                    └──────────┬───────────┘
                               │
                    ┌──────────▼───────────┐
                    │     StoryServer      │ (조립 & 부트스트랩)
                    └──────────┬───────────┘
                               │
         ┌─────────────────────┼─────────────────────┐
         │                     │                     │
    ┌────▼────┐      ┌─────────▼──────┐      ┌───────▼──────┐
    │ Network │      │    Session     │      │  Event Bus   │
    │ Server  │      │    Manager     │      │              │
    └────┬────┘      └─────────┬──────┘      └───────┬──────┘
         │                     │                     │
         │           ┌─────────▼──────┐              │
         │           │   Game State   │◄─────────────┤
         │           │    Manager     │              │
         │           └─────────┬──────┘              │
         │                     │                     │
         │    ┌──────┬──────────┴──┬──────┐          │
         │    │      │             │      │          │
         │ ┌──▼──┐┌──▼──┐    ┌────▼┐┌────▼───┐      │
         │ │ Map ││Msg. │    │Act. ││  End   │      │
         │ │ Eng.││Rout.│    │Proc.││  Cond. ├──────► SessionManager
         │ └─────┘└─────┘    └──┬──┘└────────┘      │
         │                      │                    │
         │              ┌───────▼──────┐             │
         │              │   AI Layer   │◄────────────── SessionManager (StartGame)
         │              └──────────────┘             │
         │                                           │
         └──────────► 모든 모듈은 EventBus를 통해 이벤트를 발행/구독
```

## Server Bootstrap (`internal/server/server.go`)

모든 모듈을 조립하고 의존성을 주입하여 서버를 시작.

```go
type StoryServer struct {
    network            *network.NetworkServer
    eventBus           *eventbus.EventBus
    session            *session.SessionManager
    gameState          *game.GameStateManager
    messageRouter      *message.MessageRouter
    mapEngine          *mapengine.MapEngine
    actionProcessor    *action.ActionProcessor
    endConditionEngine *end.EndConditionEngine
    aiLayer            *ai.AILayer
}

func NewStoryServer(config ServerConfig) *StoryServer {
    // 의존성 조립
    bus    := eventbus.NewEventBus()
    net    := network.NewNetworkServer(network.NetworkConfig{Port: config.Port})
    gs     := game.NewGameStateManager(bus)
    ail, _ := ai.NewAILayer(ai.AILayerConfig{
        QualityMode:     config.QualityMode,
        ProviderConfigs: config.ProviderConfigs,
        RuntimeProvider: config.RuntimeProvider,
    })
    sess   := session.NewSessionManager(net, bus, gs, ail)
    router := message.NewMessageRouter(net, gs, bus)
    me     := mapengine.NewMapEngine(gs)
    ece    := end.NewEndConditionEngine(gs, ail, bus, sess)
    ap     := action.NewActionProcessor(gs, me, ail, ece, bus)

    s := &StoryServer{
        network:            net,
        eventBus:           bus,
        session:            sess,
        gameState:          gs,
        messageRouter:      router,
        mapEngine:          me,
        actionProcessor:    ap,
        endConditionEngine: ece,
        aiLayer:            ail,
    }
    s.wireUpHandlers()
    return s
}

type ServerInfo struct {
    RoomCode string
    Port     int
}

func (s *StoryServer) Start() (ServerInfo, error)
func (s *StoryServer) Stop() error
// Stop() 종료 순서:
//   1. session.Shutdown(ctx)  — 플레이어에게 알림 및 WebSocket 연결 정리
//   2. network.Stop()         — HTTP 서버 종료
//   3. eventBus.Close()       — 모든 구독 채널 닫기 및 리소스 정리
```
