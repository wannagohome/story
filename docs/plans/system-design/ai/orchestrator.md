# Orchestrator (`internal/ai/orchestrator/`)

## 책임

멀티 모델 세계 생성 파이프라인. 여러 AI 프로바이더의 모델을 역할별로 배정하여 세계를 생성한다.

**핵심 원칙: "사실은 틀려도 되지만 구조는 틀리면 안 된다."**

단일 모델이 아닌 전문화된 논리적 에이전트가 협력하여 세계를 생성한다. 각 에이전트는 서로 다른 프로바이더의 모델로 구동되며, 역할에 최적화된 모델을 선택한다. StoryBibleCompressor는 사전 처리(pre-processing) 단계이고, Polish는 사후 처리(post-processing) 단계로, 파이프라인 에이전트(8종)와는 별도로 동작한다.

## 의존하는 모듈

ProviderRegistry, StoryBible, StoryValidator

## 인터페이스

```go
// internal/ai/orchestrator/orchestrator.go

type QualityMode string
const (
    QualityModeFast    QualityMode = "fast"    // 번개 집필
    QualityModePremium QualityMode = "premium" // 시네마틱 집필
)

type OrchestratorConfig struct {
    Mode     QualityMode
    Registry *ProviderRegistry
    Bible    *StoryBible
}

type Orchestrator struct {
    config OrchestratorConfig
}

func NewOrchestrator(config OrchestratorConfig) *Orchestrator

func (o *Orchestrator) GenerateWorld(
    ctx context.Context,
    playerCount int,
    themeHint string,
    onProgress ProgressFunc,
) (*WorldGeneration, error)
```

## 논리적 에이전트 구성

**사전 처리 (pre-processing, 파이프라인 외부):**

| Agent | 책임 | 입력 | 출력 |
|-------|------|------|------|
| StoryBibleCompressor | concept/PRD → 3종 캐시 압축 | concept.md, prd.md | CreativeBrief, SchemaBrief, ValidationChecklist |

**파이프라인 에이전트 (8종):**

| # | Agent | 책임 | 입력 | 출력 |
|---|-------|------|------|------|
| 1 | ChaosMuse (×2~3) | 미친 훅, 장르/사건/배신 포인트 | CreativeBrief, playerCount, themeHint | SeedProposal[] |
| 2 | ConflictEngineer | 핵심 갈등, 종료 조건, 승리 판정 | 선택된 Seed | ConflictDesign |
| 3 | CastSecretMaster | 역할, 개인 목표, 비밀, 관계망 | Seed + ConflictDesign | CastDesign |
| 4 | MapClueSmith | 맵, 방, 단서 배치, NPC 배치 | Seed + ConflictDesign + CastDesign | MapClueDesign |
| 5 | Showrunner | 통합, 톤 통일, WorldGeneration JSON | 모든 Design 결과 | WorldGeneration (draft) |
| 6 | ContinuityCop | 구조적 결함만 검사 (재미 죽이는 상식 교정 금지) | WorldGeneration draft | ValidationReport |
| 7 | SchemaEditor | JSON 스키마 준수, 필드 보정, 최종 패키징 | draft + ValidationReport | WorldGeneration (final) |
| 8 | Polish | 브리핑 문장 윤기, 문학적 마무리 | WorldGeneration (final) | WorldGeneration (polished) |

**사후 처리 (post-processing, 파이프라인 외부):** Polish 에이전트는 파이프라인 완료 후 선택적으로 실행된다 (프로바이더 가용 시).

## 파이프라인 흐름

```
┌─────────────────────────────────────────────────────────────────┐
│  Phase 1: Seed Generation  (병렬, 3~5초)                        │
│                                                                 │
│   ChaosMuse-A ──┐                                               │
│   ChaosMuse-B ──┼──► SeedProposal[]                            │
│   ChaosMuse-C ──┘                                               │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│  Phase 2: Seed Selection   (직렬, ~1초)                          │
│                                                                 │
│   SeedScorer (규칙 기반, AI 불필요)                               │
│                                                                 │
│   점수 가중치:                                                    │
│     재미/말문열림  45%   사회적긴장    20%                         │
│     비밀충돌      15%   방이동유발    10%                          │
│     solvability   5%   schema repair cost  5%                  │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│  Phase 3: Parallel Design                                       │
│                                                                 │
│   [빠른 모드]  Showrunner 1회 호출로 통합 처리                    │
│                                                                 │
│   [품질 모드]  ConflictEngineer                                  │
│                    │                                            │
│                    ▼                                            │
│               CastSecretMaster                                  │
│                    │                                            │
│                    ▼                                            │
│               MapClueSmith                                      │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│  Phase 4: Integration                                           │
│                                                                 │
│   Showrunner → WorldGeneration (draft)                          │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│  Phase 5: Validation + Repair                                   │
│                                                                 │
│   ContinuityCop → ValidationReport                              │
│        │                                                        │
│        ▼                                                        │
│   SchemaEditor → WorldGeneration (final)                        │
└─────────────────────────────────────────────────────────────────┘
```

