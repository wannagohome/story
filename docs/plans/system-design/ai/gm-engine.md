# GMEngine (`internal/ai/gm/`)

## 책임

GM 서술 이벤트 생성. 게임 진행 모니터링 및 필요시 개입. GM이 없는 게임에서는 비활성화.

## 의존하는 모듈

AIProvider

## 인터페이스

```go
// internal/ai/gm/gm_engine.go

type GMEngine struct {
    provider AIProvider
    active   bool
}

func NewGMEngine(provider AIProvider) *GMEngine {
    return &GMEngine{provider: provider}
}

// ── GM 활성 여부 ──
func (g *GMEngine) SetActive(active bool)
func (g *GMEngine) IsActive() bool

// ── 게임 시작 시 오프닝 ──
func (g *GMEngine) GenerateOpening(ctx context.Context, gameCtx *GameContext) (*NarrationEvent, error)

// ── 플레이어 행동에 반응한 서술 ──
// nil 반환 = 개입 불필요
func (g *GMEngine) RespondToAction(ctx context.Context, gameCtx *GameContext, triggeringEvent GameEvent) (*NarrationEvent, error)

// ── 주기적 체크: 게임이 정체되었는가? ──
// P1 - MVP 이후 구현
func (g *GMEngine) CheckPacing(ctx context.Context, gameCtx *GameContext) (*StoryEventEvent, error)

// ── 게임 후반부 수렴 유도 (FR-061) ──
// P1 - MVP 이후 구현
func (g *GMEngine) CheckConvergence(ctx context.Context, gameCtx *GameContext) (*NarrationEvent, error)
```

## GM이 개입하는 조건

| 조건 | 트리거 |
|------|--------|
| 게임 정체 | 일정 시간(5분) 동안 단서 발견 없음 + 이동 없음 |
| 스토리 이벤트 트리거 | 특정 단서 발견, 특정 장소 방문 등 |
| 게임 후반부 | 남은 시간 < 전체의 30%일 때 수렴 유도 |

## GM 비활성 모드 Fallback

GM이 없는 게임(`active == false`)에서의 동작:

| 기능 | GM 활성 시 | GM 비활성 시 |
|------|-----------|------------|
| `GenerateOpening` | AI 서술 생성 | 빈 내레이션 또는 세계 제목만 반환. 브리핑 화면(FR-078)은 GMEngine과 무관하게 서버에서 처리. GenerateOpening은 브리핑 이후 게임 시작 시점의 분위기 서술에만 해당. |
| `RespondToAction` | AI 판단 후 개입 | 항상 `nil` 반환 (개입 없음) |
| `CheckPacing` (P1) | AI가 긴장감 조율 (FR-060) | 미적용 — MVP 이후 구현 |
| `CheckConvergence` (P1) | AI가 스토리 수렴 유도 (FR-061) | 미적용 — MVP 이후 구현 |
| `time_warning` 이벤트 | GM과 무관 | EndConditionEngine에서 독립적으로 발생 |

```go
func (g *GMEngine) RespondToAction(ctx context.Context, gameCtx *GameContext, trigger GameEvent) (*NarrationEvent, error) {
    if !g.active {
        return nil, nil // GM 비활성 시 즉시 반환
    }
    // ... AI 호출 로직
}
```

> FR-060(긴장감 조율), FR-061(스토리 수렴)은 P1이므로 MVP에서는 GM 없는 게임에 미적용.
> `time_warning` 이벤트는 GM과 무관하게 `EndConditionEngine`에서 타임아웃 계산 시 발생.

## 오프닝 내레이션

게임 시작 직후 분위기를 설정하는 GM 서술.

```go
func (g *GMEngine) GenerateOpening(ctx context.Context, gameCtx *GameContext) (*NarrationEvent, error) {
    prompt := fmt.Sprintf(`
