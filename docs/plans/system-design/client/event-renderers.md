# EventRenderers (`internal/client/renderers/`)

> **Import 경로:** `charm.land/lipgloss/v2`

## 책임

이벤트 타입별 시각적 렌더링. 확장 가능한 레지스트리 패턴.
각 렌더러는 `GameEvent`를 받아 Lip Gloss v2로 스타일링된 문자열을 반환하는 순수 함수다.

> **클라이언트 GameEvent 역직렬화 방식:** 클라이언트에서 GameEvent는 JSON 역직렬화 시 `struct { Type string; Data map[string]any }` 형태의 flat 구조로 처리한다. events.md의 Go interface 정의와 달리, 네트워크 수신 시 `json.RawMessage`를 `map[string]any`로 언마샬하여 사용한다. 따라서 모든 렌더러는 `event.Data["fieldName"].(type)` 방식으로 데이터에 접근하며, 중첩 맵 접근 시 nil 체크가 필수다.

> **GameEvent data 승격 규칙:** GameEvent 역직렬화 시 JSON의 `data` 하위 객체가 GameEvent.Data로 승격된다. 즉, `{"type":"narration","data":{"text":"..."}}` 수신 시 `event.Data["text"]`로 접근 가능하다. 중첩 없이 최상위 Data map에서 바로 접근한다.

## 의존하는 모듈

없음 (순수 함수, Lip Gloss 스타일링)

## 안전한 데이터 접근 헬퍼

> **주의:** 모든 렌더러는 `event.Data["key"].(string)` 형태의 직접 타입 단언 대신 아래의 안전한 접근자 함수를 사용해야 한다. 직접 타입 단언은 키가 없거나 타입이 다를 경우 런타임 패닉을 유발한다.

```go
// internal/client/renderers/helpers.go

// getString은 map[string]any에서 string 값을 안전하게 추출한다.
// 키가 없거나 값이 string이 아니면 빈 문자열을 반환한다 (패닉 없음).
func getString(data map[string]any, key string) string {
    if v, ok := data[key].(string); ok {
        return v
    }
    return ""
}

// getFloat64는 map[string]any에서 float64 값을 안전하게 추출한다.
func getFloat64(data map[string]any, key string) float64 {
    if v, ok := data[key].(float64); ok {
        return v
    }
    return 0
}

// getMap은 map[string]any에서 중첩 map을 안전하게 추출한다.
func getMap(data map[string]any, key string) map[string]any {
    if v, ok := data[key].(map[string]any); ok {
        return v
    }
    return nil
}
```

## 레지스트리 패턴

```go
// internal/client/renderers/registry.go

// EventRendererFn은 게임 이벤트를 렌더링된 문자열로 변환한다.
type EventRendererFn func(event GameEvent) string

var rendererRegistry = map[string]EventRendererFn{
    "narration":       renderNarration,
    "npc_dialogue":    renderNPCDialogue,
    "clue_found":      renderClueFound,
    "story_event":     renderStoryEvent,
    "game_end":        renderGameEnd,
    "examine_result":  renderExamineResult,
    "action_result":   renderActionResult,
    "player_move":     renderPlayerMove,
    "time_warning":    renderTimeWarning,
    // 투표(vote_started/progress/ended)는 GameEvent가 아닌 ServerMessage로 처리.
    // → VoteOverlay에서 state.ActiveVote 기반으로 렌더링.
    // end_proposed(종료 투표)도 ServerMessage로 처리. 렌더링 시 투표 안내 메시지를 함께 표시한다:
    // "'/endvote yes' 또는 '/endvote no'로 투표하세요 (60초 제한)"
    "npc_moved":       renderNPCMoved,
    "npc_give_item":   renderNPCGiveItem,
    "npc_receive_item": renderNPCReceiveItem,
    "npc_reveal":      renderNPCReveal,
}
```

```go
// internal/client/renderers/message_renderer.go

// RenderMessage는 DisplayMessage를 화면 문자열로 변환한다.
func RenderMessage(msg DisplayMessage) string {
    switch msg.Kind {
    case "chat":
        return renderChatMessage(msg)
    case "event":
        fn, ok := rendererRegistry[msg.Event.Type]
        if ok {
            return fn(msg.Event)
        }
        return renderFallback(msg.Event)
    case "system":
        return renderSystemMessage(msg.Content)
    }
    return ""
}
```

