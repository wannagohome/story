# EndConditionEngine (`internal/server/end/`)

## 책임

종료 조건 평가, 투표/합의 시스템 관리, 타임아웃 관리, 엔딩 트리거.

## 의존하는 모듈

GameStateManager, AILayer, EventBus, SessionManager

## 인터페이스

```go
type EndConditionEngine struct {
    timeout        *time.Timer
    activeVote     *ActiveVote
    endProposal    *EndProposal
    gameState      *game.GameStateManager
    aiLayer        *ai.AILayer
    eventBus       *eventbus.EventBus
    sessionManager *session.SessionManager
    mu             sync.Mutex
}

func NewEndConditionEngine(
    gs  *game.GameStateManager,
    ail *ai.AILayer,
    bus *eventbus.EventBus,
    sm  *session.SessionManager,
) *EndConditionEngine {
    ece := &EndConditionEngine{
        gameState:      gs,
        aiLayer:        ail,
        eventBus:       bus,
        sessionManager: sm,
    }
    // 게임 이벤트 구독: goroutine에서 종료 조건 감시
    go ece.listenGameEvents(bus.SubscribeGameEvent())
    return ece
}

// ── 게임 시작 시 호출 ──
func (ece *EndConditionEngine) StartMonitoring(endConditions []EndCondition, timeoutMinutes int)

// ── 투표 시스템 ──
func (ece *EndConditionEngine) StartVote(reason string, candidates []string, timeoutSeconds int)
func (ece *EndConditionEngine) CastVote(playerId string, targetId string)

// ── 합의 시스템 ──
func (ece *EndConditionEngine) SubmitSolution(playerId string, answer string)

// ── /end 발의 시스템 ──
func (ece *EndConditionEngine) ProposeEnd(proposerId string)
func (ece *EndConditionEngine) RespondToEndProposal(playerId string, agree bool)

// ── 정리 ──
func (ece *EndConditionEngine) StopMonitoring()
```

## 종료 조건 평가 흐름

`checkEndConditions()`는 의미 있는 이벤트에만 트리거됨: `vote_ended`, `story_event`, `clue_found`, `action_result`. 채팅이나 이동 등 의미 없는 이벤트에는 실행하지 않아 불필요한 연산을 방지한다.

```
게임 이벤트 발생
      │
      ▼
checkEndConditions(event)
      │
      ├── endConditions를 순회
      │
      ├── triggerType == "timeout"
      │     └── 타이머가 별도 관리 (startTimeout)
      │
      ├── triggerType == "vote"
      │     └── 투표 결과와 triggerCriteria 비교
      │
      ├── triggerType == "consensus"
      │     └── 합의 상태와 triggerCriteria 비교
      │
      ├── triggerType == "event"
      │     └── 이벤트 데이터와 triggerCriteria 비교
      │
      └── triggerType == "ai_judgment"
            └── aiLayer.JudgeEndCondition(condition, context)
                  └── AI가 현재 상태를 분석하여 종료 여부 결정
      │
      ▼
  조건 충족 시 → triggerEnding(reason)
```

## 투표 시스템

게임 구조에 투표가 포함된 경우 활성화.

```go
type ActiveVote struct {
    Reason      string
    Candidates  []string
    Votes       map[string]string  // voterId → candidateId
    TotalVoters int
    TimeoutMs   int64
    Timer       *time.Timer
}
```

```
StartVote()
    │
    ▼
Server → All: { type: 'vote_started', reason, candidates, timeoutSeconds }
    │
    ▼
각 플레이어: CastVote(playerId, targetId)
    │
    ├── Server → All: { type: 'vote_progress', votedCount, totalVoters }
    │
    └── 모든 플레이어 투표 완료 또는 타임아웃
          │
          ▼
    집계 → Server → All: { type: 'vote_ended', results, outcome }
          │
          ▼
     EndCondition과 대조 → 조건 충족 여부 확인
```

## 합의 시스템

게임 구조에 합의가 포함된 경우 활성화.

```go
type ActiveConsensus struct {
    Answers     map[string]string  // playerId → answer
    TotalPlayers int
    TimeoutMs   int64
    Timer       *time.Timer
}
```

```
SubmitSolution(playerId, answer)
     │
     ▼
 답변 저장 → answers[playerId] = answer
     │
     ├── Server → All: { type: 'solve_progress', submittedCount, totalPlayers }
     │
     └── 모든 플레이어 답변 완료 또는 타임아웃
           │
           ▼
     모든 답변 수집 → 답변들을 triggerCriteria와 대조
           │
           ├── 답변이 triggerCriteria와 일치
           │     │
           │     ▼
           │   triggerEnding("consensus")
           │
           └── 답변이 triggerCriteria와 불일치
                 │
                 ▼
           Server → All: { type: 'solve_result', outcome, answers }
```

## /end 발의 시스템

```go
type EndProposal struct {
    ProposerID  string
    Responses   map[string]bool  // playerId → agree/disagree
    TotalVoters int
    Timer       *time.Timer
}
```

