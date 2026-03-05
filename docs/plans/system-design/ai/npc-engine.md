# NPCEngine (`internal/ai/npc/`)

## 책임

NPC 대화 생성. 퍼소나 유지, 정보 공개 제어, 기믹 트리거 판단, 대화 이력 관리.

## 의존하는 모듈

AIProvider

## 인터페이스

```go
// internal/ai/npc/npc_engine.go

// NPCEngine는 자체 대화 이력을 저장하지 않는다.
// 대화 이력의 유일한 소스는 GameState.NPCStates[npcID].ConversationHistory이다.
// Chat() 호출 시 gameCtx.CurrentState.NPCStates를 통해 이력을 읽고,
// 응답 후 ActionProcessor가 GameStateManager.AddConversation()을 호출하여 이력을 갱신한다.
type NPCEngine struct {
    provider AIProvider
}

func NewNPCEngine(provider AIProvider) *NPCEngine {
    return &NPCEngine{provider: provider}
}

// ── NPC와 대화 ──
func (e *NPCEngine) Chat(
    ctx context.Context,
    npc *NPC,
    playerID string,
    playerMessage string,
    gameCtx *GameContext,
) (*NPCResponse, error)

// 대화 이력은 GameState.NPCStates[npcID].ConversationHistory에서 읽는다.
// NPCEngine은 자체 이력을 보관하지 않음.
```

## NPCResponse 구조

```go
// NPCResponse는 schemas.NPCResponse를 참조. 이 파일에서는 편의상 재정의하되,
// Events 타입은 schemas.md와 동일하게 []json.RawMessage를 사용한다.
type NPCResponse struct {
    Dialogue         string            `json:"dialogue"`         // NPC가 하는 말
    Emotion          string            `json:"emotion"`          // 현재 감정 상태 (events.md NPCDialogueData.Emotion)
    InternalThought  string            `json:"internalThought"`  // 내부 판단 (디버깅용, 플레이어 비공개)
    InfoRevealed     []string          `json:"infoRevealed"`     // 이번 대화에서 공개한 정보
    TrustChange      float64           `json:"trustChange"`      // 신뢰도 변화 (-1 ~ +1)
    TriggeredGimmick bool              `json:"triggeredGimmick"` // 기믹 트리거 여부 — MVP에서는 항상 false (기믹 미지원, FR P1)
    Events           []json.RawMessage `json:"events"`           // 추가 발생 이벤트 (schemas.md와 동일)
}
```

> **MVP 참고:** MVP에서 기믹 트리거는 미지원. `NPCResponse.TriggeredGimmick` 필드는 항상 `false`.

## 대화 흐름

```
Chat(ctx, npc, playerID, playerMessage, gameCtx)
    │
    ├── 대화 이력 조회: gameCtx.CurrentState.NPCStates[npc.ID].ConversationHistory
    │
    ├── 프롬프트 구성: buildNPCPrompt()
    │     - NPC 퍼소나, 보유 정보, 숨기는 정보
    │     - 플레이어와의 신뢰도
    │     - 이전 대화 이력
    │     - 현재 게임 상황
    │
    ├── provider.GenerateStructured(ctx, StructuredRequest{
    │     Temperature: 0.8,
    │     MaxTokens:   500,
    │   })
    │
    ├── json.Unmarshal → NPCResponse
    │
    ├── validateNPCResponse(response, npc) — 정보 누출 검사
    │     검사 실패(숨겨야 할 정보 감지) → 더 강한 제약 조건으로 재생성 (1회)
    │
    └── return NPCResponse, nil
    // 대화 이력 업데이트는 ActionProcessor가 GameStateManager.AddConversation()으로 처리
```

## 정보 누출 방지 (`validateNPCResponse`)

AI 응답이 NPC의 숨겨야 할 정보를 의도치 않게 노출하는지 검사한다.

```go
func validateNPCResponse(response NPCResponse, npc NPC) error {
    for _, hidden := range npc.HiddenInfo {
        // 숨겨야 할 정보의 핵심 키워드가 dialogue에 포함되었는지 확인
        // 완전 일치보다 핵심 명사/고유명사 기반 부분 일치로 검사
        if containsHiddenInfo(response.Dialogue, hidden) {
            return fmt.Errorf("NPC 응답이 숨겨야 할 정보를 노출: %s", hidden)
        }
    }
    return nil
}

// containsHiddenInfo는 hiddenInfo에서 핵심 키워드를 추출하여
// dialogue에 부분 일치 여부를 검사한다.
// 단순 문자열 포함 검사이므로 false positive가 발생할 수 있으나,
// 보수적(정보 보호 우선)으로 동작하도록 설계한다.
func containsHiddenInfo(dialogue, hiddenInfo string) bool {
    // 핵심 키워드(고유명사, 특정 숫자, 장소명 등) 추출 후 포함 여부 검사
    // 구현은 단순 strings.Contains 기반으로 시작하여 필요 시 고도화
    keywords := extractKeywords(hiddenInfo)
    for _, kw := range keywords {
        if strings.Contains(dialogue, kw) {
            return true
        }
    }
    return false
}
```

