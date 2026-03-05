# L1: Session Management (세션 관리)

**Traces**: FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, FR-008, FR-082, FR-083, FR-086
**Layer**: L1 Command Agent
**인터랙션**: Protocol level (+ TUI 부트스트랩/프롬프트 확인)
**AI 사용**: 없음 (fixture AI)
> 프로토콜 메시지 타입과 필드명은 /docs/plans/system-design/shared/protocol.md 및 /docs/plans/system-design/shared/events.md 기준

---

## L1 시나리오

| ID | 시나리오 | 사전조건 | 에이전트 행동 | 검증 (필드 레벨) | 관련 FR AC |
|----|---------|---------|-------------|------------------|------------|
| L1-SM-001 | 게임 호스팅 | 활성 세션 없음, 호스트 CLI 실행 가능 | Host가 `npx story host` 실행 | `joined.roomCode`가 `^[A-Z]+-\d{4}$` 정규식 일치, TUI 헤더 `header.roomCode === joined.roomCode`, 터미널 출력에 동일 코드가 강조 스타일(ANSI bold/inverse)로 1회 이상 표시, `activeSessions.has(roomCode) === true` | FR-001 AC1, AC2, AC3 |
| L1-SM-002 | 정상 참가 (프롬프트 기반) | 유효 룸 코드 존재, 참가자 아직 미입장 | Joiner가 `npx story join <roomCode>` 실행 후 닉네임 `Alice` 입력 | 닉네임 입력 전 TUI 프롬프트 `ui.prompt.id === 'nickname_input'` 노출, 입력 후 `joined.playerId` 비어있지 않음, `joined.roomCode === hostRoomCode`, 직후 `lobby_update.players.some(p => p.nickname === 'Alice') === true` | FR-002 AC1, FR-003 AC1 |
| L1-SM-003 | 무효 룸 코드 | 존재하지 않는 룸 코드 준비 | Joiner가 `npx story join FAKE-9999` 실행 | `error.code === 'INVALID_ROOM_CODE'`, `error.message`에 입력 코드 포함, 동일 세션 기준 `lobby_update` 미수신 | FR-002 AC2 |
| L1-SM-004 | 시작된 게임 참가 차단 | 세션 상태가 `playing` | Joiner가 `join` 시도 | `error.code === 'GAME_ALREADY_STARTED'`, `error.message`에 재시도 불가 사유 포함, `lobby_update.players.length` 변화 없음 | FR-002 AC3 |
| L1-SM-005 | 닉네임 유효성 검사 | 유효 룸 코드 존재 | 빈 문자열, 21자 문자열, 제어문자 포함 문자열로 각각 `join` 시도 | 빈 문자열/21자/제어문자 입력 모두 `error.code === 'INVALID_NICKNAME'`, 각 오류에 `error.message`가 허용 규칙(1~20자/제어문자 금지)을 명시 | FR-003 AC2, AC4 |
| L1-SM-005a | 닉네임 경계값 허용 검증 | 유효 룸 코드 존재, 닉네임 슬롯 사용 가능 | 1자 닉네임 `A`로 `join` 시도 후 성공 확인; 별도 세션에서 정확히 20자 닉네임 `ABCDEFGHIJKLMNOPQRST`로 `join` 시도 후 성공 확인 | 1자 닉네임: `error` 미수신, `joined.playerId` non-empty, `lobby_update.players.some(p => p.nickname === 'A') === true`; 20자 닉네임: `error` 미수신, `joined.playerId` non-empty, `lobby_update.players.some(p => p.nickname === 'ABCDEFGHIJKLMNOPQRST') === true`; 두 경우 모두 `INVALID_NICKNAME` 미발생 | FR-003 AC2 |
| L1-SM-006 | 닉네임 중복 거부 | Agent A가 `Alice`로 이미 입장 | Agent B가 동일 닉네임 `Alice`로 `join` | Agent B가 `error.code === 'DUPLICATE_NICKNAME'` 수신, `lobby_update.players.filter(p => p.nickname === 'Alice').length === 1` 유지 | FR-003 AC3 |
| L1-SM-007 | 로비 실시간 플레이어 목록 갱신 | Host만 접속한 로비 | Alice, Bob, Carol 순차 `join` | 각 `join` 직후 전원에게 `lobby_update` 브로드캐스트, 단계별 `lobby_update.players.length === 2/3/4`, 마지막 메시지의 nickname 집합이 `{Host, Alice, Bob, Carol}`와 정확히 일치 | FR-004 AC1 |
| L1-SM-008 | 호스트 게임 시작 트리거 | Host 포함 2명 이상 `connected` | Host가 `{type:'start_game'}` 전송 | Host 요청에 `error` 미발생, 전원에게 첫 `generation_progress` 수신(`type === 'generation_progress'`, `step` 비어있지 않음, `progress >= 0`), 서버 상태 `lobby -> generating` 전이 | FR-005 AC1, AC3 |
| L1-SM-009 | 비호스트 시작 시도 거부 | Host + Player 1명 이상 로비 | 비호스트가 `{type:'start_game'}` 전송 | 비호스트에게 `error.code === 'NOT_HOST'`, `error.message`에 권한 부족 포함, 상태가 계속 `lobby` 유지 | FR-005 AC1 |
| L1-SM-010 | 1명(호스트만) 시작 거부 | 로비에 Host 1명만 존재 | Host가 `{type:'start_game'}` 전송 | Host에게 `error.code === 'NOT_ENOUGH_PLAYERS'`, `error.message`에 `minPlayers=2`, 현재 연결 수 `currentPlayers=1`, 상태 `lobby` 유지 | FR-005 AC2, FR-006 AC1 |
| L1-SM-011 | 최대 인원 초과 참가 거부 | Host 포함 8명 이미 접속 | 9번째 플레이어가 `join` 시도 | 9번째 참가자 `error.code === 'ROOM_FULL'`, `error.message`에 `maxPlayers=8`, 기존 참가자 대상 `lobby_update.players.length === 8` 유지 | FR-006 AC1, AC2 |
| L1-SM-012 | 연결 해제 알림 브로드캐스트 | `playing` 상태, 3명 이상 접속 | Bob의 WebSocket 강제 종료 | 다른 플레이어가 `player_disconnected` 수신(`player_disconnected.playerId===bobId`, `player_disconnected.nickname==='Bob'`), `/who` 응답에서 `who_info.players.find(p => p.id===bobId).status === 'disconnected'` | FR-007 AC1 |
| L1-SM-013 | 30초 내 재접속 복원 | Bob이 `disconnected`, 끊긴 시각 기록 | T+25초에 Bob이 `{type:'rejoin', playerId:bobId}` 전송 | 전원에게 `player_reconnected.playerId === bobId`, `player_reconnected.nickname === 'Bob'`, Bob의 `request_inventory` 결과가 disconnect 직전 스냅샷과 동일, `/who`에서 Bob 상태 `connected` | FR-007 AC2 |
| L1-SM-014 | 30초 초과 비활성 전환 + AI 대응 | Bob이 `playing` 중 `disconnected` | 35초 경과 대기 | `/who`에서 Bob 상태가 `inactive`, 전원에게 AI 대응 이벤트 수신(`game_event.event.type === 'narration'`, `game_event.event.data.text`에 Bob 부재 대응 문구 포함), 세션은 종료되지 않고 진행 유지 | FR-007 AC3 |
| L1-SM-015 | API 키 검증 실패 | API 키가 잘못된 값(`bad-key`) | Host 실행 후 API 키 제출 | `error.code === 'INVALID_API_KEY'`, `error.message`에 검증 실패 원인 포함, 프로세스 exit code != 0, `activeSessions`에 신규 세션 미생성 | FR-008 AC4 |
| L1-SM-016 | 게임 시작 후 추가 참가 차단 | 상태가 `generating` | 신규 참가자가 `join` 시도 | `error.code === 'GAME_ALREADY_STARTED'`, 기존 플레이어 기준 `lobby_update.players.length` 불변, 신규 참가자에게 `joined` 미전송 | FR-005 AC4 |
| L1-SM-017 | 호스트 게임 취소 | 로비에 Host + 2명 이상 접속 | Host가 `{type:'cancel_game'}` 전송 | 전원에게 `game_cancelled.reason`에 호스트 취소 사유 포함, 모든 클라이언트 `connectionStatus === 'disconnected'` 전이, 서버 `activeSessions.has(roomCode) === false`, `networkServer.peerCount === 0` | FR-082 AC1, AC2, AC3 |
| L1-SM-018 | 비호스트 취소 시도 | Host + Player 접속 | 비호스트가 `{type:'cancel_game'}` 전송 | 요청자에게 `error.code === 'NOT_HOST'`, 다른 참가자에게 취소 알림 미전송, `activeSessions.has(roomCode) === true` 유지 | FR-082 AC1 |
| L1-SM-019 | 룸 코드 고유성/충돌 재생성 | 테스트 더블로 첫 2회 룸 코드 생성값을 동일하게 주입 | Host A/B가 동시에 세션 생성 | Host A `roomCode === 'WOLF-0001'`, Host B는 충돌 감지 후 재생성되어 `roomCode !== 'WOLF-0001'`, 두 코드 모두 `^[A-Z]+-\d{4}$` 일치, 충돌 로그 카운트 1회 | FR-086 AC1, AC2, AC3 |
| L1-SM-020 | 월드 생성 중 호스트 연결 해제 | 상태가 `generating`, 참가자 1명 이상 존재 | Host 채널 강제 종료 | 참가자에게 `error.code === 'CONNECTION_LOST'` 전송, 2초 내 `activeSessions.has(roomCode) === false`, 참가자 추가 요청 전송 시 로컬에서 연결 종료 상태 유지 | FR-007 AC1 |
| L1-SM-021 | 전원 동시 연결 해제 정리 | Host 포함 전원 접속 중 | 모든 플레이어 채널 동시 종료 | grace window 종료 후 `activeSessions.has(roomCode) === false`, `networkServer.peerCount === 0`, `resourceTracker.openWebSockets === 0`, orphan timer/job 수 `0` | FR-007 AC1, FR-082 AC3 |
| L1-SM-022 | 재접속 시 닉네임 변경 시도 무시 | Bob이 disconnect 후 재접속 준비 | Bob 클라이언트 표시 닉네임을 `Robert`로 바꿔 `rejoin` 수행 | `player_reconnected.nickname === 'Bob'`, `lobby_update.players.find(p => p.id===bobId).nickname === 'Bob'`, 동일 세션 내 nickname 중복/변조 없음 | FR-007 AC2 |
| L1-SM-023 | 첫 실행 API 키 입력 프롬프트 | `STORY_OPENAI_KEY` unset, `~/.story/config.json` 없음 | Host가 `npx story host` 실행 | 룸 코드 발급 전 `ui.prompt.id === 'api_key_input'` 표시, `ui.prompt.masked === true`, API 키 제출 전 `createSession()` 호출 횟수 `0` | FR-008 AC1 |
| L1-SM-024 | API 키 로컬 설정 파일 저장 | 첫 실행, 유효 API 키 입력 가능 | Host가 유효 키 입력 후 진행 | `~/.story/config.json` 생성, JSON `apiKey`가 입력값과 동일, 파일 권한 `0600`, 동일 머신 재실행 시 API 키 프롬프트 미표시 | FR-008 AC2 |
| L1-SM-025 | 환경 변수 우선순위 | config 파일 `apiKey=KEY_FILE`, 환경변수 `STORY_OPENAI_KEY=KEY_ENV` | Host 실행 | API 키 프롬프트 미표시, AI provider 초기화 인자 `apiKey === 'KEY_ENV'`, config 파일 값은 변경되지 않음(`KEY_FILE` 유지) | FR-008 AC3 |
| L1-SM-026 | 새 참가자 접속 알림 (lobby_update와 분리) | Host와 Alice가 로비 대기 중 | Bob이 `join` | Host/Alice가 `system_message` 수신(`content`에 `Bob` + `참가` 포함), 동일 틱에 `lobby_update`와 별개 메시지 타입임을 확인(`type === 'system_message'`) | FR-004 AC2 |
| L1-SM-027 | 로비 정원 표기 `3/6명` | 세션 최대 인원 6, Host+Alice+Bob 접속 | 로비 렌더 사이클 1회 실행 | TUI 헤더 텍스트 `header.capacityLabel === '3/6명 접속 중'`, 데이터 소스 `lobby_update.players.length === 3`, `sessionConfig.maxPlayers === 6` | FR-004 AC3 |
| L1-SM-028 | 호스트 포함 인원 계산 | 최대 인원 8 설정, Host만 접속 | 7명 순차 join 후 1명 추가 join 시도 | 7명 입장 완료 시 `lobby_update.players.length === 8` 및 `players.filter(p => p.isHost).length === 1`, 다음 참가자는 `error.code === 'ROOM_FULL'` | FR-006 AC3 |
| L1-SM-029 | 호스트 Ctrl+C graceful 종료 (playing 중) | `playing` 상태, Host 포함 4명 접속, 게임 진행 중 | Host 프로세스에 SIGINT 전송 | 모든 참가자에게 `game_cancelled.reason`에 호스트 종료 사유 포함, 참가자 연결이 정리됨(`connectionStatus === 'disconnected'`), `activeSessions.has(roomCode) === false`, Host 프로세스 exit code `0` | FR-083 AC1 |
| L1-SM-030 | 참가자 Ctrl+C 종료 (playing 중) | `playing` 상태, Host 포함 4명 접속, Bob 프로세스에 SIGINT 전송 가능 | Bob 프로세스에 SIGINT 전송 | Bob만 퇴장 처리, 다른 플레이어에게 `system_message.content`에 `Bob` + `퇴장` 포함, 세션은 종료되지 않고 `sessionState === 'playing'` 유지, `/who`에서 Bob 상태 `disconnected` | FR-083 AC2 |
| L1-SM-031 | 호스트 Ctrl+C 시 데이터 저장 시도 | `playing` 상태, 게임 진행 중(행동 로그 존재), Host 프로세스에 SIGINT 전송 가능 | Host 프로세스에 SIGINT 전송 | `game_cancelled.reason`에 호스트 종료 사유(`HOST_SIGINT`) 포함, 저장 시도 로그 존재(또는 `session-<id>.json` 생성 확인), 참가자 알림 후 프로세스 종료까지 2초 이내 | FR-083 AC3 |

