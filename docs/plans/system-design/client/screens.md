# Screens (`internal/client/screens/`)

## 책임

게임 상태(phase)에 따른 화면 전환 관리. 각 화면은 `AppModel.Update()`에서 위임받아 처리하며, `View()`에서 해당 화면의 렌더링 함수를 호출한다.

## 의존하는 모듈

ClientState (AppModel)

## 화면 전환

```go
// internal/client/screens/app.go
// Elm Architecture 루트 모델

type Screen int

const (
    ScreenConnecting Screen = iota
    ScreenNickname   // 닉네임 입력 (연결 직후, 로비 입장 전)
    ScreenLobby
    ScreenGenerating
    ScreenBriefing
    ScreenGame
    ScreenEnding
    ScreenFinished
)

type AppModel struct {
    screen  Screen
    state   ClientState
    network *NetworkClient
    // 화면별 서브 모델
    connecting ConnectingModel
    nickname   NicknameModel
    lobby      LobbyModel
    generating GeneratingModel
    briefing   BriefingModel
    game       GameModel
    ending     EndingModel
}

func (m AppModel) Init() tea.Cmd {
    return m.network.ConnectCmd()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case ScreenChangeMsg:
        m.screen = msg.Screen
        return m, nil
    case ServerMsgReceived:
        // 상태 적용은 루트에서 1회만 수행. 화면 핸들러에서는 재호출하지 않는다.
        m.state = applyServerMessage(m.state, msg.Msg)
        m = m.syncScreenFromPhase()
    case tea.KeyPressMsg:
        if msg.String() == "ctrl+c" {
            return m, tea.Quit
        }
    }

    // 현재 화면에 위임
    switch m.screen {
    case ScreenConnecting:
        return m.updateConnecting(msg)
    case ScreenNickname:
        return m.updateNickname(msg)
    case ScreenLobby:
        return m.updateLobby(msg)
    case ScreenGenerating:
        return m.updateGenerating(msg)
    case ScreenBriefing:
        return m.updateBriefing(msg)
    case ScreenGame:
        return m.updateGame(msg)
    case ScreenEnding:
        return m.updateEnding(msg)
    case ScreenFinished:
        return m.updateFinished(msg)
    }

    return m, nil
}

func (m AppModel) View() tea.View {
    switch m.screen {
    case ScreenConnecting:
        return m.viewConnecting()
    case ScreenNickname:
        return m.nickname.View()
    case ScreenLobby:
        return m.lobby.View(m.state)
    case ScreenGenerating:
        return m.viewGenerating()
    case ScreenBriefing:
        return m.briefing.View(m.state)
    case ScreenGame:
        return m.game.View(m.state)
    case ScreenEnding:
        return m.ending.View(m.state)
    case ScreenFinished:
        return m.viewFinished()
    }
    return tea.NewView("")
}

// 화면 전환 메시지
type ScreenChangeMsg struct {
    Screen Screen
}

// syncScreenFromPhase는 GamePhase에 따라 Screen을 동기화한다.
// 루트 Update()에서 ServerMsgReceived 처리 후 호출.
func (m AppModel) syncScreenFromPhase() AppModel {
    switch m.state.GamePhase {
    case PhaseConnecting:
        m.screen = ScreenConnecting
    case PhaseNickname:
        // 서버 연결 완료 후, join 메시지 전송 전 닉네임 입력 단계
        m.screen = ScreenNickname
    case PhaseLobby:
        m.screen = ScreenLobby
    case PhaseGenerating:
        m.screen = ScreenGenerating
    case PhaseBriefing:
        m.screen = ScreenBriefing
    case PhasePlaying:
        m.screen = ScreenGame
    case PhaseEnding:
        m.screen = ScreenEnding
    case PhaseFinished:
        m.screen = ScreenFinished
    }
    return m
}
```

## 화면별 설계

### ConnectingScreen

서버에 연결 중 표시. `bubbles/spinner` + "연결 중..." 메시지.

