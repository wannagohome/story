# InputSystem (`internal/client/input/`)

## 책임

사용자 입력을 파싱하여 `ClientMessage`로 변환. `/`로 시작하면 명령어, 아니면 같은 방 채팅.
Bubble Tea v2에서 키 입력은 `tea.KeyPressMsg`로 수신된다. `bubbles/textinput`이 문자 축적을 담당하고, Enter 키 감지 시 `ParseInput`을 호출한다.

> **Import 경로:** `charm.land/bubbletea/v2`, `charm.land/bubbles/v2/textinput`

## 의존하는 모듈

없음 (순수 함수)

## CommandParser

```go
// internal/client/input/command_parser.go

type InputKind int

const (
    InputKindChat InputKind = iota
    InputKindCommand
)

type ParsedInput struct {
    Kind    InputKind
    Command string // Kind == InputKindCommand일 때
    Args    string // 명령어 인자 (없으면 빈 문자열)
    Content string // Kind == InputKindChat일 때
}

func ParseInput(raw string) ParsedInput {
    trimmed := strings.TrimSpace(raw)

    if strings.HasPrefix(trimmed, "/") {
        spaceIdx := strings.Index(trimmed, " ")
        if spaceIdx == -1 {
            return ParsedInput{
                Kind:    InputKindCommand,
                Command: trimmed[1:],
            }
        }
        return ParsedInput{
            Kind:    InputKindCommand,
            Command: trimmed[1:spaceIdx],
            Args:    strings.TrimSpace(trimmed[spaceIdx+1:]),
        }
    }

    return ParsedInput{
        Kind:    InputKindChat,
        Content: trimmed,
    }
}
```

## CommandHandler

`ParsedInput` → `ClientMessage` 변환.

