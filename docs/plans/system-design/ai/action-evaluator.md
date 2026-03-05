# ActionEvaluator (`internal/ai/evaluator/`)

## 책임

`/examine`, `/do` 등 플레이어 행동의 결과를 AI가 판정. 단서 발견, 스토리 이벤트 트리거 여부를 결정.

## 의존하는 모듈

AIProvider

## 인터페이스

```go
// internal/ai/evaluator/action_evaluator.go

type ActionEvaluator struct {
    provider AIProvider
}

func NewActionEvaluator(provider AIProvider) *ActionEvaluator {
    return &ActionEvaluator{provider: provider}
}

func (e *ActionEvaluator) EvaluateExamine(
    ctx context.Context,
    gameCtx *GameContext,
    room *Room,
    target string,
) (*EvaluationResult, error)

func (e *ActionEvaluator) EvaluateAction(
    ctx context.Context,
    gameCtx *GameContext,
    playerID string,
    action string,
) (*EvaluationResult, error)

// EvaluationResult는 GameResponse와 동일 구조.
// 별도 타입을 만들지 않고 GameResponse를 재사용한다.
type EvaluationResult = GameResponse
```

## /examine 평가

```go
func (e *ActionEvaluator) EvaluateExamine(
    ctx context.Context,
    gameCtx *GameContext,
    room *Room,
    target string,
) (*EvaluationResult, error) {
    // World.clues는 불변 템플릿. 발견 여부는 GameState.clueStates에서 조회.
    clueStates := gameCtx.CurrentState.ClueStates
    var undiscovered []string
    for _, c := range gameCtx.World.Clues {
        if c.RoomID == room.ID && !clueStates[c.ID].IsDiscovered {
            undiscovered = append(undiscovered,
                fmt.Sprintf("- %s: %s (발견 조건: %s)", c.Name, c.Description, c.DiscoverCondition),
            )
        }
    }

    var items []string
    for _, item := range room.Items {
        items = append(items, item.Name)
    }

    targetLine := "방 전체를 조사합니다."
    if target != "" {
        targetLine = fmt.Sprintf("조사 대상: %s", target)
    }

    prompt := fmt.Sprintf(`
[상황]
%s이(가) %s을(를) 조사합니다.
%s

[방 정보]
%s
물품: %s

[이 방의 미발견 단서]
%s

[요청 플레이어 정보]
역할: %s
개인 목표: %s

[현재 게임 상태]
경과 시간: %d분 / %d분
발견된 단서 수: %d / %d

[최근 이벤트 히스토리]
%s

[지시]
1. 조사 결과를 생동감 있게 서술하세요 (2~3문장)
2. target이 단서의 발견 조건과 일치하면 단서를 발견시키세요
3. 단서가 없더라도 방의 분위기와 디테일을 묘사하세요
4. 반드시 {"events": [...], "stateChanges": [...]} 형태의 JSON으로 응답하세요.
5. 단서를 발견시킬 경우, events에 clue_found 이벤트와 함께 stateChanges에 {"type": "discover_clue", "playerId": "...", "clueId": "..."} 를 반드시 포함하세요.
`,
        gameCtx.RequestingPlayer.Role.CharacterName, room.Name, targetLine,
        room.Description,
        strings.Join(items, ", "),
        strings.Join(undiscovered, "\n"),
        gameCtx.RequestingPlayer.Role.CharacterName,
        strings.Join(gameCtx.RequestingPlayer.Role.PersonalGoals, ", "),
        int(gameCtx.CurrentState.ElapsedTime/60), gameCtx.World.GameStructure.EstimatedDuration,
        countDiscoveredClues(gameCtx), len(gameCtx.World.Clues),
        summarizeRecentEvents(gameCtx.RecentEvents),
    )

    raw, err := e.provider.GenerateStructured(ctx, StructuredRequest{
        SystemPrompt: evaluatorSystemPrompt,
        UserPrompt:   prompt,
        Temperature:  0.7,
        MaxTokens:    500,
    })
    if err != nil {
        return nil, err
    }

    var result EvaluationResult
    if err := json.Unmarshal(raw, &result); err != nil {
        return nil, err
    }
    return &result, nil
}
```

## /do 평가

```go
func (e *ActionEvaluator) EvaluateAction(
    ctx context.Context,
    gameCtx *GameContext,
    playerID string,
    action string,
) (*EvaluationResult, error) {
    var playersInRoom []string
    for _, p := range gameCtx.PlayersInRoom {
        playersInRoom = append(playersInRoom, p.Nickname)
    }

    prompt := fmt.Sprintf(`
