# L2: GM & Story Progression (GM/스토리 진행)

**Traces**: FR-059, FR-060, FR-061, FR-062, FR-030
**Layer**: L1 Command Agent + L2 Phase Agent (Phase 2E)
**인터랙션**: Protocol level
**AI 사용**: L1 없음 (fixture AI) / L2 LLM 의사결정 (에이전트) + 실제 AI (GM)

---

## L1 시나리오

| ID | 시나리오 | 사전조건 | 에이전트 행동 | 검증 (필드 레벨) | 관련 FR AC |
|----|---------|---------|-------------|------------------|------------|
| L1-GM-001 | narration 이벤트 구조 | `hasGM=true` 세션 시작, 이벤트 캡처 훅 활성화 | idle 구간(예: 90초) 유도 후 GM 서술 트리거 행동 1회 실행 | 수신 `game_event.event.type === 'narration'`; `event.data.text` 비어있지 않은 문자열; `event.data.mood` 비어있지 않은 문자열; `event.visibility.scope==='all'`; `scope==='all'`이면 `visibility.playerIds`가 빈 배열 또는 미포함 | FR-059 AC1, AC2 |
| L1-GM-002 | story_event 구조 | `hasGM=true`, 긴장감 개입이 가능한 fixture(단서/NPC/사건 슬롯 포함) | 의미 없는 행동 반복으로 stagnation 유도 후 GM 개입 대기 | 수신 `game_event.event.type === 'story_event'`; `event.data.title`/`event.data.description`이 비어있지 않은 문자열; `event.data.consequences`가 길이 1 이상 배열; 개입 내용이 data.description 또는 data.consequences에서 다음 중 하나와 관련: 새 단서, 돌발 사건, NPC 행동 변화 | FR-060 AC2, AC3 |
| L1-GM-003 | GM 비활성 확인 | `hasGM=false` 세션, 동일 맵/단서 fixture, 이벤트 로그 초기화 | 플레이어 3명이 `/examine`, `/do`, 이동, 대기 등 N=20 행동 실행 (GM 개입을 유도하는 stagnation 포함) | 전체 로그에서 `type==='narration'` 건수 0; `story_event`는 플레이어 명령 직후(예: 5초 이내)만 발생하고 idle 타이머 단독으로는 발생하지 않음 | FR-062 AC1, AC2 |
| L1-GM-004 | 스토리 수렴 이벤트 발생 조건 검증 *(Priority: P1)* | `hasGM=true` 세션, fixture에서 `gmEngine.pacingState.elapsedPercent >= 70%` 상태 강제 주입 가능, 이벤트 캡처 훅 활성화 | GM 엔진 체크를 1회 트리거 | **긍정 케이스**: `elapsedPercent >= 70%` 상태에서 `game_event.event.type === 'story_event'` 최소 1건 발생; 해당 `story_event.data.description` 또는 `story_event.data.consequences[]`에 수렴/클라이맥스 관련 내용 포함(예: `결정`, `진실`, `수렴`, `결말` 키워드 중 1개 이상); **부정 케이스**: 동일 fixture에서 `elapsedPercent < 50%` 상태로 재설정 후 GM 체크 1회 트리거 시 수렴 유형 `story_event` 0건 발생 | FR-061 AC1, AC2 |

---

## L2 Phase 시나리오 (Phase 2E)

