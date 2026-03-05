# L1: Ending System (종료/엔딩)

**Traces**: FR-063, FR-066, FR-067, FR-068, FR-091 (※ FR-064, FR-065는 PRD에 미정의)
**Layer**: L1 Command Agent + L2 Phase Agent
**인터랙션**: Protocol level
**AI 사용**: L1 없음 (fixture AI) / L2 LLM 의사결정
> 프로토콜 메시지 타입과 필드명은 /docs/plans/system-design/shared/protocol.md 및 /docs/plans/system-design/shared/events.md 기준

---

## L1 시나리오

| ID | 시나리오 | 사전조건 | 에이전트 행동 | 검증 (필드 레벨) | 관련 FR AC |
|----|---------|---------|-------------|------------------|------------|
| L1-EN-001 | 투표 종료 판정 | 종료 조건 유형이 `vote`인 세션, 플레이어 4명 | 투표 진행 후 3명 이상 찬성으로 종료 조건 충족 | `vote_started`→`vote_progress`→`vote_ended` 순서 보장; `vote_ended.results` 합계가 총 투표자 수와 일치; `vote_ended.outcome` 문자열 존재; `game_event.event.type==='game_end'`, `event.data.reason`에 `vote` 문구 포함 | FR-063 AC1, AC2 |
| L1-EN-002 | 시간 초과 경고 | 강제 종료 타이머 활성 세션, 타이머 가속 가능 | 남은 시간 5분 전까지 진행 | `game_event` 수신, `event.type==='time_warning'`; `event.data.remainingMinutes===5` | FR-066 AC1 |
| L1-EN-003 | 시간 초과 종료 | 미해결 과제가 남아있는 상태, 타이머 만료 직전 | 타이머 만료까지 대기 | `game_event.event.type==='game_end'`, `event.data.reason`에 `timeout` 문구 포함; `game_ending.commonResult` 비어있지 않음; 미해결 요소가 commonResult에 최소 1건 언급됨 | FR-066 AC2, AC3 |
| L1-EN-004 | 개인화 엔딩 | 종료 직전 플레이어별 행동 로그가 최소 3건 이상 존재, 각 플레이어에 개인 목표가 1개 이상 배정된 세션 | 일반 종료 발생 | 각 에이전트가 수신한 `game_ending.personalEnding`이 PlayerEnding 구조: `playerId` 존재, `summary` 비어있지 않음, `narrative` 비어있지 않음; 에이전트 간 `narrative`가 완전히 동일하지 않음; `personalEnding.goalResults[]` 존재하고 길이 >= 1; 각 `goalResults[]` 원소에 `goalId`(문자열), `achieved`(boolean), `description`(문자열), `evaluation`(문자열, AI 판정 근거) 필드 포함; `goalResults[].goalId`가 해당 플레이어에게 배정된 목표 ID와 일치 | FR-067 AC1, AC2 |
| L1-EN-005 | 비밀 공개 | 플레이어 역할/목표/비밀, NPC 숨김정보, 미발견 단서가 fixture에 존재 | 종료 후 `secretReveal` payload 수집 | `secretReveal.playerSecrets[]`가 모든 플레이어를 포함하고 각 원소에 `playerId`,`characterName`,`secret` 존재; `secretReveal.npcSecrets[]` 존재; `secretReveal.undiscoveredClues[]` 존재 | FR-068 AC1, AC3, AC4 |
| L1-EN-006 | 게임 완료 이벤트 | 정상 종료 가능한 세션 | 종료 조건 달성 | `game_ending` 이후 `game_finished`가 1회 수신; `game_finished.type==='game_finished'` (status 필드 없음 — 수신 자체가 완료 의미) | FR-063 AC5 |
| L1-EN-007 | 이벤트 기반 종료 | 종료 조건 유형이 `event`인 세션 | 지정 이벤트(예: 핵심 아이템 조합) 달성 | `game_event.event.type==='game_end'`, `event.data.reason`에 `event` 문구 포함; `game_ending` 수신; `game_ending.commonResult` 비어있지 않음 | FR-063 AC1 |
| L1-EN-008 | 투표 중 /end 발의 | 이미 `vote_state='open'`인 종료 투표 존재 | 다른 플레이어가 `/end` 발의 | 활성 투표 중 추가 종료투표가 생성되지 않음; 발의자에게 `error.code==='END_VOTE_ALREADY_OPEN'` 단일 응답 반환; 기존 투표는 계속 진행 | FR-091 AC2, AC4 |
| L1-EN-009 | 연결 해제 중 게임 종료 | 4인 세션 중 1명 `disconnected` 상태 | 나머지 3인이 종료 조건 충족 | 연결된 플레이어 전원에게 `game_ending`/`game_finished` 전달; `disconnected` 플레이어는 전달 실패로 기록되되 세션 종료는 성공 | FR-063 AC5, FR-067 AC2 |
| L1-EN-010 | 합의 기반 종료 | 종료 조건 유형이 `consensus`인 세션 | 전원이 `/solve` 응답 제출 | `solve_result` 수신; `game_event.event.type==='game_end'`, `event.data.reason`에 `consensus` 문구 포함; 종료 판정이 확정 | FR-063 AC3 |
| L1-EN-011 | AI 판단 종료 | 종료 조건 유형이 `ai_judgment`인 세션 | AI 종료 판정 트리거 시점까지 플레이 진행 | `game_event.event.type==='game_end'`, `event.data.reason`에 `ai_judgment` 문구 포함; `game_ending` 수신; `game_ending.commonResult` 비어있지 않음 | FR-063 AC4 |
| L1-EN-012 | 종료 조건 미충족 | 종료 조건 임계치 직전 상태(예: 찬성 2/4) | 부분 조건만 충족 후 판정 요청 | 부분 조건만 충족 상태에서 10초 대기; `game_event.event.type==='game_end'` 미발생; `game_ending` 미수신; `sessionState==='playing'` 유지 | FR-063 AC5 |
| L1-EN-013 | 행동 요약-로그 매칭 | 플레이어별 action log에 고유 행동 3개 이상 존재 | 종료 후 각 `personalEnding.narrative` 추출, action log 키워드와 비교 | 플레이어별 상위 핵심행동 3개 중 2개 이상이 개인 엔딩 텍스트에 매칭; 타 플레이어 행동이 잘못 귀속되지 않음 | FR-067 AC1, AC4 |
| L1-EN-014 | 공통+개인 결과 동시 제공 | 종료 payload 스키마 검증 훅 활성화 | 일반 종료 1회 수행 | `game_ending.commonResult` 존재하고 비어있지 않음; `game_ending.personalEnding` PlayerEnding 구조 존재하고 `summary`,`narrative` 비어있지 않음 | FR-067 AC3 |
| L1-EN-015 | 반공개 공유 관계 공개 | 반공개 정보 공유 이벤트가 최소 1건 발생한 세션 | 종료 후 `secretReveal` 검증 | `secretReveal.semiPublicReveal[]` 존재; 각 원소가 `info`(문자열),`sharedBetween`(문자열 배열) 필드 포함 | FR-068 AC2 |
| L1-EN-016 | /end 종료의 엔딩 플로우 동일성 | 동일 fixture로 일반 종료 1회, `/end` 종료 1회 실행 가능 | `/end` 투표 과반 통과 후 엔딩 payload 비교 | `/end` 경로 `game_ending`이 일반 종료와 동일한 상위 스키마(`commonResult`,`personalEnding`,`secretReveal`)를 모두 포함 | FR-091 AC5 |
| L1-EN-017 | /end 60초 제한 + 미응답 기권 | 4인 세션, `/end` 발의 시각 기록 가능 | A가 `/end` 발의, B만 찬성, C/D 무응답으로 60초 경과 | 종료투표가 60초 시점에 자동 마감(`end_vote_result` 수신); 무응답 인원은 거부로 집계 `disagreed>=2`; 찬성률 50%는 과반 아님으로 `passed===false` | FR-091 AC6, AC4 |
| L1-EN-018 | 비호스트 /end 발의 가능 | 4인 세션, `playing` 상태, Agent A는 비호스트 | Agent A(비호스트)가 `{type:'propose_end'}` 전송 | `end_proposed` 전체 브로드캐스트; `end_proposed.proposerId===agentAId`; `error` 미수신; 호스트가 아닌 플레이어도 정상 발의 가능 | FR-091 AC1 |
| L1-EN-019 | /end 과반 경계값 (정확히 50%) | 4인 세션, `/end` 발의 완료 | A 발의, B 찬성, C 반대, D 반대 (2/4 = 50%) | `end_vote_result.agreed===2`; `end_vote_result.disagreed===2`; `end_vote_result.passed===false` (50%는 과반 미달, 과반은 초과 필요) | FR-091 AC3 |
| L1-EN-020 | 동시 /end 발의 충돌 | 4인 세션, `playing` 상태, 종료 투표 비활성 | Agent A와 Agent B가 동시(같은 tick)에 `{type:'propose_end'}` 전송 | 전원에게 `end_proposed` 1건만 브로드캐스트됨(중복 발의 없음); 두 번째 요청자(`end_proposed`에 `proposerId`로 포함되지 않은 에이전트)에게 `error.code==='END_VOTE_ALREADY_OPEN'` 수신; 기존 `end_proposed.timeoutSeconds===60` 유지; `end_proposed` 총 수신 건수 `===1` | FR-091 AC2, AC4 |
| L1-EN-021 | 개인화된 엔딩에 행동 패턴 특성화 포함 검증 | `examine` 행동을 5회 이상 수행한 플레이어가 포함된 fixture 세계, 각 플레이어의 행동 로그가 기록됨 | 해당 플레이어가 포함된 세션에서 종료 조건 충족 후 `game_ending` payload 수집 | `personalEnding.narrative`가 비어있지 않음; `personalEnding.narrative`에 해당 플레이어의 행동 패턴에서 도출된 성격/특성 키워드(예: "신중한", "적극적", "탐색형", "관찰자적", "주도적" 등 성격/패턴 묘사어) 중 1개 이상이 포함됨; 행동 빈도가 낮은 다른 플레이어의 엔딩과 `narrative` 내용이 완전히 동일하지 않음 | FR-067 AC4 |