게임이 시작됩니다. 분위기를 설정하는 짧은 오프닝 내레이션을 작성하세요.
세계: %s
분위기: %s
3~5문장으로 짧게.
`, gameCtx.World.Title, gameCtx.World.Atmosphere)

    raw, err := g.provider.GenerateStructured(ctx, StructuredRequest{
        SystemPrompt: gmSystemPrompt,
        UserPrompt:   prompt,
        Temperature:  0.9,
        MaxTokens:    500,
    })
    if err != nil {
        return nil, err
    }

    var response NarrationEvent
    if err := json.Unmarshal(raw, &response); err != nil {
        return nil, err
    }

    response.Visibility = EventVisibility{Scope: "all"}
    return &response, nil
}
```

## 행동 반응

모든 게임 이벤트에 대해 호출. GM이 반응할 필요가 있는지 AI가 판단.

```go
// gmInterventionResponse는 GM 개입 여부와 서술을 함께 담는 래퍼.
// "null 반환" 대신 shouldIntervene 필드로 개입 여부를 명시하여
// JSON 파싱 안정성을 높임.
type gmInterventionResponse struct {
    ShouldIntervene bool           `json:"shouldIntervene"`
    Events          []NarrationEvent `json:"events"`
}

func (g *GMEngine) RespondToAction(ctx context.Context, gameCtx *GameContext, trigger GameEvent) (*NarrationEvent, error) {
    if !g.active {
        return nil, nil
    }

    triggerJSON, _ := json.Marshal(trigger)
    prompt := fmt.Sprintf(`
다음 이벤트가 발생했습니다: %s

GM으로서 이 이벤트에 반응해야 하는지 판단하세요.

판단 기준:
- 스토리에 중요한 전환점인가?
- 플레이어에게 분위기/긴장감을 전달해야 하는가?
- 단순한 일상적 행동이면 개입하지 마세요.

반드시 {"shouldIntervene": bool, "events": [...]} 형태로 응답하세요.
개입 불필요 시: {"shouldIntervene": false, "events": []}
개입 필요 시: {"shouldIntervene": true, "events": [{"narration": "..."}]}
`, string(triggerJSON))

    raw, err := g.provider.GenerateStructured(ctx, StructuredRequest{
        SystemPrompt: gmSystemPrompt,
        UserPrompt:   prompt,
        Temperature:  0.9,
        MaxTokens:    500,
    })
    if err != nil {
        return nil, err
    }

    var response gmInterventionResponse
    if err := json.Unmarshal(raw, &response); err != nil {
        return nil, err
    }

    if !response.ShouldIntervene || len(response.Events) == 0 {
        return nil, nil
    }
    result := response.Events[0]
    return &result, nil
}
```

## 스토리 수렴 유도 (FR-061)

게임 후반부(남은 시간 < 전체의 30%)에 진입하면 스토리가 자연스럽게 결말로 수렴하도록 유도.

```go
func (g *GMEngine) CheckConvergence(ctx context.Context, gameCtx *GameContext) (*NarrationEvent, error) {
    // GameContext.CurrentState.ElapsedTime (초) 및 World.GameStructure로부터 계산
    elapsedSec := gameCtx.CurrentState.ElapsedTime
    totalSec := int64(gameCtx.World.GameStructure.EstimatedDuration) * 60 // GameStructure.EstimatedDuration (분 단위, schemas.md WorldGenerationMeta에서 변환됨)
    remainingSec := totalSec - elapsedSec

    // 후반부 기준: 남은 시간이 전체의 30% 미만
    if remainingSec > totalSec*30/100 {
        return nil, nil
    }

    // 미발견 단서 수 계산
    undiscoveredClues := 0
    for _, cs := range gameCtx.CurrentState.ClueStates {
        if !cs.IsDiscovered {
            undiscoveredClues++
        }
    }

    prompt := fmt.Sprintf(`
게임이 후반부에 접어들었습니다. 남은 시간: 약 %d분

[수렴 유도 원칙]
- 결정적 단서를 간접적으로 노출하거나 NPC를 통해 힌트를 제공하세요
- 갑작스러운 끝이 아닌 자연스러운 클라이맥스로 유도하세요
- 아직 해결되지 않은 핵심 갈등을 부각시키세요

