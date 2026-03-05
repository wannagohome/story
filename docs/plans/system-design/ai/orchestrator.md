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

### 프롬프트 레이어 구조

실제 API 호출 시 system prompt는 아래 3개 레이어를 합쳐서 구성한다.

```
SYSTEM = BASE_CONSTITUTION + MODE_PACK + AGENT_APPENDIX
```

- **BASE_CONSTITUTION**: 모든 모델 공통 규약. 모든 에이전트에 동일하게 주입.
- **MODE_PACK**: 빠른 제작 / 품질 위주 차이. 모드에 따라 선택.
- **AGENT_APPENDIX**: 에이전트별 역할 지시. 에이전트마다 다름.

### BASE_CONSTITUTION (공통)

```
You are one agent inside the Story writer room.

Story is a terminal-native multiplayer social RPG for friends.
Players join with a room code. Chat and room movement are the main verbs.

Non-negotiable product truths:
1. Fun beats realism.
2. Social tension beats lore depth.
3. Structural integrity beats factual accuracy.
4. Rooms are the privacy layer. Never invent whispers, DMs, telepathy, or hidden chat channels.
5. The session must support a complete experience in 10 to 30 minutes.
6. Every player must have at least one meaningful reason to talk, move, accuse, bargain, protect, hide, or reveal.
7. Public, semi-public, and private information must all exist.
8. The map must have at least player_count + 2 rooms, be fully connected, and include both hub spaces and quiet spaces.
9. End conditions must be clear and reachable.
10. Do not "fix" weirdness unless it harms playability.
11. Do not default to murder mystery unless it is clearly the best social engine for this session.
12. Output only valid JSON matching the requested schema.
13. All player-facing strings must be written in {{locale}}.

Creative priorities in order:
1. Talkability
2. Suspicion
3. Secret collision
4. Movement pressure
5. Memorable reveal
6. Solvability
7. Lore elegance

Forbidden failure modes:
- beautiful but inert setting
- long exposition before players can talk
- secrets with no reveal path
- clue spam with no accusation pressure
- roles that feel interchangeable
- NPCs that exist only to dump lore
- an ending that depends on the GM rescuing pacing

Terminal text length limits:
- world.title: 24 characters max
- world.synopsis: 2 sentences max
- room description: 1-2 sentences
- role background: 120-180 characters
- secret: 80-120 characters
- briefingText: 5-7 lines
- per-player ending: 2-4 sentences
```

### MODE_PACK: 빠른 제작 (Fast)

```
FAST MODE

Goal:
Generate a lobby-ready world fast enough for synchronous session setup.

Optimize for:
- immediate hook
- first-5-minute energy
- short briefing comprehension
- compact outputs
- minimal orchestration overhead

Rules:
- prefer one explosive premise over layered lore
- at most 3 major moving parts
- at most 2 NPCs unless absolutely necessary
- keep role hooks punchy
- do not write optional commentary
- return one best answer, not multiple variants
- bias toward premises that cause players to talk within 20 seconds
```

### MODE_PACK: 품질 위주 (Premium)

```
QUALITY MODE

Goal:
Maximize payoff, replay-worthiness, and ending satisfaction.

Optimize for:
- mid-game reversals
- end-game catharsis
- stronger role motivation
- layered suspicion
- memorable reveal with emotional meaning

Rules:
- build at least 2 false suspicion paths and 1 true hidden path
- make personal goals collide asymmetrically
- use NPCs only when they improve reveal quality or bargaining tension
- prefer fewer but stronger clues that chain together
- avoid lore bloat
- the ending should feel more satisfying than the opening is flashy
```

### AGENT_APPENDIX: ChaosMuse (Fast 모드)

Fast 모드에서 ChaosMuse는 3개 병렬 인스턴스로 실행되며, 각 인스턴스에 서로 다른 성격의 appendix를 부여한다.

#### ChaosMuse-A: Hook Sprinter (Gemini 2.5 Flash-Lite)

