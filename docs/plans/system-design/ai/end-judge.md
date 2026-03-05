# EndJudge & EndingGenerator (`internal/ai/judge/`)

## 책임

AI 기반 종료 조건 판정 및 개인화된 엔딩 생성.

## 의존하는 모듈

AIProvider

---

## EndJudge

### 인터페이스

```go
// internal/ai/judge/end_judge.go

type EndJudge struct {
    provider AIProvider
}

func NewEndJudge(provider AIProvider) *EndJudge {
    return &EndJudge{provider: provider}
}

// EvaluateEndCondition은 (shouldEnd bool, reason string, error)를 반환.
func (j *EndJudge) EvaluateEndCondition(
    ctx context.Context,
    condition *EndCondition,
    gameCtx *GameContext,
) (bool, string, error)
```

### 동작

`ai_judgment` 타입 종료 조건에 대해서만 호출됨. 다른 종료 조건 타입(vote, consensus, event, timeout)은 EndConditionEngine이 규칙 기반으로 처리.

```go
type EndJudgment struct {
    ShouldEnd bool   `json:"shouldEnd"`
    Reason    string `json:"reason"`
}

func (j *EndJudge) EvaluateEndCondition(
    ctx context.Context,
    condition *EndCondition,
    gameCtx *GameContext,
) (bool, string, error) {
    criteriaJSON, _ := json.Marshal(condition.TriggerCriteria)

    // GameContext.CurrentState.ElapsedTime (초 단위 int64)
    elapsedMinutes := int(gameCtx.CurrentState.ElapsedTime / 60)
    totalMinutes := gameCtx.World.GameStructure.EstimatedDuration // Meta에서 설정된 분 단위
    // 발견된 단서 수 계산
    discoveredClues := 0
    for _, cs := range gameCtx.CurrentState.ClueStates {
        if cs.IsDiscovered {
            discoveredClues++
        }
    }
    totalClues := len(gameCtx.World.Clues)

    prompt := fmt.Sprintf(`
[종료 조건]
%s
판정 기준: %s

[현재 게임 상태]
진행 시간: %d분 / %d분
발견된 단서: %d/%d
최근 이벤트: %s

[판단]
이 종료 조건이 충족되었는지 판단하세요.
shouldEnd: true/false와 판단 근거를 제공하세요.
`,
        condition.Description, string(criteriaJSON),
        elapsedMinutes, totalMinutes,
        discoveredClues, totalClues,
        summarizeRecentEvents(gameCtx.RecentEvents), // RecentEvents []GameEvent에서 요약 생성
    )

    raw, err := j.provider.GenerateStructured(ctx, StructuredRequest{
        SystemPrompt: "당신은 공정한 게임 심판입니다. 종료 조건이 충족되었는지만 판단하세요.",
        UserPrompt:   prompt,
        Temperature:  0.3, // 판정은 결정적이어야 함
        MaxTokens:    300,
    })
    if err != nil {
        return false, "", err
    }

    var judgment EndJudgment
    if err := json.Unmarshal(raw, &judgment); err != nil {
        return false, "", err
    }
    return judgment.ShouldEnd, judgment.Reason, nil
}
```

---

## /end 플레이어 종료 요청 흐름

플레이어가 `/end`를 입력하면 서버(EndConditionEngine)가 다수결(과반 이상 동의)을 확인한 후 `EndingGenerator.GenerateEndings()`를 호출.

| endReason | 설명 | GenerateEndings 처리 |
|-----------|------|----------------------|
| `condition_met` | 종료 조건 자동 충족 | 정상 엔딩 생성 |
| `player_vote` | `/end` 과반 동의 | 현재까지 진행 기준 엔딩 생성 |
| `timeout` | 시간 초과 | 미해결 요소 포함 엔딩 (완결감 우선) |

endReason은 `GenerateEndings()`의 명시적 파라미터로 전달되어 `buildEndingPrompt()`에서 프롬프트 분기에 사용.

### /end 투표 메커니즘 (FR-091)

투표 추적 및 타임아웃은 서버 사이드 EndConditionEngine이 처리한다. AI는 투표 집계에 관여하지 않는다.

| 항목 | 규칙 |
|------|------|
| 투표 제안자 | 자동으로 "동의"로 집계 |
| 무응답 | 기권으로 처리 (= 반대로 집계) |
| 과반 기준 | 전체 활성 플레이어의 50%+1 |
| 투표 타임아웃 | 60초. 타임아웃 후 집계하여 과반 미달 시 부결 |

