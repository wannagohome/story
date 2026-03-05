# AI Layer (`internal/ai/`)

서버에서만 사용되는 AI 통합 계층. 모든 AI 호출은 이 계층을 통해 이루어짐.

## 모듈 목록

| 모듈 | 패키지 | 핵심 책임 |
|------|--------|-----------|
| [AIProvider & Registry](./provider.md) | `internal/ai/provider/` | 멀티 프로바이더 추상화 + 레지스트리 (OpenAI, Anthropic, Gemini, Grok, DeepSeek) |
| [Orchestrator](./orchestrator.md) | `internal/ai/orchestrator/` | 멀티 모델 세계 생성 파이프라인 (8종 논리 에이전트) |
| [StoryBible](./story-bible.md) | `internal/ai/bible/` | concept/PRD → 3종 캐시 압축 (토큰 절약) |
| [WorldGenerator](./world-generator.md) | `internal/ai/worldgen/` | 세계 생성 (Orchestrator 래퍼) + 결과 변환 |
| [StoryValidator](./story-validator.md) | `internal/ai/validator/` | 구조적 결함 검사 (규칙 기반) |
| [GMEngine](./gm-engine.md) | `internal/ai/gm/` | GM 서술 생성, 긴장감 조율 |
| [NPCEngine](./npc-engine.md) | `internal/ai/npc/` | NPC 대화 생성, 퍼소나 유지 |
| [ActionEvaluator](./action-evaluator.md) | `internal/ai/evaluator/` | /examine, /do 결과 생성 |
| [EndJudge](./end-judge.md) | `internal/ai/judge/` | 종료 판정, 엔딩 생성 |

## AILayer Facade (`internal/ai/ailayer.go`)

서버가 AI 계층을 사용하는 **단일 진입점**. 내부 모듈을 조합.

> **멀티 모델 아키텍처:** 세계 생성은 Orchestrator(멀티 모델 파이프라인)를 사용하고,
> 런타임 AI(GM/NPC/행동평가/종료판정)는 단일 runtimeProvider를 사용.
> 이 분리를 통해 세계 생성의 창의성과 런타임의 응답 속도를 모두 확보.

```go
type QualityMode string
const (
    QualityModeFast    QualityMode = "fast"    // 번개 집필 (25~45초, ~$0.02)
    QualityModePremium QualityMode = "premium" // 시네마틱 집필 (90~180초, ~$0.20)
)

type AILayerConfig struct {
    QualityMode     QualityMode
    ProviderConfigs map[string]ProviderConfig // 멀티 프로바이더 레지스트리용
    RuntimeProvider string                    // 런타임 AI용 프로바이더 이름 (기본: "openai")
}

type AILayer struct {
    registry        *ProviderRegistry   // 멀티 프로바이더 레지스트리
    orchestrator    *Orchestrator       // 세계 생성용 멀티 모델 파이프라인
    runtimeProvider AIProvider          // 런타임 AI용 (단일 프로바이더)
    worldGenerator  *WorldGenerator
    storyValidator  *StoryValidator
    gmEngine        *GMEngine
    npcEngine       *NPCEngine
    actionEvaluator *ActionEvaluator
    endJudge        *EndJudge
    endingGenerator *EndingGenerator
}

func NewAILayer(config AILayerConfig) (*AILayer, error) {
    // 1. 프로바이더 레지스트리 초기화
    registry, err := NewProviderRegistry(config.ProviderConfigs)
    if err != nil {
        return nil, err
    }

    // 2. Health check — 가용 프로바이더 확인
    available := registry.HealthCheck(context.Background())

    // 3. 런타임 프로바이더 선택
    runtimeProvider, err := registry.Get(config.RuntimeProvider)
    if err != nil {
        return nil, fmt.Errorf("런타임 프로바이더 '%s' 사용 불가: %w", config.RuntimeProvider, err)
    }

    // 4. Story Bible 로드/생성
    bibleCompressor := NewStoryBibleCompressor(runtimeProvider)
    bible, err := bibleCompressor.GetOrCreate(context.Background())
    if err != nil {
        return nil, fmt.Errorf("Story Bible 생성 실패: %w", err)
    }

    // 5. Orchestrator 초기화 (가용 프로바이더는 GenerateWorld 호출 시 HealthCheck로 내부 결정)
    orchestrator := NewOrchestrator(OrchestratorConfig{
        Mode:     config.QualityMode,
        Registry: registry,
        Bible:    bible,
    })

    return &AILayer{
        registry:        registry,
        orchestrator:    orchestrator,
        runtimeProvider: runtimeProvider,
        worldGenerator:  NewWorldGenerator(orchestrator),
        storyValidator:  NewStoryValidator(),
        gmEngine:        NewGMEngine(runtimeProvider),
        npcEngine:       NewNPCEngine(runtimeProvider),
        actionEvaluator: NewActionEvaluator(runtimeProvider),
        endJudge:        NewEndJudge(runtimeProvider),
        endingGenerator: NewEndingGenerator(runtimeProvider),
    }, nil
}

// ── 세계 생성 (멀티 모델 Orchestrator 파이프라인) ──
func (a *AILayer) GenerateWorld(ctx context.Context, playerCount int, themeHint string, onProgress func(step string, message string, progress float64)) (*World, error)

// ── 게임 중 AI 호출 (단일 runtimeProvider) ──
func (a *AILayer) EvaluateExamine(ctx context.Context, gameCtx *GameContext, room *Room, target string) (*EvaluationResult, error)
func (a *AILayer) EvaluateAction(ctx context.Context, gameCtx *GameContext, playerID string, action string) (*EvaluationResult, error)
func (a *AILayer) ChatWithNPC(ctx context.Context, npc *NPC, playerID string, msg string, gameCtx *GameContext) (*NPCResponse, error)
func (a *AILayer) GetGMNarration(ctx context.Context, gameCtx *GameContext, trigger GameEvent) (*NarrationEvent, error)
func (a *AILayer) CheckGMPacing(ctx context.Context, gameCtx *GameContext) (*StoryEventEvent, error)

// ── 종료 판정 (단일 runtimeProvider) ──
func (a *AILayer) JudgeEndCondition(ctx context.Context, cond *EndCondition, gameCtx *GameContext) (bool, string, error)
func (a *AILayer) GenerateEndings(ctx context.Context, gameCtx *GameContext, endReason string) (*GameEndData, error)
```