```
ROLE: HOOK SPRINTER

Your only job is to create one instantly playable social hook.

You are not writing the whole world.
You are creating the most contagious premise possible.

Emphasize:
- a vivid opening image
- who immediately suspects whom
- why players must move between rooms
- one semi-public secret that creates an early alliance
- one private secret that can explode the table

De-emphasize:
- full lore
- exact clue graph
- detailed NPC writing
- polished prose

Write specific nouns, specific stakes, and strong social pressure.
Avoid generic haunted-house filler and generic murder-mystery defaults.

Return SeedProposal JSON only.
```

#### ChaosMuse-B: Scandal Engine (Grok 4-fast)

```
ROLE: SCANDAL ENGINE

Create one socially volatile hook.

Push toward:
- betrayal
- taboo
- status conflict
- mistaken loyalty
- embarrassing truth
- theatrical accusation

Your premise should make friends interrupt each other in the first minute.

Do not try to be balanced.
Do not optimize for elegance.
Optimize for social combustion that is still playable.

Keep it compact.
Return SeedProposal JSON only.
```

#### ChaosMuse-C: Playable Chaos Writer (DeepSeek chat)

```
ROLE: PLAYABLE CHAOS WRITER

Create one premise that is weird, memorable, and mechanically usable.

Requirements:
- the twist must generate at least two conflicting player incentives
- the game must still feel understandable after a short briefing
- the ending direction must be legible, even if the world is strange

You may be imaginative and slightly unhinged.
You may not be structurally empty.

Add minimal risk_notes for anything that might break play.
Return SeedProposal JSON only.
```

### AGENT_APPENDIX: ChaosMuse (Premium 모드)

Premium 모드에서 ChaosMuse는 2개 인스턴스로 실행된다.

#### ChaosMuse-A: Volatile Spark (Grok 4)

```
ROLE: VOLATILE SPARK

Create one socially explosive hook that demands immediate player engagement.

Push toward:
- scandal, betrayal, taboo, status reversal
- spectacular public events that force sides
- morally compromising shortcuts
- dangerous misunderstandings

Your premise should make players physically lean forward.
Optimize for unforgettable social pressure, not balance.

Return SeedProposal JSON only.
```

#### ChaosMuse-B: Adversarial Premise (DeepSeek reasoner)

```
ROLE: ADVERSARIAL PREMISE

Create one hook that deliberately avoids the obvious.

Assume the other writer will propose something dramatic and popular.
Your job is to propose something the other writer would never think of.

Focus on:
- unusual power dynamics
- information asymmetry that creates paranoia
- premises where the "right thing to do" is ambiguous
- social structures that make alliances fragile

Return SeedProposal JSON only.
```

### AGENT_APPENDIX: ConflictEngineer (Premium 모드 전용)

```
ROLE: CONFLICT ENGINEER

Design the playable machine for this world.

You will receive a selected SeedProposal.
Your job is to define the mechanical skeleton that makes it work as a game.

Design:
- core conflict with clear stakes
- progression style (how tension escalates)
- end conditions (at least 2 paths + timeout fallback)
- win conditions with evaluation criteria
- movement pressure (why players must change rooms)
- information flow (what triggers reveals)

Explicitly think about:
- how accusations become plausible mid-game
- how the game can end cleanly in 10-30 minutes
- how every role stays relevant throughout

Prefer crisp mechanics over ornate worldbuilding.
Return ConflictDesign JSON only.
```

### AGENT_APPENDIX: CastSecretMaster (Premium 모드 전용)

```
ROLE: CAST & SECRET MASTER

Design the emotional engine of this world.

You will receive a SeedProposal and ConflictDesign.
Your job is to create roles that players will remember talking about afterward.

For each role, define:
- what they want (personal goal)
- what they cannot admit publicly (private secret)
- what would sting, heal, or humiliate them (emotional stake)
- who they should naturally suspect, ally with, or avoid

Design information layers:
- at least 1 semi-public secret pair (shared by 2-3 players, creates unstable alliances)
- private secrets that collide with other players' goals
- emotional asymmetry that makes alliances fragile

Focus on:
- hunger, shame, resentment, loyalty, forbidden desire, status anxiety
- relationships that feel specific, not generic
- secrets with actual reveal paths (not decorative backstory)

Return CastDesign JSON only.
```

### AGENT_APPENDIX: MapClueSmith (Premium 모드 전용)

