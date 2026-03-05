# L1/L2: Additional Features (추가 기능)

**Traces**: FR-081, FR-082, FR-083, FR-084, FR-085, FR-086, FR-087, FR-088, FR-089
**Layer**: L1 Command Agent
**인터랙션**: Protocol level
**AI 사용**: 없음 (fixture AI)
> 프로토콜 메시지 타입과 필드명은 /docs/plans/system-design/shared/protocol.md 및 /docs/plans/system-design/shared/events.md 기준

---

## L1 시나리오

| ID | 시나리오 | 사전조건 | 에이전트 행동 | 검증 (필드 레벨) | 관련 FR AC |
|----|---------|---------|-------------|------------------|------------|
| L1-AF-001 | graceful 종료 (호스트) | `playing` 상태, Host + Agent A/B 접속, 연결/세션 리소스 모니터 활성화 | Host 프로세스에 SIGINT 1회 전송 | Agent A/B가 `game_cancelled` 1건씩 수신, `game_cancelled.reason`에 `호스트 종료` 포함, 2초 내 `activeSessions.has(roomCode)=false`, `networkServer.peerCount=0`, `resourceTracker.openWebSockets=0` | FR-083 AC1 |
| L1-AF-002 | 참가자 종료 | `playing` 상태, Host + Agent A/B 접속 | Agent B 프로세스에 SIGINT 1회 전송 | Host/Agent A가 `player_disconnected.playerId=agentBId` 수신, Agent B만 `connectionStatus='disconnected'`, Host/Agent A는 `connectionStatus='connected'` 유지, 세션 상태 `playing` 유지 | FR-083 AC2 |
| L1-AF-003 | 대기실 채팅 기본 동작 *(Priority: P1 — MVP 이후)* | 게임 시작 전 로비, Host + Agent A/B 접속 | Agent A가 `{type:'chat', content:'준비됐어요?'}` 전송 | Agent A/B/Host 전원 `chat_message` 1건 수신, `chat_message.senderId=agentAId`, `chat_message.scope='room'`, `chat_message.content='준비됐어요?'` | FR-081 AC1 |
| L1-AF-004 | 대기실 채팅 전체 전달 *(Priority: P1 — MVP 이후)* | 로비에 4명(Host, A, B, C), 수신 로그 초기화 | Agent C가 `{type:'chat', content:'시작 전 단서 공유 금지!'}` 전송 | Host/A/B/C 수신 로그에 동일 `chat_message.content='시작 전 단서 공유 금지!'` 1건 존재, 4개 클라이언트의 `chat_message.content` 값 완전 일치, 누락 수신자 수 `0` | FR-081 AC2 |
| L1-AF-005 | 귓속말/DM 차단 | 로비 상태, Agent A/B 같은 방, DM 명령 비활성 정책 적용 | Agent A가 `{type:'dm', targetPlayerId:agentBId, message:'비밀'}` 전송 | Agent A가 `error.code='NOT_SUPPORTED'` 수신, `error.message`에 `귓속말` 포함, Agent B 수신 로그 `dm_message` 0건, 서버 point-to-point 채널 생성 카운트 `0` | 관련 FR: 없음 (concept 설계 원칙: 귓속말 기능 미제공) |
| L1-AF-006 | 테마 힌트 입력(선택) *(Priority: P2 — MVP 이후)* | Host 시작 프롬프트에 테마 입력 필드 존재, 키워드 `space` 입력 가능 | Host가 `themeKeyword='space'` 입력 후 `start_game` 실행 | `start_game.themeKeyword='space'`(protocol.md StartGameMessage), `generation_progress` 정상 진행, 생성 결과 `worldGeneration.meta.theme` 비어있지 않음, 테마 입력 필드 미입력 선택도 허용됨(`themeKeyword=''` 저장 가능) | FR-089 AC1 |
| L1-AF-007 | 호스트 게임 취소 | 로비 상태, Host + Agent A/B 접속, `activeSessions`에 roomCode 등록됨 | Host가 `{type:'cancel_game'}` 전송 | Agent A/B/Host 전원 `game_cancelled` 수신, `game_cancelled.reason`에 `게임 취소` 포함, 모든 클라이언트 `connectionStatus='disconnected'`, `activeSessions.has(roomCode)=false` | FR-082 AC1, AC2, AC3 |
| L1-AF-008 | 비호스트 게임 취소 거부 | 로비 상태, Host + Agent A 접속 | Agent A가 `{type:'cancel_game'}` 전송 | Agent A가 `error.code='NOT_HOST'` 수신, Host 수신 로그 `system_message`에서 `게임 취소` 문구 0건, `activeSessions.has(roomCode)=true` 유지 | FR-082 AC1 |
| L1-AF-009 | 게임 중 데이터 저장 시도 (부분 저장) | `playing` 상태, action log 20건 이상 존재, 저장 경로 모니터링 활성화 | Host 프로세스에 SIGINT 전송 | `shutdown_report.saveAttempted=true`, partial 저장 파일 1개 생성(`*-partial.json`), 파일에 `sessionId`, `players`, `map`, `actionLogs` 필드 존재, `actionLogs.length>=1`, `status='interrupted'` 기록 | FR-083 AC3 |
| L1-AF-010 | `/who` 필드 완전성 | `playing` 상태, 4명 플레이어가 서로 다른 방에 위치 | Agent A가 `{type:'request_who'}` 전송 | Agent A가 `who_info` 수신, `who_info.players.length=4`, 각 엔트리마다 `id`,`nickname`,`roomId`,`roomName`,`status` 필드 존재, 각 `status` 값이 허용 enum 목록 `['connected','disconnected']` 내부 값과 정확히 일치 | FR-084 AC1 |
| L1-AF-011 | `/who` 본인 전용 노출 | `playing` 상태, Agent A/B/C 접속 | Agent A가 `{type:'request_who'}` 전송 | Agent A 수신 로그 `who_info` 1건, Agent B/C 수신 로그 `who_info` 0건, Agent A 응답의 `who_info.players[]`에 `agentAId` 포함(WhoInfoMessage에 requesterId 없음 — 응답 수신 자체가 본인 전용 증거) | FR-084 AC2 |
| L1-AF-012 | AI 호출 실패 재시도 + 지수 백오프 | AI provider 스텁이 첫 3회 `timeout` 반환, 4번째 호출 성공, retry 로그 활성화 | Host가 `start_game` 실행 | `retryLog.count=3`, `retryLog[0].delayMs=1000`, `retryLog[1].delayMs=2000`, `retryLog[2].delayMs=4000`, 4번째 시도에서 세계 생성 성공 후 세션 상태 `briefing` 진입 | FR-085 AC1, AC2 |
| L1-AF-013 | 재시도 전체 실패 시 세션 종료 | AI provider 스텁이 모든 호출 실패, 참가자 2명 이상 접속 | Host가 `start_game` 실행 후 실패까지 대기 | 재시도 로그 `retryLog.count=3`, 마지막 실패 직후 Host/참가자에 `error.message` 전달, `error.message`에 `세계 생성 실패` 포함, 세션 상태 `terminated`, `activeSessions.has(roomCode)=false` | FR-085 AC3 |
| L1-AF-014 | 활성 세션 중복 검사 (동시 시작) | Host A/B를 별도 프로세스로 동시 시작, 룸 코드 RNG 시드 강제로 동일 설정 | Host A/B가 같은 시점에 `create_session` 실행 | `hostA.roomCode != hostB.roomCode`, Host B 로그에 `roomCodeCollisionDetected=true` 1회, 두 코드 모두 `activeSessions`에 동시 등록, 충돌 코드 재사용 건수 `0` | FR-086 AC1, AC2 |
| L1-AF-015 | 룸 코드 엔트로피 형식 검증 | 룸 코드 사전(dictionary) 로드 완료, 코드 생성 훅으로 1000회 샘플링 가능 | 코드 생성 1000회 실행 후 샘플 분석 | 모든 코드가 `^[A-Z]+-\d{4}$` 정규식 일치 (PRD: "단어-4자리숫자" 형태, 단어 길이 가변), 접두사가 사전 단어 집합에 포함, 1000개 샘플 고유 코드 수 >= 995 | FR-086 AC3 |
| L1-AF-016 | 방 방문자 기록 노출 *(Priority: P1 — MVP 이후)* | `playing` 상태, Agent A/B/C가 `library`를 순차 방문 후 이탈, 방문 로그 저장 활성화 | Agent A가 `library`에서 `{type:'examine'}` 전송 | `game_event.event.type==='examine_result'` 이벤트 수신, `game_event.event.data.description`에 방문자 관련 서술 포함(AI가 방문 흔적을 묘사 텍스트에 반영), `game_event.event.data.clueFound` 필드 존재 | FR-087 AC1 |
| L1-AF-017 | 이동 이력 알리바이 검증 활용 *(Priority: P1 — MVP 이후)* | 살인 시각 `22:15` 고정 fixture, Agent B의 이동 로그 존재 | 1) Agent A가 사건 방 `library` 조사 2) Agent A가 `/do 알리바이를 검증한다` 실행 | `game_event.event.type==='examine_result'`의 `game_event.event.data.description`에 방문 흔적 관련 서술 포함, 서술 내용이 `22:15` 이전 방문을 시사하면 `game_event.event.type==='action_result'`의 `game_event.event.data.result`에 `알리바이 성립` 문구 포함, 검증 결과가 이동 로그와 모순되지 않음 | FR-087 AC2 |
| L1-AF-018 | NPC 조건부 스토리 분기 *(Priority: P1 — MVP 이후)* | `hasNPC=true`, `raymond` 조건부 행동(`trustLevel>=0.7`)이 분기 플래그 `vault_unlocked`와 연결됨 | Agent A가 신뢰 조건 달성 후 NPC 트리거 대화 실행 | `game_event.event.type==='story_event'` 1건 발생, `game_event.event.data.title` 비어있지 않음, `game_event.event.data.description` 비어있지 않음, `game_event.event.data.consequences.length>=1`, 분기 후 `gameState.gimmickStates['vault'].isTriggered===true` | FR-088 AC1, AC2 |
| L1-AF-019 | 분기 관련 플레이어 한정 전달 *(Priority: P1 — MVP 이후)* | Agent A/B는 `raymond`와 같은 방, Agent C는 다른 방, NPC 분기 트리거 준비 | Agent A가 분기 트리거 대화 실행 | Agent A/B 수신 로그에 동일 `game_event.event.id` 1건 존재, Agent C 수신 로그 동일 `event.id` 0건, 전달된 `game_event.event.visibility.scope='room'`, `game_event.event.visibility.roomId='library'` | FR-088 AC3 |
| L2-AF-020 | 미입력 = 완전 자유 생성 (다양성 검증) *(Priority: P2 — MVP 이후)* | Host가 테마 입력을 비워서 10회 새 세션 생성, 생성 결과 수집 | 각 실행에서 `themeKeyword=''`로 `start_game` 실행 | 10회 모두 `start_game.themeKeyword=''`, 생성 결과 `worldGeneration.meta.theme` 비어있지 않음, 10회 결과의 장르 라벨 고유 수 >= 5, 상위 2개 장르 합산 비율 <= 70% | FR-089 AC2 |
| L2-AF-021 | 키워드 참고하되 구속되지 않음 (`space`) *(Priority: P2 — MVP 이후)* | Host가 `themeKeyword='space'` 입력, 생성 결과 텍스트 분석 훅 활성화 | `start_game` 5회 실행 후 결과 비교 | 5회 모두 `worldGeneration.meta.theme`에 우주 관련 토큰 1개 이상 포함, 5회 모두 `worldGeneration.meta.theme != 'space'`, 5회 결과의 `world.title` 고유 수 >= 3, 동일 키워드 입력이어도 스토리 시놉시스 코사인 유사도 평균 < 0.85 | FR-089 AC3 |