```go
// internal/client/screens/connecting.go

type ConnectingModel struct {
    spinner spinner.Model
}

func (m AppModel) viewConnecting() tea.View {
    return lipgloss.NewStyle().
        Padding(2, 4).
        Render(fmt.Sprintf("%s 서버에 연결 중...", m.connecting.spinner.View()))
}
```

### NicknameScreen

서버 연결 완료 후, 로비 입장 전에 표시. 닉네임을 입력받아 `join` 메시지를 전송한다.

```
┌─────────────────────────────────────┐
│         Story - 닉네임 입력          │
│                                     │
│  닉네임을 입력하세요 (1~20자):       │
│  > _                                │
│                                     │
│  Enter로 참가                       │
└─────────────────────────────────────┘
```

- 닉네임 길이 1~20자 검증 (FR-003 AC2)
- 제어 문자 포함 시 오류 표시 (FR-003 AC4)
- 서버에서 중복 닉네임 오류 수신 시 오류 메시지와 함께 재입력 요청
- Enter 제출 시 `join` 메시지에 닉네임 포함하여 전송

```go
// internal/client/screens/nickname.go

// 화면 흐름: ConnectingScreen → NicknameScreen → LobbyScreen
// PhaseNickname은 서버 TCP 연결 성공 직후 클라이언트가 로컬로 설정하는 단계.
// 서버의 "joined" 응답 수신 시 PhaseLobby로 전환.

type NicknameModel struct {
    textInput textinput.Model
    errorMsg  string
}

func (nm NicknameModel) View() tea.View {
    title := lipgloss.NewStyle().Bold(true).Render("Story - 닉네임 입력")

    errLine := ""
    if nm.errorMsg != "" {
        errLine = "\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(nm.errorMsg)
    }

    body := fmt.Sprintf(
        "  닉네임을 입력하세요 (1~20자):\n  %s%s\n\n  Enter로 참가",
        nm.textInput.View(),
        errLine,
    )
    return boxStyle.Render(title + "\n\n" + body)
}

func (m AppModel) updateNickname(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyPressMsg:
        if msg.String() == "enter" {
            nick := strings.TrimSpace(m.nickname.textInput.Value())

            // FR-003 AC2: 길이 검증
            if len([]rune(nick)) < 1 || len([]rune(nick)) > 20 {
                m.nickname.errorMsg = "닉네임은 1자 이상 20자 이하여야 합니다."
                return m, nil
            }

            // FR-003 AC4: 제어 문자 검증
            for _, r := range nick {
                if r < 0x20 {
                    m.nickname.errorMsg = "닉네임에 제어 문자를 포함할 수 없습니다."
                    return m, nil
                }
            }

            m.nickname.errorMsg = ""
            m.state.Nickname = nick
            return m, m.network.SendCmd(ClientMessage{Type: "join", Nickname: nick})
        }
        var cmd tea.Cmd
        m.nickname.textInput, cmd = m.nickname.textInput.Update(msg)
        return m, cmd

    case ServerMsgReceived:
        // 참고: ServerMsgReceived의 전역 상태 변경은 루트 Update()에서 처리.
        // 화면별 핸들러는 화면 로컬 상태만 처리.
        if msg.Msg.Type == "error" && msg.Msg.Code == "DUPLICATE_NICKNAME" {
            // 중복 닉네임: 오류 표시 후 재입력 요청
            m.nickname.errorMsg = "이미 사용 중인 닉네임입니다. 다른 닉네임을 입력하세요."
            m.nickname.textInput.Reset()
        }
        return m, nil
    }
    return m, nil
}
```

### LobbyScreen

```
┌─────────────────────────────────────┐
│         Story - 대기실               │
│                                     │
│  룸 코드: WOLF-7423                 │
│  친구에게 이 코드를 공유하세요!      │
│                                     │
│  접속 중 (3/8):                     │
│    ● Alice (호스트)                 │
│    ● Bob                            │
│    ● Carol                          │
│                                     │
│  [호스트] Enter로 게임 시작          │
└─────────────────────────────────────┘
```

