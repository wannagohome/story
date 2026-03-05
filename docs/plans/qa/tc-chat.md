# L1: Chat & Communication (채팅/커뮤니케이션)

**Traces**: FR-036, FR-037, FR-038, FR-039, FR-040, FR-081
**Layer**: L1 Command Agent
**인터랙션**: Protocol level
**AI 사용**: 없음 (fixture AI)
> 프로토콜 메시지 타입과 필드명은 /docs/plans/system-design/shared/protocol.md 및 /docs/plans/system-design/shared/events.md 기준

---

## L1 시나리오

| ID | 시나리오 | 사전조건 | 에이전트 행동 | 검증 (필드 레벨) | 관련 FR AC |
|----|---------|---------|-------------|------------------|-----------|
| L1-CH-001 | 같은 방 채팅 | Agent A, B는 `living_room`, Agent C는 `kitchen`에 위치 | Agent A가 `{type:'chat', content:'단서 공유'}` 전송 | Agent B 수신 payload에서 `chat_message.senderId===agentA.id`, `chat_message.senderName===agentA.nickname`, `chat_message.scope==='room'`, `chat_message.content==='단서 공유'`; Agent C는 동일 `chat_message.content==='단서 공유'` 미수신 | FR-036 AC1, AC2, AC3; FR-054 AC1, AC2 |
| L1-CH-002 | 채팅 메시지 형식 | Agent A, B 같은 방 | Agent A가 `{type:'chat', content:'안녕하세요'}` 전송 | 수신 메시지에 `chat_message.senderName` 존재, `chat_message.content==='안녕하세요'`, `chat_message.scope==='room'`, `chat_message.timestamp` 존재 및 유효한 Unix timestamp (int64 정수) 형식 | FR-036 AC1, AC2, AC3 |
| L1-CH-003 | 글로벌 채팅 | Agent A, B, C가 서로 다른 방에 위치 | Agent A가 `{type:'shout', content:'전체 공지'}` 전송 | A/B/C 모두 `chat_message` 수신, `chat_message.scope==='global'`, `chat_message.senderId===agentA.id`, `chat_message.senderName===agentA.nickname`, `chat_message.senderLocation` 필드 존재, **`chat_message.senderLocation` 값이 전송 시점 Agent A의 실제 현재 방 ID와 일치**, `chat_message.content==='전체 공지'`; `scope==='global'` 필드가 존재하여 TUI에서 시각적 구분(접두사/색상)의 기반이 됨을 확인. 실제 시각적 렌더링은 L3 터미널 UI 테스트(tc-terminal-ui.md)에서 검증. | FR-037 AC1, AC2, AC3 |
| L1-CH-004 | 시스템/이벤트 메시지 구분 | 이동 이벤트, GM 서술 이벤트, 단서 발견 이벤트 생성 가능한 fixture 로드 | 순서대로 이동 발생 -> GM 서술 트리거 -> 단서 발견 트리거 | 각 이벤트가 `game_event.event.type`으로 래핑되어 수신되고, 순서대로 `player_move`, `narration`, `clue_found` 타입이 확인되며 서로 다른 타입으로 구분됨 | FR-038 AC1, AC2, AC3 |
| L1-CH-005 | 입장/퇴장 알림 | Agent B가 `study`에 대기, Agent A가 인접 방에서 이동 준비 | Agent A가 `study`에 입장 후 다시 다른 방으로 이동 | 입장 시 `player_joined_room.type==='player_joined_room'`, `player_joined_room.nickname` 존재; 퇴장 시 `player_left_room.type==='player_left_room'`, `player_left_room.nickname` 존재, `player_left_room.destination` 존재 | FR-040 AC1, AC2 |
| L1-CH-006 | 대기실 채팅 (P1: MVP 이후) | 세션 상태 `lobby`, 대기 플레이어 3명 접속 | Agent A가 `{type:'chat', content:'준비됐나요?'}` 전송 | 모든 대기 플레이어가 `chat_message` 수신, `chat_message.senderId===agentA.id`, `chat_message.content==='준비됐나요?'`, `chat_message.scope==='room'` | FR-081 AC1, AC2 |
| L1-CH-007 | 빈 메시지 전송 | Agent A 접속 완료 | Agent A가 `{type:'chat', content:''}` 전송 | Agent A가 `error.code==='EMPTY_MESSAGE'` 수신, 어떤 수신자에게도 `chat_message.scope==='room'` 브로드캐스트 없음 | FR-036 AC1 |
| L1-CH-008 | 초장문 메시지 | 메시지 최대 길이 정책 활성화 | Agent A가 길이 5001자의 `{type:'chat', content:'...5001 chars...'}` 전송 | Agent A가 `error.code` 수신 (메시지 길이 초과 관련), `error.message`에 최대 길이 안내 포함, 다른 에이전트에게 `chat_message` 브로드캐스트 없음 | NFR-015 |
| L1-CH-009 | 유니코드 메시지 | Agent A, B 같은 방 | Agent A가 `{type:'chat', content:'테스트🙂مرحبا'}` 전송 | Agent B 수신 payload에서 `chat_message.content==='테스트🙂مرحبا'`, `chat_message.scope==='room'` | NFR-029 |
| L1-CH-010 | NPC 대화 방 범위 | Agent A는 `hall`(NPC 동방), Agent B는 `kitchen`(다른 방) | Agent A가 `{type:'talk', npcId:'raymond', message:'무엇을 봤나요?'}` 전송 | Agent A가 `game_event.event.type==='npc_dialogue'` 수신, `hall`의 동방 플레이어는 동일 `game_event.event.id` 수신, Agent B는 동일 `event.id` 미수신, 수신 이벤트 `game_event.event.visibility.scope==='room'` | FR-054 AC1, AC2, AC3 (cross-check FR-043 AC3) |
| L1-CH-011 | /examine 결과 방 범위 | Agent A, B는 `library`, Agent C는 `kitchen` | Agent A가 `{type:'examine', target:'desk'}` 전송 | Agent A/B는 동일 `game_event.event.id` 수신 및 `game_event.event.type==='examine_result'`, 수신 이벤트 `game_event.event.visibility.scope==='room'`, Agent C는 동일 `event.id` 미수신 | FR-041 AC4; FR-054 AC1, AC2, AC3 |
| L1-CH-012 | 시스템 메시지 스타일 구분 (TUI 참조) | L1-CH-003, L1-CH-004, L1-CH-005 통과, L3 TUIAgent 실행 가능 | `tc-terminal-ui.md`의 FG-007 실행 | `tuiEvaluation.messageTypesDistinguishable===true`, `tuiEvaluation.layoutReadable===true`, 검증 근거를 `tc-terminal-ui.md` 리포트 경로에 기록 | FR-037 AC2; FR-038 AC2, AC3 |
| L1-CH-013 | 채팅 200개 보관 | 단일 방에서 2명 이상 접속 | 205개 메시지 연속 전송 후 클라이언트 측 `state.Messages` 길이 확인 | 클라이언트 로컬 `state.Messages.length===200` (store.md maxMessages=200에 의해 최근 200개만 유지), 가장 오래된 메시지가 6번째 전송부터 시작 (최초 5개 삭제됨) | FR-039 AC2 |
| L1-CH-014 | 스크롤 지원 (TUI) | 채팅 로그 200개 이상 누적, L3 TUIAgent 실행 가능 | `tc-terminal-ui.md` 기준으로 PageUp/PageDown 스크롤 및 auto-scroll 토글 수행 | `tuiEvaluation.scrollWorking===true`, `tuiEvaluation.commandsResponsive===true`, `tuiEvaluation.autoScrollToggleVisible===true`, `tuiEvaluation.autoScrollStateChangeDetected===true` | FR-039 AC1, AC3 |