---

## L2 Phase 참조

> DM/귓속말은 프로토콜에 정의되지 않은 메시지 유형. 테스트 하네스는 미정의 타입에 NOT_SUPPORTED 응답을 반환해야 함.

- Phase 2A: 대기실 채팅(FR-081), 호스트 게임 취소(FR-082), 세계 생성 재시도(FR-085), 룸 코드 고유성(FR-086), 테마 힌트 입력(FR-089 AC1)
- Phase 2D: `/who` 조회(FR-084), 이동 이력(FR-087), NPC 조건부 분기(FR-088)
- Graceful 종료(FR-083)는 Phase 2A, FG 시나리오에서 종료 유형별로 반복 검증

> **L2-AF-020/021 이동 사유**: 키워드 기반 다양성 검증(L2-AF-020/021)은 본질적으로 확률적(stochastic) 특성을 가지므로 L1 fixture 테스트가 아닌 L2 Phase 에이전트 테스트로 분류. 실제 AI 호출이 필요하며 다회 실행을 통한 통계적 검증이 필수.

---

## L3 참조

- FG-* 전체 게임: graceful 종료(FR-083), `/who` 접근 범위(FR-084), 이동 이력 알리바이 활용(FR-087)
- FG-008: NPC 분기 트리거(FR-088)와 관련 플레이어 전달 범위 재검증
- 다회 실행 리그레션: 세계 생성 재시도(FR-085), 키워드 미입력 다양성(FR-089 AC2), 키워드 참고 비구속성(FR-089 AC3)