## 렌더링 스타일

PRD FR-074 기반. 색상과 텍스트/기호 양쪽으로 구분 (색상 의존 금지, NFR-023).

### 채팅

```go
// internal/client/renderers/chat.go

var globalScopeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))  // yellow
var globalNameStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
var boldStyle        = lipgloss.NewStyle().Bold(true)

func renderChatMessage(msg DisplayMessage) string {
    if msg.Scope == "global" {
        // FR-037 AC3: 전체 채팅은 발신자 위치 포함 표시
        location := ""
        if msg.SenderLocation != nil {
            location = " (" + *msg.SenderLocation + ")"
        }
        return fmt.Sprintf("%s %s: %s",
            globalScopeStyle.Render("[전체]"),
            globalNameStyle.Render(msg.SenderName+location),
            msg.Content,
        )
    }
    return fmt.Sprintf("%s: %s",
        boldStyle.Render(msg.SenderName),
        msg.Content,
    )
}
```

### GM 서술

```go
// internal/client/renderers/narration.go

var narrationStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("5")).  // magenta
    Italic(true).
    PaddingTop(1).
    PaddingBottom(1)

// getString 헬퍼를 사용하여 패닉 없이 안전하게 접근한다.
// 모든 렌더러는 이 패턴을 따라야 한다.
func renderNarration(event GameEvent) string {
    return narrationStyle.Render("[GM] " + getString(event.Data, "text"))
}
```

### NPC 대화

```go
// internal/client/renderers/npc_dialogue.go

var npcNameStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("2")).  // green
    Bold(true)

func renderNPCDialogue(event GameEvent) string {
    npcName := getString(event.Data, "npcName")
    text    := getString(event.Data, "text")
    return fmt.Sprintf("%s %s",
        npcNameStyle.Render("["+npcName+"]"),
        text,
    )
}
```

### 단서 발견

```go
// internal/client/renderers/clue.go

var clueBoxStyle = lipgloss.NewStyle().
    BorderStyle(lipgloss.RoundedBorder()).
    BorderForeground(lipgloss.Color("6")).  // cyan
    Foreground(lipgloss.Color("6")).
    PaddingLeft(1).
    PaddingRight(1)

func renderClueFound(event GameEvent) string {
    playerName := getString(event.Data, "playerName")
    clueMap := getMap(event.Data, "clue")
    clueName := getString(clueMap, "name")
    text := fmt.Sprintf("* %s이(가) [%s]을(를) 발견했습니다!", playerName, clueName)
    return clueBoxStyle.Render(text)
}
```

### 스토리 이벤트

```go
// internal/client/renderers/story_event.go

var storyEventBoxStyle = lipgloss.NewStyle().
    BorderStyle(lipgloss.DoubleBorder()).
    BorderForeground(lipgloss.Color("1")).  // red
    Foreground(lipgloss.Color("1")).
    PaddingLeft(1).
    PaddingRight(1).
    PaddingTop(1).
    PaddingBottom(1)

var storyEventTitleStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("1")).
    Bold(true)

func renderStoryEvent(event GameEvent) string {
    title := getString(event.Data, "title")
    desc  := getString(event.Data, "description")
    body  := lipgloss.JoinVertical(lipgloss.Left,
        storyEventTitleStyle.Render("[사건] "+title),
        desc,
    )
    return storyEventBoxStyle.Render(body)
}
```

### 시스템 메시지

```go
// internal/client/renderers/system.go

var systemStyle = lipgloss.NewStyle().Faint(true)

func renderSystemMessage(content string) string {
    return systemStyle.Render("--- " + content + " ---")
}
```

### 이동

```go
// internal/client/renderers/player_move.go

func renderPlayerMove(event GameEvent) string {
    playerName := getString(event.Data, "playerName")
    from       := getString(event.Data, "from")
    to         := getString(event.Data, "to")
    msg := fmt.Sprintf("--- %s이(가) %s에서 %s(으)로 이동했습니다 ---", playerName, from, to)
    return systemStyle.Render(msg)
}
```

### 게임 종료

```go
// internal/client/renderers/game_end.go

func renderGameEnd(event GameEvent) string {
    reason := getString(event.Data, "reason")
    result := getString(event.Data, "commonResult")
    body := lipgloss.JoinVertical(lipgloss.Left,
        storyEventTitleStyle.Render("[게임 종료] "+reason),
        result,
    )
    return storyEventBoxStyle.Render(body)
}
```

