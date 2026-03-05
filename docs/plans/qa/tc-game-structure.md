# L1+L2: Game Structure (게임 구조/종료 조건)

**Traces**: FR-026, FR-027, FR-028, FR-030, FR-014, FR-019, FR-063, FR-092
**Layer**: L1 Command Agent + L2 Phase Agent
**인터랙션**: Protocol level
**AI 사용**: L1 없음 (fixture AI) / L2 LLM 의사결정
> 프로토콜 메시지 타입과 필드명은 /docs/plans/system-design/shared/protocol.md 및 /docs/plans/system-design/shared/events.md 기준

---

## L1 시나리오

| ID | 시나리오 | 사전조건 | 에이전트 행동 | 검증 (필드 레벨) | 관련 FR AC |
|----|---------|---------|-------------|------------------|-----------|
| L1-GS-001 | 게임 구조 JSON 존재 | 세계 생성 완료 후 서버 `world` 객체 접근 가능 상태 | 서버 내부 `server.world.gameStructure` 직접 검사 | `gameStructure.coreConflict` 문자열 존재, `gameStructure.progressionStyle` 문자열 존재, `gameStructure.endConditions` 배열 길이 `>= 1`, `gameStructure.winConditions` 배열 길이 `>= 1` | FR-026 AC1, AC3; FR-014 AC2, AC3 |
| L1-GS-002 | 종료 조건 triggerType 명시 | L1-GS-001 통과, `endConditions` 배열 존재 | 서버 내부 `server.world.gameStructure.endConditions[]` 순회 검사 | 모든 `endConditions[i].triggerType` 문자열 존재, 모든 `endConditions[i].triggerCriteria` 객체 존재, `triggerCriteria` 키 개수 `>= 1` | FR-027 AC2, AC5; FR-019 AC1 |
| L1-GS-003 | 시간초과 fallback 포함 | L1-GS-001 통과 | `endConditions`에서 `isFallback === true` 항목 검색 | `endConditions` 내 `isFallback === true` 항목 개수 `>= 1`, 해당 항목 `triggerType === 'timeout'`, 해당 항목 `triggerCriteria.timeoutMinutes` 숫자 존재 | FR-027 AC4; FR-019 AC2 |
| L1-GS-004 | requiredSystems 명시 | L1-GS-001 통과, 종료 조건에 `triggerType` 존재 | 서버 내부 종료 조건 타입별 필요 시스템 집합 계산 후 `requiredSystems[]` 비교 | `server.world.gameStructure.requiredSystems` 배열 존재, `triggerType==='vote'` 조건이 있으면 `requiredSystems`에 `'vote'` 포함, `triggerType==='consensus'` 조건이 있으면 `requiredSystems`에 `'consensus'` 포함, `triggerType==='ai_judgment'` 조건이 있으면 `requiredSystems`에 `'ai_judge'` 포함 | FR-019 AC3 |
| L1-GS-005 | 투표 종료 플로우 | 종료 조건 중 `triggerType==='vote'` 존재, 투표 가능 플레이어 3명 이상 | `vote_started` 발생 후 모든 에이전트가 `{type:'vote', targetId:'...'}` 제출 | 이벤트 순서가 `vote_started -> vote_progress -> vote_ended -> game_ending`과 일치, `vote_started.reason` 존재, `vote_started.candidates` 배열 존재, `vote_ended.results` 배열 존재, `vote_ended.outcome` 문자열 존재, `game_event.event.type==='game_end'`, `game_event.event.data.reason`에 `vote` 문구 포함 | FR-028 AC3; FR-063 AC1, AC2 |
| L1-GS-006 | 합의 종료 플로우 | 종료 조건 중 `triggerType==='consensus'` 존재, 합의 시스템 활성 상태 | 모든 에이전트가 `end_proposed` 수신 후 `{type:'end_vote', agree:true}` 제출 | `end_proposed.proposerId` 존재, `end_vote_result.passed===true`, `end_vote_result.agreed===totalPlayers`, `game_event.event.type==='game_end'`, `game_event.event.data.reason`에 `consensus` 문구 포함 | FR-028 AC4; FR-063 AC1, AC3 |
| L1-GS-007 | AI 판정 종료 | 종료 조건 중 `triggerType==='ai_judgment'` 존재, AI 판정 시스템 활성 | AI 판정 시스템이 자동으로 종료 조건 평가 (서버 내부에서 주기적 호출) | AI 판정에 의해 종료 조건 충족 시 `game_event.event.type==='game_end'`, `game_event.event.data.reason`에 `ai_judgment` 문구 포함, `game_ending.commonResult` 비어있지 않음 | FR-028 AC5; FR-063 AC1, AC4 |
| L1-GS-008 | 종료 미충족 시 계속 | 현재 상태가 어떤 종료 조건도 만족하지 않는 fixture 준비 | 서버 내부 종료 판정 엔진을 3회 tick 실행 | 매 평가마다 종료 조건 미충족, 3회 평가 동안 `game_ending` 이벤트 미발생, `game_event.event.type==='game_end'` 미발생, 서버 상태 `playing` 유지 | FR-063 AC5; FR-028 AC1 |
| L1-GS-009 | 진행 방식 자유 문자열 허용 | `gameStructure.progressionStyle='증언 릴레이 기반 비선형 추적'` fixture를 검증 가능한 환경 | 서버 내부 `server.world.gameStructure.progressionStyle` 직접 검사 | `progressionStyle`가 non-empty 문자열이고 fixture 값과 동일하며, 사전 정의 enum 고정값으로만 제한되지 않음 | FR-026 AC2 |
| L1-GS-010 | 전원 의미 있는 행동 경로 보장 | StoryValidator 검증 결과 조회 가능, 플레이어 4명 이상 | 서버 내부 `storyValidator.Validate()` 결과의 `playerActionPaths` 검사 | `playerActionPaths`에 모든 플레이어 id 키가 존재, 모든 `playerActionPaths[playerId].length>=1`, 빈 배열 플레이어 수 `0` | FR-026 AC4; FR-093 AC2 |
| L1-GS-011 | 종료 조건 유형 다양성 | L1-GS-001 통과, `endConditions` 배열 존재 | `endConditions[].triggerType` 유니크 집합 계산 | `uniqueTriggerTypes.length>=2`, `timeout` 외 최소 1개 이상의 추가 `triggerType` 존재 | FR-027 AC1 |
| L1-GS-012 | 종료 조건 달성 가능성 검증 | StoryValidator 검증 결과 조회 가능 | 서버 내부 `storyValidator.Validate()` 결과 검사 | `feasible===true`, `reachableEndConditionIds.length>=1`, `unreachableEndConditionIds.length===0`(필드 존재 시), 검증 결과 상태가 `passed` | FR-027 AC3; FR-093 AC1 |
| L1-GS-013 | 게임 종료 시 game_end 이벤트 발생 | 하나 이상의 종료 조건을 충족 가능한 세션 | 종료 조건 1개를 충족시켜 종료 트리거 | `game_ending` 이후 `game_event.event.type==='game_end'` 1회 수신, `game_event.event.data.reason` non-empty, `game_event.event.data.commonResult` non-empty | FR-028 AC2 |
| L1-GS-014 | sessionConfig.targetMinutes 범위 경계값 검증 *(Priority: P1 — MVP 이후)* | 세션 시작 직후, `sessionManager.timingConfig` 접근 가능 | 서버 내부 `sessionManager.timingConfig.targetMinutes` 직접 검사 | `targetMinutes >= 10`; `targetMinutes <= 30`; 경계값 10분(최솟값)과 30분(최댓값)이 허용됨; `targetMinutes < 10` 또는 `targetMinutes > 30`인 경우 설정 오류로 간주 | FR-030 AC1 |
| L1-GS-015 | time_warning AI 조정 시간과 무관하게 5분 전 발송 검증 *(Priority: P1 — MVP 이후)* | `targetMinutes`가 AI에 의해 기본값(20분)과 다르게 조정된 세션(예: 24분), 타이머 가속 가능 | 타이머를 가속하여 잔여 시간이 5분이 되는 시점까지 진행 | `game_event.event.type==='time_warning'` 수신; `event.data.remainingMinutes===5`; `time_warning` 발송 시각이 `targetMinutes - 5` 분 경과 시점과 일치(±10초); AI 조정 전 하드코딩 값(예: 15분)이 아닌 조정된 `targetMinutes` 기준으로 계산됨 | FR-030 AC2 |
| L1-GS-016 | timeout fallback이 AI 조정 시간에서 트리거됨 검증 *(Priority: P1 — MVP 이후)* | `targetMinutes`가 AI에 의해 조정된 세션(예: 24분), `isFallback===true` 종료 조건 존재, 타이머 가속 가능 | 타이머를 가속하여 `targetMinutes` 경과 시점까지 진행 | `game_event.event.type==='game_end'`, `event.data.reason`에 `timeout` 문구 포함; 트리거 시각이 AI 조정 `targetMinutes`(예: 24분) 기준과 일치(±10초); 하드코딩된 기본 `baseMinutes`(20분) 시점에서는 `game_end` 미발생; `game_ending.commonResult` 비어있지 않음 | FR-030 AC3 |

