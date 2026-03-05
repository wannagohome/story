# Story — Agent-Driven QA Plan

**작성일:** 2026-03-03
**기반 문서:** concept.md, prd.md, system-design/
**테스트 방법론:** LLM 에이전트 플레이테스트 (기존 Unit/Integration/E2E/Manual Playtest 완전 대체)

---

## 1. 개요

기존 QA 플랜(Unit/Integration/E2E/Manual Playtest)을 **LLM 에이전트 플레이테스트**로 완전 대체한다.

여러 AI 에이전트가 실제 플레이어처럼 게임에 접속하여 플레이하고, 플레이 과정에서 모든 FR/NFR을 체험적으로 검증한다.

> **📒 QA Journal**: 세션 간 기억을 유지하기 위한 living document → [`qa-journal.md`](./qa-journal.md)
> QA 세션 시작 시 **반드시 먼저 읽고**, 세션 종료 시 발견 사항을 **반드시 기록**할 것.

```
기존:  TestCase → Assert(expected, actual)
새로:  Agent.play() → Agent.observe() → Agent.evaluate() → Report
```

### 핵심 결정 사항

| 항목 | 결정 |
|------|------|
| 기존 QA 플랜과의 관계 | **완전 대체** |
| 에이전트 유형 | LLM Agent Players (Claude/GPT) |
| 상호작용 방식 | **Protocol level (WebSocket) + TUI (stdin/stdout)** 둘 다 |
| 테스트 구조 | 3-Layer 피라미드 (Command → Phase → Full Game) |

---

## 2. 테스트 피라미드

```
        ╱╲
       ╱  ╲        Layer 4: Subagent Playtest (탐색적, 수동 트리거)
      ╱ TUI╲       └ Claude subagent 2~4명이 TUI로 실제 플레이. 체험 기반 발견.
     ╱──────╲
    ╱        ╲      Layer 3: Full Game (느림, 비쌈)
   ╱ Protocol ╲     └ 코딩된 에이전트 4~8명이 한 판 완주. 속성 기반 검증.
  ╱  + TUI     ╲
 ╱──────────────╲
╱  L2: Phase     ╲   └ 페이즈별 에이전트 2~4명이 특정 구간 집중 테스트
╲────────────────╱
╱  L1: Command    ╲   └ 단일 명령 → 응답 검증. 기존 Unit/Integration 대체
╲──────────────────╱
```

| 레이어 | 인터랙션 | 속도 | LLM 비용 | 커버리지 |
|--------|---------|------|---------|---------|
| **L1: Command** | Protocol | 초 단위 | 없음~최소 (fixture AI) | 개별 FR 기능 검증 |
| **L2: Phase** | Protocol | 분 단위 | 중간 (에이전트 의사결정) | 페이즈 내 통합, 정보 비대칭 교차 검증 |
| **L3: Full Game** | Protocol + TUI | 10분+ | 높음 (자유 플레이) | 전체 플로우, UX, 스토리 품질, TUI 렌더링 |
| **L4: Subagent Playtest** | TUI (tmux) | 15~40분 | 매우 높음 (subagent 전체) | 탐색적 버그 발견, AI 품질 정성 평가, UX 체감 |

### 테스트 시나리오 ID 규칙