## 모듈 의존성

```
                    ┌──────────────┐
                    │   AILayer    │ (Facade)
                    │  (ailayer)   │
                    └──────┬───────┘
                           │
          ┌────────────────┼────────────────┐
          │                │                │
          ▼                ▼                ▼
    ┌───────────┐   ┌────────────┐   ┌───────────┐
    │Orchestrator│   │런타임 모듈들│   │  Story    │
    │(세계 생성) │   │GM/NPC/Act/ │   │ Validator │
    └─────┬─────┘   │EndJudge/   │   │(규칙 기반)│
          │         │Ending      │   └───────────┘
          │         └─────┬──────┘
          │               │
    ┌─────┤               ▼
    │     │        runtimeProvider
    ▼     │         (단일 AIProvider)
 Story    │
 Bible    │
 (캐시)   │
          ▼
   ProviderRegistry
          │
   ┌──────┼──────┬──────┬──────┬──────┐
   │      │      │      │      │      │
OpenAI Anthropic Gemini  Grok  Deep
Provider Provider Provider     Seek
```

**구조 설명:**
- **Orchestrator**: 세계 생성 시 ProviderRegistry에서 역할별 최적 모델을 가져와 멀티 모델 파이프라인 실행
- **런타임 모듈**: 게임 중 AI 호출은 단일 runtimeProvider 사용 (기존 인터페이스 유지)
- **StoryValidator**: AI를 호출하지 않음 — 규칙 기반 검증
- **StoryBible**: concept/PRD를 캐시 압축하여 프롬프트 토큰 절약

## AI 토큰 관리

### 세계 생성 (Orchestrator 멀티 모델 파이프라인)

| 단계 | 모델 (빠른 제작) | 모델 (품질 위주) | 호출 빈도 |
|------|-----------------|-----------------|-----------|
| Seed 생성 ×3 | Flash-Lite, Grok fast, DeepSeek | Grok 4, DeepSeek reasoner | 게임당 1회 |
| Showrunner 통합 | GPT-5 mini | GPT-5 | 게임당 1회 |
| ContinuityCop 검증 | Gemini Flash | Claude Sonnet 4.6 | 게임당 1~2회 |
| SchemaEditor 패치 | GPT-5 mini | GPT-5 | 게임당 0~1회 |
| Polish | Claude Haiku 4.5 | Claude Opus 4.6 | 게임당 1회 |

**예상 비용:** 빠른 제작 ~$0.01~$0.02, 품질 위주 ~$0.16~$0.25 / 스토리

### 런타임 AI (단일 runtimeProvider)

| AI 호출 유형 | max_tokens | temperature | 호출 빈도 |
|-------------|-----------|-------------|-----------|
| GM 서술 | 500 | 0.9 | 필요 시 |
| NPC 대화 | 500 | 0.8 | 플레이어 요청 시 |
| /examine | 500 | 0.7 | 플레이어 요청 시 |
| /do | 500 | 0.8 | 플레이어 요청 시 |
| 종료 판정 | 300 | 0.3 | 이벤트 발생 시 |
| 엔딩 생성 | 3000 | 0.9 | 게임당 1회 |