## Seed 설계

### SeedProposal

```go
// internal/ai/orchestrator/seed.go

type SeedProposal struct {
    Hook            string   `json:"hook"`
    Genre           string   `json:"genre"`
    Setting         string   `json:"setting"`
    CoreConflict    string   `json:"coreConflict"`
    TwistPoints     []string `json:"twistPoints"`
    SocialTension   string   `json:"socialTension"`
    SecretPotential int      `json:"secretPotential"` // 1-10
    MovementDriver  string   `json:"movementDriver"`
}
```

### SeedScore

```go
// internal/ai/orchestrator/seed.go

type SeedScore struct {
    Fun              float64
    SocialTension    float64
    SecretConflict   float64
    MovementDriver   float64
    Solvability      float64
    SchemaRepairCost float64
    Total            float64
}

func ScoreSeed(seed SeedProposal) SeedScore
```

`ScoreSeed`는 규칙 기반이며 AI를 호출하지 않는다.

| 항목 | 가중치 | 측정 방법 |
|------|--------|-----------|
| Fun | 45% | Hook 길이/문장 수, TwistPoints 수, Genre 다양성 키워드 |
| SocialTension | 20% | SocialTension 필드 길이와 구체성 |
| SecretConflict | 15% | SecretPotential 값 (1-10 → 0-1 정규화) |
| MovementDriver | 10% | MovementDriver 필드 존재/구체성 |
| Solvability | 5% | CoreConflict에 해결 가능 힌트 존재 여부 |
| SchemaRepairCost | 5% | 제안 구조의 복잡도 역수 (단순할수록 높음) |

```go
func ScoreSeed(seed SeedProposal) SeedScore {
    score := SeedScore{}

    // Fun (45%): Hook 비어있지 않음 + TwistPoints 수 + Genre 키워드
    if seed.Hook != "" {
        score.Fun += 0.3
    }
    score.Fun += math.Min(float64(len(seed.TwistPoints))*0.1, 0.4)
    if containsAnyKeyword(seed.Genre, diverseGenreKeywords) {
        score.Fun += 0.3
    }
    score.Fun = math.Min(score.Fun, 1.0)

    // SocialTension (20%): 필드 길이와 구체성
    score.SocialTension = math.Min(float64(len(seed.SocialTension))/200.0, 1.0)

    // SecretConflict (15%): SecretPotential 1-10 → 0-1
    score.SecretConflict = float64(seed.SecretPotential-1) / 9.0

    // MovementDriver (10%): 필드 존재/구체성
    if seed.MovementDriver != "" {
        score.MovementDriver = math.Min(float64(len(seed.MovementDriver))/100.0, 1.0)
    }

    // Solvability (5%): CoreConflict에 해결 힌트
    if containsAnyKeyword(seed.CoreConflict, solvabilityKeywords) {
        score.Solvability = 1.0
    }

    // SchemaRepairCost (5%): 복잡도 역수
    complexity := len(seed.TwistPoints) + len(seed.CoreConflict)/50
    score.SchemaRepairCost = math.Max(1.0-float64(complexity)*0.1, 0.1)

    score.Total = score.Fun*0.45 +
        score.SocialTension*0.20 +
        score.SecretConflict*0.15 +
        score.MovementDriver*0.10 +
        score.Solvability*0.05 +
        score.SchemaRepairCost*0.05

    return score
}

func selectBestSeed(seeds []SeedProposal) SeedProposal {
    best := seeds[0]
    bestScore := ScoreSeed(best)
    for _, s := range seeds[1:] {
        if sc := ScoreSeed(s); sc.Total > bestScore.Total {
            best = s
            bestScore = sc
        }
    }
    return best
}
```

## 에이전트 역할 배정

