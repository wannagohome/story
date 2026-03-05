# L1: Save & Feedback (저장/피드백)

> **MVP 우선순위:** P1/P2 — 이 파일의 모든 테스트 케이스는 MVP 이후 실행 대상입니다. (FR-069: P1, FR-070: P2, FR-071: P1, FR-072: P2)

**Traces**: FR-069, FR-070, FR-071, FR-072
**Layer**: L1 Command Agent
**인터랙션**: Protocol level
**AI 사용**: 없음 (fixture AI)

---

## L1 시나리오

| ID | 시나리오 | 사전조건 | 에이전트 행동 | 검증 (필드 레벨) | 관련 FR AC |
|----|---------|---------|-------------|------------------|------------|
| L1-SF-001 | 세션 데이터 저장 | 게임이 정상 종료 가능한 fixture, 호스트 저장 경로 writable | 게임 종료 후 저장 디렉터리 스캔 | `session-<id>.json` 1개 생성; 파일 파싱 성공; 루트에 `sessionId`, `savedAt`, `story`, `map` 존재 | FR-069 AC1, AC2, AC5 |
| L1-SF-002 | 저장 내용 검증 | L1-SF-001에서 생성된 저장 파일 확보 | 저장 JSON 로드 | `story.world`,`story.background`,`roles[]`,`clues[]`,`npcs[]` 필드 존재; 배열 길이가 fixture 데이터와 일치; `npcs[]` 각 원소에 `persona`(문자열, non-empty), `knownInfo`(배열), `hiddenInfo`(배열), `behaviorPrinciple`(문자열, non-empty) 서브필드 존재 | FR-069 AC1, AC2, AC4 |
| L1-SF-003 | 이전 게임 조회 | 저장 세션 2개 이상 존재 | `npx story history` 실행 | 목록 응답이 배열이며 각 항목에 `sessionId`,`createdAt`,`title` 포함 | FR-072 AC1 |
| L1-SF-004 | 기믹 및 단서 배치 저장 | 게임 내 기믹 2개 이상, 단서 2개 이상 배치된 fixture | 게임 종료 후 세션 JSON 검증 | 저장 JSON에 `gimmicks[]` 존재; 각 원소에 `id`,`roomId` 존재; 런타임 상태 참조 시 `gimmickStates[id].isTriggered`(bool),`gimmickStates[id].triggeredAt`(*int64) 확인; 배치 수가 런타임 상태와 일치; 저장 JSON에 `clues[]` 존재; 각 원소에 `id`,`roomId`(또는 `location`),`name` 존재; `clues[].roomId`가 런타임 방 ID 집합 내 유효 값; `clues.length`가 런타임 단서 수와 일치 | FR-069 AC3 |
| L1-SF-005 | 행동 로그 저장 opt-in | 호스트가 `saveActionLog=true` 설정, 플레이어 동의 완료 | 채팅/이동/`/do`/투표를 각각 1회 이상 수행 후 종료 | `session-<id>-actions.json` 생성; 로그에 `chat`,`move`,`action`,`vote` 타입 모두 존재; 각 로그에 `timestamp`(ISO-8601) 포함 | FR-070 AC1, AC2, AC3 |
| L1-SF-006 | 행동 로그 동의 | `saveActionLog=true`, 플레이어 4명 세션 | 로그 시작 전 동의 프롬프트 진행 | 모든 플레이어에게 로그 동의 프롬프트(`system_message.content`에 '행동 로그 저장 동의' 문구 포함) 1회 수신; 각 플레이어가 동의/거부 응답 제출 후 `server.logConsent[playerId]`에 저장됨 | FR-070 AC4 |
| L1-SF-007 | 로그 미동의 시 미저장 | 플레이어 A가 `logConsent=false` 응답 | A/B/C가 동일 타입 행동 수행 후 종료 | 로그 파일에서 `playerId===A` 엔트리 0건; B/C 엔트리는 정상 저장; 시스템 로그에 A 비저장 사유 기록 | FR-070 AC4 |
| L1-SF-008 | 피드백 UI 표시 | 엔딩 화면(`game_ending` 수신 후) 진입 상태 | 엔딩 화면 다음 단계로 이동 | `feedback_request` 이벤트 수신; 입력 스키마에 `funRating`(1~5), `immersionRating`(1~5), `comment`(optional) 포함 | FR-071 AC1, AC2, AC3 |
| L1-SF-009 | 피드백 제출 | 피드백 프롬프트 활성 상태 | `funRating=4`, `immersionRating=5`, `comment='긴장감 좋음'` 제출 | `feedback_ack` 수신; 저장 레코드에 `sessionId`,`funRating`,`immersionRating`,`comment`,`submittedAt` 존재 | FR-071 AC1, AC2, AC3 |
| L1-SF-010 | 피드백 스킵 | 피드백 프롬프트 활성 상태 | `skip_feedback` 실행 | `game_finished` 수신(피드백 스킵 후 세션 정상 종료); `feedback` 저장 파일/레코드 미생성; 세션 종료 상태는 `completed` 유지 | FR-071 AC4 |
| L1-SF-011 | 피드백-세션 연결 | 피드백 1건 이상 제출된 세션 존재 | feedback 저장소와 session 저장소를 조인 조회 | `feedback.sessionId === session.sessionId`; 동일 `sessionId`로 스토리 데이터 조회 가능 | FR-071 AC5 |
| L1-SF-012 | 특정 게임 선택 | history 목록에 2개 이상 게임 존재 | history 목록에서 특정 `sessionId` 선택 조회 | 선택 응답이 요청한 `sessionId`와 일치; 상세 payload 반환 | FR-072 AC2 |
| L1-SF-013 | 상세 정보 열람 | L1-SF-012의 상세 payload 확보 | 상세 필드 스키마 검증 | 상세에 `story`,`map`,`roles`,`ending` 필드 모두 존재하고 비어있지 않음 | FR-072 AC3 |
| L1-SF-014 | 행동 로그 저장 opt-out | 호스트가 `saveActionLog=false` 설정, 게임 진행 후 종료 | 게임 종료 후 저장 디렉터리 스캔 | `session-<id>-actions.json` 파일 0건 생성, 동의 프롬프트 미표시, 서버 로그에 `saveActionLog=false` 기록 | FR-070 AC1 |
| L1-SF-015 | 저장소 쓰기 불가 오류 처리 | 저장 경로 권한을 읽기 전용으로 설정, 게임 종료 | 게임 종료 후 에러 메시지 확인 | `error.code==='SAVE_FAILED'`, `error.message`에 `저장 불가` 포함, 사일런트 실패 없음, 사용자에게 명확한 오류 메시지 전달 | FR-069 AC1 |
| L1-SF-016 | 빈 게임 이력 조회 | 저장된 세션 0건 상태, `npx story history` 실행 | 조회 응답 확인 | 응답이 배열이며 `length === 0`, 오류 없음, 빈 목록 메시지 표시 | FR-072 AC1 |

