# StoryBible Compressor (`internal/ai/bible/`)

## 책임

concept.md + prd.md를 3종 캐시로 압축. **배포 시 1회 / 문서 변경 시 1회**만 실행.
전체 concept/PRD를 매번 프롬프트에 넣지 않고, 역할별 요약본을 캐시하여 토큰 절약.

## 의존하는 모듈

AIProvider (단일 — GPT-5 mini 권장)

## 인터페이스

```go
// internal/ai/bible/story_bible.go

type StoryBible struct {
    Creative   CreativeBrief        `json:"creative"`
    Schema     SchemaBrief          `json:"schema"`
    Validation ValidationChecklist  `json:"validation"`
    SourceHash string               `json:"sourceHash"` // SHA256 of source docs
}

type CreativeBrief struct {
    CorePhilosophy string   `json:"corePhilosophy"`  // 핵심 설계 철학
    TonePrinciples []string `json:"tonePrinciples"`   // 톤 원칙들
    Taboos         []string `json:"taboos"`           // 금기사항
    InfoAsymmetry  string   `json:"infoAsymmetry"`    // 정보 비대칭 원칙
    GameTimeRange  string   `json:"gameTimeRange"`    // "10~30분"
}

type SchemaBrief struct {
    RequiredTopLevelKeys []string          `json:"requiredTopLevelKeys"`
    RequiredFieldRules   []SchemaRule      `json:"requiredFieldRules"`
    MinimumCounts        map[string]string `json:"minimumCounts"` // e.g. "rooms": "playerCount + 2"
}

type SchemaRule struct {
    Path        string `json:"path"`
    Constraint  string `json:"constraint"`
    Description string `json:"description"`
}

type ValidationChecklist struct {
    StructuralRules []ValidationRule `json:"structuralRules"`
}

type ValidationRule struct {
    ID       string `json:"id"`
    Category string `json:"category"` // "map" | "clue" | "npc" | "end_condition" | "player_path" | "information"
    Rule     string `json:"rule"`
}

func NewStoryBible(provider AIProvider) *StoryBibleCompressor

type StoryBibleCompressor struct {
    provider  AIProvider
    cacheDir  string // ~/.story/bible-cache/
}

func (c *StoryBibleCompressor) GetOrCreate(ctx context.Context) (*StoryBible, error)
func (c *StoryBibleCompressor) Invalidate() error
```

## 캐시 산출물

| 캐시 | 용도 | 토큰 목표 |
|------|------|---------|
| CreativeBrief | Muse/Writer용 요약 (톤, 원칙, 금기사항) | ~800 토큰 |
| SchemaBrief | Showrunner/SchemaEditor용 (JSON 스키마 요약, 필수 필드) | ~600 토큰 |
| ValidationChecklist | ContinuityCop용 (구조 검증 규칙 목록) | ~400 토큰 |

## 캐시 전략

- 파일 해시 기반: concept.md + prd.md의 SHA256 → 캐시 키
- 캐시 저장: `~/.story/bible-cache/{hash}.json`
- 서버 시작 시 캐시 존재하면 로드, 없으면 생성
- 생성 모델: GPT-5 mini (저렴, 빠름, 요약 능력 충분)

## GetOrCreate 흐름

```
GetOrCreate(ctx)
    │
    ├── sourceHash 계산 (concept.md + prd.md SHA256)
    │
    ├── 캐시 파일 존재 확인: ~/.story/bible-cache/{hash}.json
    │     ├── 존재 → 로드 후 반환
    │     └── 미존재 → 생성
    │
    ├── concept.md + prd.md 읽기
    │
    ├── provider.GenerateStructured(ctx, StructuredRequest{
    │     SystemPrompt: bibleCompressorSystemPrompt,
    │     UserPrompt:   sourceDocuments,
    │     Temperature:  0.3,  // 결정적 요약
    │     MaxTokens:    2000,
    │   })
    │
    ├── json.Unmarshal → StoryBible
    │
    ├── 캐시 파일 저장
    │
    └── return bible, nil
```

## 프롬프트 설계

### System Prompt

```
당신은 게임 설계 문서 요약 전문가입니다.

주어진 concept.md와 prd.md를 분석하여 3종 요약본을 생성하세요:

1. creative: 창작자(작가)를 위한 요약
   - 게임의 핵심 철학 (1~2문장)
   - 톤 원칙 (5개 이내)
   - 금기사항 (하지 말아야 할 것)
   - 정보 비대칭 원칙
   - 게임 시간 범위

2. schema: JSON 컴파일러를 위한 요약
   - WorldGeneration 최상위 필수 키
   - 각 필드의 필수 규칙 (path + constraint 형태)
   - 최소 개수 규칙 (방 수, 단서 수 등)

3. validation: 구조 검증자를 위한 체크리스트
   - 카테고리별 검증 규칙 (ID + rule 형태)
   - 규칙 기반으로 자동 검증 가능한 항목만 포함

반드시 JSON으로 응답하세요.
```

## 소스 문서 임베딩

StoryBibleCompressor는 빌드 시 concept.md와 prd.md를 Go의 `embed` 패키지로 바이너리에 포함.

```go
import "embed"

//go:embed docs/concept.md
var conceptDoc string

//go:embed docs/prd.md
var prdDoc string
```

이를 통해 런타임에 파일 시스템 의존 없이 문서에 접근.