**참고:** `/solve` 명령 테스트(L1-PA-020)는 [tc-player-actions.md](./tc-player-actions.md)에 정의.

**FR-092 커버리지:** 개인 목표 시스템(FR-092)은 tc-world-generation.md의 L2-WG-011(독립성/고유 동기), L2-WG-012(충돌 관계 탐지), L2-WG-013(엔딩 평가 가능 데이터)에서 주로 검증. 게임 구조 관점에서는 L1-GS-010(전원 의미 있는 행동 경로)이 FR-092 AC2를 간접 커버.

---

## L2 Phase 참조 (상세 절차 + payload assertion)

### Phase 2B: 게임 구조 검증 (FR-014, FR-019, FR-026, FR-027, FR-093)

1. **세계 생성 결과 수집**
   - 방법: 서버 내부 `server.world.gameStructure` 직접 검사 (L2 에이전트가 테스트 API를 통해 접근)
   - 기대 구조 예시:

```json
{
  "gameStructure": {
    "coreConflict": "...",
    "progressionStyle": "...",
    "endConditions": [
      {
        "id": "ec_vote_01",
        "triggerType": "vote",
        "triggerCriteria": {"requiredMajority": 0.5},
        "isFallback": false
      },
      {
        "id": "ec_timeout_01",
        "triggerType": "timeout",
        "triggerCriteria": {"timeoutMinutes": 20},
        "isFallback": true
      }
    ],
    "winConditions": ["..."],
    "requiredSystems": ["vote"]
  }
}
```

   - 검증: `coreConflict`, `progressionStyle`, `endConditions(>=1)`, `winConditions(>=1)` 존재.