`L{레이어}-{카테고리약어}-{순번}` (예: L1-SM-001 = Layer 1 Session Management #001)

| 약어 | 카테고리 |
|------|---------|
| SM | Session Management (세션 관리) |
| WG | World Generation (세계 생성) |
| SV | Story Verification (스토리 검증) |
| MP | Map System (맵) |
| CH | Chat & Communication (채팅) |
| PA | Player Actions (플레이어 행동) |
| IA | Information Asymmetry (정보 비대칭) |
| NP | NPC System (NPC) |
| GM | GM System (GM 시스템) |
| EN | Ending System (엔딩) |
| SF | Save & Feedback (저장/피드백) |
| AF | Additional Features (추가 기능) |
| TU | Terminal UI (터미널 UI) |
| NF | Non-Functional (비기능) |
| GS | Game Structure (게임 구조) |
| FG | Full Game (풀 게임) |

**L2 ID 권고 규칙:** 현재 L2는 `Phase 2A/2B/...`로만 표기되어 있으므로, 추적성 강화를 위해 신규/보강 시나리오에는 `L2-{Phase}-{순번}` 형식을 권고한다.
- 예시: `L2-2A-001`, `L2-2B-001`
- 표기 방식: 기존 Phase 표기와 병기 (`Phase 2B (L2-2B-001)`)

---

## 3. 에이전트 아키텍처

### 3.1 ProtocolAgent

WebSocket으로 서버와 직접 통신. TUI 없이 프로토콜 메시지만 주고받음.

```
┌─────────────────────────────────┐
│         ProtocolAgent           │
│                                 │
│  ┌──────────┐  ┌─────────────┐  │
│  │  Persona  │  │   Memory    │  │
│  │ (성격/전략)│  │(받은 메시지, │  │
│  └──────────┘  │ 알려진 정보, │  │
│                │ 방문한 방)   │  │
│  ┌──────────┐  └─────────────┘  │
│  │ Decision  │                  │
│  │ Engine   │ ← LLM 호출       │
│  │ (다음행동) │                  │
│  └──────────┘                   │
│  ┌──────────────────────────┐   │
│  │  Assertion Rules         │   │
│  │  (FR 검증 로직, 비동기)   │   │
│  └──────────────────────────┘   │
│            │                    │
│    WebSocket                    │
└────────────┼────────────────────┘
             │
         Server (Host)
```

**용도:** L1, L2, L3 (프로토콜 검증 담당)

### 3.2 TUIAgent

실제 CLI 프로세스를 child_process로 스폰하고 stdin/stdout으로 상호작용.

```
┌─────────────────────────────────┐
│           TUIAgent              │
│                                 │
│  ┌──────────┐  ┌─────────────┐  │
│  │  Persona  │  │   Memory    │  │
│  └──────────┘  │(화면 텍스트  │  │
│                │ 히스토리)    │  │
│  ┌──────────┐  └─────────────┘  │
│  │ Decision  │                  │
│  │ Engine   │ ← LLM 호출       │
│  └──────────┘                   │
│  ┌──────────────────────────┐   │
│  │  Screen Parser           │   │
│  │  (ANSI strip → clean text)│  │
│  └──────────────────────────┘   │
│            │                    │
│    child_process (stdin/stdout) │
│    → npx story join ROOM-CODE  │
└─────────────────────────────────┘
```

**용도:** L3 전용 (TUI 렌더링, 사용자 경험 검증)

**TUI 파싱 방식:**
1. `child_process.spawn('npx', ['story', 'join', roomCode])`
2. stdout 데이터를 ANSI escape code strip (`strip-ansi` 패키지)
3. 정제된 텍스트를 LLM에 "화면에 보이는 내용"으로 전달
4. LLM이 다음 입력(채팅 or `/명령어`)을 결정
5. stdin에 텍스트 write + `\n`

### 3.3 공통 구성 요소

```go
type AgentConfig struct {
    ID              string
    Persona         AgentPersona
    InteractionMode string // "protocol" | "tui"
    AssertionRules  []AssertionRule
}

type AgentPersona struct {
    Name        string
    PlayStyle   string   // "explorer" | "diplomat" | "strategist" | "chaotic" | "observer"
    Description string   // LLM 시스템 프롬프트에 포함
    Traits      []string // e.g., "모든 방을 탐색한다", "NPC와 적극 대화한다"
}

type AgentMemory struct {
    ReceivedMessages []ServerMessage
    KnownInfo        struct {
        PublicBriefing  *PublicInfo
        PrivateBriefing *PlayerRole
        DiscoveredClues []Clue
        VisitedRooms    []string
        NPCConversations map[string][]string
    }
    CurrentRoom *RoomView
    Inventory   []Item
}
```

---

## 4. 에이전트 페르소나

각 에이전트에 다른 성격을 부여해서 다양한 플레이 패턴을 커버한다.

| 페르소나 | 플레이 스타일 | 커버하는 영역 |
|---------|-------------|-------------|
| **Explorer** | 맵 전체를 돌아다니며 모든 방 조사 | 이동(FR-032), /examine(FR-041), 맵(FR-031), 단서(FR-012) |
| **Diplomat** | 대화 위주, NPC와 적극 소통 | 채팅(FR-036~037), NPC 대화(FR-043), 아이템 교환(FR-047~048) |
| **Strategist** | 목표 지향, 효율적 행동 | 개인 목표(FR-092), 투표(FR-049), /end(FR-091), 종료 조건 |
| **Chaotic** | 예상 밖 행동, 엣지 케이스 유발 | 잘못된 명령, 비연결 방 이동, 빈 입력, 존재하지 않는 NPC |
| **Observer** | 최소 행동, 주로 관찰 | 정보 비대칭(FR-050~054), 방 범위 채팅 격리, 수동적 수신 검증 |

---

## 5. Layer 2: Phase Agent 시나리오

### 5.1 개요

- **인터랙션**: Protocol level
- **AI 사용**: LLM이 에이전트 의사결정 (매 턴 1회 호출)
- **목적**: 게임 페이즈별 2~4 에이전트가 집중 테스트
- **서버 AI**: Phase 2B는 실제 AI, 나머지는 fixture 가능

### 5.2 Phase 시나리오 (상세는 개별 tc-*.md 참조)

| Phase | 구간 | Traces | 상세 파일 |
|-------|------|--------|----------|
| Phase 2A | Lobby | FR-001~006, FR-008, FR-081~082, FR-085~086 | [tc-session-management.md](./tc-session-management.md), [tc-additional.md](./tc-additional.md) (FR-085~086) |
| Phase 2B | World Generation & Verification | FR-009~019, FR-020~023, FR-025, FR-092, FR-093 | [tc-world-generation.md](./tc-world-generation.md), [tc-story-verification.md](./tc-story-verification.md) |
| Phase 2C | Briefing & Information Asymmetry | FR-050~054, FR-078 | [tc-info-asymmetry.md](./tc-info-asymmetry.md) |
| Phase 2D | Exploration & Actions | FR-031~035, FR-041~048, FR-055~058, FR-084, FR-087~088 | [tc-map-system.md](./tc-map-system.md), [tc-player-actions.md](./tc-player-actions.md), [tc-npc.md](./tc-npc.md), [tc-additional.md](./tc-additional.md) (FR-087~088) |
| Phase 2E | GM & Story Progression | FR-059~062, FR-030 | [tc-gm.md](./tc-gm.md) |
| Phase 2F | Ending | FR-063, FR-066~068, FR-091, FR-092 | [tc-ending.md](./tc-ending.md) |

---

## 6. Layer 3: Full Game Agent 시나리오

### 6.1 개요

- **인터랙션**: Protocol + TUI (동시에 두 종류 에이전트 참여)
- **AI 사용**: 실제 AI (세계 생성, NPC, GM, 행동 평가)
- **LLM 의사결정**: 에이전트가 자유롭게 플레이 (페르소나에 따라)
- **목적**: 완전한 게임 한 판 체험. 기능 검증 + 스토리 품질 + UX 평가.

### 6.2 오케스트레이션

```
┌──────────────────────────────────────────┐
│           AgentOrchestrator              │
│                                          │
│  Host(TUIAgent) ─── npx story host       │
│  Agent1(TUIAgent) ── npx story join XXX  │
│  Agent2(ProtocolAgent) ── WebSocket     │
│  Agent3(ProtocolAgent) ── WebSocket     │
│  Agent4(TUIAgent) ── npx story join XXX  │
│                                          │
│  ┌─────────────────────────────┐         │
│  │    Assertion Collector      │         │
│  │  (실시간 프로토콜 검증)      │         │
│  └─────────────────────────────┘         │
│  ┌─────────────────────────────┐         │
│  │    Game Transcript Logger   │         │
│  │  (전체 메시지 타임라인 기록) │         │
│  └─────────────────────────────┘         │
└──────────────────────────────────────────┘
```

### 6.3 풀 게임 시나리오

| ID | 시나리오 | 에이전트 | 페르소나 | 특수 검증 |
|----|---------|---------|---------|----------|
| **FG-001** | Happy Path 풀 게임 | 4 (2 TUI + 2 Protocol) | Explorer, Diplomat, Strategist, Observer | 전체 플로우 완주, 크래시 없음 |
| **FG-002** | 최대 인원 스트레스 | 8 (2 TUI + 6 Protocol) | 혼합 | 8명 동시 안정성 (FR-006, NFR-004) |
| **FG-003** | 최소 인원 게임 | 2 (2 TUI) | Explorer, Diplomat | 2인 세계 적절성 |
| **FG-004** | Adversarial Play | 4 (1 TUI + 3 Protocol) | Chaotic×2, Observer×2 | 잘못된 입력 내성, 서버 안정성 |
| **FG-005** | 연결 해제/복구 | 4 (Protocol) | 혼합 + 1명 중간 disconnect | FR-007 재접속 + 상태 복원 |
| **FG-006** | 정보 비대칭 집중 | 4 (Protocol) | Observer×4 | 전체 메시지 교차 검증, 정보 누출 0건 |
| **FG-007** | TUI 경험 검증 | 4 (4 TUI) | 혼합 | 렌더링, 레이아웃, 메시지 구분, UX |
| **FG-008** | NPC 집중 상호작용 | 4 (Protocol) | Diplomat×2, Explorer, Strategist | NPC 퍼소나, 기믹, 아이템 교환 |
| **FG-009** | 투표 종료 게임 | 4 (2 TUI + 2 Protocol) | Strategist×2, Explorer, Observer | 투표 → 판정 → 엔딩 풀 플로우; 개인 목표 달성/미달성 평가 정확성 확인 |
| **FG-010** | /end 플레이어 종료 | 4 (Protocol) | Strategist×4 | 조기 종료 발의 → 투표 → 엔딩; 개인 목표 달성/미달성 평가 정확성 확인 |

---

## 7. 에이전트 QA 리포트

### 7.1 개별 리포트

게임 종료 후 각 에이전트가 LLM으로 자신의 플레이 경험을 평가한다.

```go
type AgentQAReport struct {
    AgentID             string
    Persona             string
    InteractionMode     string  // "protocol" | "tui"
    GameDurationMinutes float64

    // === 기능 검증 ===
    FunctionalIssues []struct {
        Description string
        Severity    string   // "critical" | "major" | "minor"
        RelatedFR   []string
        Evidence    string   // 실제 받은 메시지/상황
        Timestamp   int64
    }

    // === 스토리 품질 평가 ===
    StoryEvaluation struct {
        Coherence             int    // 1-5: 설정 일관성, 모순 여부
        Engagement            int    // 1-5: 재미, 몰입도
        Pacing                int    // 1-5: 진행 속도 적절성
        PersonalGoalClarity   int    // 1-5: 개인 목표 명확성
        PersonalGoalAchievable bool  // 개인 목표 달성 경로 존재 여부
        EndingSatisfaction    int    // 1-5: 엔딩 만족도
        NarrativeComments     string // 자유 텍스트 평가
    }

    // === 정보 비대칭 검증 ===
    InformationIntegrity struct {
        ReceivedOthersSecrets bool     // 다른 플레이어 비밀 수신 여부 (false여야 함)
        MissingOwnInfo        bool     // 자기 정보 누락 여부 (false여야 함)
        CrossRoomLeaks        []string // 다른 방 채팅 수신 내역 (비어야 함)
        SemiPublicCorrect     bool     // 반공개 정보 올바른 그룹에만 수신
    }

    // === TUI 경험 (TUIAgent만) ===
    TUIEvaluation *struct {
        LayoutReadable              bool     // 레이아웃 가독성
        MessageTypesDistinguishable bool     // 메시지 유형 시각적 구분
        CommandsResponsive          bool     // 명령어 즉각 응답
        ScrollWorking               bool     // 스크롤 동작
        RenderingGlitches           []string // 렌더링 결함 상세
        HeaderInfoPresent           bool     // 헤더에 룸코드, 위치, 플레이어 현황
        BriefingScreenClear         bool     // 브리핑 화면 명확성
        EndingScreenComplete        bool     // 엔딩 화면 완결성
    }

    // === 행동 로그 요약 ===
    ActionSummary struct {
        TotalActions      int
        MoveCount         int
        ChatCount         int
        ExamineCount      int
        NPCTalkCount      int
        UniqueRoomsVisited int
        CluesFound        int
    }
}
```

### 7.2 집계 리포트

```go
type AggregatedTestReport struct {
    Scenario   string
    AgentCount int
    Duration   float64

    // Pass/Fail 판정
    Verdict        string // "PASS" | "FAIL"
    CriticalIssues int    // > 0 이면 FAIL
    MajorIssues    int    // > 3 이면 FAIL
    MinorIssues    int

    // 스토리 품질 평균
    AvgCoherence          float64
    AvgEngagement         float64
    AvgPacing             float64
    AvgEndingSatisfaction float64

    // 정보 비대칭 PASS/FAIL
    InformationLeaks int // > 0 이면 FAIL

    // TUI (TUI Agent가 있는 경우만)
    TUIIssues []string

    // 전체 Assertion 결과
    AssertionResults struct {
        Total   int
        Passed  int
        Failed  int
        Details []AssertionResult
    }
}
```

---

## 8. L1 테스트 시나리오 (상세)

### 8.0 프로토콜 필드명 참고사항

**프로토콜 필드명 정렬 안내**: TC 문서의 메시지 타입 및 필드명은 현재 설계 의도를 반영하며, 구현 단계에서 `docs/plans/system-design/shared/protocol.md` 및 `events.md`와 최종 정렬됩니다. 구현 시 프로토콜 문서가 변경되면 해당 TC의 필드명도 함께 갱신해야 합니다.

### 8.1 PRD 용어 ↔ 프로토콜 메시지 매핑 (종료 이벤트)

| PRD 용어 | 프로토콜 메시지 | 의미 |
|----------|----------------|------|
| `game_end` | `game_ending` | 종료 판정 결과 전달 시점 (종료 조건 충족 직후) |
| `game_end` | `game_finished` | 엔딩 처리 완료 후 세션 종료 확정 시점 |

> 모든 `tc-*.md` 문서는 종료 이벤트 검증 시 위 매핑을 기준으로 해석한다.
> 즉, PRD의 `game_end` 요구사항은 프로토콜 레벨에서 `game_ending` + `game_finished` 두 메시지로 분해해 검증한다.

개별 L1 Command Agent 테스트 시나리오 파일 참조:

- [tc-session-management.md](./tc-session-management.md) — 세션 관리 (FR-001~008)
- [tc-map-system.md](./tc-map-system.md) — 맵 시스템 (FR-031~035)
- [tc-chat.md](./tc-chat.md) — 채팅/커뮤니케이션 (FR-036~040)
- [tc-player-actions.md](./tc-player-actions.md) — 플레이어 행동 (FR-041~049)
- [tc-info-asymmetry.md](./tc-info-asymmetry.md) — 정보 비대칭 (FR-050~054)
- [tc-npc.md](./tc-npc.md) — NPC 시스템 (FR-055~058)
- [tc-ending.md](./tc-ending.md) — 종료/엔딩 (FR-063~068, FR-091)
- [tc-save-feedback.md](./tc-save-feedback.md) — 저장/피드백 (FR-069~072)
- [tc-additional.md](./tc-additional.md) — 추가 기능 (FR-081~089)
- [tc-world-generation.md](./tc-world-generation.md) — 세계 생성 & 구조 검증 (Phase 2B)
- [tc-story-verification.md](./tc-story-verification.md) — 스토리 검증 (Phase 2B)
- [tc-game-structure.md](./tc-game-structure.md) — 게임 구조 (FR-026~028, FR-030)
- [tc-gm.md](./tc-gm.md) — GM 시스템 (FR-059~062, Phase 2E)
- [tc-terminal-ui.md](./tc-terminal-ui.md) — 터미널 UI (FR-073~080, L3 TUI)
- [tc-nonfunctional.md](./tc-nonfunctional.md) — 비기능 요구사항 (NFR)

---

## 9. FR/NFR 전체 커버리지 매트릭스

### 기능 요구사항 (FR)

| FR | 설명 | L1 | L2 | L3 |
|----|------|:--:|:--:|:--:|
| FR-001 | 게임 호스팅 | L1-SM-001 | Phase 2A | FG-* |
| FR-002 | 게임 참가 | L1-SM-002~004 | Phase 2A | FG-* |
| FR-003 | 닉네임 설정 | L1-SM-005~006 | Phase 2A | FG-* |
| FR-004 | 대기실 | L1-SM-007,026,027 | Phase 2A | FG-* |
| FR-005 | 게임 시작 | L1-SM-008~010,016 | Phase 2A | FG-* |
| FR-006 | 플레이어 수 제한 | L1-SM-010,011,028 | Phase 2A | FG-002 |
| FR-007 | 연결 해제 처리 | L1-SM-012~014,020~022 | — | FG-005 |
| FR-008 | API 키 설정 | L1-SM-015,023~025 | Phase 2A | FG-* |
| FR-009 | 세계 자동 생성 | — | Phase 2B | FG-* |
| FR-010 | 맵 구조 생성 | — | Phase 2B | FG-* |
| FR-011 | 역할 배정 | — | Phase 2B | FG-* |
| FR-012 | 단서/기믹 생성 | — | Phase 2B | FG-* |
| FR-013 | 정보 레이어 구성 | — | Phase 2B,2C | FG-006 |
| FR-014 | 게임 구조 생성 | L1-GS-001 | Phase 2B | FG-* |
| FR-015 | GM 필요 여부 | — | Phase 2B | FG-* |
| FR-016 | NPC 생성 | L1-NP-015 | Phase 2B | FG-008 |
| FR-017 | 세계 생성 진행 표시 | — | Phase 2B (L2-WG-024) | FG-* |
| FR-018 | JSON 스키마 준수 | — | Phase 2B | FG-* |
| FR-019 | 종료 조건 스키마 | L1-GS-002,003,004 | Phase 2B | FG-* |
| FR-020 | 구조적 모순 검사 | — | Phase 2B | FG-* |
| FR-021 | 복선 회수 경로 | — | Phase 2B | FG-* |
| FR-022 | 단서 배치 검증 | — | Phase 2B | FG-* |
| FR-023 | NPC 정보 정합성 | — | Phase 2B | FG-* |
| FR-025 | 검증 실패 부분 재생성 | — | Phase 2B | — |
| FR-026 | 게임 구조 자유 설계 | L1-GS-001,009 | Phase 2B | FG-* |
| FR-027 | 종료 조건 자유 설계 | L1-GS-002,011 | Phase 2B | FG-* |
| FR-028 | 종료 판정 엔진 | L1-PA-020, L1-GS-005~008,013 | Phase 2F | FG-009 |
| FR-030 | 세션 시간 유동 조절 | — | Phase 2E | FG-* |
| FR-031 | 맵 조회 | L1-MP-001 | Phase 2D | FG-* |
| FR-032 | 방 이동 | L1-MP-002~005 | Phase 2D | FG-* |
| FR-034 | 방 설명 표시 | L1-MP-005~006 | Phase 2D | FG-* |
| FR-035 | 맵 크기 동적 결정 | L1-MP-007 | Phase 2B | FG-002,003 |
| FR-036 | 같은 방 대화 | L1-CH-001~002,007 | Phase 2D | FG-* |
| FR-037 | 글로벌 채팅 | L1-CH-003 | Phase 2D | FG-* |
| FR-038 | 시스템 메시지 | L1-CH-004 | Phase 2D | FG-* |
| FR-039 | 채팅 로그 스크롤 | L1-CH-013~014 | — | FG-007 |
| FR-040 | 입장/퇴장 알림 | L1-CH-005 | Phase 2D | FG-* |
| FR-041 | 방 조사 | L1-PA-001~004,021 | Phase 2D | FG-* |
| FR-042 | 행동 서술 | L1-PA-005,025,026 | Phase 2D | FG-* |
| FR-043 | NPC 대화 | L1-PA-006~008,023, L1-NP-004~005 | Phase 2D | FG-008 |
| FR-044 | 인벤토리 | L1-PA-009 | Phase 2D | FG-* |
| FR-045 | 역할 재확인 | L1-PA-010 | Phase 2C | FG-* |
| FR-046 | 도움말 | L1-PA-011 | — | FG-* |
| FR-047 | NPC 아이템 전달 | L1-PA-012~013,022 | Phase 2D | FG-008 |
| FR-048 | NPC 아이템 수령 | L1-PA-024 | Phase 2D | FG-008 |
| FR-049 | 투표 | L1-PA-014~015 | Phase 2F | FG-009 |
| FR-050 | 공개 정보 전달 | L1-IA-001 | Phase 2C | FG-006 |
| FR-051 | 반공개 정보 전달 | L1-IA-006 | Phase 2C | FG-006 |
| FR-052 | 비공개 정보 전달 | L1-IA-002,005,007 | Phase 2C | FG-006 |
| FR-053 | 위치 정보 전체 공개 | L1-IA-003 | Phase 2D | FG-* |
| FR-054 | 대화 방 범위 제한 | L1-IA-004 | Phase 2D | FG-006 |
| FR-055 | NPC 퍼소나 유지 | L1-NP-001,006 | Phase 2D | FG-008 |
| FR-056 | NPC 정보 공개 제어 | L1-NP-002 | Phase 2D | FG-008 |
| FR-057 | NPC 기믹 실행 | L1-NP-003 | Phase 2D | FG-008 |
| FR-058 | NPC 위치 | L1-NP-007 | Phase 2D | FG-008 |
| FR-059 | GM 서술 이벤트 | L1-GM-001 | Phase 2E | FG-* |
| FR-060 | 긴장감 조율 | L1-GM-002 | Phase 2E | FG-* |
| FR-061 | 스토리 수렴 유도 | — | Phase 2E | FG-* |
| FR-062 | GM 비활성 모드 | L1-GM-003 | Phase 2E | FG-* |
| FR-063 | 종료 조건 판정 | L1-EN-001,006~012 | Phase 2F | FG-009,010 |
| FR-066 | 시간 초과 종료 | L1-EN-002~003 | Phase 2F | FG-* |
| FR-067 | 개인화 엔딩 | L1-EN-004,009 | Phase 2F | FG-* |
| FR-068 | 전체 비밀 공개 | L1-EN-005 | Phase 2F | FG-* |
| FR-069 | 세션 데이터 저장 | L1-SF-001~002 | — | FG-* |
| FR-070 | 행동 로그 저장 | L1-SF-005~007 | — | — |
| FR-071 | 피드백 수집 | L1-SF-008~011 | — | — |
| FR-072 | 저장 데이터 조회 | L1-SF-003 | — | FG-* |
| FR-073 | 레이아웃 구성 | — | — | FG-007 |
| FR-074 | 메시지 스타일 구분 | — | — | FG-007 |
| FR-075 | 입력 모드 | — | — | FG-007 |
| FR-076 | 자동완성 | — | — | FG-007 |
| FR-077 | 이벤트 렌더러 | — | — | FG-007 |
| FR-078 | 브리핑 화면 | — | Phase 2C | FG-007 |
| FR-079 | 엔딩 화면 | — | Phase 2F | FG-007 |
| FR-080 | 반응형 레이아웃 | — | — | FG-007 |
| FR-081 | 대기실 채팅 | L1-CH-006 | Phase 2A | FG-* |
| FR-082 | 호스트 게임 취소 | L1-SM-017~018 | Phase 2A | — |
| FR-083 | graceful 종료 | L1-AF-001~002,009 | — | FG-* |
| FR-084 | /who 조회 | L1-PA-019 | Phase 2D | FG-* |
| FR-085 | 세계 생성 재시도 | L1-AF-012~013 | Phase 2A | — |
| FR-086 | 룸 코드 고유성 | L1-SM-019 | — | — |
| FR-087 | 이동 이력 | L1-AF-016~017 | Phase 2D | FG-* |
| FR-088 | NPC 스토리 분기 | L1-AF-018~019 | Phase 2D | FG-008 |
| FR-089 | 테마 힌트 | L1-AF-006, L2-AF-020~021 | — | — |
| FR-091 | 플레이어 주도 종료 | L1-PA-016~018, L1-EN-008 | Phase 2F | FG-010 |
| FR-092 | 개인 목표 시스템 | — | Phase 2B | FG-* |
| FR-093 | 게임 구조 검증 | L1-GS-010,012 | Phase 2B | — |

**참고:** FR-024, FR-029, FR-033, FR-064, FR-065는 PRD에 미정의 (번호 갭)

**커버리지 보정 메모:** `(pending)` 표기는 해당 레이어에 시나리오/케이스 ID가 아직 정의되지 않았음을 의미한다.

### 비기능 요구사항 (NFR)

| NFR | 설명 | 검증 방식 |
|-----|------|----------|
| NFR-001 | AI 응답 시간 p95 < 3초 | L1: 모든 AI 응답에 타이밍 assertion |
| NFR-003 | 메시지 전달 < 500ms | L1: chat→수신 시점 측정 |
| NFR-004 | 세션당 최대 8명 | L3: FG-002 (8명 풀 게임) |
| NFR-005 | 서버 시작 < 5초 | L1: 서버 시작 시간 측정 |
| NFR-006 | npm 패키지 < 50MB | CI: 빌드 후 패키지 크기 확인 |
| NFR-007 | 서버 메모리 < 512MB | L1,L2,L3: RSS 샘플링 |
| NFR-008 | 클라이언트 메모리 < 256MB | L3: TUI 프로세스 RSS |
| NFR-009 | 이벤트 타입 확장성 | L1: 커스텀 이벤트 핸들러 추가 테스트 |
| NFR-010 | 게임 구조 확장성 | Phase 2B: 다양한 게임 구조 생성 확인 |
| NFR-011 | AI 프로바이더 교체 | L1: 어댑터 교체 후 동작 확인 |
| NFR-012 | 다국어 지원 가능성 | L1: 텍스트 외부화 구조 확인 |
| NFR-013 | API 키 보호 | L1: 모든 메시지에 키 미포함 |
| NFR-014 | 정보 격리 | L1,Phase 2C,FG-006: 교차 검증 |
| NFR-015 | 입력 검증 | L1: 특수문자/긴 문자열 전송 |
| NFR-016 | 세션 보안 (룸 코드 엔트로피) | L1: 룸 코드 생성 엔트로피 확인 (단어+4자리 ≥ 100만 조합) |
| NFR-017 | API 키 저장 보안 | L1: 설정 파일 권한 600 확인, 키 평문 미노출 |
| NFR-018 | 네트워크 통신 보안 (TLS/WSS) | L1: 호스트-참가자 간 TLS/WSS 암호화 사용 확인 |
| NFR-019 | 즉시 시작 (npx) | CI: `npx story host` 한 줄로 설치/회원가입 없이 실행 확인 |
| NFR-020 | 자기 설명적 UI | FG-007: TUI Agent가 /help 없이 명령어 사용 가능 여부 평가 |
| NFR-021 | 오류 메시지 명확성 | L1: 모든 error에 code+message+해결방법 포함 확인 |
| NFR-022 | 대기 시간 피드백 | L1: AI 처리 중 스피너/진행 메시지 표시 확인 |
| NFR-023 | 색상 의존 금지 | FG-007: 색상 외 텍스트/기호로도 메시지 유형 구분 가능 확인 |
| NFR-024 | Go 호환성 | CI: Go 1.25, 1.26에서 L1 실행 |
| NFR-025 | OS 호환성 | CI: macOS, Linux, Windows(WSL)에서 L1 실행 |
| NFR-026 | 터미널 에뮬레이터 지원 | CI: 다중 터미널에서 L3 실행 |
| NFR-027 | 최소 터미널 크기 | FG-007: 80x24에서 TUI 테스트 |
| NFR-028 | npm 호환성 | CI: npm 7+, npx 실행 테스트 |
| NFR-029 | 유니코드 지원 | L1: UTF-8 다국어 닉네임/메시지 전송 및 수신 확인 |

**QA 추가 검증 항목** (PRD 외 QA 자체 추가)

| QA-NFR | 설명 | 검증 방식 |
|--------|------|----------|
| QA-NFR-101 | Replay Attack 방지 | L1: 메시지 타임스탬프/시퀀스 확인 |
| QA-NFR-102 | 네트워크 장애 내성 | FG-005: 연결 해제/복구 |
| QA-NFR-103 | 로깅 구조화 | L1: 로그 포맷 확인 |
| QA-NFR-104 | 명령어 직관성 | FG-007: TUI Agent UX 평가 |
| QA-NFR-105 | 무상태 아키텍처 | L1: 서버 재시작 복원 테스트 |
| QA-NFR-106 | 에러 복구 | L1: 에러 후 계속 동작 확인 |

### 9.5 User Story ↔ FR ↔ Test ID 매핑

다음 표는 PRD §3 사용자 스토리를 FR 및 `tc-*.md` 테스트 식별자와 연결한 추적 매트릭스다.
- `Test IDs`의 `— (pending)`은 현재 해당 레이어에 시나리오/케이스 ID가 없는 항목이다.
- PRD §3에는 `US-G11`이 정의되어 있지 않아 갭으로 표시했다.

| US ID | FR IDs | Test IDs (L1/L2/L3) |
|------|--------|----------------------|
| US-H01 | FR-001 | L1: `L1-SM-001` / L2: `Phase 2A` / L3: `FG-*` |
| US-H02 | FR-001, FR-086 | L1: `L1-SM-001`, `L1-SM-019` / L2: `Phase 2A` / L3: `—` |
| US-H03 | FR-004, FR-005 | L1: `L1-SM-007`, `L1-SM-008~010,016` / L2: `Phase 2A` / L3: `FG-*` |
| US-H04 | FR-017 | L1: `—` / L2: `L2-WG-024`, `Phase 2B` / L3: `FG-*` |
| US-H05 | FR-009, FR-010, FR-011, FR-035 | L1: `L1-MP-007`(FR-035) / L2: `Phase 2B` / L3: `FG-*`, `FG-002`, `FG-003` |
| US-H06 | FR-008 | L1: `L1-SM-015` / L2: `Phase 2A` / L3: `FG-*` |
| US-J01 | FR-002 | L1: `L1-SM-002~004` / L2: `Phase 2A` / L3: `FG-*` |
| US-J02 | FR-003, FR-004 | L1: `L1-SM-005~007` / L2: `Phase 2A` / L3: `FG-*` |
| US-J03 | FR-050, FR-078 | L1: `L1-IA-001` / L2: `Phase 2C` / L3: `FG-006`, `FG-007` |
| US-J04 | FR-011, FR-052, FR-092 | L1: `L1-IA-002,005,007` / L2: `Phase 2B`, `Phase 2C` / L3: `FG-*`, `FG-006` |
| US-G01 | FR-036, FR-054 | L1: `L1-CH-001~002,007`, `L1-IA-004` / L2: `Phase 2D` / L3: `FG-*`, `FG-006` |
| US-G02 | FR-037 | L1: `L1-CH-003` / L2: `Phase 2D` / L3: `FG-*` |
| US-G03 | FR-032 | L1: `L1-MP-002~004` / L2: `Phase 2D` / L3: `FG-*` |
| US-G04 | FR-041 | L1: `L1-PA-001~004,021` / L2: `Phase 2D` / L3: `FG-*` |
| US-G05 | FR-042 | L1: `L1-PA-005` / L2: `Phase 2D` / L3: `FG-*` |
| US-G06 | FR-043 | L1: `L1-PA-006~008,023`, `L1-NP-004~005` / L2: `Phase 2D` / L3: `FG-008` |
| US-G07 | FR-031, FR-053, FR-084 | L1: `L1-MP-001`, `L1-IA-003`, `L1-PA-019` / L2: `Phase 2D` / L3: `FG-*` |
| US-G08 | FR-044 | L1: `L1-PA-009` / L2: `Phase 2D` / L3: `FG-*` |
| US-G09 | FR-045, FR-092 | L1: `L1-PA-010` / L2: `Phase 2B`, `Phase 2C` / L3: `FG-*` |
| US-G10 | FR-059, FR-060, FR-061, FR-062 | L1: `—` / L2: `Phase 2E` / L3: `FG-*`, `FG-001` |
| US-G11 | — (PRD §3 미정의) | L1/L2/L3: `—` |
| US-G12 | FR-047, FR-048 | L1: `L1-PA-012~013,022,024` / L2: `Phase 2D` / L3: `FG-008` |
| US-G13 | FR-046 | L1: `L1-PA-011` / L2: `—` / L3: `FG-*` |
| US-E01 | FR-028, FR-063 | L1: `L1-PA-020`, `L1-EN-001,007` / L2: `Phase 2F` / L3: `FG-009`, `FG-010` |
| US-E02 | FR-067 | L1: `L1-EN-004,009` / L2: `Phase 2F` / L3: `FG-*` |
| US-E03 | FR-068 | L1: `L1-EN-005,006` / L2: `Phase 2F` / L3: `FG-*` |
| US-E04 | FR-071 | L1: `L1-SF-008~011` / L2: `—` / L3: `—` |
| US-E05 | FR-069, FR-072 | L1: `L1-SF-001~003` / L2: `—` / L3: `FG-*` |
| US-E06 | FR-066 | L1: `L1-EN-002~003` / L2: `Phase 2F` / L3: `FG-*` |
| US-E07 | FR-049, FR-091 | L1: `L1-PA-014~018`, `L1-EN-008` / L2: `Phase 2F` / L3: `FG-009`, `FG-010` |

---

## 10. 테스트 인프라

| 컴포넌트 | 역할 | 구현 |
|---------|------|------|
| **AgentOrchestrator** | 호스트 서버 + N개 에이전트 스폰, 라이프사이클 관리 | Go, go test 기반 |
| **ProtocolClient** | WebSocket 경량 클라이언트 (TUI 없이 프로토콜만) | ws 패키지 사용 |
| **TUIHarness** | CLI 스폰, stdin/stdout 파이프, ANSI strip | child_process + strip-ansi |
| **TestServer** | L1/L2용 fixture AI 응답 서버 | AI Provider Adapter mock |
| **LLMDecisionEngine** | 에이전트 의사결정 (커스텀 AI Provider interface) | Claude/GPT API 호출 |
| **AssertionCollector** | 모든 에이전트 assertion 수집, 리포트 생성 | Custom reporter |
| **TranscriptLogger** | 전체 프로토콜 메시지 타임라인 기록 | JSON 로그 파일 |
| **ReportGenerator** | AgentQAReport → AggregatedTestReport 변환 | Markdown/JSON 출력 |

---

## 11. 실행 전략

| 환경 | 실행 대상 | 예상 시간 | 빈도 |
|------|---------|----------|------|
| **CI (PR)** | L1 전체 (~70 시나리오) | ~2분 | 매 PR |
| **CI (Nightly)** | L1 + L2 전체 (6 Phase) | ~10분 | 매일 |
| **Release** | L1 + L2 + L3 (FG-001~003) | ~30분 | 릴리스 전 |
| **Weekly QA** | L1 + L2 + L3 전체 (FG-001~010) | ~2시간 | 주 1회 |
| **Manual** | L4 Subagent Playtest (L4-001~004) | ~15~40분/판 | 수동 트리거 |
| **Manual** | AI Quality Eval Suite | ~10~20분 | 수동 트리거 |

### 비용 추정 (Weekly QA 기준)

| 항목 | 예상 LLM 호출 | 비용 (Claude Sonnet 기준) |
|------|-------------|------------------------|
| L1 (fixture AI) | 0 | $0 |
| L2 에이전트 의사결정 | ~200 calls | ~$2 |
| L2 게임 AI (실제 AI) | ~50 calls | ~$5 |
| L3 에이전트 의사결정 | ~500 calls | ~$5 |
| L3 게임 AI (실제 AI) | ~200 calls | ~$20 |
| L3 QA 리포트 생성 | ~40 calls | ~$2 |
| **합계 (Weekly)** | ~990 calls | **~$34/주** |

### 비용 추정 (L4 Subagent Playtest 기준, 수동 트리거)

| 항목 | 예상 LLM 호출 | 비용 (Claude Sonnet 기준) |
|------|-------------|------------------------|
| L4 subagent 플레이 (3명) | ~90~150 calls | ~$5~15 |
| L4 게임 AI (실제 AI) | ~50~100 calls | ~$5~10 |
| **합계 (1판)** | ~140~250 calls | **~$10~25/판** |

### 비용 추정 (AI Quality Eval Suite, 수동 트리거)

| 항목 | 예상 LLM 호출 | 비용 (Claude Sonnet 기준) |
|------|-------------|------------------------|
| WorldGen Eval (4회 생성 + 평가) | ~12 calls | ~$5~10 |
| NPC Eval (고정 시나리오) | ~25 calls | ~$3~5 |
| GM Eval (트랜스크립트 분석) | ~5 calls | ~$1~2 |
| ActionEval Eval (고정 시나리오) | ~15 calls | ~$2~3 |
| Ending Eval (트랜스크립트 분석) | ~5 calls | ~$1~2 |
| Orchestration Eval (12회 생성 + 평가) | ~36 calls | ~$3~8 |
| **합계 (전체 Suite)** | ~98 calls | **~$15~30/회** |

---

## 12. L4: Subagent Playtest

### 12.1 개요

- **인터랙션**: TUI 직접 조작 (tmux)
- **실행 주체**: Claude Code task() subagent
- **AI 사용**: subagent 자체 판단 + 실제 게임 AI
- **목적**: 탐색적 버그 발견 + **tc 케이스 TUI 교차 검증** + UX 문제 발견 + AI 품질 정성 평가
- **검증 방식**: 자유 플레이 + **체크리스트 기반 검증** (하이브리드)
- **실행 빈도**: 수동 트리거

> **체크리스트 방식**: 각 `tc-*.md` 문서에 L4 체크리스트가 정의되어 있다. subagent는 자유 플레이를 유지하되, 플레이 중 체크리스트 항목에 해당하는 상황을 만나면 반드시 검증하고 리포트에 기록한다. 체크리스트 항목을 의도적으로 커버하는 행동도 포함한다.

### 12.2 L3 vs L4

| 항목 | L3 (기존) | L4 (신규) |
|------|-----------|-----------|
| 실행 주체 | Go 코드 | Claude task() subagent |
| 상호작용 | Protocol JSON + TUI 파싱 | TUI 직접 조작 (tmux) |
| 의사결정 | 코딩된 LLM 호출 | subagent 자체 판단 |
| assertion | 코드로 작성된 속성 검증 | **체크리스트 기반 검증** + 자유 서술 |
| 목적 | 불변 규칙 위반 탐지 | tc 케이스 TUI 교차검증 + UX + 스토리 + 탐색적 버그 |

### 12.3 오케스트레이션

```
Sisyphus (orchestrator)
  │
  │ 1. tmux에서 npx story host 실행
  │    → 화면에서 룸 코드 파싱 (예: WOLF-7423)
  │
  │ 2. task() × N — 각 subagent에게 전달:
  │    - 룸 코드
  │    - 페르소나 (Explorer / Diplomat / Chaotic / Observer)
  │    - **L4 체크리스트** (tc-*.md에서 추출한 TUI 검증 항목)
  │    - 플레이 가이드라인 + 에러 핸들링 지침
  │    - 리포트 작성 지침
  │
  │ 3. 각 subagent:
  │    a. tmux 세션 생성
  │    b. npx story join WOLF-7423 실행
  │    c. 닉네임 입력
  │    d. 게임 플레이 루프:
  │       - tmux capture-pane으로 화면 읽기
  │       - 상황 판단 후 명령 입력
  │       - 반복 (게임 종료까지)
  │    e. 게임 종료 후 PlayReport 파일 작성
  │
  │ 4. Sisyphus가 모든 리포트 수집 → 종합 분석
```

### 12.4 에러 분류

| 유형 | 설명 | subagent 행동 |
|------|------|--------------|
| **Hard Error** | 크래시, 프로세스 죽음, 화면 무응답, 게임 진행 불가 | **즉시 중단** + 에러 리포트 작성 후 반환 |
| **Soft Issue** | UX 불편, 스토리 부자연스러움, 경미한 이상 동작 | **기록하고 계속 플레이** → 종료 후 리포트에 포함 |

### 12.5 Fix-Replay 루프

1. subagent가 Hard Error를 만남 → 에러 리포트 파일 작성 → task() 종료
2. Sisyphus가 리포트를 읽음 → 에러 원인 분석 → 코드 수정
3. 새 게임 세션으로 재시작 — 같은 시나리오, 새 룸코드
4. 반복 — 에러 없이 완주할 때까지 (최대 N회 제한)

### 12.6 리포트 구조

```go
type PlayExperienceReport struct {
    Completed   bool   // false면 에러로 중단된 것
    AbortReason string `json:",omitempty"`

    Errors []GameError

    // === tc 케이스 체크리스트 검증 ===
    Checklist struct {
        Items        []ChecklistItem
        CoveredCount int     // 검증 완료 항목 수
        TotalCount   int     // 전체 체크리스트 항목 수
        CoverageRate float64 // CoveredCount / TotalCount
    }

    Story struct {
        Coherence   int    // 1-5
        Engagement  int    // 1-5
        Pacing      int    // 1-5
        Creativity  int    // 1-5
        FreeComment string
    }

    NPC struct {
        PersonaConsistency int      // 1-5
        InformationControl int      // 1-5
        Naturalness        int      // 1-5
        GimmickWorked      bool
        Interactions       []string
    }

    GM *struct {
        TimingAppropriate  int // 1-5
        TensionManagement  int // 1-5
        ConvergenceNatural int // 1-5
        NarrativeQuality   int // 1-5
    }

    PersonalGoal struct {
        Clarity      int  // 1-5
        Achievable   bool
        Achieved     bool
        ConflictFelt bool
    }

    Ending struct {
        Satisfaction           int  // 1-5
        ReflectedActions       bool
        SecretRevealSurprising bool
    }

    UX struct {
        Readability      int      // 1-5
        CommandDiscovery int      // 1-5
        Responsiveness   int      // 1-5
        ConfusingMoments []string
    }
}

type ChecklistItem struct {
    SourceID    string  // 원본 tc 케이스 ID (예: L1-PA-001)
    Category    string  // tc 파일 카테고리 (예: player-actions)
    Description string  // 체크 항목 설명
    Verified    bool    // 검증 완료 여부
    Passed      *bool   // 통과 여부 (미검증 시 nil)
    Observation string  // subagent가 관찰한 실제 결과
    Screenshot  string  `json:",omitempty"` // tmux capture-pane 스냅샷 (이슈 발견 시)
}

type GameError struct {
    ID       string // E-001, E-002, ...
    Type     string // "crash" | "unresponsive" | "render_broken" | "logic_error" | "protocol_error" | "ux_issue" | "story_issue"
    Severity string // "critical" | "major" | "minor"
    Blocking bool   // true → 게임 중단 사유

    Context struct {
        Phase         string // lobby, briefing, playing, ending
        Room          string
        LastAction    string
        ScreenContent string // tmux capture-pane
    }

    Description  string
    Expected     string
    Actual       string
    Reproduction string
}
```

### 12.7 체크리스트 운용 방식

#### 12.7.1 체크리스트 소스

각 `tc-*.md` 문서의 `## L4 체크리스트` 섹션에 TUI로 검증 가능한 항목이 정의되어 있다.

| tc 파일 | 체크리스트 항목 수 (예상) | 주요 커버리지 |
|---------|----------------------|-------------|
| tc-session-management | ~10 | 룸 코드, 닉네임, 로비, 게임 시작 |
| tc-map-system | ~6 | 맵 조회, 이동, 방 정보 |
| tc-chat | ~7 | 방 채팅, 글로벌 채팅, 시스템 메시지 |
| tc-player-actions | ~13 | 조사, 행동, NPC, 투표, 인벤토리 |
| tc-info-asymmetry | ~6 | 브리핑, 정보 격리, 위치 공개 |
| tc-npc | ~8 | 퍼소나, 기믹, 대화 이력, 위치 |
| tc-gm | ~4 | GM 서술, 긴장감, 수렴 |
| tc-ending | ~7 | 종료 판정, 개인 엔딩, 비밀 공개 |
| tc-terminal-ui | ~17 | 레이아웃, 자동완성, 브리핑/엔딩 화면 |
| tc-nonfunctional | ~7 | UI 직관성, 오류 메시지, 대기 피드백 |
| tc-additional | ~5 | 대기실 채팅, 테마, 방문자 기록 |
| tc-save-feedback | ~4 | 피드백 UI, 히스토리 |
| tc-game-structure | ~3 | 투표/합의 종료 플로우 |
| tc-world-generation | ~2 | 생성 진행 표시 |
| tc-story-verification | ~3 | 모순/복선/NPC 정합성 체감 |

#### 12.7.2 subagent별 체크리스트 배분

모든 subagent가 전체 체크리스트를 수행하는 것이 아니라, **페르소나에 맞는 항목을 우선 커버**한다.

| 페르소나 | 우선 체크리스트 | 이유 |
|---------|---------------|------|
| **Explorer** | map-system, player-actions (examine/do), additional (방문자 기록) | 맵 탐색 위주 플레이 |
| **Diplomat** | npc, chat, info-asymmetry | 대화/소통 위주 플레이 |
| **Strategist** | ending, game-structure, player-actions (vote/end) | 목표 지향 플레이 |
| **Chaotic** | session-management (오류 케이스), nonfunctional (오류 복구) | 엣지 케이스 유발 |
| **Observer** | info-asymmetry, terminal-ui, chat | 관찰/수신 위주 플레이 |

모든 subagent는 공통으로 **terminal-ui**, **session-management** (기본 플로우), **save-feedback** (종료 후) 체크리스트를 수행한다.

#### 12.7.3 검증 프로토콜

1. **플레이 전**: subagent가 자신에게 배정된 체크리스트를 확인
2. **플레이 중**: 자유 플레이 유지 + 체크리스트 항목에 해당하는 상황 발생 시 검증
3. **의도적 커버**: 자연스러운 플레이 흐름 내에서 미검증 항목을 위한 행동 추가
4. **즉시 기록**: 각 체크리스트 항목의 Pass/Fail + 관찰 결과를 즉시 기록
5. **이슈 발견 시**: tmux capture-pane 스냅샷 첨부

#### 12.7.4 Pass/Fail 기준 (체크리스트)

| 조건 | 판정 |
|------|------|
| 체크리스트 커버율 < 70% | **WARNING** (탐색적 플레이 특성상 100% 불필요) |
| 체크리스트 Fail 항목 중 critical 이슈 1건 이상 | **FAIL** |
| 체크리스트 Fail 항목 3건 초과 (minor 포함) | **FAIL** |
| 위 조건 모두 미해당 | **PASS** |

### 12.8 시나리오

| ID | 시나리오 | subagent | 페르소나 | 특수 목적 |
|----|---------|----------|---------|----------|
| L4-001 | 기본 플레이테스트 | 3 | Explorer, Diplomat, Strategist | 전체 플로우 체험 + 체크리스트 교차 커버 |
| L4-002 | 스트레스 플레이 | 4 | Chaotic×2, Observer×2 | 엣지 케이스 + 오류 복구 체크리스트 |
| L4-003 | NPC 집중 | 3 | Diplomat×3 | NPC 체크리스트 심층 커버 |
| L4-004 | 최소 인원 | 2 | Explorer, Diplomat | 2인 게임 + 기본 체크리스트 커버 |

---

## 13. AI Quality Evaluation Suite

L4와 별개로, 각 AI 모듈의 출력을 독립적으로 평가하는 프레임워크.

### 13.1 WorldGen Eval — 세계 생성 품질

```
입력: 플레이어 수 (2, 4, 6, 8)
실행: 각 플레이어 수로 세계 생성 3~5회
평가 항목:
  - 장르/설정 다양성 (반복되지 않는가)
  - 역할 간 긴장 관계 (갈등 구조가 흥미로운가)
  - 맵 구조 적절성 (방 수, 연결성, 공개/밀실 비율)
  - 개인 목표 달성 가능성 (경로가 실제 존재하는가)
  - 종료 조건 명확성 (판정 가능한가)
  - 정보 레이어 균형 (공개:반공개:비공개 비율)
```

### 13.2 NPC Eval — NPC 대화 품질

```
입력: 고정 세계 + NPC 설정
실행:
  - 일반 질문 10개 → 퍼소나 유지 확인
  - 비밀 유도 질문 5개 → 정보 통제 확인
  - 반복 질문 3개 → 맥락 기억 확인
  - 기믹 트리거 시도 → 조건부 동작 확인
평가: LLM evaluator가 대화 로그를 분석
```

### 13.3 GM Eval — GM 서술 품질

```
입력: 게임 트랜스크립트 (L4 또는 L3에서 수집)
평가 항목:
  - 개입 빈도 (너무 잦은가, 너무 뜸한가)
  - 개입 타이밍 (정체된 순간에 개입했는가)
  - 수렴 유도 (후반부에 결말로 이끌었는가)
  - 서술 품질 (분위기, 긴장감, 문체)
```

### 13.4 ActionEval Eval — 행동 평가 품질

```
입력: 고정 세계 + 다양한 행동 시나리오
실행:
  - 존재하는 물건 조사 → 관련 설명 생성 확인
  - 없는 물건 조사 → 적절한 불발 응답
  - 창의적 행동 (/do) → 판정 공정성과 창의성
  - 단서가 있는 물건 조사 → clue_found 이벤트 발생
```

### 13.5 Ending Eval — 엔딩 품질

```
입력: 게임 트랜스크립트 + 생성된 엔딩
평가 항목:
  - 플레이어 행동 반영도 (실제 한 일이 언급되는가)
  - 개인 목표 평가 정확성 (달성/미달성 판정이 맞는가)
  - 서사 완결성 (열린 결말이 아닌 완결감)
  - 비밀 공개의 극적 효과
```

### 13.6 Orchestration Eval — 오케스트레이션 품질

```
입력: 플레이어 수 (4, 6), qualityMode (fast, premium)
실행: 각 조합으로 세계 생성 3회 (총 12회)
평가 항목:
  - seed 다양성 (3개 seed의 장르/설정 분포 — 동일 장르 비율 < 50%)
  - 통합 일관성 (Showrunner 산출물의 톤 통일도, LLM evaluator 1-5)
  - 모드별 품질 차이 (fast vs premium engagement/coherence 정량 비교)
  - fallback 안정성 (의도적 프로바이더 2개 차단 후 생성 성공률 100%)
  - 생성 시간 (fast < 60초, premium < 180초)
비용: ~$3~8/회
```

### 13.7 실행 방식

| 평가 | 실행 방법 | 빈도 | 비용 |
|------|----------|------|------|
| PlayExperience | L4 subagent 플레이 후 자동 | L4 실행 시 | L4에 포함 |
| WorldGen Eval | 독립 스크립트 (세계 생성 N회 + LLM 평가) | 수동 | ~$5~10/회 |
| NPC Eval | 독립 스크립트 (고정 시나리오 + LLM 평가) | 수동 | ~$3~5/회 |
| GM Eval | L4 트랜스크립트 사후 분석 | L4 후 | ~$1~2/회 |
| ActionEval Eval | 독립 스크립트 (고정 시나리오) | 수동 | ~$2~3/회 |
| Ending Eval | L4 트랜스크립트 사후 분석 | L4 후 | ~$1~2/회 |
| Orchestration Eval | 독립 스크립트 (멀티 모델 파이프라인 12회 생성 + LLM 평가) | 수동 | ~$3~8/회 |

---

## 14. Pass/Fail 기준

| 조건 | 판정 |
|------|------|
| Critical 이슈 1건 이상 | **FAIL** |
| Major 이슈 3건 초과 | **FAIL** |
| 정보 누출 (informationLeaks) 1건 이상 | **FAIL** |
| 스토리 품질 평균 < 3.0/5.0 | **FAIL** (L3, L4) |
| TUI 렌더링 결함 3건 초과 | **FAIL** (FG-007, L4) |
| L1 Assertion 실패 1건 이상 | **FAIL** |
| L4 Hard Error (blocking=true) 1건 이상 | **FAIL** (L4) |
| L4 AI 품질 평균 < 3.0/5.0 | **FAIL** (L4) |
| 위 조건 모두 미해당 | **PASS** |