---

## L2 Phase 참조: Phase 2A (Lobby)

Phase 2A에서 위 시나리오들이 통합 검증됨:

```
Host(ProtocolAgent) 서버 시작
  → Agent1 join "Alice"  → lobby_update(players.length=2) 확인
  → Agent2 join "Bob"    → lobby_update(players.length=3) 확인
  → Agent3 join "Carol"  → lobby_update(players.length=4) 확인
  → 대기실 채팅 교환 (FR-081)
  → Host가 start_game
  → 전원 generation_progress(step, progress) 동기 수신 확인
```

**Phase 2A Assertions:**
- 각 join마다 전원 수신 `lobby_update.players[].id/nickname/isHost`가 동일 스냅샷으로 정렬 일치.
- 새 참가자 입장 시 `system_message`가 `lobby_update`와 별개로 1회 브로드캐스트.
- `start_game` 직후 신규 join은 `error.code='GAME_ALREADY_STARTED'`로 차단.

---

## L3 참조

- FG-001~FG-010: 모든 풀 게임 시나리오에서 세션 관리 기능 검증
- FG-002: 최대 인원(8명) 스트레스 테스트 (FR-006)
- FG-005: 연결 해제/복구 (FR-007)

---

## L4 체크리스트

> L4 subagent가 TUI 플레이 중 아래 항목에 해당하는 상황을 만나면 **반드시 검증하고** PlayExperienceReport의 `checklist` 필드에 기록한다.
> 자유 플레이를 유지하되, 체크리스트 항목을 의도적으로 커버하는 행동을 포함한다.

