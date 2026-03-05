# L1: Player Actions (플레이어 행동)

**Traces**: FR-041, FR-042, FR-043, FR-044, FR-045, FR-046, FR-047, FR-048, FR-049, FR-084, FR-091, FR-028
**Note**: FR-048 (NPC 아이템 수령)은 L1-PA-024에서 조건 충족 흐름으로 검증
**Layer**: L1 Command Agent
**인터랙션**: Protocol level
**AI 사용**: 없음 (fixture AI)

---

## L1 시나리오

| ID | 시나리오 | 사전조건 | 에이전트 행동 | 검증 (필드 레벨) | 관련 FR AC |
|----|---------|---------|-------------|------------------|------------|
| L1-PA-001 | 방 조사 (대상 지정) | Agent A/B는 같은 방(거실), Agent C는 다른 방(부엌), `desk` 조사 가능 | Agent A가 `{type:'examine', target:'desk'}` 전송 | Agent A/B가 `examine_result` 1건 수신, `data.playerId=AgentA`, `data.target='desk'`, `data.description` 비어있지 않음, `data.clueFound` boolean, Agent C 미수신 | FR-041 AC1, AC2, AC4 |
| L1-PA-002 | 방 조사 (대상 미지정) | Agent A가 조사 가능한 방에 있음 | Agent A가 `{type:'examine'}` 전송 | Agent A/B가 `examine_result` 수신, `data.target` 비어있지 않음, `data.description` 비어있지 않음, 응답 타입이 `examine_result`로 고정 | FR-041 AC1, AC2 |
| L1-PA-003 | 조사로 단서 발견 | fixture에 `desk` 조사 시 단서가 나오도록 설정됨 | Agent A가 `{type:'examine', target:'desk'}` 전송 | `clue_found` 이벤트 발생, `data.playerId=AgentA`, `data.clue.id`, `data.clue.name`, `data.location` 필드 존재, 같은 방 수신 범위 유지 | FR-041 AC3, AC4 |
| L1-PA-004 | 조사 결과 같은 방 공개 | Agent A/B는 거실, Agent C는 부엌 | Agent A가 `/examine` 실행 | Agent B가 `examine_result` 수신, Agent C는 `examine_result`와 `clue_found` 모두 미수신 | FR-041 AC4 |
| L1-PA-005 | 행동 서술 기본 판정 | Agent A/B는 같은 방, `/do` 허용 상태 | Agent A가 `{type:'do', action:'문을 잠근다'}` 전송 | Agent A/B가 `action_result` 수신, `data.playerId=AgentA`, `data.action='문을 잠근다'`, `data.result` 비어있지 않음, `data.triggeredEvents` 배열 존재 | FR-042 AC1, AC2 |
| L1-PA-006 | NPC 대화 | Agent A/B는 NPC `raymond`와 같은 방 | Agent A가 `{type:'talk', npcId:'raymond', message:'어젯밤 뭘 봤지?'}` 전송 | `npc_dialogue` 이벤트 수신, `data.npcId='raymond'`, `data.npcName` 비어있지 않음, `data.text` 비어있지 않음, 같은 방 공개 유지 | FR-043 AC2, AC3, AC4 |
| L1-PA-007 | 없는 NPC 대화 거부 | Agent A 방에 `raymond` 없음 | Agent A가 `{type:'talk', npcId:'raymond', message:'...'}` 전송 | `error` 수신, `error.code='NPC_NOT_IN_ROOM'`, 성공 이벤트(`npc_dialogue`) 미발생 | FR-043 AC1 |
| L1-PA-008 | 방에 NPC 1명일 때 자동 대상 | Agent A 방에 NPC가 1명만 존재 | Agent A가 `{type:'talk', npcId:'', message:'안녕하세요'}` 전송 (npcId 빈 문자열 — 자동 대상) | `npc_dialogue.data.npcId`가 해당 단일 NPC id와 일치, 별도 대상 지정 없이 정상 처리 | FR-043 AC1 |
| L1-PA-009 | 인벤토리 조회 기본 | Agent A는 아이템 1개, 단서 1개 보유 | Agent A가 `{type:'request_inventory'}` 전송 | Agent A만 `inventory` 수신, `items[]` 각 엔트리에 `id`, `name`, `description` 필드 존재; `clues[]` 각 엔트리에 `id`, `name`, `description` 필드 존재 | FR-044 AC1, AC2, AC3 |
| L1-PA-010 | 역할 재확인 기본 | 게임 시작 브리핑 완료 상태 | Agent A가 `{type:'request_role'}` 전송 | Agent A가 `role_info` 수신, `role.characterName`, `role.background`, `role.personalGoals[]`, `role.secret` 필드 존재 | FR-045 AC1, AC2 |
| L1-PA-011 | 도움말 조회 | Agent A/B/C 접속 상태 | Agent A가 `{type:'request_help'}` 전송 | Agent A만 `help_info` 수신, 각 명령어 엔트리에 `command`와 `description` 필드 존재, Agent B/C는 `help_info` 미수신 | FR-046 AC1, AC2, AC3 |
| L1-PA-012 | NPC에게 아이템 전달 | Agent A는 `letter` 보유, `raymond`와 같은 방 | Agent A가 `{type:'give', npcId:'raymond', itemId:'letter'}` 전송 | `npc_receive_item` 이벤트 발생, `data.npcId='raymond'`, `data.playerId=AgentA`, `data.item.id='letter'` | FR-047 AC1, AC3 |
| L1-PA-013 | 없는 아이템 전달 거부 | Agent A 인벤토리에 `fake_key` 없음 | Agent A가 `{type:'give', npcId:'raymond', itemId:'fake_key'}` 전송 | `error.code='ITEM_NOT_FOUND'`, `npc_receive_item` 미발생 | FR-047 AC2 |
| L1-PA-014 | 투표 기본 진행 | 서버가 `vote_started` 발행 완료, 투표 대상 목록 존재 | Agent A가 `{type:'vote', targetId:'bob'}` 전송 | `vote_progress` 수신, `votedCount` 정수 증가, Agent A 중복 투표 시도는 거부 코드 반환 | FR-049 AC1, AC2, AC4 |
| L1-PA-015 | 투표 결과 집계 | 4인 게임, `vote_started` 상태, 전원 미투표 | Agent A/B/C/D가 순차 투표 | 마지막 투표 후 `vote_ended` 1회 발생, `results[]`에 4표 합계 반영, `results[].votes` 총합이 4 | FR-049 AC3 |
| L1-PA-016 | `/end` 발의 브로드캐스트 | 게임 진행 중, 종료 투표 비활성 상태 | Agent A가 `{type:'propose_end'}` 전송 | 전원에게 `end_proposed` 전달, `proposerId=AgentA`, `timeoutSeconds===60` 필드 존재 | FR-091 AC1, AC2, AC6 |
| L1-PA-017 | `/end` 과반수 동의 종료 | 4인 게임, `end_proposed` 활성 | Agent A/B/C가 찬성표 전송 | `end_vote_result.passed=true`, 이후 `game_ending` 이벤트에 `personalEnding`(PlayerEnding 구조)와 `secretReveal` 필드 포함 | FR-091 AC3, AC5 |
| L1-PA-018 | `/end` 과반수 미달 지속 | 4인 게임, `end_proposed` 활성 | Agent A만 찬성, B/C/D 반대 | `end_vote_result.passed=false`, 게임 상태가 `playing` 유지, `game_ending` 미발생 | FR-091 AC4 |
| L1-PA-019 | `/who` 조회 | 4인 플레이어가 서로 다른 방에 분산 | Agent A가 `{type:'request_who'}` 전송 | Agent A만 `who_info` 수신, 엔트리마다 `id`, `nickname`, `roomId`, `roomName`, `status` 필드 존재, Agent B/C/D는 `who_info` 미수신 | FR-084 AC1, AC2 |
| L1-PA-020 | `/solve` 합의 시스템 연동 | `gameStructure.requiredSystems=['consensus']`, 해결안 제출 가능 상태 | Agent A가 `{type:'solve', answer:'열쇠는 지하 금고에 있다'}` 전송 | `solve_progress` 수신, `submittedCount` 증가, `totalPlayers`가 현재 플레이어 수와 일치, 즉시 `game_end` 미발생 | FR-028 AC1, AC4 |
| L1-PA-021 | 동시 `/examine` 충돌 없음 | Agent A/B 같은 방, 둘 다 조사 가능 | Agent A/B가 같은 tick에 `/examine` 전송 | 두 요청 모두 `examine_result` 수신, 응답 상호 덮어쓰기 없음, 서버 오류 미발생 | FR-041 AC1, AC2 |
| L1-PA-022 | `/give` 잘못된 아이템 재검증 | Agent A 인벤토리에 `poison` 없음 | Agent A가 `{type:'give', npcId:'raymond', itemId:'poison'}` 전송 | `error.code='ITEM_NOT_FOUND'` 고정, 인벤토리 변화 없음 | FR-047 AC2 |
| L1-PA-023 | 다른 방 NPC 대화 차단 | Agent A=거실, `raymond`=서재 | Agent A가 `{type:'talk', npcId:'raymond', message:'열쇠 줘'}` 전송 | `error.code='NPC_NOT_IN_ROOM'`, `npc_dialogue` 미발생 | FR-043 AC1 |
| L1-PA-024 | NPC 아이템 수령 조건 충족 흐름 | `raymond` NPC의 `trustLevel=0.5`(fixture), 필요 조건은 `trustLevel>=0.7`, Agent A/B 같은 방, Agent C 다른 방 | 1) Agent A가 신뢰 대화 2회 수행하여 trustLevel 상승 2) 조건 달성 후 Agent A가 트리거 질문 전송 | 조건 미충족 구간(trustLevel<0.7)에서는 `npc_give_item` 미발생, 조건 충족 직후 `npc_give_item` 1회 발생, `data.item.id='master_key'`, Agent A 인벤토리에 `master_key` 추가, Agent A/B 수신, Agent C 미수신 | FR-048 AC1, AC2, AC3, AC4 |
| L1-PA-025 | `/do` 결과 같은 방 공개 검증 | Agent A/B 같은 방, Agent C 다른 방 | Agent A가 `{type:'do', action:'창문을 연다'}` 전송 | Agent A/B가 `action_result` 수신, Agent C는 `action_result` 미수신 | FR-042 AC3 |
| L1-PA-026 | `/do`가 스토리 이벤트 트리거 | fixture에 `action='숨겨진 레버를 당긴다'`가 분기 트리거로 등록됨 | Agent A가 `{type:'do', action:'숨겨진 레버를 당긴다'}` 전송 | `action_result.data.triggeredEvents`에 `story_event` 포함, 별도 `story_event` 이벤트 발생, `data.title`, `data.description`, `data.consequences[]` 필드 존재 | FR-042 AC4 |
| L1-PA-027 | 인벤토리 본인 전용 검증 | Agent A/B 같은 방, 둘 다 접속 상태 | Agent A가 `{type:'request_inventory'}` 전송 | Agent A는 `inventory` 수신, Agent B 수신 로그에 `inventory` 타입 0건, Agent B 페이로드에 `items`, `clues` 키 0건 | FR-044 AC1 |
| L1-PA-028 | 역할 정보 본인 전용 검증 | Agent A/B 같은 방, 게임 진행 중 | Agent A가 `{type:'request_role'}` 전송 | Agent A는 `role_info` 수신, Agent B 수신 로그에 `role_info` 타입 0건 | FR-045 AC1 |
| L1-PA-029 | `/role` 응답과 시작 브리핑 동일성 | 게임 시작 시 Agent A의 `briefing_private.role` 원문 바이트 저장 완료 | Agent A가 `{type:'request_role'}` 전송 | `role_info.role`의 `characterName`, `background`, `personalGoals`, `secret` 등이 저장한 `briefing_private.role`과 일치, 문자열 길이와 바이트 해시(SHA-256) 동일 | FR-045 AC3 |
| L1-PA-030 | NPC 아이템 수령 이벤트 같은 방 공개 | Agent A/B 같은 방, Agent C 다른 방, Agent A가 `letter` 보유 | Agent A가 `{type:'give', npcId:'raymond', itemId:'letter'}` 전송 | Agent A/B가 `npc_receive_item` 수신, Agent C는 `npc_receive_item` 미수신, `data.item.id='letter'` 일치 | FR-047 AC4 |
| L1-PA-031 | 투표 시작 전 `/vote` 거부 | 현재 라운드에 `vote_started` 미발행 | Agent A가 `{type:'vote', targetId:'bob'}` 전송 | `error.code='VOTE_NOT_ACTIVE'`, `vote_progress` 미발생 | FR-049 AC1 |
| L1-PA-032 | 실시간 투표 현황 카운트 | 4인 게임, `vote_started` 활성, 전원 미투표 | Agent A, Agent B, Agent C가 순서대로 투표 | 각 투표 직후 전원에게 `vote_progress` 방송, `votedCount`가 1,2,3으로 증가, `totalVoters===4` 고정 | FR-049 AC4 |
| L1-PA-033 | 투표 비활성 구조에서 `/vote` 거부 | `gameStructure.requiredSystems`에 `vote` 없음 | Agent A가 `{type:'vote', targetId:'bob'}` 전송 | `error.code='VOTING_DISABLED'`, `vote_progress`와 `vote_ended` 모두 미발생 | FR-049 AC5 |
| L1-PA-034 | `/end` 60초 제한과 미응답 처리 | 4인 게임, `end_proposed` 발행, 제한시간 60초 | Agent A만 찬성 전송, Agent B/C/D는 60초 동안 미응답 유지 | 60초 경과 시 `end_vote_result` 자동 발생, `passed===false`, `agreed===1`, `disagreed===3`, 미응답 3명이 거부로 집계됨 | FR-091 AC6 |
| L1-PA-035 | 호스트 투표 개시 트리거 | 호스트가 투표 개시 명령을 전송할 수 있는 상태 (`playing`), 4인 세션, 투표 구조 활성화 (`gameStructure.requiredSystems`에 `vote` 포함) | 1) 호스트가 투표 개시 명령 전송 2) 서버가 `vote_started` 발행 3) 플레이어들이 각자 투표 | `vote_started` 메시지가 전체 플레이어(4인)에게 전달됨, `vote_started.initiator`에 호스트 식별자 존재, 이후 `vote_progress` 수신 정상, 최종 `vote_ended` 1건 발생, `results[]` 합계 일치 | FR-049 AC1 |
| L1-PA-036 | `/do` 빈 행동 문자열 오류 | Agent A/B 같은 방, `/do` 허용 상태 | Agent A가 `{type:'do', action:''}` 전송 (action 빈 문자열) | Agent A가 `error.code==='EMPTY_ACTION'` 수신; `action_result` 미발생; Agent B에게 `action_result` 미전송; 서버 상태 정상 유지 | FR-042 AC1 |
| L1-PA-037 | `/do` 극도로 긴 행동 텍스트 처리 | Agent A/B 같은 방, `/do` 허용 상태 | Agent A가 `{type:'do', action:'A'.repeat(1001)}` 전송 (1001자 action 문자열) | `error` 수신(`error.code`가 길이 초과 관련 코드 포함, 예: `ACTION_TOO_LONG`) 또는 `action_result` 수신 시 `data.action.length <= 1000` (truncation 처리); 서버 크래시 없음; Agent B 수신 로그에 1001자 이상 `action` 필드 0건 | FR-042 AC1 |