- 플레이어 목록 실시간 갱신
- 호스트에게만 게임 시작 컨트롤 표시
- 새 참가자 입장 시 알림
- 대기 중 간단한 채팅 가능 (FR-081, P1 — MVP 이후)

```go
// internal/client/screens/lobby.go

// MVP에서는 로비 채팅 미지원. 아래 채팅 코드는 P1 구현을 위한 설계 참조용이며, MVP 빌드에서는 비활성화한다.
// (FR-081은 P1. MVP에서는 textInput, messages 필드 사용 안 함.)
type LobbyModel struct{
    textInput textinput.Model   // 로비 채팅 입력 (FR-081, MVP 이후)
    messages  []string          // 로비 채팅 메시지 목록 (FR-081, MVP 이후)
}

func (lm LobbyModel) View(state ClientState) tea.View {
    title := lipgloss.NewStyle().Bold(true).Render("Story - 대기실")

    playerLines := make([]string, 0, len(state.LobbyPlayers))
    for _, p := range state.LobbyPlayers {
        label := p.Nickname
        if p.IsHost {
            label += " (호스트)"
        }
        playerLines = append(playerLines, "  ● "+label)
    }

    hint := ""
    if state.IsHost {
        hint = "\n  [호스트] Enter로 게임 시작"
    }

    body := fmt.Sprintf(
        "  룸 코드: %s\n  친구에게 이 코드를 공유하세요!\n\n  접속 중 (%d/%d):\n%s%s",
        state.RoomCode,
        len(state.LobbyPlayers),
        state.MaxPlayers,
        strings.Join(playerLines, "\n"),
        hint,
    )

    return boxStyle.Render(title + "\n\n" + body)
}

func (m AppModel) updateLobby(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case ServerMsgReceived:
        // 상태 적용은 루트 Update()의 applyServerMessage에서 완료됨. 여기서는 재호출하지 않음.
        // 로비 채팅 메시지 수신 (FR-081, MVP 이후)
        // 주의: lobby_chat은 protocol.md에 미정의. MVP에서는 로비 채팅 미지원.
        // MVP 이후 구현 시 protocol.md에 lobby_chat 메시지 타입 추가 필요.
        if msg.Msg.Type == "chat_message" && m.state.GamePhase == PhaseLobby {
            m.lobby.messages = append(m.lobby.messages,
                msg.Msg.SenderName+": "+msg.Msg.Content)
        }
        return m, nil
    case tea.KeyPressMsg:
        if msg.String() == "enter" {
            // 호스트: 빈 입력 시 게임 시작
            raw := strings.TrimSpace(m.lobby.textInput.Value())
            if raw == "" && m.state.IsHost {
                return m, m.network.SendCmd(ClientMessage{Type: "start_game"})
            }
            // 로비 채팅 전송 (FR-081, MVP 이후)
            // MVP에서는 로비 채팅 미지원. MVP 이후 구현 시 protocol.md에 메시지 타입 추가 필요.
            if raw != "" {
                m.lobby.textInput.Reset()
                return m, m.network.SendCmd(ClientMessage{Type: "chat", Content: raw})
            }
        }
        var cmd tea.Cmd
        m.lobby.textInput, cmd = m.lobby.textInput.Update(msg)
        return m, cmd
    }
    return m, nil
}
```

### GeneratingScreen

```
┌─────────────────────────────────────┐
│                                     │
│  AI가 세계를 만들고 있습니다...      │
│                                     │
│  ████████████░░░░░░░  60%           │
│  역할을 배정하는 중...               │
│                                     │
└─────────────────────────────────────┘
```

- 단계별 진행 메시지 표시
- `bubbles/progress` 바

```go
// internal/client/screens/generating.go

type GeneratingModel struct {
    progress progress.Model
}

func (m AppModel) viewGenerating() tea.View {
    msg := m.state.GenerationMessage
    if msg == "" {
        msg = "세계를 생성하는 중..."
    }
    bar := m.generating.progress.View()
    return lipgloss.NewStyle().Padding(2, 4).Render(
        "AI가 세계를 만들고 있습니다...\n\n" + bar + "\n" + msg,
    )
}
```