| ID | 시나리오 | 사전조건 | 에이전트 행동 | 검증 (필드 레벨) | 관련 FR AC |
|----|---------|---------|-------------|------------------|------------|
| L2-2E-001 | GM 활성 게임 *(Priority: P1 — MVP 이후)* | 3 에이전트, `hasGM=true`, 세션 타이머/이벤트 로그 수집 활성화 | 10턴 동안 의미 없는 행동 반복(진전 없음) 후 후반부(경과시간 70%+)까지 진행 | idle 구간에서 `narration` 발생; 모든 narration에 `type:'narration'` 및 유효 `target/targetIds` 존재; 개입 이벤트가 `story_event`로 전달되고 `title`, `description`, `consequences[]` 포함; 후반부 수렴 이벤트에 `decisive clue`가 명시됨; 저활동 구간에서 GM 개입 간격이 고활동 구간보다 짧거나 개입 횟수가 증가(진행 속도 적응); **FR-059 AC3 품질 체크**: GM 서술이 현재 장면/분위기와 관련성 있음 — 평가자 3인 중 2인 이상 '관련성 있음' 판정 | FR-059 AC1, AC2, AC3, FR-060 AC1, AC2, AC3, FR-061 AC1, AC2, FR-030 AC2 |
| L2-2E-002 | GM 비활성 게임 | 동일 난이도 fixture로 `hasGM=false` 세션 시작 | GM이 개입할 만한 상황(장시간 무진전, 반복 `/do`)을 포함해 10턴 플레이 | `narration` 이벤트 0건; 스토리 이벤트는 플레이어 행동 기반 트리거만 발생(autonomous GM 개입 없음); 기본 시간 제한 종료 메커닉(`time_warning`, `game_ending`) 유지 | FR-062 AC1, AC2, AC3 |
| L2-2E-001a | GM 활성 게임 클라이맥스 자연스러움 | 3 에이전트, `hasGM=true`, 후반부 수렴 이벤트 로그 수집 | 게임 진행 후 후반부 수렴 이벤트 발생까지 진행 | **FR-061 AC3 품질 체크**: 수렴 이벤트가 기존 플롯라인과 연결됨 — 평가자 3인 중 2인 이상 '자연스러움' 판정 | FR-061 AC3 |
| L2-2E-003 | 세션 시간 유동 조절 범위 *(Priority: P1 — MVP 이후)* | 플레이어 수 3/5/8 각각 별도 세션 시작, 세션 설정 payload 수집 | 각 세션 시작 후 `gameConfig.durationMinutes`(또는 동등 필드) 기록, 활동량 높은/낮은 라운드 각각 1회 실행 | 기본값 20분이 기본 프로파일로 설정됨; AI 조정값이 항상 10~30분 범위(포함); 플레이어 수/활동량 변화에 따라 pacing 메시지 또는 개입 빈도 변화가 관측되어 진행 속도 모니터링이 실제 반영됨 | FR-030 AC1, AC2, AC3 |

---

## L3 참조

- FG-*: 모든 풀 게임에서 GM 활성/비활성에 따른 `narration`/`story_event` 프로토콜 정확성 재검증
- FG-001: FR-059 AC3 품질 체크(분위기 일관성, 긴장감 유지) 및 FR-061 AC3 품질 체크(급종결 없이 클라이맥스 유도)

---

## L4 체크리스트

> L4 subagent가 TUI 플레이 중 아래 항목에 해당하는 상황을 만나면 **반드시 검증하고** PlayExperienceReport의 `checklist` 필드에 기록한다.
> 자유 플레이를 유지하되, 체크리스트 항목을 의도적으로 커버하는 행동을 포함한다.

| 원본 ID | 체크 항목 | TUI 검증 방법 | Pass 기준 |
|---------|---------|--------------|----------|
| L1-GM-001 | GM 서술 이벤트 표시 | 게임 중 GM 서술 발생 시 관찰 | GM 서술 텍스트가 화면에 표시되고 다른 메시지와 시각적으로 구분됨 |
| L2-2E-001 | GM 개입 타이밍 | 비활동 구간에서 관찰 | 비활동 구간에서 GM이 서술/이벤트로 개입하여 진행을 유도함 |
| L2-2E-001 | 스토리 이벤트 표시 | GM이 스토리 이벤트 발생 시 | 스토리 이벤트 제목과 설명이 강조 스타일로 표시됨 |
| L1-GM-003 | GM 비활성 시 개입 없음 | GM 없는 게임에서 관찰 | GM 서술 메시지가 나타나지 않음 |

### 정성 평가 (기존 L4 참조)

- L4-001~004: subagent가 PlayExperienceReport의 `gm` 항목으로 정성 평가
  - `timingAppropriate`: 개입 타이밍 적절성
  - `tensionManagement`: 분위기/긴장감 유지 여부 (FR-059 AC3)
  - `convergenceNatural`: 결말 수렴의 자연스러움/클라이맥스 형성 (FR-061 AC3)
  - `narrativeQuality`: 서술 품질 및 개연성

---

## AI Quality Eval 참조

- **GM Eval**: L3/L4 트랜스크립트에서 GM 개입을 추출해 독립 평가 (개입 빈도, 타이밍, 분위기 유지, 긴장감 조율, 수렴 자연스러움)