---

## L4 체크리스트

> L4 subagent가 TUI 플레이 중 아래 항목에 해당하는 상황을 만나면 **반드시 검증하고** PlayExperienceReport의 `checklist` 필드에 기록한다.
> 자유 플레이를 유지하되, 체크리스트 항목을 의도적으로 커버하는 행동을 포함한다.

| 원본 ID | 체크 항목 | TUI 검증 방법 | Pass 기준 |
|---------|---------|--------------|----------|
| L1-AF-003 | 대기실 채팅 동작 | 로비에서 채팅 전송 | 로비 참가자 전원에게 채팅 메시지가 표시됨 |
| L1-AF-005 | 귓속말/DM 차단 | DM 시도 | 지원하지 않는 기능 오류 메시지가 표시됨 |
| L1-AF-006 | 테마 힌트 입력 | 게임 시작 시 테마 키워드 입력 | 테마 입력 필드가 표시되고 입력/미입력 모두 허용됨 |
| L1-AF-007 | 호스트 게임 취소 알림 | 호스트가 게임 취소 시 | 모든 참가자에게 게임 취소 메시지가 표시됨 |
| L1-AF-010 | /who 정보 표시 | `/who` 실행 | 플레이어 닉네임, 위치, 접속 상태가 표시됨 |
| L1-AF-016 | 방 방문자 기록 | `/examine` 실행 후 방문자 정보 확인 | 조사 결과의 설명 텍스트에 방문 흔적 관련 서술이 포함되어 표시됨 |