---

## L2 Phase 참조: Phase 2F (Ending)

| ID | 시나리오 | 사전조건 | 에이전트 행동 | 검증 (필드 레벨) | 관련 FR AC |
|----|---------|---------|-------------|------------------|------------|
| L2-2F-001 | 투표 종료 플로우 | 종료 유형 `vote` + 4인 세션 | 다중 에이전트가 투표를 순차 제출 | `vote_started`→`vote_progress`→`vote_ended`→`game_ending`→`game_finished` 순서 불변 | FR-063 AC1, AC2 |
| L2-2F-002 | /end 플레이어 주도 종료 플로우 | `/end` 허용 세션, 과반 동의 가능 | A가 `/end` 발의, 3/4 동의 | `end_proposed` 전체 브로드캐스트; `end_vote_result.passed===true`; 엔딩 구조가 일반 종료와 동일 | FR-091 AC2, AC3, AC5 |
| L2-2F-003 | 시간 초과 종료 플로우 | 강제 종료 타이머 활성 | 타이머 가속으로 경고/만료 연속 검증 | `game_event.event.type==='time_warning'` 후 `game_event.event.type==='game_end'`, `event.data.reason`에 `timeout` 문구 포함; 미해결 항목 참조 + 완결 문장 포함 | FR-066 AC1, AC2, AC3 |

