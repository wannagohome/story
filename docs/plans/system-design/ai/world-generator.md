# WorldGenerator (`internal/ai/worldgen/`)

## 책임

게임 시작 시 세계를 생성. 플레이어 수에 맞는 세계, 맵, 역할, 단서를 AI에게 요청하고 Go struct로 검증.

> **멀티 모델 오케스트레이션:** 세계 생성은 [Orchestrator](./orchestrator.md)를 통해 멀티 모델 파이프라인으로 실행됨. WorldGenerator는 Orchestrator의 얇은 래퍼로, AILayer가 Orchestrator를 직접 사용.

## 의존하는 모듈

[Orchestrator](./orchestrator.md) (멀티 모델 파이프라인), [StoryBible](./story-bible.md) (캐시)

## 인터페이스

```go
// internal/ai/worldgen/world_generator.go

type WorldGenerator struct {
    orchestrator *Orchestrator
}

func NewWorldGenerator(orchestrator *Orchestrator) *WorldGenerator {
    return &WorldGenerator{orchestrator: orchestrator}
}

type ProgressFunc func(step string, message string, progress float64)

func (wg *WorldGenerator) Generate(
    ctx context.Context,
    playerCount int,
    themeHint string,
    onProgress ProgressFunc,
) (*World, error)
```

## 생성 흐름

> **핵심 변경:** Generate()는 더 이상 단일 AI 호출이 아니라 [Orchestrator](./orchestrator.md)의 멀티 모델 파이프라인을 호출.
> 진행 상황(onProgress)은 Orchestrator 내부에서 단계별로 호출됨.

```
Generate(ctx, playerCount, themeHint, onProgress)
    │
    ├── orchestrator.GenerateWorld(ctx, playerCount, themeHint, onProgress)
    │     ├── Phase 1: Seeds (병렬) — 3~5초
    │     ├── Phase 2: Score & Select — ~1초
    │     ├── Phase 3: Design (모드별) — 5~15초
    │     ├── Phase 4: Integration — 5~10초
    │     └── Phase 5: Validate + Repair — 2~5초
    │
    ├── json.Unmarshal → WorldGeneration
    │   Validate() 실패 → Orchestrator 내부에서 처리
    │
    ├── playerCount 일치 확인
    │
    └── return transformToWorld(result, playerCount)
```

구현:

```go
func (wg *WorldGenerator) Generate(
    ctx context.Context,
    playerCount int,
    themeHint string,
    onProgress ProgressFunc,
) (*World, error) {
    // Orchestrator가 멀티 모델 파이프라인으로 세계 생성
    // 내부에서 seed 생성 → 선택 → 설계 → 통합 → 검증 → 수리를 처리
    result, err := wg.orchestrator.GenerateWorld(ctx, playerCount, themeHint, onProgress)
    if err != nil {
        return nil, fmt.Errorf("세계 생성 실패: %w", err)
    }

    if err := result.Validate(); err != nil {
        return nil, fmt.Errorf("최종 검증 실패: %w", err)
    }

    if len(result.Characters.PlayerRoles) != playerCount {
        return nil, fmt.Errorf("역할 수(%d)가 플레이어 수(%d)와 불일치",
            len(result.Characters.PlayerRoles), playerCount)
    }

    return wg.transformToWorld(result, playerCount), nil
}
```

## 프롬프트 설계