```go
// internal/ai/orchestrator/roles.go

type AgentRole string
const (
    RoleChaosMuse        AgentRole = "chaos_muse"
    RoleConflictEngineer AgentRole = "conflict_engineer"
    RoleCastMaster       AgentRole = "cast_master"
    RoleMapSmith         AgentRole = "map_smith"
    RoleShowrunner       AgentRole = "showrunner"
    RoleContinuityCop    AgentRole = "continuity_cop"
    RoleSchemaEditor     AgentRole = "schema_editor"
    RolePolisher         AgentRole = "polisher"
)

type RoleAssignment struct {
    Role         AgentRole
    ProviderName string // ProviderRegistry key
    ModelID      string // 모델 ID override
}

// GetRoleAssignments는 available 프로바이더 맵을 참고하여 편성을 결정한다.
// 불가능한 프로바이더가 있으면 대체 편성을 반환한다.
func GetRoleAssignments(mode QualityMode, available map[string]bool) []RoleAssignment
```

`GetRoleAssignments`는 `available` 맵에서 각 프로바이더의 가용 여부를 확인하고, 비가용 프로바이더가 배정된 역할에 대해 fallback 편성을 반환한다. 모든 외부 프로바이더가 불가할 경우 OpenAI(GPT-5 mini) 단독 편성을 반환한다.

## 모드별 편성

### 빠른 제작 모드 (기본) — 목표 25~45초

빠른 모드에서는 ConflictEngineer, CastSecretMaster, MapClueSmith를 **Showrunner 1회 호출에 통합**한다. Phase 3 직렬 체인이 없으므로 전체 소요 시간이 단축된다.

| Agent | Model | 이유 |
|-------|-------|------|
| ChaosMuse-A | Gemini 2.5 Flash-Lite | 싸고 빠른 seed |
| ChaosMuse-B | Grok 4-fast (non-reasoning) | 튀는 아이디어 |
| ChaosMuse-C | DeepSeek chat | 싸게 구조 힌트 |
| Showrunner (통합) | GPT-5 mini | canon/JSON 안정성 |
| ContinuityCop | Gemini 2.5 Flash | 빠른 구조 검증 |
| SchemaEditor | GPT-5 mini | Showrunner 재사용 |
| Polish | Claude Haiku 4.5 | 브리핑 문장 윤기 |

예상 비용: ~$0.01~$0.02/스토리

### 품질 위주 모드 (프리미엄) — 목표 90~180초

품질 모드에서는 Phase 3 직렬 체인을 모두 실행한다. 각 에이전트가 이전 에이전트의 결과를 입력으로 받아 순차적으로 세계를 정교화한다.

| Agent | Model | 이유 |
|-------|-------|------|
| ChaosMuse-A | Grok 4 | 위험한 아이디어 |
| ChaosMuse-B | DeepSeek reasoner | 반대 논리 |
| ConflictEngineer | Gemini 2.5 Pro | 구조적 설계 |
| CastSecretMaster | Claude Opus 4.6 | 서사적 캐릭터 |
| MapClueSmith | Gemini 2.5 Pro | 구조+공간 |
| Showrunner | GPT-5 | canon/스키마 |
| ContinuityCop | Claude Sonnet 4.6 | 냉정한 비평 |
| SchemaEditor | GPT-5 | 스키마 보정 |
| Final polish | Claude Opus 4.6 | 엔딩/브리핑 문학성 |

예상 비용: ~$0.16~$0.25/스토리

## Health Check

세션 시작 시 등록된 프로바이더에 최소 요청을 발송하여 가용 여부를 확인한다. 실패한 프로바이더는 해당 세션에서 제외되고 `GetRoleAssignments`에 available 맵으로 전달된다.

```go
// internal/ai/orchestrator/registry.go

func (r *ProviderRegistry) HealthCheck(ctx context.Context) map[string]bool
```

Health check는 경량 요청(토큰 최소화)으로 수행한다. 타임아웃은 5초 (provider.md의 HealthCheck 구현과 동일). 실패 시 false를 기록하고 계속 진행한다.

## Timeout/Fallback 정책

| 상황 | 정책 |
|------|------|
| 개별 Muse 타임아웃 (10초) | 해당 seed 무시, 다른 seed로 진행 |
| 모든 Muse 타임아웃 | GPT-5 mini 단독 fallback으로 전체 생성 |
| Showrunner 타임아웃 (30초) | 1회 재시도 후 실패 → 에러 반환 |
| Validator 실패 | 기존 부분 재생성 로직 재사용 (WorldGenerator.RegeneratePartial) |
| API 에러 | 기존 retry 래퍼 재사용 (1s→2s→4s) |
| 모든 외부 프로바이더 불가 | OpenAI(GPT-5 mini) 단독 fallback |