---

## NFR 임베딩

| 검증 항목 | 방식 | 기준 |
|----------|------|------|
| AI 응답 시간 (FR-041 AC5) | 실서버 AI 호출 로그 기반 타이밍 측정 | p95 < 3초 (NFR-001). fixture AI 기반 L1에서는 측정 불가. Not measurable with fixture AI. Measured in L2/L3 with real AI. → tc-nonfunctional.md L3-NF-001 참조 (NFR-001 응답시간 3초 p95) |

---

## L2 Phase 참조: Phase 2D (Exploration & Actions)

3 에이전트 (Explorer, Diplomat, Chaotic)가 15턴 자유 플레이:

**Explorer 에이전트:**
- 모든 인접 방 순회 이동, 각 방에서 `/examine` 실행, 발견된 단서 인벤토리 확인

**Diplomat 에이전트:**
- NPC 있는 방으로 이동, NPC와 3회 이상 대화, 아이템 교환 시도

**Chaotic 에이전트:**
- 존재하지 않는 NPC 대화 시도 후 `NPC_NOT_IN_ROOM` 오류 확인
- 빈 명령, 긴 문자열, 특수문자 입력 시 서버 crash 없음 확인

**교차 검증:**
- Agent A(거실)의 examine 결과는 Agent B(거실) 수신, Agent C(부엌) 미수신
- Agent A의 NPC 대화는 같은 방 에이전트만 수신