---

## NFR 임베딩

| 검증 항목 | 방식 | 기준 |
|----------|------|------|
| 메시지 전달 지연 | chat 전송 시각과 수신 시각의 `timestamp` 차이 측정 | p95 < 500ms (NFR-003) |

---

## L2 Phase 참조: Phase 2D

1. **같은 방 범위 전달 검증**
   - Agent A(거실) chat -> Agent B(거실), Agent C(부엌) 관찰.
   - 검증: B만 `chat_message.scope==='room'` 수신, C는 동일 내용 메시지 미수신.

2. **/examine 방 범위 검증**
   - Agent A가 `/examine` 실행.
   - 검증: 같은 방 플레이어만 `game_event.event.type==='examine_result'` 수신, 다른 방 플레이어 미수신.

3. **NPC 대화 방 범위 검증**
   - Agent A가 NPC와 대화.
   - 검증: 같은 방 플레이어만 `game_event.event.type==='npc_dialogue'` 수신, 다른 방 플레이어 미수신.

---

## L3 참조

- FG-007: TUI 경험 검증에서 메시지 유형 시각적 구분 확인
- FR-039 (채팅 로그 스크롤)는 FG-007의 `tuiEvaluation.scrollWorking`, `autoScrollStateChangeDetected`로 검증

---

## L4 체크리스트

> L4 subagent가 TUI 플레이 중 아래 항목에 해당하는 상황을 만나면 **반드시 검증하고** PlayExperienceReport의 `checklist` 필드에 기록한다.
> 자유 플레이를 유지하되, 체크리스트 항목을 의도적으로 커버하는 행동을 포함한다.

| 원본 ID | 체크 항목 | TUI 검증 방법 | Pass 기준 |
|---------|---------|--------------|----------|
| L1-CH-001 | 같은 방 채팅 수신 확인 | 같은 방에서 채팅 전송 후 수신 관찰 | 같은 방 플레이어에게만 메시지가 표시됨 |
| L1-CH-003 | 글로벌 채팅(외치기) | `/shout` 명령 사용 | 모든 플레이어에게 메시지가 표시되고 `[전체]` 등 구분 표시 있음 |
| L1-CH-005 | 입장/퇴장 알림 | 방 이동 시 관찰 | `[닉네임]이(가) 들어왔습니다/나갔습니다` 시스템 메시지 표시 |
| L1-CH-006 | 대기실 채팅 | 로비에서 채팅 전송 | 대기 중 모든 플레이어에게 채팅 메시지가 전달됨 |
| L1-CH-007 | 빈 메시지 전송 거부 | 빈 입력 후 Enter | 오류 메시지 표시, 빈 메시지가 다른 플레이어에게 보이지 않음 |
| L1-CH-010 | NPC 대화 방 범위 확인 | NPC와 대화 후 다른 방 화면 관찰 | 같은 방에서만 NPC 대화 내용이 표시됨 |
