# Story - System Design

**작성일:** 2026-03-01
**기반 문서:** concept.md, prd.md

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Host Machine                             │
│                                                                 │
│  ┌──────────────────────────────────┐  ┌─────────────────────┐  │
│  │          Server (Backend)        │  │   Host Client (TUI) │  │
│  │                                  │  │                     │  │
│  │  ┌────────────┐ ┌─────────────┐  │  │  동일한 Client 코드  │  │
│  │  │  Session    │ │  Game State │  │  │  (로컬 WebSocket)    │  │
│  │  │  Manager    │ │  Manager    │  │  └──────────┬──────────┘  │
│  │  └────────────┘ └─────────────┘  │             │             │
│  │  ┌────────────┐ ┌─────────────┐  │  로컬 WebSocket 연결       │
│  │  │  Message    │ │  Action     │  │             │             │
│  │  │  Router     │ │  Processor  │◄─┼─────────────┘             │
│  │  └────────────┘ └─────────────┘  │                           │
│  │  ┌────────────┐ ┌─────────────┐  │                           │
│  │  │  End Cond.  │ │  Event Bus  │  │                           │
│  │  │  Engine     │ │             │  │                           │
│  │  └────────────┘ └─────────────┘  │                           │
│  │          │                       │                           │
│  │          ▼                       │                           │
│  │  ┌──────────────────────────────────────┐  │                │
│  │  │           AI Layer (Facade)          │  │                │
│  │  │                                      │  │                │
│  │  │  ┌─────────────┐  ┌──────────────┐  │  │                │
│  │  │  │ Orchestrator │  │ 런타임 모듈들  │  │  │                │
│  │  │  │ (세계 생성)   │  │ GM / NPC /   │  │  │                │
│  │  │  │ 멀티 모델     │  │ Action Eval /│  │  │                │
│  │  │  │ 파이프라인    │  │ EndJudge /   │  │  │                │
│  │  │  └──────┬──────┘  │ Ending       │  │  │                │
│  │  │         │         └──────┬───────┘  │  │                │
│  │  │  ┌──────┴──────┐        │           │  │                │
│  │  │  │ Story Bible │   runtimeProvider  │  │                │
│  │  │  │ (캐시)      │   (단일 AIProvider) │  │                │
│  │  │  └─────────────┘        │           │  │                │
│  │  │  ┌──────────────┐ ┌─────┴────────┐  │  │                │
│  │  │  │Story Validator│ │   Provider   │  │  │                │
│  │  │  │(규칙 기반)    │ │  Registry    │  │  │                │
│  │  │  └──────────────┘ └──────┬───────┘  │  │                │
│  │  └──────────────────────────┼──────────┘  │                │
│  │          │                  │              │                │
│  │          ▼                  ▼              │                │
│  │  ┌──────────────────────────────────────┐ │                │
│  │  │       AI Provider Adapters           │ │                │
│  │  │  OpenAI / Anthropic / Gemini /       │ │                │
│  │  │  Grok / DeepSeek                     │ │                │
│  │  └──────────────────────────────────────┘ │                │
│  │       WebSocket Server          │                           │
│  └──────────────────────────────────┘                           │
│                 │                                               │
└─────────────────┼───────────────────────────────────────────────┘
                  │ WebSocket (클라이언트-서버)
       ┌──────────┼──────────┐
       │          │          │
       ▼          ▼          ▼
  ┌─────────┐┌─────────┐┌─────────┐
  │ Client 1 ││ Client 2 ││ Client N │
  │  (TUI)   ││  (TUI)   ││  (TUI)   │
  └─────────┘└─────────┘└─────────┘
```

## 핵심 설계 원칙

| 원칙 | 설명 |
|------|------|
| **서버가 유일한 진실의 원천** | 모든 게임 상태는 서버에서 관리. 클라이언트는 필터링된 뷰만 수신 |
| **이벤트 기반 아키텍처** | 모든 게임 내 변화는 타입화된 이벤트로 흐름 |
| **서버 사이드 가시성 필터링** | 정보 비대칭의 핵심. 클라이언트를 신뢰하지 않음 |
| **AI 출력은 반드시 구조화** | 모든 AI 응답은 JSON 스키마를 따름. 자유 형식 금지 |
| **어댑터 패턴으로 교체 가능** | AI 프로바이더, 향후 네트워크 방식 등을 교체 가능하게 설계 |

## Tech Stack

| 영역 | 기술 | 이유 |
|------|------|------|
| Language | Go 1.26 | 타입 안전성, 정적 바이너리 배포, 단일 바이너리로 클라이언트/서버 포함. Green Tea GC(GC 오버헤드 10~40% 감소), `errors.AsType[T]()` 제네릭 헬퍼, post-quantum TLS 기본 활성화 등 최신 기능 활용 |
| Runtime | Go runtime | 단일 바이너리 배포, 경량 고루틴 기반 비동기 I/O |
| Network | WebSocket ([gorilla/websocket](https://github.com/gorilla/websocket) v1.5.3) | 경량 양방향 통신, 순서 보장, Go 생태계 표준. 서버 비용 $0 (호스트 머신에서 실행) |
| CLI | [Cobra](https://github.com/spf13/cobra) v1.10.2 (`github.com/spf13/cobra`) | CLI 인자 파싱, 서브커맨드(host/join) 분기 |
| TUI | [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) (`charm.land/bubbletea/v2` v2.0.1) + [Lip Gloss v2](https://github.com/charmbracelet/lipgloss) (`charm.land/lipgloss/v2` v2.0.0) + [Bubbles v2](https://github.com/charmbracelet/bubbles) (`charm.land/bubbles/v2` v2.0.0) | ELM 아키텍처 기반 TUI, 선언적 렌더링, 풍부한 스타일링. v2에서 `View()` 반환 타입 `tea.View`, `tea.KeyPressMsg` 등 주요 변경 |
| AI SDK | 커스텀 AI Provider interface ([openai-go v3](https://github.com/openai/openai-go) v3.24.0, [anthropic-sdk-go](https://github.com/anthropics/anthropic-sdk-go) v1.26.0, [google.golang.org/genai](https://pkg.go.dev/google.golang.org/genai) v1.49.0) | 멀티 프로바이더 추상화 (OpenAI, Anthropic, Gemini, Grok, DeepSeek), structured output 지원. Grok/DeepSeek는 OpenAI-compatible API로 `openai-go/v3` + `WithBaseURL()` 사용 |
| Validation | Go struct tags + encoding/json + 커스텀 validator | 런타임 타입 검증, AI 출력 스키마 검증 |
| Build | go build | 표준 Go 빌드 도구, 크로스 컴파일 지원 |
| Test | go test | 표준 Go 테스트 도구, 빠른 실행 |

## 소스 코드 디렉토리 구조

```
cmd/
└── story/
    └── main.go          # CLI 진입점 (Cobra, host / join 분기)