---

## L2 Phase 참조

없음 — 저장/피드백은 게임 종료 후 동작으로 L1에서 직접 검증.

---

## L3 참조

- FG-*: 모든 풀 게임 시나리오에서 게임 종료 시 세션 데이터 저장 자동 검증
- FR-070/071은 L1에서 스키마/동의/스킵 검증, L3에서는 장시간 플레이 후 누락 없이 저장되는지 회귀 검증

---

## L4 체크리스트

> L4 subagent가 TUI 플레이 중 아래 항목에 해당하는 상황을 만나면 **반드시 검증하고** PlayExperienceReport의 `checklist` 필드에 기록한다.
> 자유 플레이를 유지하되, 체크리스트 항목을 의도적으로 커버하는 행동을 포함한다.

| 원본 ID | 체크 항목 | TUI 검증 방법 | Pass 기준 |
|---------|---------|--------------|----------|
| L1-SF-003 | 이전 게임 조회 | `npx story history` 실행 | 이전 게임 목록이 표시됨 |
| L1-SF-008 | 피드백 UI 표시 | 게임 종료 후 피드백 화면 확인 | 재미/몰입 점수 입력 필드와 코멘트 필드가 표시됨 |
| L1-SF-009 | 피드백 제출 | 점수 입력 후 제출 | 피드백 제출 완료 메시지가 표시됨 |
| L1-SF-010 | 피드백 스킵 | 피드백 건너뛰기 선택 | 피드백 없이 정상 종료됨 |
| L1-SF-016 | 빈 게임 이력 조회 | 저장된 게임 없을 때 조회 | 빈 목록 메시지가 표시되고 오류 없음 |
