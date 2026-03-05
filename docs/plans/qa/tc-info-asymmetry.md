# L1: Information Asymmetry (정보 비대칭)

**Traces**: FR-050, FR-051, FR-052, FR-053, FR-054
**Layer**: L1 Command Agent
**인터랙션**: Protocol level
**AI 사용**: 없음 (fixture AI)

---

## L1 시나리오

| ID | 시나리오 | 사전조건 | 에이전트 행동 | 검증 (필드 레벨) | 관련 FR AC |
|----|---------|---------|-------------|------------------|------------|
| L1-IA-001 | 공개 브리핑 필드 완전성 + 동일성 | 4 에이전트(A/B/C/D) 입장 완료, 게임 시작 직후 브리핑 단계 | 각 에이전트가 `briefing_public` 수신 payload 저장 | 4명 `briefing_public` 원문 바이트 동일, payload에 `info.synopsis`, `info.characterList[]`, `info.relationships`, `info.mapOverview`, `info.npcList[]` 필드 존재, `info.npcList[]` 각 엔트리에 `name`, `location` 필드 존재 | FR-050 AC1, AC2, AC3 |
| L1-IA-002 | 비공개 브리핑 고유성 + 필수 필드 | 4 에이전트 브리핑 수신 상태 | 각 에이전트의 `briefing_private` 저장 후 비교 | 각 `briefing_private`가 플레이어별 고유 값, 각 payload에 `role.personalGoals[]`, `role.secret`, `role.specialRole` 필드 존재, `secrets[]` 배열 존재 | FR-052 AC1 |
| L1-IA-003 | 위치 정보 전체 공개 + 헤더 실시간 반영 | A/B/C/D가 서로 다른 방으로 분산 가능 상태, TUIAgent 관찰 활성 | Agent A가 인접 방으로 이동 1회 수행 | 프로토콜: 전원에게 `player_move` 전달, 맵/위치 데이터 동기화. TUI 참조: `tc-terminal-ui.md`의 `tuiEvaluation.headerInfoPresent` 체크와 FG-007 스냅샷에서 상단 방별 인원 표시가 이동 직후 갱신됨 | FR-053 AC1, AC2, AC3 |
| L1-IA-004 | 방 채팅 격리 (서버 필터 + 클라이언트 미렌더링) | Agent A/B=거실, Agent C=부엌 | Agent A가 일반 채팅 1회 전송 | 서버: 수신 대상이 거실 플레이어 집합으로 제한됨. 프로토콜: Agent C 수신 로그에 해당 chat 이벤트 0건. TUI 참조: FG-007 stdout 캡처에서 Agent C 화면에 해당 메시지 텍스트 미출력 | FR-054 AC1, AC2, AC3 |
| L1-IA-005 | 비공개 정보 네트워크 격리 | Agent A의 `briefing_private.role.secret` 값이 고유 문자열로 설정됨 | Agent B/C/D의 전체 수신 메시지 덤프 검색 | Agent B/C/D 메시지에 Agent A의 `role.secret` 문자열 0회 출현, Agent A 단말에서만 해당 `secret` 확인 가능 | FR-052 AC2, AC3 |
| L1-IA-006 | 반공개 정보 그룹 정의 + 비수신 검증 | fixture에 반공개 항목 1개 이상 존재, 공유 그룹은 `[AgentA, AgentB]` | 모든 에이전트 `briefing_private` payload 수집 | `semiPublicInfo[]` 각 엔트리에 `targetPlayerIds: string[]` 필드 존재, `content` 필드 존재, `targetPlayerIds`에 없는 Agent C/D에게 해당 엔트리 payload 미전달 | FR-051 AC1, AC2, AC3 |
| L1-IA-007 | 재접속 후 비공개 정보 유지 | Agent A가 브리핑 완료 후 연결 끊김 가능 상태 | Agent A disconnect 후 재접속 | 복구된 `briefing_private`가 disconnect 전 저장본과 바이트 동일, `personalGoals[]`, `secret`, `specialRole` 값 변형 없음 | FR-052 AC1, AC2 |
| L1-IA-008 | DM/귓속말 프로토콜 거부 | 기본 채팅은 room/global만 허용, A/B 같은 방 | Agent A가 `{type:'dm', targetPlayerId:'AgentB', message:'비밀'}` 전송 | `error.code='NOT_SUPPORTED'` 반환, Agent B 단말에 DM 메시지 미도착, 서버가 point-to-point 채널 생성하지 않음 | FR-054 AC1, AC2 |
| L1-IA-009 | NPC 대화 방 범위 제한 | Agent A/B는 NPC와 같은 방, Agent C는 다른 방 | Agent A가 `{type:'talk', npcId:'raymond', message:'열쇠가 있나?'}` 전송 | Agent A/B가 `npc_dialogue` 수신, Agent C 수신 로그에 `npc_dialogue` 0건, Agent C TUI 화면에 NPC 발화 텍스트 미출력 | FR-054 AC2, AC3 |
| L1-IA-010 | 위치 변경 push 이벤트 (요청 없이 수신) | Agent A/B/C/D가 각자 방에 위치, `/map` 요청 없이 이벤트 수신 대기 | Agent A가 `{type:'move', targetRoomId:'kitchen'}` 전송 후 B/C/D는 어떠한 요청도 하지 않음 | B/C/D가 요청 없이 `game_event`(`event.type==='player_move'`)를 push로 수신(`event.data.playerId===AgentA`, `event.data.to` 존재), A의 이동 전 방 플레이어에게 `player_left_room` 수신(`playerId===AgentA`, `destination` 존재), A의 이동 후 방(kitchen) 플레이어에게 `player_joined_room` 수신(`playerId===AgentA`), B/C/D가 `/map` 미요청 상태에서도 위치 정보가 갱신됨 | FR-053 AC2, AC3 |
| L1-IA-011 | `briefing_private` 전원 ready 후 수신 | 4 에이전트(A/B/C/D) 브리핑 단계 진입, `briefing_public` 수신 완료 | Agent A만 `{type:'ready', phase:'briefing_read'}` 전송; B/C/D는 아직 미전송 | Agent A가 `ready` 전송 직후 `briefing_private` 미수신 확인(`briefing_private` 수신 이벤트 0건), B/C/D도 순차적으로 `{type:'ready', phase:'briefing_read'}` 전송 후 전원이 `{type:'ready', phase:'game_ready'}` 전송, 마지막 플레이어(D)의 `game_ready` 전송 직후 A/B/C/D 전원에게 `briefing_private` 수신(`briefing_private.role` 필드 존재), 전원 수신 타임스탬프가 마지막 `ready` 전송 이후임을 확인 | FR-078 AC2 |
| L1-IA-012 | 전원 ready 이전 `request_role` 전송 시 오류 | 4 에이전트 브리핑 단계, Agent A만 `{type:'ready', phase:'briefing_read'}` 전송 완료, B/C/D는 미전송 상태 | Agent A가 `{type:'request_role'}` 전송 | Agent A가 `error` 수신, `error.code==='BRIEFING_NOT_COMPLETE'`; `role_info` 또는 `briefing_private` 미수신; 다른 에이전트(B/C/D)에게 관련 이벤트 미전송; 서버 상태 `briefing` 유지 | FR-078 AC2 |
| L1-IA-013 | Agent가 `ready`를 2회 전송 시 중복 브리핑 미발송 | 4 에이전트 브리핑 단계, 전원이 `ready` 전송하여 `briefing_private` 수신 완료 | Agent A가 `{type:'ready'}`를 추가로 1회 더 전송 | Agent A에게 `briefing_private` 추가 수신 0건(총 1건만 수신); 다른 에이전트(B/C/D)에게 추가 `briefing_private` 미전송; 서버 상태가 이미 `briefing` 이후 단계(`playing` 또는 `game_started`)를 유지하며 회귀하지 않음 | FR-078 AC2 |