internal/
├── shared/              # 공유 타입, 프로토콜
│   ├── types/           # 도메인 엔티티 타입
│   ├── schema/          # AI 출력 검증용 struct 정의
│   ├── protocol/        # 클라이언트-서버 메시지 타입
│   └── events/          # 게임 이벤트 타입
│
├── server/              # 백엔드 서버
│   ├── network/         # WebSocket 서버, 연결 관리
│   ├── session/         # 세션/로비 관리
│   ├── game/            # 게임 상태 관리
│   ├── message/         # 메시지 라우팅 (가시성 필터링)
│   ├── map/             # 맵 엔진
│   ├── action/          # 플레이어 행동 처리
│   ├── end/             # 종료 조건 판정
│   └── eventbus/        # 내부 이벤트 버스
│
├── client/              # TUI 클라이언트 (Bubble Tea)
│   ├── network/         # WebSocket 클라이언트
│   ├── screens/         # 화면별 모델
│   ├── components/      # 재사용 UI 컴포넌트
│   ├── input/           # 입력 파싱, 명령어 처리
│   ├── renderers/       # 이벤트 타입별 렌더러
│   └── store/           # 로컬 상태 관리
│
└── ai/                  # AI 통합 계층
    ├── provider/        # AI 프로바이더 어댑터 + ProviderRegistry
    ├── orchestrator/    # 멀티 모델 세계 생성 파이프라인
    ├── bible/           # Story Bible 캐시 압축
    ├── worldgen/        # 세계 생성 (Orchestrator 래퍼)
    ├── validator/       # 스토리 검증 (규칙 기반)
    ├── gm/              # GM 엔진
    ├── npc/             # NPC 엔진
    ├── evaluator/       # 행동 평가 (examine, do)
    ├── judge/           # 종료 판정, 엔딩 생성
    └── prompts/         # 프롬프트 템플릿

go.mod
go.sum
```

## 문서 구조

| 문서 | 내용 |
|------|------|
| [shared/](./shared/) | 공유 계층 - 타입, 이벤트, 프로토콜, 스키마 |
| [backend/](./backend/) | 서버 모듈 8개의 설계 |
| [client/](./client/) | TUI 클라이언트 모듈 6개의 설계 |
| [ai/](./ai/) | AI 계층 모듈 10개의 설계 (Provider Registry, Orchestrator, Story Bible 포함) |
| [data-flow.md](./data-flow.md) | 주요 시나리오별 데이터 흐름 |
| [cross-cutting.md](./cross-cutting.md) | 횡단 관심사 (에러, 로깅, shutdown, 토큰) |

## CLI Entry Point

`story host` / `story join` 명령어 분기.

```go
// cmd/story/main.go
package main

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
)

func main() {
    rootCmd := &cobra.Command{Use: "story"}

    hostCmd := &cobra.Command{
        Use:   "host",
        Short: "새 게임 세션을 호스트합니다",
        RunE: func(cmd *cobra.Command, args []string) error {
            port, _ := cmd.Flags().GetInt("port")
            apiKey, err := resolveAPIKey()
            if err != nil {
                return err
            }
            server := NewStoryServer(ServerConfig{APIKey: apiKey, Port: port})
            roomCode, err := server.Start()
            if err != nil {
                return err
            }
            return renderClient(ClientConfig{
                RoomCode:  roomCode,
                ServerURL: fmt.Sprintf("ws://localhost:%d", port),
                IsHost:    true,
            })
        },
    }
    hostCmd.Flags().Int("port", 3000, "WebSocket 서버 포트")

    joinCmd := &cobra.Command{
        Use:   "join <roomCode>",
        Short: "기존 게임에 참가합니다",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            roomCode := args[0]
            serverURL, _ := cmd.Flags().GetString("server")
            return renderClient(ClientConfig{
                RoomCode:  roomCode,
                ServerURL: serverURL,
                IsHost:    false,
            })
        },
    }
    // MVP에서는 --server 플래그 필수. 향후 mDNS/Bonjour 기반 로컬 네트워크 자동 탐색 검토.
    joinCmd.Flags().String("server", "", "WebSocket 서버 URL")

    rootCmd.AddCommand(hostCmd, joinCmd)
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

**호스트 디스커버리:** 호스트가 WebSocket 서버를 시작하면 서버 URL이 표시된다. 원격 참가자는 룸 코드와 서버 URL을 함께 사용해 연결한다. `story join WOLF-7423 --server ws://192.168.1.5:3000` → WebSocket 연결 수립.