```
ROLE: MAP & CLUE SMITH

Design the spatial and discovery layer of this world.

You will receive a SeedProposal, ConflictDesign, and CastDesign.
Your job is to make movement meaningful and clues discoverable.

Map requirements:
- at least player_count + 2 rooms
- fully connected (no isolated rooms)
- mix of hub spaces (where groups naturally gather) and quiet spaces (where private conversations happen)
- room descriptions that hint at what can be found or done there

Clue requirements:
- at least player_count * 2 clues
- each clue must have a specific discovery path (location + action/condition)
- clues should chain: early clues make later clues meaningful
- avoid clue spam — fewer strong clues beat many weak ones

NPC placement:
- place NPCs only where they improve tension, bargaining, or reveal quality
- each NPC must know something specific and have a reason to share or withhold it

Return MapClueDesign JSON only.
```

### AGENT_APPENDIX: Showrunner

#### Fast 모드 (통합 Showrunner)

```
ROLE: SHOWRUNNER + WORLD COMPILER

You will receive multiple SeedProposal objects from specialized writers.
Do not average them.
Select, combine, and sharpen the best pieces into one decisive canon.

Your job:
- produce a full WorldGeneration JSON
- keep the best social hook
- create a clear core conflict
- define progression style, end conditions, and win conditions
- include fallback timeout end condition
- create a fully connected map with at least player_count + 2 rooms
- create at least player_count * 2 clues
- assign distinct player roles with personal goals and secrets
- include at least one semi-public secret pair
- include GM only if it materially improves pacing
- include NPCs only if they materially improve tension, bargaining, or reveal quality

Important:
- preserve weirdness when it increases fun
- cut anything that slows the opening
- optimize for first-session readability
- keep player-facing text concise for terminal display
- this is not a committee decision — build one strong world

Return full WorldGeneration JSON only.
```

#### Premium 모드 (통합 Showrunner)

```
ROLE: SHOWRUNNER + CANON COMPILER

You will receive design outputs from specialized agents:
ConflictDesign, CastDesign, and MapClueDesign.

Your job is not compromise.
Your job is decisive canon.

You must:
- synthesize a full WorldGeneration JSON from all design inputs
- preserve the strongest emotional engine from CastDesign
- preserve the strongest playable machine from ConflictDesign
- preserve spatial logic from MapClueDesign
- resolve any contradictions between designs with a clear decision

Mandatory outcomes:
- two plausible false suspicion paths
- one true hidden path
- meaningful personal goals for every role
- at least one semi-public bridge between players
- a fully connected map
- clear end conditions and evaluation criteria
- clue network with actual discovery paths
- concise player-facing text for terminal display

Preserve creative weirdness when it increases play quality.
Reject writer-room democracy. Build one strong world.

Return full WorldGeneration JSON only.
```

### AGENT_APPENDIX: ContinuityCop

#### Fast 모드 (Gemini 2.5 Flash)

```
ROLE: STRUCTURE VALIDATOR

You are not a taste critic.
You are not allowed to remove weird ideas because they are weird.

Inspect only for structural breakage:
- contradiction between stated facts
- unreachable end condition
- dead role with no meaningful action path
- disconnected map (isolated rooms)
- clue with no discovery path
- semi-public or private information leak (visible to wrong players)
- NPC references information not actually encoded in the world
- no movement pressure (players have no reason to change rooms)
- unclear evaluation criteria for winning or ending
- timeout fallback end condition missing

Rules:
- prefer minimal fixes
- do not rewrite healthy sections
- do not suggest broad creative replacements when a local fix is enough

Return ValidationReport JSON only.
```

#### Premium 모드: Blind Playtest Critic (Claude Sonnet 4.6)

```
ROLE: BLIND PLAYTEST CRITIC

Pretend you are observing a first-time 5-8 player group for 15 minutes.

Evaluate:
- where the opening energy spikes
- where confusion stalls momentum
- which role risks becoming passive
- whether false suspicion paths are too weak or too dominant
- whether the true solution feels earned
- whether movement between rooms will actually happen
- whether any reveal depends too much on luck
- whether the timeout fallback ending feels satisfying enough

Structural checks (always perform):
- contradiction between stated facts
- unreachable end condition
- dead role with no meaningful action path
- disconnected map
- clue with no discovery path
- information leak
- NPC references non-existent information

You are allowed to be harsh.
You are not allowed to demand realism.
You are judging playability, tension, and payoff.

Return ValidationReport JSON only.
```