> **프롬프트 레이어 전체 정의는 [orchestrator.md](./orchestrator.md#프롬프트-설계) 참조.**
> WorldGenerator는 Orchestrator의 얇은 래퍼이므로, 프롬프트 구성은 Orchestrator가 담당한다.
> 아래 User Prompt 템플릿만 WorldGenerator에서 직접 관리한다.

### User Prompt (`text/template` 사용)

```go
// internal/ai/worldgen/prompts.go

const worldGenUserPromptTmpl = `
{{.PlayerCount}}명의 플레이어가 함께 할 게임을 만들어 주세요.
{{- if .ThemeHint}}
테마 힌트: {{.ThemeHint}}. 이 방향을 참고하되 구속되지 마세요.
{{- else}}
장르, 배경, 사건 모두 자유롭게 만들어 주세요.
{{- end}}
`

// MVP에서 themeHint는 항상 빈 문자열. FR-089 (P2).
type worldGenPromptData struct {
    PlayerCount int
    ThemeHint   string
}

func buildWorldGenPrompt(playerCount int, themeHint string) (string, error) {
    tmpl, err := template.New("worldgen").Parse(worldGenUserPromptTmpl)
    if err != nil {
        return "", err
    }
    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, worldGenPromptData{
        PlayerCount: playerCount,
        ThemeHint:   themeHint,
    }); err != nil {
        return "", err
    }
    return buf.String(), nil
}
```

## 부분 재생성 (RegeneratePartial)

검증 실패 시 전체가 아닌 영향받은 섹션만 재생성.

> **오케스트레이터 통합:** 부분 재생성은 Orchestrator 내부의 SchemaEditor 에이전트가 담당.
> Orchestrator.GenerateWorld() 내부 Phase 5에서 ContinuityCop 검증 실패 시 자동 호출됨.
> 아래 코드는 SchemaEditor가 사용하는 로직으로, Orchestrator에서 호출됨.

```go
// internal/ai/orchestrator/schema_editor.go
// SchemaEditor 에이전트가 부분 재생성을 처리

func (o *Orchestrator) repairDraft(
    ctx context.Context,
    world *WorldGeneration,
    issues []ValidationError,
) (*WorldGeneration, error) {
    editor := o.getProvider(RoleSchemaEditor)

    worldJSON, err := json.Marshal(world)
    if err != nil {
        return nil, fmt.Errorf("기존 세계 직렬화 실패: %w", err)
    }

    var issueLines []string
    for _, issue := range issues {
        issueLines = append(issueLines, fmt.Sprintf("- [%s] %s (영향 ID: %s)",
            issue.Category, issue.Message, strings.Join(issue.AffectedIDs, ", ")))
    }

    systemPrompt := `당신은 텍스트 RPG 세계의 교정자입니다.
기존 세계 데이터에서 지적된 문제만 수정하세요.
수정하지 않은 섹션은 원본 그대로 유지하세요.
반드시 전체 WorldGeneration JSON을 반환하세요.`

    userPrompt := fmt.Sprintf(`[기존 세계 데이터]
%s

[수정 필요 항목]
%s

위 문제들만 수정한 완전한 WorldGeneration JSON을 반환하세요.`,
        string(worldJSON),
        strings.Join(issueLines, "\n"),
    )

    raw, err := editor.GenerateStructured(ctx, StructuredRequest{
        SystemPrompt: systemPrompt,
        UserPrompt:   userPrompt,
        Temperature:  0.7,
        MaxTokens:    8000,
    })
    if err != nil {
        return nil, fmt.Errorf("부분 재생성 AI 호출 실패: %w", err)
    }

    var regenerated WorldGeneration
    if err := json.Unmarshal(raw, &regenerated); err != nil {
        return nil, fmt.Errorf("부분 재생성 파싱 실패: %w", err)
    }

    // 머지: 재생성된 섹션을 기존 세계에 덮어씌움
    merged := *world
    for _, issue := range issues {
        switch issue.Category {
        case "map":
            merged.Map = regenerated.Map
        case "clue":
            merged.Clues = regenerated.Clues
        case "npc":
            merged.Characters.NPCs = regenerated.Characters.NPCs
        case "end_condition":
            merged.GameStructure.EndConditions = regenerated.GameStructure.EndConditions
        case "player_path":
            merged.Characters.PlayerRoles = regenerated.Characters.PlayerRoles
        case "information":
            merged.Information = regenerated.Information
        }
    }

    return &merged, nil
}
```

## AILayer 오케스트레이션 흐름

> **변경:** 기존 단일 프로바이더 Generate → Validate → RegeneratePartial 루프가
> Orchestrator 내부 파이프라인으로 대체됨. AILayer는 Orchestrator를 직접 호출.

```
AILayer.GenerateWorld(ctx, playerCount, themeHint, onProgress)
    │
    ├── Orchestrator.GenerateWorld()
    │     │
    │     ├── Phase 1: Seeds (병렬, 멀티 모델)
    │     ├── Phase 2: Seed Score & Select (규칙 기반)
    │     ├── Phase 3: Design (모드별 분기)
    │     ├── Phase 4: Integration (Showrunner)
    │     └── Phase 5: Validate + Repair (FR-025)
    │           │
    │           ├── StoryValidator.ValidateStructure() (규칙 기반, 기존 유지)
    │           ├── 통과 → 반환
    │           └── 실패 → SchemaEditor로 부분 패치 (최대 3회)
    │                 │
    │                 ├── 재검증 통과 → 반환
    │                 └── 3회 모두 실패 → Showrunner 재호출 (전체 재생성 1회)
    │                       │
    │                       ├── 재검증 통과 → 반환
    │                       └── 재실패 → error 반환 (세션 종료)
    │
    ├── WorldGenerator.transformToWorld(result, playerCount)
    │
    └── return world, nil
```

## 변환: AI 출력 → World

AI 응답(`WorldGeneration`)을 서버 내부 `World` 타입으로 변환.

```go
// internal/ai/worldgen/transform.go

func (wg *WorldGenerator) transformToWorld(raw *WorldGeneration, playerCount int) *World {
    // ── 키 매핑 (WorldGeneration → World) ──
    // raw.Characters.NPCs       → world.NPCs       (characters.npcs → npcs로 플랫화)
    // raw.Characters.PlayerRoles → world.PlayerRoles (characters.playerRoles → playerRoles로 플랫화)
    // raw.Meta.EstimatedDuration → world.GameStructure.EstimatedDuration
    //   (estimatedDuration은 AI 출력의 Meta에서 생성되어 transformToWorld() 시 GameStructure로 복사된다.)
    //
    // ── 초기화 ──
    // playerRoles 수가 playerCount와 일치하는지 확인
    // NPC에 초기 trustLevel 설정 (float64 0~1, 기본 0.5 — types.md NPC.InitialTrust 참조)
    // clue에 isDiscovered: false 초기화
    // gimmick에 isTriggered: false 초기화
    // endConditions에 timeout fallback 포함 확인
}
```

`WorldGeneration`의 검증 메서드:

```go
// Validate는 schemas.md의 WorldGeneration.Validate()를 호출한다.
// playerCount 의존적 검증(역할 수 == 플레이어 수)은 Generate() 내에서 별도 수행.
func (r *WorldGeneration) Validate() error {
    if len(r.Characters.PlayerRoles) == 0 {
        return fmt.Errorf("playerRoles가 비어있음")
    }
    if len(r.Map.Rooms) < len(r.Characters.PlayerRoles)+2 {
        return fmt.Errorf("방 수(%d)가 최소 요구치(%d)보다 적음", len(r.Map.Rooms), len(r.Characters.PlayerRoles)+2)
    }
    if len(r.GameStructure.EndConditions) == 0 {
        return errors.New("종료 조건이 없음")
    }
    // FR-012: 단서 최소 플레이어 수 × 2개
    playerCount := len(r.Characters.PlayerRoles)
    if len(r.Clues) < playerCount*2 {
        return fmt.Errorf("단서 수(%d)가 최소 요구치(%d)보다 적음", len(r.Clues), playerCount*2)
    }
    // FR-013: 반공개 정보가 최소 1쌍 이상
    if len(r.Information.SemiPublic) < 1 {
        return errors.New("반공개 정보가 최소 1쌍 이상 필요")
    }
    // FR-092/FR-011: 각 역할에 개인 목표 1개 이상
    for i, role := range r.Characters.PlayerRoles {
        if len(role.PersonalGoals) == 0 {
            return fmt.Errorf("playerRoles[%d]에 개인 목표가 없음", i)
        }
    }
    // timeout fallback 종료 조건 필수
    hasTimeoutFallback := false
    for _, ec := range r.GameStructure.EndConditions {
        if ec.IsFallback {
            hasTimeoutFallback = true
            break
        }
    }
    if !hasTimeoutFallback {
        return errors.New("timeout fallback end condition required")
    }
    return nil
}
```