[상황]
%s이(가) 행동합니다: "%s"
위치: %s
같은 방: %s

[세계 설정]
%s

[요청 플레이어 정보]
역할: %s
개인 목표: %s

[현재 게임 상태]
경과 시간: %d분 / %d분
발견된 단서 수: %d / %d

[최근 이벤트 히스토리]
%s

[지시]
1. 이 행동의 결과를 판정하고 서술하세요 (2~3문장)
2. 행동이 스토리에 영향을 줄 수 있으면 이벤트를 트리거하세요
3. 불가능한 행동이면 왜 안 되는지 서술하세요
4. 행동의 결과는 게임 세계관에 일관적이어야 합니다
5. 반드시 {"events": [...], "stateChanges": [...]} 형태의 JSON으로 응답하세요.
6. 행동으로 단서가 발견되면 stateChanges에 {"type": "discover_clue", "playerId": "...", "clueId": "..."} 를 반드시 포함하세요.
7. 행동으로 NPC 신뢰도가 변화하면 stateChanges에 {"type": "update_npc_trust", "npcId": "...", "delta": 0.1} 를 포함하세요.
`,
        gameCtx.RequestingPlayer.Role.CharacterName, action,
        gameCtx.CurrentRoom.Name,
        strings.Join(playersInRoom, ", "),
        gameCtx.World.Synopsis,
        gameCtx.RequestingPlayer.Role.CharacterName,
        strings.Join(gameCtx.RequestingPlayer.Role.PersonalGoals, ", "),
        int(gameCtx.CurrentState.ElapsedTime/60), gameCtx.World.GameStructure.EstimatedDuration,
        countDiscoveredClues(gameCtx), len(gameCtx.World.Clues),
        summarizeRecentEvents(gameCtx.RecentEvents),
    )

    raw, err := e.provider.GenerateStructured(ctx, StructuredRequest{
        SystemPrompt: evaluatorSystemPrompt,
        UserPrompt:   prompt,
        Temperature:  0.8,
        MaxTokens:    500,
    })
    if err != nil {
        return nil, err
    }

    var result EvaluationResult
    if err := json.Unmarshal(raw, &result); err != nil {
        return nil, err
    }
    return &result, nil
}
```

## Evaluator System Prompt

```
당신은 텍스트 RPG의 행동 판정자입니다.

[원칙]
- 플레이어의 행동에 공정하게 반응하세요
- 세계 설정에 일관된 결과를 생성하세요
- 단서 발견은 관련된 행동을 했을 때만 허용하세요
- 결과를 생동감 있게 서술하되 짧게 (2~3문장)
- 같은 방 플레이어 모두에게 보이는 결과입니다
- 응답은 같은 방의 모든 플레이어에게 공개됩니다. 요청 플레이어의 개인 목표나 비밀을 암시하거나 언급하지 마세요.

[출력 형식]
반드시 {"events": [...], "stateChanges": [...]} 형태의 JSON으로 응답하세요.
단서를 발견시킬 경우, events에 clue_found 이벤트와 함께 stateChanges에 {"type": "discover_clue", "playerId": "...", "clueId": "..."} 를 반드시 포함하세요.
설명 텍스트나 마크다운 코드 블록 없이 JSON만 반환하세요.
```

## 이벤트 타입 매핑 (ActionProcessor)

AI는 `{"events": [...], "stateChanges": [...]}` 형태의 원시 콘텐츠를 반환한다. 서버의 `ActionProcessor`가 이를 받아 적절한 이벤트 타입으로 래핑하여 클라이언트에 전송한다.

| AI 반환 내용 | ActionProcessor가 생성하는 이벤트 타입 |
|-------------|--------------------------------------|
| `/examine` 결과 텍스트 + events | `examine_result` (playerId, target, content 포함) |
| `/do` 결과 텍스트 + events | `action_result` (playerId, action, content 포함) |

AI가 반환한 `events` 배열의 개별 이벤트(예: `clue_found`)는 ActionProcessor가 별도 이벤트로 브로드캐스트한다.

> **참고:** `/look` 명령은 AI 호출 없이 서버에서 직접 처리한다 (방 설명 + NPC/플레이어 목록 반환). `backend/action-processor.md` 참조.