### 모든 종료 시나리오 공통 Assertions

- `game_ending` payload가 `commonResult`와 `personalEnding`을 동시에 포함
- 개인 목표 달성/실패 판정 필드(`personalEnding.goalResults[]`)가 플레이어별로 존재
- `secretReveal`에 역할/목표/비밀(`playerSecrets[]`) + 반공개 공유관계(`semiPublicReveal[]`) + NPC 숨김정보(`npcSecrets[]`) + 미발견 단서(`undiscoveredClues[]`) 포함
- `game_finished`는 세션당 1회만 발행되고 `game_ending` 이후에만 수신

---

## L3 참조

- FG-009: 투표 종료 게임 (투표 → 판정 → 엔딩 풀 플로우)
- FG-010: /end 플레이어 종료 (조기 종료 발의 → 투표 → 엔딩)
- FR-066 AC3 평가 기준: (1) ending 텍스트가 미해결 항목을 최소 1개 이상 참조 (2) 마지막 문단이 결과 요약/정서적 마무리 문장을 포함

---

## L4 체크리스트

> L4 subagent가 TUI 플레이 중 아래 항목에 해당하는 상황을 만나면 **반드시 검증하고** PlayExperienceReport의 `checklist` 필드에 기록한다.
> 자유 플레이를 유지하되, 체크리스트 항목을 의도적으로 커버하는 행동을 포함한다.

| 원본 ID | 체크 항목 | TUI 검증 방법 | Pass 기준 |
|---------|---------|--------------|----------|
| L1-EN-002 | 시간 초과 경고 표시 | 남은 시간 5분 전 관찰 | 시간 경고 메시지가 화면에 표시됨 |
| L1-EN-004 | 개인화 엔딩 표시 | 게임 종료 시 엔딩 화면 확인 | 자신만의 개인 엔딩 텍스트가 표시됨 |
| L1-EN-005 | 비밀 공개 표시 | 게임 종료 후 비밀 공개 섹션 확인 | 각 플레이어의 역할/비밀/목표가 공개되어 표시됨 |
| L1-EN-008 | 투표 중 /end 거부 | 이미 투표 진행 중 `/end` 시도 | 중복 종료투표 불가 오류 메시지가 표시됨 |
| L1-EN-017 | /end 60초 제한 | `/end` 발의 후 60초 경과 관찰 | 60초 후 투표 자동 마감 결과가 표시됨 |

### 정성 평가 (기존 L4 참조)

- L4-001~004: subagent가 엔딩까지 완주 후 PlayExperienceReport의 `ending` 항목으로 평가
  - satisfaction (만족도), reflectedActions (행동 반영), secretRevealSurprising (비밀 공개 놀라움)
  - unresolvedButClosed (미해결 요소를 남기되 완결감 제공; FR-066 AC3)
  - personalityReflection (플레이어 성격/행동 패턴 반영; FR-067 AC4)
  - **FR-067 AC4 평가 기준**: 개인 엔딩이 해당 플레이어의 실제 행동 패턴을 반영 — 평가자 3인 중 2인 이상 '반영됨' 판정

---

## AI Quality Eval 참조

- **Ending Eval**: 게임 트랜스크립트 + 생성된 엔딩 텍스트를 독립적으로 평가 (행동 반영도, 개인 목표 판정 정확성, 서사 완결성, 비밀 공개 극적 효과)