---

## L3 참조

- FG-001~FG-010: 모든 풀 게임에서 플레이어 행동 검증
- FG-008: NPC 집중 상호작용 (아이템 교환 포함)
- FG-009: 투표 종료 게임
- FG-010: `/end` 플레이어 종료

---

## L4 체크리스트

> L4 subagent가 TUI 플레이 중 아래 항목에 해당하는 상황을 만나면 **반드시 검증하고** PlayExperienceReport의 `checklist` 필드에 기록한다.
> 자유 플레이를 유지하되, 체크리스트 항목을 의도적으로 커버하는 행동을 포함한다.

| 원본 ID | 체크 항목 | TUI 검증 방법 | Pass 기준 |
|---------|---------|--------------|----------|
| L1-PA-001 | 방 조사 결과 표시 | `/examine <대상>` 실행 | 조사 결과와 단서 발견 여부가 화면에 표시됨 |
| L1-PA-005 | 행동 서술 결과 | `/do <행동>` 실행 | 행동 결과 서술이 화면에 표시됨 |
| L1-PA-006 | NPC 대화 응답 | `/talk <NPC> <메시지>` 실행 | NPC 응답이 화면에 표시됨 |
| L1-PA-007 | 없는 NPC 대화 거부 | 방에 없는 NPC와 대화 시도 | 오류 메시지가 표시됨 |
| L1-PA-009 | 인벤토리 조회 | `/inventory` 실행 | 보유 아이템과 단서 목록이 표시됨 |
| L1-PA-010 | 역할 재확인 | `/role` 실행 | 역할명, 배경, 개인 목표, 비밀이 표시됨 |
| L1-PA-011 | 도움말 조회 | `/help` 실행 | 사용 가능한 명령어 목록과 설명이 표시됨 |
| L1-PA-014 | 투표 진행 표시 | 투표 시작 후 투표 참여 | 투표 현황(N/M voted)이 표시됨 |
| L1-PA-016 | /end 발의 알림 | `/end` 실행 | 종료 발의 알림이 전원에게 표시됨 |
| L1-PA-019 | /who 조회 | `/who` 실행 | 플레이어 닉네임, 위치, 접속 상태 목록이 표시됨 |