**누출 감지 시 처리:**
1. 더 강한 제약 조건을 시스템 프롬프트에 추가하여 1회 재생성
2. 재생성 후에도 누출 감지 시 안전한 기본 응답 반환 ("그것에 대해서는 말하기 어렵네요.")

## NPC Prompt 구성

```go
// internal/ai/npc/prompts.go

func buildNPCPrompt(
    npc *NPC,
    playerID string,
    playerMessage string,
    history []ConversationRecord,
    gameCtx *GameContext,
) string {
    var knownInfo strings.Builder
    for _, info := range npc.KnownInfo {
        fmt.Fprintf(&knownInfo, "- %s\n", info)
    }

    var hiddenInfo strings.Builder
    for _, info := range npc.HiddenInfo {
        fmt.Fprintf(&hiddenInfo, "- %s\n", info)
    }

    var historyBuf strings.Builder
    for _, h := range history {
        // ConversationRecord: PlayerID, Message (플레이어 입력), Response (NPC 응답)
        fmt.Fprintf(&historyBuf, "플레이어: %s\n%s: %s\n", h.Message, npc.Name, h.Response)
    }

    // 런타임 NPC 상태는 GameState.NPCStates에서 가져옴 (NPC 템플릿은 불변)
    npcState := gameCtx.CurrentState.NPCStates[npc.ID]
    trustLevel := npcState.TrustLevels[playerID]

    gimmickSection := ""
    if npc.Gimmick != nil {
        triggered := "미발동"
        if npcState.GimmickTriggered {
            triggered = "이미 발동됨"
        }
        gimmickSection = fmt.Sprintf(`
[기믹]
%s
트리거 조건: %s
현재 상태: %s`, npc.Gimmick.Description, npc.Gimmick.TriggerCondition, triggered)
    }

    return fmt.Sprintf(`
[당신의 정체]
이름: %s
퍼소나: %s
행동 원칙: %s

[당신이 아는 것]
%s
[당신이 숨기고 있는 것]
%s
[현재 상대 플레이어]
이름: %s
신뢰도: %.1f / 1.0
%s
[이전 대화]
%s
[현재 입력]
%s: "%s"
`,
        npc.Name, npc.Persona, npc.BehaviorPrinciple,
        knownInfo.String(),
        hiddenInfo.String(),
        gameCtx.RequestingPlayer.Role.CharacterName,
        trustLevel,
        gimmickSection,
        historyBuf.String(),
        gameCtx.RequestingPlayer.Role.CharacterName, playerMessage,
    )
}
```

## NPC System Prompt

```
당신은 텍스트 RPG의 NPC입니다. 주어진 퍼소나를 철저히 유지하세요.

[정보 공개 원칙]
- 숨기는 정보는 쉽게 내놓지 마세요
- 신뢰도가 높을수록 더 많은 정보를 공개하세요
- 행동 원칙을 반드시 따르세요
- 숨기는 정보를 플레이어가 직접 묻지 않으면 절대 먼저 말하지 마세요 (FR-056)

[기믹]
- 트리거 조건이 충족되면 triggeredGimmick: true로 설정하세요
- 조건이 충족되지 않았으면 절대 발동하지 마세요

[응답]
- 퍼소나에 맞는 말투를 사용하세요
- 짧게 (1~3문장)
- internalThought에 판단 근거를 남기세요
```

## 대화 이력 관리

ConversationRecord는 types.md에 정의된 공유 타입을 사용한다:
`{PlayerID string, Message string, Response string, Timestamp int64}`

NPC 단위로 이력을 관리하며, 프롬프트 구성 시 PlayerID와 Message/Response를 사용하여 대화 흐름을 재현한다.

ConversationRecord는 types.md의 정의를 사용한다 (NPCEngine에서 재정의하지 않음).

NPCEngine은 자체 이력을 보관하지 않는다. Chat() 호출 시 `gameCtx.CurrentState.NPCStates[npcID].ConversationHistory`를 통해 이력을 읽고, 응답 후 ActionProcessor가 `GameStateManager.AddConversation()`을 호출하여 이력을 갱신한다. 최대 20턴 유지는 `GameStateManager.AddConversation()`에서 관리한다.

**대화 이력은 NPC 단위로 관리 (플레이어 단위가 아님).** 이유: 같은 방 다른 플레이어가 NPC와 한 대화도 NPC가 기억해야 함 (같은 방에서 다 들림).