| 원본 ID | 체크 항목 | TUI 검증 방법 | Pass 기준 |
|---------|---------|--------------|----------|
| L1-SM-001 | 게임 호스팅 후 룸 코드 표시 | `npx story host` 실행 후 TUI 화면 확인 | 룸 코드(`XXXX-0000` 형식)가 화면에 강조 스타일로 표시됨 |
| L1-SM-002 | 참가 후 닉네임 입력 프롬프트 | `npx story join <code>` 실행 | 닉네임 입력 프롬프트가 표시되고, 입력 후 로비 진입 확인 |
| L1-SM-005 | 닉네임 유효성 오류 메시지 | 빈 문자열 또는 21자 이상 닉네임 입력 | 오류 메시지가 화면에 표시되고 허용 규칙이 안내됨 |
| L1-SM-007 | 로비 플레이어 목록 실시간 갱신 | 새 플레이어 참가 관찰 | 로비 화면에 참가자 목록이 즉시 갱신됨 |
| L1-SM-012 | 연결 해제 알림 표시 | 플레이어 연결 끊김 발생 시 | 시스템 메시지로 `[닉네임] 연결 끊김` 알림이 표시됨 |
| L1-SM-015 | API 키 검증 실패 메시지 | 잘못된 API 키 입력 | 검증 실패 오류 메시지가 화면에 표시됨 |
| L1-SM-023 | API 키 입력 프롬프트 (첫 실행) | 첫 실행 시 화면 확인 | API 키 입력 프롬프트가 마스킹되어 표시됨 |
| L1-SM-027 | 로비 정원 표기 | 로비 화면 헤더 확인 | `N/M명 접속 중` 형태의 정원 표기가 헤더에 표시됨 |