### BriefingScreen

4단계로 구성 (FR-078: 공개 정보 → 읽음 확인 → 개인 정보 → 준비 확인):

**Phase 1: 공개 브리핑**
```
┌─────────────────────────────────────┐
│         ═══ 브리핑 ═══              │
│                                     │
│  [저택의 밤]                        │
│                                     │
│  1920년 어느 겨울밤, 블랙우드       │
│  저택에 6명의 손님이 초대되었다...   │
│                                     │
│  등장인물:                          │
│    - 박사 에드워드: 피해자의 친구    │
│    - 메이드 클라라: 저택의 하녀      │
│    ...                              │
│                                     │
│  Enter를 눌러 다음으로              │
└─────────────────────────────────────┘
```

**Phase 2: 개인 역할**
```
┌─────────────────────────────────────┐
│     ═══ 당신의 역할 (비공개) ═══    │
│                                     │
│  이름: 박사 에드워드 크레인          │
│  배경: 피해자의 오랜 친구이자 라이벌 │
│                                     │
│  [개인 목표]                        │
│  - 피해자가 훔쳐간 연구 노트를      │
│    되찾아라                         │
│                                     │
│  [비밀]                             │
│  어젯밤 11시, 당신은 서재에 있었다   │
│                                     │
│  준비가 되면 Enter를 누르세요        │
└─────────────────────────────────────┘
```

- 공개 브리핑은 모든 플레이어에게 동일
- 개인 역할은 본인에게만 표시
- 모든 플레이어가 읽음 확인 후 개인 정보 전달 (FR-078 AC2)
- 모든 플레이어가 "준비 완료"하면 게임 시작

```go
// internal/client/screens/briefing.go

type BriefingPhase int

const (
    BriefingPublic         BriefingPhase = iota // 공개 브리핑 표시
    BriefingWaitingPrivate                      // 읽음 확인 전송 후 다른 플레이어 대기
    BriefingPrivate                             // 개인 역할 표시
    BriefingWaitingReady                        // 준비 완료 전송 후 게임 시작 대기
)

type BriefingModel struct {
    phase    BriefingPhase
    viewport viewport.Model  // 긴 브리핑 텍스트 스크롤
}

func (m AppModel) updateBriefing(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyPressMsg:
        if msg.String() == "enter" {
            switch m.briefing.phase {
            case BriefingPublic:
                // 읽음 확인을 서버에 전송, 다른 플레이어 대기 (FR-078 AC2)
                // protocol.md ReadyMessage (type: "ready", phase: "briefing_read")
                m.briefing.phase = BriefingWaitingPrivate
                return m, m.network.SendCmd(ClientMessage{Type: "ready", Phase: "briefing_read"})
            case BriefingPrivate:
                // 준비 완료 전송 (FR-078 AC4)
                // protocol.md ReadyMessage (type: "ready", phase: "game_ready")
                m.briefing.phase = BriefingWaitingReady
                return m, m.network.SendCmd(ClientMessage{Type: "ready", Phase: "game_ready"})
            }
        }
    case ServerMsgReceived:
        // 참고: ServerMsgReceived의 전역 상태 변경은 루트 Update()에서 처리.
        // 화면별 핸들러는 화면 로컬 상태만 처리.
        // 루트 Update()가 applyServerMessage()로 ClientState를 갱신하고,
        // syncScreenFromPhase()로 screen 전환을 처리한다.
        // 여기서는 BriefingModel.phase (화면 로컬) 전환만 담당한다.
        if msg.Msg.Type == "briefing_private" {
            m.briefing.phase = BriefingPrivate
        }
        // 화면 전환은 syncScreenFromPhase()에서 처리
        return m, nil
    }
    return m, nil
}
```

### GameScreen

메인 게임 화면. 3분할 레이아웃. [components.md](./components.md) 참조.