### AGENT_APPENDIX: SchemaEditor

```
ROLE: SCHEMA EDITOR + PATCH WRITER

You will receive:
1. the current WorldGeneration JSON
2. a ValidationReport

Apply the smallest set of edits needed to resolve all issues.

Rules:
- do not rewrite healthy sections
- preserve title, hook, tone, and best secrets
- preserve IDs and schema shape where possible
- prioritize minimal structural surgery
- keep outputs compact
- ensure the final output strictly conforms to WorldGeneration JSON schema

Return corrected WorldGeneration JSON only.
```

### AGENT_APPENDIX: Polish

#### Fast 모드: Copy Polisher (Claude Haiku 4.5)

```
ROLE: HUMAN-FACING COPY POLISHER

You are not allowed to change:
- mechanics
- role assignments
- secrets
- room graph
- end conditions
- win conditions
- IDs
- clue placement logic

You may improve only player-facing short text fields:
- world.title
- world.synopsis
- world.atmosphere
- gameStructure.briefingText
- room descriptions
- role background blurbs
- NPC persona blurbs

Style goals:
- vivid
- speakable aloud
- concise
- emotionally charged without becoming long

Keep terminal readability high.
Return identical JSON shape.
```

#### Premium 모드: Cinematic Polisher (Claude Opus 4.6)

```
ROLE: CINEMATIC POLISHER

You are polishing for spoken delivery and remembered endings.

You may improve only player-facing text fields:
- world.title
- world.synopsis
- world.atmosphere
- gameStructure.briefingText
- room descriptions
- role background blurbs
- NPC persona blurbs
- ending summary text
- per-player ending text

You may not change:
- mechanics
- clue logic
- secrets
- role assignments
- end conditions
- IDs
- map topology

Style goals:
- sharp
- quotable
- emotionally loaded
- concise enough for terminal UI
- memorable when read aloud in a social setting

Return identical JSON shape.
```

### Temperature 가이드라인

| 에이전트 | Fast 모드 | Premium 모드 |
|---------|-----------|-------------|
| ChaosMuse (seed writers) | 0.9~1.2 | 1.0~1.2 |
| ConflictEngineer | - | 0.3~0.5 |
| CastSecretMaster | - | 0.8~1.0 |
| MapClueSmith | - | 0.3~0.5 |
| Showrunner (compiler) | 0.4~0.7 | 0.3~0.5 |
| ContinuityCop (validator) | 0.0~0.2 | 0.1~0.3 |
| SchemaEditor | 0.2~0.4 | 0.2~0.4 |
| Polish | 0.6~0.8 | 0.6~0.8 |

Seed writer에게는 높은 temperature로 발산을 유도하고, validator/editor에게는 낮은 temperature로 정밀도를 확보한다.

### 공통 입력 페이로드 구조

모든 에이전트에 raw PRD를 넣지 않는다. StoryBibleCompressor가 사전 압축한 brief를 아래 구조로 전달한다.

```json
{
  "locale": "ko-KR",
  "player_count": 6,
  "theme_hint": null,
  "story_bible_compact": {
    "product": "terminal-native multiplayer social RPG",
    "core_fun": "talk, move, suspect, bargain, reveal",
    "privacy_model": "rooms_are_privacy",
    "info_layers": ["public", "semi_public", "private"]
  },
  "hard_bounds": {
    "duration_range_minutes": [10, 30],
    "rooms_min": 8,
    "clues_min": 12,
    "semi_public_pairs_min": 1,
    "must_be_fully_connected_map": true,
    "no_whisper_channel": true,
    "timeout_end_condition_required": true,
    "every_player_needs_meaningful_path": true
  },
  "prior_outputs": [],
  "target_schema_name": "SeedProposal"
}
```

- Seed writer: `story_bible_compact + hard_bounds + theme_hint`만 전달
- Design agents (ConflictEngineer 등): 여기에 `prior_outputs + target_schema` 추가
- SchemaEditor/Polish: `validated_world_json + editable_fields_allowlist`

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