## GenerateWorld 흐름

```go
// internal/ai/orchestrator/orchestrator.go

func (o *Orchestrator) GenerateWorld(
    ctx context.Context,
    playerCount int,
    themeHint string,
    onProgress ProgressFunc,
) (*WorldGeneration, error) {
    onProgress("seeds", "아이디어를 모으는 중...", 0.1)

    // Phase 1: 병렬 seed 생성
    seeds := o.generateSeeds(ctx, playerCount, themeHint)
    if len(seeds) == 0 {
        // fallback: 단일 모델로 전체 생성
        return o.fallbackGenerate(ctx, playerCount, themeHint, onProgress)
    }

    onProgress("select", "최적의 아이디어를 선택하는 중...", 0.2)

    // Phase 2: seed 선택 (규칙 기반, AI 불필요)
    best := selectBestSeed(seeds)

    onProgress("design", "세계를 설계하는 중...", 0.3)

    // Phase 3: 상세 설계 (모드에 따라 분기)
    var designs *DesignBundle
    if o.config.Mode == QualityModePremium {
        designs = o.designPremium(ctx, best, playerCount)
    } else {
        designs = nil // fast 모드에서는 Showrunner가 통합 처리
    }

    onProgress("integrate", "세계를 구축하는 중...", 0.6)

    // Phase 4: 통합
    draft, err := o.integrate(ctx, best, designs, playerCount)
    if err != nil {
        return nil, fmt.Errorf("세계 통합 실패: %w", err)
    }

    onProgress("validate", "검증하는 중...", 0.8)

    // Phase 5: 검증 + 수리
    final, err := o.validateAndRepair(ctx, draft, best, designs, playerCount)
    if err != nil {
        return nil, fmt.Errorf("검증/수리 실패: %w", err)
    }

    onProgress("polish", "마무리하는 중...", 0.9)

    // Polish (선택적 — 프로바이더 가용 시)
    if polisher := o.getPolisher(); polisher != nil {
        final = o.polish(ctx, final)
    }

    onProgress("complete", "완료!", 1.0)
    return final, nil
}
```

### generateSeeds (Phase 1)

```go
func (o *Orchestrator) generateSeeds(
    ctx context.Context,
    playerCount int,
    themeHint string,
) []SeedProposal {
    assignments := GetRoleAssignments(o.config.Mode, o.config.Registry.HealthCheck(ctx))
    museAssignments := filterByRole(assignments, RoleChaosMuse)

    type result struct {
        seed SeedProposal
        err  error
    }

    ch := make(chan result, len(museAssignments))
    for _, a := range museAssignments {
        go func(assignment RoleAssignment) {
            provider, err := o.config.Registry.Get(assignment.ProviderName)
            if err != nil {
                ch <- result{SeedProposal{}, err}
                return
            }
            seed, err := o.callMuse(
                ctx,
                provider,
                assignment.ModelID,
                playerCount,
                themeHint,
            )
            ch <- result{seed, err}
        }(a)
    }

    var seeds []SeedProposal
    deadline := time.After(10 * time.Second)
    for range museAssignments {
        select {
        case r := <-ch:
            if r.err == nil {
                seeds = append(seeds, r.seed)
            }
        case <-deadline:
            // 타임아웃: 받은 seed만 사용
            return seeds
        }
    }
    return seeds
}
```

### validateAndRepair (Phase 5)

ContinuityCop의 ValidationReport를 받아, StoryValidator와 동일한 재생성 루프를 재사용한다.