```
ProposeEnd(proposerId)
    │
    ├── Responses[proposerId] = true  ← 발의자는 자동으로 동의로 집계
    │
    ▼
Server → All: { type: 'end_proposed', proposerId, proposerName, timeoutSeconds: 60 }
    │
    ▼
각 플레이어: RespondToEndProposal(playerId, agree)
    │
    └── 과반수 동의 (>50%)
          │
          ├── true → triggerEnding("player_proposed")
          └── false → Server → All: { type: 'end_vote_result', passed: false }

    └── 60초 타임아웃 → 미응답은 기권(=동의하지 않음)
```

## 타임아웃 관리

`timeoutMinutes`는 AI가 세계 생성 시 결정 (범위 10~30분, 기본 20분. Concept: "10~30분 내 완결, 최대 30분").
시간 초과에 의한 강제 종료는 항상 EndCondition 중 `isFallback: true`로 포함됨 (FR-019, FR-027).

```go
func (ece *EndConditionEngine) startTimeout(minutes int) {
    // 5분 전 경고 (게임 시간이 5분 이하이면 건너뜀)
    if minutes > 5 {
        time.AfterFunc(time.Duration(minutes-5)*time.Minute, func() {
            ece.eventBus.PublishGameEvent(TimeWarningEvent{
                BaseEvent: BaseEvent{
                    ID: uuid.New().String(), Timestamp: time.Now().UnixMilli(),
                    Visibility: EventVisibility{Scope: "all"},
                },
                Type: "time_warning",
                Data: TimeWarningData{RemainingMinutes: 5},
            })
        })
    }

    // 1분 전 경고
    time.AfterFunc(time.Duration(minutes-1)*time.Minute, func() {
        ece.eventBus.PublishGameEvent(TimeWarningEvent{
            BaseEvent: BaseEvent{
                ID: uuid.New().String(), Timestamp: time.Now().UnixMilli(),
                Visibility: EventVisibility{Scope: "all"},
            },
            Type: "time_warning",
            Data: TimeWarningData{RemainingMinutes: 1},
        })
    })

    // 타임아웃
    ece.timeout = time.AfterFunc(time.Duration(minutes)*time.Minute, func() {
        ece.triggerEnding("timeout")
    })
}
```

## 엔딩 트리거

```go
func (ece *EndConditionEngine) triggerEnding(reason string) {
    ece.StopMonitoring()

    // SessionManager에 상태 전이 알림
    ece.sessionManager.StartEnding()

    // AI 엔딩 생성은 goroutine에서 처리 (블로킹 방지)
    go func() {
        ctx := ece.buildFullContext()

        // 재시도 로직: 최대 3회, 지수 백오프
        var endData GameEndData
        var err error
        for attempt := 0; attempt < 3; attempt++ {
            if attempt > 0 {
                backoff := time.Duration(1<<uint(attempt-1)) * time.Second // 1s, 2s
                time.Sleep(backoff)
            }
            endData, err = ece.aiLayer.GenerateEndings(ctx, reason)
            if err == nil {
                break
            }
            slog.Warn("엔딩 생성 실패, 재시도", "attempt", attempt+1, "error", err)
        }

        if err != nil {
            // 모든 재시도 실패 시 하드코딩된 fallback 엔딩 전송
            slog.Error("엔딩 생성 최종 실패, fallback 엔딩 전송", "error", err)
            endData = ece.buildFallbackEnding(reason)
        }

        // 각 플레이어에게 개인화된 엔딩 전달 (per-player ServerMessage)
        // game_ending은 개인별 PersonalEnding이 다르므로 개별 전송 필요.
        // MessageRouter가 채널을 통해 수신하여 각 플레이어에게 전송.
        ece.eventBus.PublishSendEndings(endData)
        // → MessageRouter가 수신하여 각 플레이어에게:
        //   { type: 'game_ending', commonResult, personalEnding, secretReveal }
        //   (personalEnding은 해당 플레이어의 PlayerEnding에서 추출)

        // 이후 game_finished 브로드캐스트는 SessionManager가 처리
    }()
}

// buildFallbackEnding: AI 엔딩 생성이 모두 실패했을 때 사용하는 하드코딩 fallback
func (ece *EndConditionEngine) buildFallbackEnding(reason string) GameEndData {
    commonResult := "게임이 종료되었습니다. 모든 플레이어가 최선을 다했습니다."
    playerIDs := ece.gameState.GetAllPlayerIDs()
    endings := make([]PlayerEnding, len(playerIDs))
    for i, id := range playerIDs {
        endings[i] = PlayerEnding{
            PlayerID:    id,
            Summary:     "게임 종료",
            GoalResults: []GoalResult{},
            Narrative:   "이야기가 막을 내렸습니다.",
        }
    }
    // SecretReveal은 서버가 규칙 기반으로 구성하므로 AI 실패와 무관하게 항상 포함한다.
    // GameStateManager에서 플레이어 비밀, NPC 숨긴 정보, 미발견 단서를 수집하여 구성한다.
    secretReveal := ece.buildSecretRevealFromGameState()
    return GameEndData{
        CommonResult:  commonResult,
        PlayerEndings: endings,
        SecretReveal:  secretReveal,
    }
}
```