### 조사 결과

```go
// internal/client/renderers/examine_result.go

func renderExamineResult(event GameEvent) string {
    playerName := getString(event.Data, "playerName")
    target     := getString(event.Data, "target")
    desc       := getString(event.Data, "description")
    return narrationStyle.Render(
        fmt.Sprintf("[조사] %s이(가) %s을(를) 살펴봤습니다.\n%s", playerName, target, desc),
    )
}
```

### 행동 결과

```go
// internal/client/renderers/action_result.go

func renderActionResult(event GameEvent) string {
    playerName := getString(event.Data, "playerName")
    action     := getString(event.Data, "action")
    result     := getString(event.Data, "result")
    return narrationStyle.Render(
        fmt.Sprintf("[행동] %s: %s\n%s", playerName, action, result),
    )
}
```

### NPC 이동

```go
// internal/client/renderers/npc_moved.go

func renderNPCMoved(event GameEvent) string {
    npcName := getString(event.Data, "npcName")
    from    := getString(event.Data, "from")
    to      := getString(event.Data, "to")
    return systemStyle.Render(
        fmt.Sprintf("--- %s이(가) %s에서 %s(으)로 이동했습니다 ---", npcName, from, to),
    )
}
```

### NPC 아이템 전달

```go
// internal/client/renderers/npc_give_item.go

func renderNPCGiveItem(event GameEvent) string {
    npcName    := getString(event.Data, "npcName")
    playerName := getString(event.Data, "playerName")
    item       := getMap(event.Data, "item")
    itemName   := getString(item, "name")
    return fmt.Sprintf("%s %s에게 [%s]을(를) 건넸습니다.",
        npcNameStyle.Render("["+npcName+"]"), playerName, itemName)
}
```

### NPC 아이템 수령

```go
// internal/client/renderers/npc_receive_item.go

func renderNPCReceiveItem(event GameEvent) string {
    npcName    := getString(event.Data, "npcName")
    playerName := getString(event.Data, "playerName")
    item       := getMap(event.Data, "item")
    itemName   := getString(item, "name")
    return fmt.Sprintf("%s이(가) %s에게 [%s]을(를) 전달했습니다.",
        playerName, npcNameStyle.Render("["+npcName+"]"), itemName)
}
```

### NPC 정보 공개

```go
// internal/client/renderers/npc_reveal.go

var revealBoxStyle = lipgloss.NewStyle().
    BorderStyle(lipgloss.RoundedBorder()).
    BorderForeground(lipgloss.Color("2")).
    Foreground(lipgloss.Color("2")).
    PaddingLeft(1).
    PaddingRight(1)

func renderNPCReveal(event GameEvent) string {
    npcName    := getString(event.Data, "npcName")
    revelation := getString(event.Data, "revelation")
    return revealBoxStyle.Render(
        fmt.Sprintf("%s %s", npcNameStyle.Render("["+npcName+"]"), revelation),
    )
}
```

### 시간 경고

```go
// internal/client/renderers/time_warning.go

var timeWarningBoxStyle = lipgloss.NewStyle().
    BorderStyle(lipgloss.NormalBorder()).
    BorderForeground(lipgloss.Color("3")).  // yellow
    Foreground(lipgloss.Color("3")).
    Bold(true).
    PaddingLeft(1).
    PaddingRight(1)

func renderTimeWarning(event GameEvent) string {
    remaining := int(getFloat64(event.Data, "remainingMinutes"))
    return timeWarningBoxStyle.Render(
        fmt.Sprintf("⏰ 남은 시간: %d분", remaining),
    )
}
```

### 폴백 렌더러

```go
// internal/client/renderers/fallback.go

func renderFallback(event GameEvent) string {
    // 레지스트리에 없는 이벤트는 JSON으로 표시 (개발 중 디버깅용)
    b, _ := json.Marshal(event)
    return systemStyle.Render("[unknown event] " + string(b))
}
```

## 확장

새로운 이벤트 타입 추가 시:
1. `shared/events/`에 이벤트 타입 추가
2. `rendererRegistry`에 렌더러 함수 등록
3. 해당 렌더러 함수 구현

레지스트리에 없는 이벤트는 `renderFallback`이 JSON으로 표시 (개발 중 디버깅용).