```go
func (o *Orchestrator) validateAndRepair(
    ctx context.Context,
    draft *WorldGeneration,
    seed SeedProposal,
    designs *DesignBundle,
    playerCount int,
) (*WorldGeneration, error) {
    // ContinuityCop: 구조적 결함 검사
    report, err := o.runContinuityCop(ctx, draft)
    if err != nil {
        return draft, nil // Cop 실패 시 draft 그대로 진행
    }
    if report.Valid {
        // SchemaEditor: 스키마 준수 및 필드 보정
        return o.runSchemaEditor(ctx, draft, report)
    }

    // critical issues 있음 → 부분 재생성 (FR-025: 최대 3회 시도)
    repaired := draft
    for attempt := 0; attempt < 3; attempt++ {
        repaired, err = o.worldGen.RegeneratePartial(ctx, repaired, report.Issues)
        if err != nil {
            continue
        }
        report, _ = o.runContinuityCop(ctx, repaired)
        if report.Valid {
            break
        }
    }

    if !report.Valid {
        // 3회 부분 재생성 실패 → 전체 재생성 1회 (FR-025)
        repaired, err = o.integrate(ctx, seed, designs, playerCount)
        if err != nil {
            return nil, fmt.Errorf("전체 재생성 실패: %w", err)
        }
        report, _ = o.runContinuityCop(ctx, repaired)
        if !report.Valid {
            // 전체 재생성도 실패 → 에러 반환 (세션 종료)
            return nil, fmt.Errorf("세계 생성 최종 실패: critical 오류 해소 불가 (FR-025)")
        }
    }

    return o.runSchemaEditor(ctx, repaired, report)
}
```

## 프롬프트 설계

### ChaosMuse System Prompt

```
당신은 텍스트 RPG의 '미친 아이디어 제조기'입니다.

[핵심 원칙]
- 재미와 충격이 최우선. 개연성은 신경 쓰지 마세요.
- 플레이어들이 서로 의심하고, 동맹을 맺고, 배신하게 만드는 설정을 만드세요.
- 장르에 구속되지 마세요. 호러, SF, 판타지, 현대극, 코미디 무엇이든.
- "이거 미쳤는데?" 싶은 설정이 좋은 설정입니다.

[출력 형식]
반드시 JSON으로 응답하세요.
```

### Showrunner System Prompt

```
당신은 텍스트 RPG의 쇼러너(총괄 연출)입니다.
여러 작가가 제안한 아이디어들을 하나의 완결된 게임 세계로 통합합니다.

[핵심 원칙]
- 톤과 세계관의 일관성을 유지하세요
- 각 작가의 장점을 살리되, 모순되는 부분은 자연스럽게 해소하세요
- 정보 비대칭(공개/반공개/비공개)을 반드시 설계하세요
- 모든 플레이어에게 의미 있는 행동 경로를 부여하세요
- 출력은 반드시 WorldGeneration JSON 스키마를 따르세요

[구조 요구사항]
- 방 수 >= 플레이어 수 + 2
- 단서 수 >= 플레이어 수 × 2
- 반공개 정보 >= 1쌍
- 모든 플레이어에게 personalGoals >= 1개
- timeout fallback 종료 조건 필수
```

### ContinuityCop System Prompt

```
당신은 텍스트 RPG의 '구조경찰'입니다.
세계 설정의 구조적 결함만 찾으세요.

[검사 대상]
- 맵 연결성: 고립된 방 없는지
- 단서 배치: 존재하는 방에 배치되었는지
- NPC 위치: 유효한 방에 있는지
- 종료 조건: 달성 가능한지, timeout fallback 있는지
- 역할 완전성: 모든 플레이어에 목표/비밀 있는지

[검사하지 않는 것]
- "현실에서 이건 불가능해" 같은 상식 교정 → 하지 마세요
- "이 캐릭터가 왜 이러지?" 같은 동기 의문 → 재미를 죽입니다
- 문학적 품질 → 당신의 관할이 아닙니다

[출력 형식]
{"valid": bool, "issues": [{"category": "...", "severity": "critical|warning", "message": "...", "affectedIDs": [...]}]}
```

## 런타임 AI와의 관계

세계 생성과 달리, **런타임 AI(GM/NPC/행동평가/종료판정/엔딩)는 Orchestrator를 사용하지 않는다.** 런타임은 단일 `runtimeProvider`를 통해 기존과 동일하게 동작한다.

Orchestrator는 게임 시작 전 세계 생성 단계에만 관여한다. 생성 완료 후에는 `WorldGeneration`을 반환하고 역할이 종료된다.

### 런타임 권장 프로바이더

| 역할 | 빠른 제작 | 품질 위주 |
|------|-----------|-----------|
| GM narrator | Gemini Flash / GPT-5 mini | GPT-5 |
| NPC dialogue | Claude Haiku 4.5 | Claude Sonnet 4.6 |
| Action evaluator | GPT-5 mini | GPT-5 |
| End judge | GPT-5 mini | GPT-5 |
| Ending writer | Claude Haiku 4.5 | Claude Sonnet 4.6 / Opus 4.6 |