```go
// internal/client/screens/game.go

type GameModel struct {
    chatViewport viewport.Model
    textInput    textinput.Model
}

func (gm GameModel) View(state ClientState) tea.View {
    header := renderHeader(state)
    chat := renderChatLog(state.Messages, gm.chatViewport)
    input := renderInputBar(gm.textInput)

    return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, header, chat, input))
}
```

### EndingScreen

```
┌─────────────────────────────────────┐
│         ═══ 게임 종료 ═══           │
│                                     │
│  범인은 Bob이었습니다.              │
│                                     │
│  ── 당신의 이야기 ──                │
│  Alice는 끝까지 Carol을 의심했지만, │
│  마지막 순간 Bob의 거짓말을         │
│  간파했습니다.                      │
│                                     │
│  [개인 목표 결과]                   │
│  ✓ 연구 노트를 되찾았습니다         │
│  ✗ 비밀이 발각되었습니다            │
│                                     │
│  Enter를 눌러 비밀 공개 보기         │
└─────────────────────────────────────┘
```

- 공통 결과 → 개인 엔딩 → 비밀 공개 → 피드백 입력 순서로 표시 (FR-079 AC4)
- `bubbles/viewport`로 스크롤 가능한 긴 텍스트
- 비밀 공개 확인 후 피드백 입력 UI로 전환

**EndingFeedback 피드백 입력 화면:**
```
┌─────────────────────────────────────┐
│         ═══ 피드백 ═══              │
│                                     │
│  스토리 재미도: ★★★☆☆ (3/5)        │
│  ← → 키로 조정                      │
│                                     │
│  몰입도: ★★★★☆ (4/5)               │
│  ← → 키로 조정                      │
│                                     │
│  한 줄 감상 (선택): _               │
│                                     │
│  Enter: 제출   Esc: 건너뛰기        │
└─────────────────────────────────────┘
```

- 좌우 화살표 키로 1~5 평점 조정
- Tab 키로 재미도 ↔ 몰입도 ↔ 감상 입력 간 포커스 이동
- 감상 텍스트 입력은 선택 사항
- Enter로 제출, Esc로 건너뛰기 (skip_feedback 전송)

