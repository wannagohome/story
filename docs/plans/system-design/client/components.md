# UI Components (`internal/client/components/`)

> **Import 경로:** `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, `charm.land/bubbles/v2/viewport`, `charm.land/bubbles/v2/textinput`

## 책임

GameScreen의 재사용 가능한 UI 렌더링 함수. 3분할 레이아웃을 구성.
Bubble Tea v2에서 컴포넌트는 상태를 가진 `tea.Model`이거나 순수 렌더링 함수다. 여기서는 두 방식 모두 사용한다.

## 의존하는 모듈

ClientState, EventRenderers

## GameScreen 레이아웃

```
┌─────────────────────────────────────────────────┐
│ HeaderBar                                       │
│ [WOLF-7423] 저택의 밤 - 1층 거실               │
│ 여기: Alice, Bob  /  부엌: Carol  /  서재: Dave │
├─────────────────────────────────────────────────┤
│ ChatLog                                         │
│                                                 │
│  [GM] 갑자기 저택의 전등이 꺼집니다...           │
│  Alice: 누가 스위치를 건드린 거야?               │
│  Bob: 나 아니야. Carol 어디 갔어?               │
│  [GM] 부엌에서 무언가 쓰러지는 소리가 들립니다. │
│                                                 │
├─────────────────────────────────────────────────┤
│ InputBar                                        │
│ > _                            [/help] [/map]   │
└─────────────────────────────────────────────────┘
```

```go
// internal/client/screens/game.go

func (gm GameModel) View(state ClientState) tea.View {
    header := renderHeader(state)
    chat := renderChatLog(state.Messages, gm.chatViewport)
    input := renderInputBar(gm.textInput)

    return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, header, chat, input))
}
```

---

## HeaderBar

**역할:** 룸 코드, 현재 위치, 각 방 플레이어 현황 실시간 표시.

```go
// internal/client/components/header.go

var headerStyle = lipgloss.NewStyle().
    BorderStyle(lipgloss.NormalBorder()).
    BorderBottom(true).
    PaddingLeft(1).
    PaddingRight(1)

var titleStyle = lipgloss.NewStyle().Bold(true)
var dimStyle   = lipgloss.NewStyle().Faint(true)

func renderHeader(state ClientState) string {
    // 게임 시작 전(CurrentRoom/MapOverview가 nil)일 수 있으므로 방어적 처리
    roomName := ""
    if state.CurrentRoom != nil {
        roomName = state.CurrentRoom.Name
    }
    title := fmt.Sprintf("[%s] %s - %s",
        state.RoomCode,
        state.WorldTitle,
        titleStyle.Render(roomName),
    )

    if state.MapOverview == nil {
        return headerStyle.Render(title)
    }

    rooms := make([]string, 0, len(state.MapOverview.Rooms))
    for _, room := range state.MapOverview.Rooms {
        if state.CurrentRoom != nil && room.ID == state.CurrentRoom.ID {
            rooms = append(rooms, "여기: "+strings.Join(room.PlayerNames, ", "))
        } else {
            names := strings.Join(room.PlayerNames, ", ")
            if names == "" {
                names = "비어있음"
            }
            rooms = append(rooms, room.Name+": "+names)
        }
    }

    // 방이 많을 경우 플레이어가 있는 방만 표시하고 나머지는 생략. 80자 터미널에서 오버플로 방지.
    filtered := make([]string, 0, len(rooms))
    for i, room := range state.MapOverview.Rooms {
        isCurrentRoom := state.CurrentRoom != nil && room.ID == state.CurrentRoom.ID
        if isCurrentRoom || room.PlayerCount > 0 {
            filtered = append(filtered, rooms[i])
        }
    }
    if len(filtered) == 0 {
        filtered = rooms // 모두 비어있으면 전체 표시
    }

    overview := dimStyle.Render(strings.Join(filtered, "  /  "))

    return headerStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title, overview))
}
```

**갱신 시점:** 플레이어 이동 시 서버가 `map_info`를 자동 push → `AppModel.state.MapOverview` 갱신 → `View()` 재호출로 HeaderBar 자동 반영.

---

## ChatLog

**역할:** 채팅 메시지와 게임 이벤트를 시간순으로 통합 표시. `bubbles/viewport`로 스크롤 지원.

```go
// internal/client/components/chatlog.go

func renderChatLog(messages []DisplayMessage, vp viewport.Model) string {
    lines := make([]string, 0, len(messages))
    for _, msg := range messages {
        lines = append(lines, renderMessage(msg))
    }
    content := strings.Join(lines, "\n")
    vp.SetContent(content)
    return vp.View()
}
```

`messages`는 채팅과 이벤트를 시간순으로 통합한 슬라이스. `DisplayMessage` 타입은 [store.md](./store.md)에 정의.

viewport 크기는 `tea.WindowSizeMsg` 수신 시 갱신한다:

```go
case tea.WindowSizeMsg:
    headerHeight := 3
    inputHeight  := 3
    m.game.chatViewport.Width  = msg.Width
    m.game.chatViewport.Height = msg.Height - headerHeight - inputHeight
```

---

## InputBar

**역할:** 텍스트 입력 캡처, Enter 시 전송. `bubbles/textinput` 모델 사용.

```go
// internal/client/components/inputbar.go

var inputBarStyle = lipgloss.NewStyle().
    BorderStyle(lipgloss.NormalBorder()).
    BorderTop(true).
    PaddingLeft(1).
    PaddingRight(1)

func renderInputBar(ti textinput.Model) string {
    hint := lipgloss.NewStyle().Faint(true).Render("[/help] [/map]")
    // textinput.View()가 "> " 프롬프트를 포함
    row := lipgloss.JoinHorizontal(lipgloss.Top, ti.View(), "  ", hint)
    return inputBarStyle.Render(row)
}
```

GameModel의 Update에서 Enter 키를 감지하여 입력을 처리한다:

```go
func (m AppModel) updateGame(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    switch msg := msg.(type) {
    case tea.KeyPressMsg:
        if msg.String() == "enter" {
            raw := m.game.textInput.Value()
            m.game.textInput.Reset()
            if raw = strings.TrimSpace(raw); raw == "" {
                return m, nil
            }
            parsed := input.ParseInput(raw)
            result := input.CommandToClientMessage(parsed)
            if result.Message != nil {
                return m, m.network.SendCmd(*result.Message)
            }
            // FR-075 AC3: 구체적인 오류 메시지 표시
            m.state = addSystemMessage(m.state, result.ErrorMsg)
            return m, nil
        }
    }
    // textinput으로 나머지 키 이벤트 전달
    m.game.textInput, cmd = m.game.textInput.Update(msg)
    return m, cmd
}
```

textinput 초기 설정:

```go
func newGameModel() GameModel {
    ti := textinput.New()
    ti.Placeholder = "명령어를 입력하세요 (/help)"
    ti.Prompt = "> "
    ti.Focus()

    return GameModel{
        textInput:    ti,
        chatViewport: viewport.New(viewport.WithWidth(0), viewport.WithHeight(0)),
    }
}
```

**입력 → 메시지 변환:** [input-system.md](./input-system.md) 참조.