미발견 단서: %d개
`, remainingSec/60, undiscoveredClues)

    raw, err := g.provider.GenerateStructured(ctx, StructuredRequest{
        SystemPrompt: gmSystemPrompt,
        UserPrompt:   prompt,
        Temperature:  0.8,
        MaxTokens:    500,
    })
    if err != nil {
        return nil, err
    }

    // null, "null", {} 등 빈 응답 처리
    trimmed := strings.TrimSpace(string(raw))
    if trimmed == "null" || trimmed == `"null"` || trimmed == "{}" {
        return nil, nil
    }

    var response NarrationEvent
    if err := json.Unmarshal(raw, &response); err != nil {
        return nil, err
    }
    response.Visibility = EventVisibility{Scope: "all"}
    return &response, nil
}
```

## 페이싱 체크

> **P1 참고 구현 — MVP 미포함:** CheckPacing과 CheckConvergence는 P1 기능이다. 아래 코드는 향후 구현을 위한 참고 설계이며 MVP 빌드에 포함되지 않는다.

주기적으로 (매 2분 또는 5개 이벤트마다) 게임 진행 속도를 체크.

```go
func (g *GMEngine) CheckPacing(ctx context.Context, gameCtx *GameContext) (*StoryEventEvent, error) {
    recentEvents := gameCtx.RecentEvents

    // 규칙 기반 사전 필터: 최근 5분간 의미 있는 이벤트가 있으면 스킵
    hasProgress := false
    for _, e := range recentEvents {
        if e.EventType() == "clue_found" || e.EventType() == "story_event" || e.EventType() == "npc_reveal" {
            hasProgress = true
            break
        }
    }
    if hasProgress {
        return nil, nil
    }

    // AI에게 개입 여부 판단 요청
    // → 긴장감을 높이는 새로운 사건 생성
    raw, err := g.provider.GenerateStructured(ctx, StructuredRequest{
        SystemPrompt: gmSystemPrompt,
        UserPrompt:   buildPacingPrompt(gameCtx),
        Temperature:  0.9,
        MaxTokens:    500,
    })
    if err != nil {
        return nil, err
    }

    trimmedPacing := strings.TrimSpace(string(raw))
    if trimmedPacing == "null" || trimmedPacing == `"null"` || trimmedPacing == "{}" {
        return nil, nil
    }

    var event StoryEventEvent
    if err := json.Unmarshal(raw, &event); err != nil {
        return nil, err
    }
    return &event, nil
}
```

## GM System Prompt

GM 시스템 프롬프트는 고정 지시와 동적 컨텍스트로 구성된다. 동적 컨텍스트는 각 호출 시 `GameContext`에서 빌드된다.

```
당신은 이 게임의 GM(Game Master)입니다.

[세계 개요]
제목: ${world.title}
시놉시스: ${world.synopsis}
분위기: ${world.atmosphere}

[게임 구조]
핵심 갈등: ${world.gameStructure.coreConflict}
종료 조건: ${world.gameStructure.endConditions 요약}

[현재 게임 상태]
경과 시간: ${elapsedMinutes}분 / ${totalMinutes}분
발견된 단서: ${discoveredClues} / ${totalClues}
플레이어 위치: ${각 플레이어의 현재 방 목록}

[플레이어 역할 (GM 전용 정보)]
${각 플레이어: 캐릭터명, 역할 요약, 현재 상황}

[역할]
- 스토리의 분위기와 긴장감을 유지하세요
- 플레이어 행동에 반응하여 세계를 생생하게 만드세요
- 지루해지면 새로운 사건을 투입하세요
- 결말을 향해 자연스럽게 유도하세요

[원칙]
- 짧고 임팩트 있게 (1~3문장)
- 너무 자주 개입하지 마세요 — 플레이어 간 대화가 핵심입니다
- 정보를 직접 주지 말고, 단서를 암시하세요
- 공정하게 — 특정 플레이어를 편들지 마세요
```

> **time_warning 이벤트:** `time_warning` 이벤트는 AI 생성이 아닌 서버 타이머 기반으로 EndConditionEngine이 생성한다. `backend/end-condition-engine.md` 참조.