> **참고:** 실제 투표 상태 추적 및 타임아웃 스케줄링은 `backend/end-condition-engine.md` 참조.

---

## /solve 합의 시스템

`/solve` 명령은 플레이어가 게임의 핵심 수수께끼에 대한 해답을 제출하는 명령이다.

### 흐름

```
플레이어 → /solve <해답 텍스트>
    │
    ├── EndConditionEngine: consensus 타입 종료 조건 존재 여부 확인
    │
    ├── AILayer.EvaluateSolve(ctx, solutionText, gameCtx)
    │     └── EndJudge.EvaluateEndCondition() 호출
    │           - condition.Type == "consensus"인 종료 조건 사용
    │           - 해답 텍스트를 평가 기준(TriggerCriteria)에 대조
    │           - shouldEnd: true/false + 판단 근거 반환
    │
    ├── shouldEnd == false → 해답 불충분 메시지 반환, 게임 계속
    │
    └── shouldEnd == true → 합의 카운터 증가
          │
          ├── 합의 조건 미충족 (일부 동의) → 진행 상황 브로드캐스트
          │
          └── 합의 조건 충족 (모든 활성 플레이어 동의 또는 임계치 초과)
                └── game_end 이벤트 발생 → EndingGenerator.GenerateEndings()
```

### 합의 추적

```
// 합의 추적은 EndConditionEngine이 서버 사이드에서 관리
// AI는 개별 해답 텍스트의 타당성만 판단

type SolveConsensus struct {
    RequiredCount int            // 합의에 필요한 플레이어 수 (기본: 전체 활성 플레이어)
    AgreedPlayers map[string]bool // playerID → 동의 여부
}

// 임계치: 기본값은 모든 활성 플레이어 동의 (100%)
// 세계 설계에서 EndCondition.TriggerCriteria로 임계치 비율 오버라이드 가능
```

### AI 평가 (EndJudge 재사용)

`/solve` 해답 텍스트는 `ai_judgment` 타입 종료 조건으로 평가된다. EndJudge.EvaluateEndCondition()이 해답 텍스트를 prompt에 포함하여 승리 조건 충족 여부를 판정한다.

> **참고:** consensus 타입 종료 조건의 상태 추적은 `backend/end-condition-engine.md` 참조.

---

## EndingGenerator

### 인터페이스

```go
// internal/ai/judge/ending_generator.go

type EndingGenerator struct {
    provider AIProvider
}

func NewEndingGenerator(provider AIProvider) *EndingGenerator {
    return &EndingGenerator{provider: provider}
}

func (g *EndingGenerator) GenerateEndings(ctx context.Context, gameCtx *GameContext, endReason string) (*GameEndData, error)
```

### GameEndData 구조

```go
type GameEndData struct {
    CommonResult  string         `json:"commonResult"`
    PlayerEndings []PlayerEnding `json:"playerEndings"`
    SecretReveal  SecretReveal   `json:"secretReveal"`
}
```

### PlayerEnding 구조

```go
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
```

### SecretReveal 구조

```go
type SecretReveal struct {
    PlayerSecrets      []PlayerSecretEntry     `json:"playerSecrets"`
    SemiPublicReveal   []SemiPublicRevealEntry `json:"semiPublicReveal"`
    UndiscoveredClues  []UndiscoveredClueEntry `json:"undiscoveredClues"`
    NPCSecrets         []NPCSecretEntry        `json:"npcSecrets"`
    UntriggeredGimmicks []GimmickEntry         `json:"untriggeredGimmicks"` // world.Gimmicks 중 isTriggered == false인 항목
}

// SecretReveal 하위 타입은 types.md의 정의를 사용한다:
// - PlayerSecretEntry {PlayerID, CharacterName, Secret, SpecialRole}
// - SemiPublicRevealEntry {Info, SharedBetween}
// - UndiscoveredClueEntry {Clue, RoomName}
// - NPCSecretEntry {NPCName, HiddenInfo}
// - GimmickEntry {GimmickID, Description, TriggerCondition, NPCID}
// FR-068: 개인 목표 공개는 PlayerEnding.GoalResults를 통해 제공 (PersonalEnding에 포함)

// UntriggeredGimmicks는 buildSecretReveal()에서 world.Gimmicks를 순회하여
// GameState.NPCStates[npc.ID].GimmickTriggered == false인 항목으로 채워진다.
```