```go
// internal/client/screens/ending.go

type EndingPhase int

const (
    EndingResult   EndingPhase = iota // 공통 결과 + 개인 엔딩
    EndingReveal                      // 비밀 공개
    EndingFeedback                    // 피드백 입력 (FR-079 AC4)
)

type EndingModel struct {
    phase        EndingPhase
    viewport     viewport.Model
    funRating    int              // 1~5 스토리 재미도
    immersion    int              // 1~5 몰입도
    commentInput textinput.Model  // 자유 텍스트 (선택)
    focusField   int              // 0: funRating, 1: immersion, 2: comment
}

func renderStars(rating int) string {
    stars := ""
    for i := 1; i <= 5; i++ {
        if i <= rating {
            stars += "★"
        } else {
            stars += "☆"
        }
    }
    return stars
}

func (em EndingModel) viewFeedback() tea.View {
    title := lipgloss.NewStyle().Bold(true).Render("═══ 피드백 ═══")

    funLine := fmt.Sprintf("  스토리 재미도: %s (%d/5)", renderStars(em.funRating), em.funRating)
    immLine := fmt.Sprintf("  몰입도:        %s (%d/5)", renderStars(em.immersion), em.immersion)

    // 포커스 표시
    arrowHint := "  ← → 키로 조정"
    if em.focusField == 0 {
        funLine = lipgloss.NewStyle().Bold(true).Render(funLine) + "\n" + arrowHint
        immLine = "  " + immLine[2:]
    } else if em.focusField == 1 {
        immLine = lipgloss.NewStyle().Bold(true).Render(immLine) + "\n" + arrowHint
    }

    commentLine := fmt.Sprintf("  한 줄 감상 (선택): %s", em.commentInput.View())

    body := lipgloss.JoinVertical(lipgloss.Left,
        funLine,
        "",
        immLine,
        "",
        commentLine,
        "",
        "  Enter: 제출   Esc: 건너뛰기",
    )

    return boxStyle.Render(title + "\n\n" + body)
}

func (m AppModel) updateEnding(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyPressMsg:
        if msg.String() == "enter" {
            switch m.ending.phase {
            case EndingResult:
                m.ending.phase = EndingReveal
            case EndingReveal:
                // 비밀 공개 확인 후 피드백 입력 UI 대기.
                // 피드백 입력 전환은 서버의 'feedback_request' 메시지 수신 시 ServerMsgReceived 핸들러에서 처리.
                // (local Enter keypress로 EndingFeedback으로 직접 전환하지 않음 — Fix #4)
            case EndingFeedback:
                // 피드백 전송 후 종료
                var comment *string
                if v := m.ending.commentInput.Value(); v != "" {
                    comment = &v
                }
                return m, m.network.SendCmd(ClientMessage{
                    Type:            "submit_feedback",
                    FunRating:       m.ending.funRating,
                    ImmersionRating: m.ending.immersion,
                    Comment:         comment,  // protocol.md: *string
                })
            }
        }
        if m.ending.phase == EndingFeedback {
            switch msg.String() {
            case "esc":
                // 피드백 건너뛰기
                return m, m.network.SendCmd(ClientMessage{Type: "skip_feedback"})
            case "tab":
                // 포커스 순환: funRating → immersion → comment → funRating
                m.ending.focusField = (m.ending.focusField + 1) % 3
                return m, nil
            case "left":
                // 좌: 현재 포커스 필드 평점 -1 (최소 1)
                switch m.ending.focusField {
                case 0:
                    if m.ending.funRating > 1 { m.ending.funRating-- }
                case 1:
                    if m.ending.immersion > 1 { m.ending.immersion-- }
                }
                return m, nil
            case "right":
                // 우: 현재 포커스 필드 평점 +1 (최대 5)
                switch m.ending.focusField {
                case 0:
                    if m.ending.funRating < 5 { m.ending.funRating++ }
                case 1:
                    if m.ending.immersion < 5 { m.ending.immersion++ }
                }
                return m, nil
            }
            // 감상 입력 포커스 시 텍스트 입력 위임
            if m.ending.focusField == 2 {
                var cmd tea.Cmd
                m.ending.commentInput, cmd = m.ending.commentInput.Update(msg)
                return m, cmd
            }
        }
        if msg.String() == "esc" && m.ending.phase != EndingFeedback {
            // EndingFeedback 외에서의 Esc는 무시
            return m, nil
        }
    case ServerMsgReceived:
        // 상태 적용은 루트 Update()에서 완료됨. 화면 전환도 syncScreenFromPhase()가 처리.
        // 'feedback_request' 수신 시 EndingFeedback으로 전환 (서버 주도 전환, local Enter keypress 아님).
        if msg.Msg.Type == "feedback_request" {
            m.ending.phase = EndingFeedback
            m.ending.funRating = 3   // 기본값
            m.ending.immersion = 3   // 기본값
            m.ending.focusField = 0  // 재미도 포커스 시작
        }
        return m, nil
    }
    return m, nil
}
```

### FinishedScreen

게임 완전 종료. "다시 플레이하려면..." 안내 메시지. `Ctrl+C`로 종료.

```go
// internal/client/screens/finished.go

func (m AppModel) viewFinished() tea.View {
    title := lipgloss.NewStyle().Bold(true).Render("═══ 게임 종료 ═══")
    body := lipgloss.JoinVertical(lipgloss.Left,
        "  게임이 완전히 종료되었습니다.",
        "",
        "  다시 플레이하려면 새 게임을 시작하세요.",
        "  Ctrl+C로 프로그램을 종료합니다.",
    )
    return boxStyle.Render(title + "\n\n" + body)
}

func (m AppModel) updateFinished(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Ctrl+C는 루트 Update()에서 처리됨. 여기서는 추가 처리 없음.
    return m, nil
}
```