2. **종료 조건 스키마 검증**
   - 검증: 모든 `endConditions[]`에 `triggerType`, `triggerCriteria` 존재.
   - 검증: `isFallback===true` 조건이 최소 1개 존재.
   - 검증: fallback 조건의 `triggerType==='timeout'`.

3. **필요 시스템 매핑 검증**
   - 검증: 종료 조건에서 파생된 시스템 요구사항이 `requiredSystems[]`에 누락 없이 기록.
   - 검증: `vote`, `consensus`, `ai_judgment` 트리거와 `requiredSystems` 항목이 1:1 매핑.

4. **달성 가능성 검증 리포트 확인**
   - 방법: 서버 내부 `storyValidator.Validate()` 결과 검사
   - 기대 구조 예시:

```json
{
  "feasible": true,
  "reachableEndConditionIds": ["ec_vote_01", "ec_timeout_01"],
  "playerActionPaths": {
    "agent_a": ["talk", "move", "vote"],
    "agent_b": ["talk", "examine", "vote"]
  },
  "autoEvaluable": true
}
```

   - 검증: `feasible===true`, `reachableEndConditionIds.length>=1`, 모든 플레이어에 `playerActionPaths[playerId].length>=1`, `autoEvaluable===true`.

### Phase 2E: 세션 시간 유동 조절 (FR-030)

1. **세션 시간 계획 값 확인**
   - 방법: 서버 내부 `sessionManager.timingConfig` 직접 검사 (세션 시작 직후)
   - 기대 구조 예시:

```json
{
  "baseMinutes": 20,
  "targetMinutes": 24,
  "minMinutes": 10,
  "maxMinutes": 30,
  "decisionReason": "players=6, pace=normal"
}
```

   - 검증: `baseMinutes===20`, `targetMinutes>=10`, `targetMinutes<=30`.

2. **진행 속도 모니터링 확인**
   - 방법: 서버 내부 `gmEngine.pacingState` 직접 검사 (주기적 호출)
   - 기대 구조 예시:

```json
{
  "elapsedMinutes": 12,
  "progressScore": 0.64,
  "recommendedEndingWindow": {"start": 18, "end": 24}
}
```

   - 검증: `elapsedMinutes` 증가 추세 유지, `progressScore` 갱신, `recommendedEndingWindow.start <= recommendedEndingWindow.end`.

3. **간접 진행도 전달 확인**
   - 기대 payload 예시:

```json
{
  "type": "game_event",
  "event": {
    "id": "evt_narr_01",
    "timestamp": 1710000000000,
    "visibility": {"scope": "all"},
    "type": "narration",
    "data": {
      "text": "밤이 깊어지며 모두의 선택이 결말로 수렴합니다.",
      "mood": "urgent"
    }
  }
}
```

   - 검증: `game_event.event.type==='narration'`, `game_event.event.data.text`가 존재하고 플레이어 메시지 로그에 직접 숫자 카운트다운 대신 서술형 진행 힌트가 기록.

### Phase 2F: 종료 판정 엔진 (FR-028, FR-063)

1. **투표 기반 종료 케이스**
   - 절차: 조건 충족 상태 만들기 -> `vote_started` 수신 -> 전원 투표 -> `vote_ended` -> `game_ending` -> `game_event(event.type='game_end')`.
   - 검증 payload: `game_event.event.type==='game_end'`, `game_event.event.data.reason`에 `vote` 문구 포함, `game_event.event.data.commonResult` 존재.

2. **합의 기반 종료 케이스**
   - 절차: `end_proposed` 수신 -> 전원 `{type:'end_vote', agree:true}` 제출 -> `end_vote_result` -> `game_ending` -> `game_event(event.type='game_end')`.
   - 검증 payload: `end_vote_result.passed===true`, `game_event.event.data.reason`에 `consensus` 문구 포함.

3. **AI 판단 기반 종료 케이스**
   - 절차: 서버 내부에서 AI가 종료 조건 충족 판정 -> `game_ending` -> `game_event(event.type='game_end')`.
   - 검증 payload: `game_event.event.data.reason`에 `ai_judgment` 문구 포함, `game_ending.commonResult` 비어있지 않음.

4. **종료 미충족 지속 케이스**
   - 절차: 종료 조건 미충족 상태로 서버 내부 종료 판정 엔진을 3회 tick 실행.
   - 검증: 각 tick 후 종료 조건 미충족 확인, `game_event.event.type==='game_end'` 미발생, 서버 상태 `playing` 유지.

---

## L3 참조

- FG-009: 투표 종료 게임 풀 플로우
- FG-010: /end 플레이어 주도 종료 풀 플로우
- FG-*: 모든 풀 게임에서 종료 조건 판정 엔진 동작 검증

---

## L4 체크리스트

> L4 subagent가 TUI 플레이 중 아래 항목에 해당하는 상황을 만나면 **반드시 검증하고** PlayExperienceReport의 `checklist` 필드에 기록한다.
> 자유 플레이를 유지하되, 체크리스트 항목을 의도적으로 커버하는 행동을 포함한다.

| 원본 ID | 체크 항목 | TUI 검증 방법 | Pass 기준 |
|---------|---------|--------------|----------|
| L1-GS-005 | 투표 종료 플로우 체험 | 투표를 통한 게임 종료 시도 | 투표 시작→진행→결과→엔딩 순서로 화면에 표시됨 |
| L1-GS-006 | 합의 종료 플로우 체험 | `/end` 발의 후 전원 동의 | 종료 발의→투표→결과→엔딩이 화면에 표시됨 |
| L1-GS-009 | 진행 방식 자유도 체감 | 게임 플레이 중 다양한 행동 시도 | 다양한 행동 경로가 열려 있고 자유롭게 탐색 가능 |