---

## NFR 임베딩

| 검증 항목 | 방식 | 기준 |
|----------|------|------|
| API 키 보호 | 모든 메시지에 API 키 미포함 | 0건 (NFR-013) |

---

## L2 Phase 참조: Phase 2C (Briefing & Information Asymmetry)

4명 에이전트가 briefing 수신 → 교차 검증:

**교차 검증 프로토콜:**
1. 모든 에이전트의 `briefing_public` 메시지 수집 후 바이트 동일 확인 (FR-050 AC3)
2. `briefing_public` 필수 필드 `info.synopsis`, `info.characterList[]`, `info.relationships`, `info.mapOverview`, `info.npcList[]` 존재 확인 (FR-050 AC1)
3. `briefing_public.info.npcList[]` 각 엔트리의 `name`, `location` 필드 존재 확인 (FR-050 AC2)
4. 모든 에이전트의 `briefing_private` 메시지 수집 후 플레이어별 고유성 확인 (FR-052)
5. `briefing_private` 필수 필드 `role.personalGoals[]`, `role.secret`, `role.specialRole` 존재 확인 (FR-052 AC1)
6. 반공개 정보 엔트리에 `targetPlayerIds`, `content` 필드 존재 확인 후 비그룹 미수신 검증 (FR-051 AC1, AC2, AC3)
7. Agent A 비공개 문자열이 Agent B/C/D 메시지 전체 덤프에 0건인지 확인 (FR-052 AC2, AC3)
8. 위치 이동 시 프로토콜 갱신과 TUI 헤더 갱신을 함께 캡처 (FR-053 AC2)
9. 같은 방 채팅/NPC 대화가 다른 방에서 미렌더링되는지 TUI stdout diff로 확인 (FR-054 AC3)
10. 모든 에이전트 `ready` 전송 후 `game_started` 수신 (FR-078 AC4)

---

## L3 참조

- FG-006: 정보 비대칭 집중 (Observer×4, 전체 메시지 교차 검증, 정보 누출 0건)

---

## L4 체크리스트

> L4 subagent가 TUI 플레이 중 아래 항목에 해당하는 상황을 만나면 **반드시 검증하고** PlayExperienceReport의 `checklist` 필드에 기록한다.
> 자유 플레이를 유지하되, 체크리스트 항목을 의도적으로 커버하는 행동을 포함한다.

| 원본 ID | 체크 항목 | TUI 검증 방법 | Pass 기준 |
|---------|---------|--------------|----------|
| L1-IA-003 | 위치 정보 실시간 반영 | 이동 후 헤더/정보패널 확인 | 이동 직후 방별 인원 표시가 즉시 갱신됨 |
| L1-IA-004 | 방 채팅 격리 확인 | 다른 방 플레이어의 채팅 관찰 | 다른 방에서 보낸 메시지가 자신의 화면에 표시되지 않음 |
| L1-IA-009 | NPC 대화 방 범위 제한 | 다른 방 NPC 대화 결과 관찰 | 다른 방에서 이루어진 NPC 대화가 자신의 화면에 미표시 |
