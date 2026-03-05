# Client - TUI (`internal/client/`)

모든 플레이어(호스트 포함)가 실행하는 터미널 UI. **Bubble Tea v2 (Elm Architecture for CLI)** 기반.

> **Bubble Tea v2 / Lip Gloss v2 / Bubbles v2 마이그레이션 노트:**
> - Import 경로: `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, `charm.land/bubbles/v2`
> - `View()` 반환 타입: `string` → `tea.View` (`tea.NewView("...")` 사용)
> - 키 이벤트: `tea.KeyMsg` → `tea.KeyPressMsg`, 키 매칭은 `msg.String()` 사용 (예: `"ctrl+c"`, `"enter"`, `"esc"`, `"tab"`, `"left"`, `"right"`)
> - `tea.Sequentially()` → `tea.Sequence()`
> - `lipgloss.Color()` → `image/color.Color` 반환 (ANSI 문자열 "1"~"255" 그대로 사용 가능)

## 모듈 목록

| 모듈 | 파일 | 핵심 책임 |
|------|------|-----------|
| [Screens](./screens.md) | `internal/client/screens/` | 화면 전환, 화면별 Update/View |
| [Components](./components.md) | `internal/client/components/` | 재사용 UI 렌더링 함수 (HeaderBar, ChatLog, InputBar) |
| [InputSystem](./input-system.md) | `internal/client/input/` | 입력 파싱, 명령어 → ClientMessage 변환 |
| [EventRenderers](./event-renderers.md) | `internal/client/renderers/` | 이벤트 타입별 시각적 렌더링 |
| [NetworkClient](./network-client.md) | `internal/client/network/` | WebSocket 연결, 메시지 송수신, tea.Cmd 통합 |
| [ClientState](./store.md) | `internal/client/state/` | 클라이언트 상태 구조체, AppModel |

## 모듈 의존성 그래프

```
              ┌─────────┐
              │   App   │ (tea.Model 루트, Elm Architecture)
              └────┬────┘
                   │
          ┌────────┼────────┐
          │        │        │
     ┌────▼───┐┌───▼────┐┌──▼───────┐
     │ Screen ││ State  ││ Network  │
     │Manager ││(Model) ││ Client   │
     └────┬───┘└───┬────┘└──┬───────┘
          │        │        │
   ┌──────┼───┐    │        │
   │      │   │    │        │
┌──▼──┐┌──▼┐┌▼──┐  │        │
│Lobby││Game││End│  │        │
│View ││View││View│  │        │
└─────┘└──┬─┘└───┘  │        │
          │         │        │
    ┌─────┼─────┐   │        │
    │     │     │   │        │
┌───▼┐┌───▼┐┌──▼──┐│        │
│Chat││Head││Input│◄┘        │
│Log ││Bar ││Sys. │          │
└────┘└────┘└──┬──┘          │
               │             │
          ┌────▼────┐        │
          │ Command │        │
          │ Parser  │────────┘
          └─────────┘   (파싱된 메시지를 NetworkClient로 전송)
```

## 데이터 흐름

```
서버에서 메시지 수신
    │
    ▼
NetworkClient goroutine이 읽기 대기 중
    │
    ▼
ServerMsgReceived (tea.Msg) 반환
    │
    ▼
AppModel.Update()가 메시지 타입에 따라 Model 필드 갱신
    │
    ▼
View()가 갱신된 Model로 화면 문자열 생성 (Lip Gloss 스타일링)
```

```
사용자 입력
    │
    ▼
tea.KeyPressMsg로 수신 (Bubble Tea v2 런타임)
    │
    ▼
textinput 모델이 문자 축적, Enter 감지
    │
    ▼
CommandParser.Parse(raw)
    │
    ▼
commandToClientMessage(parsed)
    │
    ▼
NetworkClient.Send(clientMessage) tea.Cmd로 실행
```