```go
// internal/client/input/command_handler.go

// CommandResult는 명령어 처리 결과를 나타낸다.
// FR-075 AC3: 유효하지 않은 명령어 입력 시 오류 메시지를 표시하기 위해
// 단순 nil 대신 구체적인 오류 정보를 반환한다.
type CommandResult struct {
    Message  *ClientMessage
    ErrorMsg string // 비어있지 않으면 오류 (사용자에게 표시)
}

// CommandToClientMessage는 ParsedInput을 ClientMessage로 변환한다.
// 유효하지 않은 명령어이거나 인자가 부족하면 ErrorMsg에 오류와 사용법을 포함한다.
func CommandToClientMessage(parsed ParsedInput) CommandResult {
    if parsed.Kind == InputKindChat {
        return CommandResult{Message: &ClientMessage{Type: "chat", Content: parsed.Content}}
    }

    switch parsed.Command {
    case "shout":
        if parsed.Args == "" {
            return CommandResult{ErrorMsg: "사용법: /shout <메시지>"}
        }
        return CommandResult{Message: &ClientMessage{Type: "shout", Content: parsed.Args}}

    case "move", "go":
        if parsed.Args == "" {
            return CommandResult{ErrorMsg: "사용법: /move <방 이름>"}
        }
        return CommandResult{Message: &ClientMessage{Type: "move", TargetRoomID: parsed.Args}}

    case "examine":
        var target *string
        if parsed.Args != "" {
            target = &parsed.Args
        }
        return CommandResult{Message: &ClientMessage{Type: "examine", Target: target}}

    case "do":
        if parsed.Args == "" {
            return CommandResult{ErrorMsg: "사용법: /do <행동 설명>"}
        }
        return CommandResult{Message: &ClientMessage{Type: "do", Action: parsed.Args}}

    case "talk":
        if parsed.Args == "" {
            return CommandResult{ErrorMsg: "사용법: /talk <NPC 이름> [대화 내용]"}
        }
        // "/talk 레이몬드 어젯밤 무슨 일이 있었죠?" → npcID: "레이몬드", message: "어젯밤 무슨 일이 있었죠?"
        // NPC 이름에 공백이 포함될 수 있으므로(예: '집사 레이몬드'), 서버에서 부분 매칭을 지원한다.
        // 클라이언트는 첫 번째 공백 기준으로 분리하되, 서버가 NPC 이름을 퍼지 매칭으로 해결한다.
        // Tab 자동완성(P1)이 이 문제를 근본적으로 해결한다.
        spaceIdx := strings.Index(parsed.Args, " ")
        if spaceIdx == -1 {
            return CommandResult{Message: &ClientMessage{Type: "talk", NPCID: parsed.Args, Message: ""}}
        }
        return CommandResult{Message: &ClientMessage{
            Type:    "talk",
            NPCID:   parsed.Args[:spaceIdx],
            Message: parsed.Args[spaceIdx+1:],
        }}

    case "give":
        // P1 — MVP에서는 미구현 (FR-047은 P1).
        return CommandResult{ErrorMsg: "/give는 다음 버전에서 지원될 예정입니다."}

    case "vote":
        if parsed.Args == "" {
            return CommandResult{ErrorMsg: "사용법: /vote <대상 이름>"}
        }
        return CommandResult{Message: &ClientMessage{Type: "vote", TargetID: parsed.Args}}

    case "solve":
        if parsed.Args == "" {
            return CommandResult{ErrorMsg: "사용법: /solve <해결안>"}
        }
        return CommandResult{Message: &ClientMessage{Type: "solve", Answer: parsed.Args}}

    case "end":
        return CommandResult{Message: &ClientMessage{Type: "propose_end"}}

    case "endvote":
        // FR-091: /end 발의에 대한 찬반 투표
        if parsed.Args == "" {
            return CommandResult{ErrorMsg: "사용법: /endvote yes 또는 /endvote no"}
        }
        switch strings.ToLower(parsed.Args) {
        case "yes", "y", "찬성":
            return CommandResult{Message: &ClientMessage{Type: "end_vote", Agree: true}}
        case "no", "n", "반대":
            return CommandResult{Message: &ClientMessage{Type: "end_vote", Agree: false}}
        default:
            return CommandResult{ErrorMsg: "사용법: /endvote yes 또는 /endvote no"}
        }

    case "look":      return CommandResult{Message: &ClientMessage{Type: "request_look"}}
    case "map":       return CommandResult{Message: &ClientMessage{Type: "request_map"}}
    case "inventory",
         "inv":       return CommandResult{Message: &ClientMessage{Type: "request_inventory"}}
    case "role":      return CommandResult{Message: &ClientMessage{Type: "request_role"}}
    case "who":       return CommandResult{Message: &ClientMessage{Type: "request_who"}}
    case "help":      return CommandResult{Message: &ClientMessage{Type: "request_help"}}

    default:
        return CommandResult{ErrorMsg: "알 수 없는 명령어: /" + parsed.Command + ". /help로 목록 확인."}
    }
}
```

## 명령어 목록

| 입력 | ClientMessage type | 인자 |
|------|-------------------|------|
| `안녕하세요` | `chat` | content |
| `/shout 모두 들어!` | `shout` | content |
| `/move 부엌` | `move` | targetRoomId (방 이름) |
| `/go 서재` | `move` | targetRoomId (방 이름) |
| `/examine` | `examine` | target (선택) |
| `/examine 책상` | `examine` | target |
| `/do 문을 조심스럽게 연다` | `do` | action |
| `/talk 레이몬드` | `talk` | npcId (NPC 이름) |
| `/give 레이몬드 열쇠` | `give` | npcId, itemId (아이템 이름) |
| `/vote Bob` | `vote` | targetId |
| `/solve 범인은 Bob` | `solve` | answer (해결안 텍스트) |
| `/end` | `propose_end` | - |
| `/endvote yes` | `end_vote` | agree: true |
| `/endvote no` | `end_vote` | agree: false |
| `/look` | `request_look` | - |
| `/map` | `request_map` | - |
| `/inv` | `request_inventory` | - |
| `/role` | `request_role` | - |
| `/who` | `request_who` | - |
| `/help` | `request_help` | - |

## Bubble Tea 통합

`bubbles/textinput`이 문자 입력을 관리하고, `AppModel.Update()`에서 Enter 키를 감지한다:

```go
// internal/client/screens/game.go (Update 내 일부)

case tea.KeyPressMsg:
    if msg.String() == "enter" {
        raw := m.game.textInput.Value()
        m.game.textInput.Reset()
        // textinput이 문자 축적 → Enter 시 ParseInput 호출
        if raw = strings.TrimSpace(raw); raw != "" {
            parsed := input.ParseInput(raw)
            result := input.CommandToClientMessage(parsed)
            if result.Message != nil {
                return m, m.network.SendCmd(*result.Message)
            }
            // FR-075 AC3: 구체적인 오류 메시지 표시
            m.state = addSystemMessage(m.state, result.ErrorMsg)
        }
        return m, nil
    }
    // Enter 외의 키는 textinput에 전달
    m.game.textInput, cmd = m.game.textInput.Update(msg)
```

## Tab 자동완성 (FR-076)

`/` 입력 후 Tab 키로 명령어 및 대상 이름을 자동완성한다.

```go
// internal/client/input/autocomplete.go

// CompletionContext는 자동완성에 필요한 현재 게임 상태 정보를 제공한다.
type CompletionContext struct {
    Commands  []string // 사용 가능한 명령어 목록
    NPCNames  []string // 현재 방의 NPC 이름 목록
    RoomNames []string // 이동 가능한 방 이름 목록
    Players   []string // 접속 중인 플레이어 닉네임 목록
}

// Complete는 현재 입력과 커서 위치를 기반으로 자동완성 후보를 반환한다.
// FR-076 AC1: "/" 뒤 명령어 자동완성
// FR-076 AC2: 명령어 뒤 대상(NPC, 방, 플레이어) 이름 자동완성
// FR-076 AC3: 여러 후보가 있으면 후보 목록 반환
func Complete(input string, ctx CompletionContext) []string {
    trimmed := strings.TrimSpace(input)

    if !strings.HasPrefix(trimmed, "/") {
        return nil
    }

    spaceIdx := strings.Index(trimmed, " ")
    if spaceIdx == -1 {
        // 명령어 자동완성: "/mo" → ["/move", "/map"]
        prefix := trimmed[1:]
        return filterPrefix(ctx.Commands, prefix)
    }

    // 대상 이름 자동완성: "/move 부" → ["부엌"]
    command := trimmed[1:spaceIdx]
    argPrefix := strings.TrimSpace(trimmed[spaceIdx+1:])

    switch command {
    case "move", "go":
        return filterPrefix(ctx.RoomNames, argPrefix)
    case "talk", "give":
        return filterPrefix(ctx.NPCNames, argPrefix)
    case "vote":
        return filterPrefix(ctx.Players, argPrefix)
    }
    return nil
}

func filterPrefix(candidates []string, prefix string) []string {
    var matches []string
    for _, c := range candidates {
        if strings.HasPrefix(c, prefix) {
            matches = append(matches, c)
        }
    }
    return matches
}
```

Tab 키 처리는 `GameModel.Update()`에서 수행:

```go
// internal/client/screens/game.go (Update 내 일부)

case tea.KeyPressMsg:
    if msg.String() == "tab" {
        raw := m.game.textInput.Value()
        ctx := CompletionContext{
            Commands:  availableCommands,
            NPCNames:  getNPCNames(m.state.CurrentRoom),
            RoomNames: getConnectedRoomNames(m.state.MapOverview),
            Players:   getPlayerNames(m.state.LobbyPlayers),
        }
        candidates := input.Complete(raw, ctx)
        if len(candidates) == 1 {
            // 유일한 후보: 자동 삽입
            m.game.textInput.SetValue(applyCompletion(raw, candidates[0]))
        } else if len(candidates) > 1 {
            // 여러 후보: 목록 표시 (FR-076 AC3)
            m.state = addSystemMessage(m.state, "후보: "+strings.Join(candidates, ", "))
        }
    }
```

## NPC/방 이름 해결

클라이언트에서 이름 → ID 변환은 하지 않는다. **서버가 이름으로 검색하여 해결한다.**
`/move 부엌` → `{ type: "move", targetRoomId: "부엌" }` → 서버의 MapEngine이 이름으로 방을 찾음.

이유: 클라이언트는 ID 매핑 정보를 최소로 유지. 서버가 권한을 가짐.