### 엔딩 생성 흐름

```
GenerateEndings(ctx, gameCtx)
    │
    ├── 전체 게임 액션 로그 수집 (gameCtx.ActionLog — 플레이어별 전체 행동 이력)
    │     ActionLog는 RecentEvents와 달리 게임 시작부터 현재까지의 전체 기록이다.
    ├── 각 플레이어의 행동 요약 구성 (ActionLog에서 플레이어별로 집계)
    ├── 개인 목표 달성 여부 판정 요청
    │
    ├── provider.GenerateStructured(ctx, StructuredRequest{
    │     Temperature: 0.9,
    │     MaxTokens:   3000,
    │   })
    │
    ├── json.Unmarshal → endingAIResult
    │
    ├── SecretReveal은 AI 호출 없이 규칙 기반 구성
    │     - playerSecrets: 각 PlayerRole.Secret
    │     - semiPublicReveal: world.Information.SemiPublic
    │     - undiscoveredClues: IsDiscovered == false인 단서
    │     - npcSecrets: 각 NPC.HiddenInfo
    │     - untriggeredGimmicks: GimmickTriggered == false인 기믹 (world.Gimmicks에서)
    │
    └── return &GameEndData{...}, nil
```

> **GameContext.ActionLog vs RecentEvents:** `buildEndingPrompt()`는 `gameCtx.RecentEvents`(최근 이벤트 요약)가 아닌 `gameCtx.ActionLog`(전체 행동 이력)를 사용한다. ActionLog는 플레이어별 전체 행동 시퀀스를 포함하며, 엔딩 생성 시 "아, 그때 내가..."와 같은 카타르시스를 위해 필수적이다.

```go
func (g *EndingGenerator) GenerateEndings(ctx context.Context, gameCtx *GameContext, endReason string) (*GameEndData, error) {
    prompt := buildEndingPrompt(gameCtx, endReason)

    raw, err := g.provider.GenerateStructured(ctx, StructuredRequest{
        SystemPrompt: endingSystemPrompt,
        UserPrompt:   prompt,
        Temperature:  0.9,
        MaxTokens:    3000,
    })
    if err != nil {
        return nil, err
    }

    var aiResult endingAIResult
    if err := json.Unmarshal(raw, &aiResult); err != nil {
        return nil, err
    }

    // SecretReveal은 규칙 기반으로 구성 (AI 호출 없음)
    secretReveal := g.buildSecretReveal(gameCtx)

    return &GameEndData{
        CommonResult:  aiResult.CommonResult,
        PlayerEndings: aiResult.PlayerEndings,
        SecretReveal:  secretReveal,
    }, nil
}
```

### 엔딩 생성 Prompt

```
[세계]
${world.title} - ${world.synopsis}

[게임 결과]
종료 이유: ${endReason}
${endReason == "timeout" ? "[주의] 시간 초과로 종료됩니다. 미해결 사항이 있더라도 완결감 있는 엔딩을 제공하세요." : ""}
${게임 구조별 결과 (투표 결과, 합의 내용 등)}

[각 플레이어 전체 행동 이력]
// gameCtx.ActionLog 사용 — 게임 시작부터 종료까지의 전체 행동 시퀀스 (플레이어별)
// RecentEvents(최근 이벤트만)가 아닌 ActionLog(전체 기록)를 사용해야
// 개인화된 엔딩에서 초반부 행동도 반영할 수 있다.
${플레이어별 전체 행동 이력 요약 (ActionLog에서 집계)}

[각 플레이어의 개인 목표]
${플레이어별 목표 목록}

[지시]
1. 전체 게임 결과를 드라마틱하게 서술하세요 (3~5문장)
2. 각 플레이어별로:
   - 행동 요약 (2~3문장)
   - 각 개인 목표의 달성/실패 여부와 근거
   - 개인화된 엔딩 서술 (3~5문장)
3. 모든 서술은 플레이어가 읽었을 때 "아, 그때 내가 그래서..." 하는 카타르시스를 줄 수 있어야 합니다
```

> **time_warning 이벤트 설계 참고:** `time_warning` 이벤트는 AI 생성이 아닌 서버 타이머 기반으로 EndConditionEngine이 생성한다. `backend/end-condition-engine.md` 참조.
