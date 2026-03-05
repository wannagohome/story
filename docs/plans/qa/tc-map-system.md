# L1: Map System (맵 시스템)

**Traces**: FR-010, FR-031, FR-032, FR-034, FR-035 (※ FR-033은 PRD에 미정의)
**Layer**: L1 Command Agent
**인터랙션**: Protocol level
**AI 사용**: 없음 (fixture AI)

---

## L1 시나리오

| ID | 시나리오 | 사전조건 | 에이전트 행동 | 검증 (필드 레벨) | 관련 FR AC |
|----|---------|---------|-------------|------------------|-----------|
| L1-MP-001 | 맵 조회 | 4인 세션 시작 완료, 플레이어 위치 초기화 완료 | `{type:'request_map'}` 전송 | `map_info.map.rooms[].name` 모든 방에서 문자열 존재, `map_info.map.rooms[].playerCount`>=0 및 `map_info.map.rooms[].playerNames[]` 배열 존재, `map_info.map.connections[]` 모든 연결이 유효 roomId 참조, `map_info.map.myRoomId` 존재하고 유효한 방 ID | FR-031 AC1, AC2, AC3 |
| L1-MP-002 | 인접 방 이동 | Agent A 현재 방과 `kitchen` 사이 연결 존재 | `{type:'move', targetRoomId:'kitchen'}` 전송 | `room_changed.room.name==='kitchen'`, `room_changed.room.description` 문자열 존재, `room_changed.room.players[]` 배열 존재, `room_changed.room.npcs[]` 배열 존재 | FR-032 AC1, AC3 |
| L1-MP-003 | 비인접 방 이동 | Agent A 현재 방과 `basement` 사이 직접 연결 없음 | `{type:'move', targetRoomId:'basement'}` 전송 | `error.code==='INVALID_MOVE'`, `error.message`에 이동 불가 사유 포함, `player_state.currentRoomId` 변경 없음 | FR-032 AC1 |
| L1-MP-004 | 이동 전체 알림 | Agent A, B, C 접속 완료 | Agent A가 `{type:'move', targetRoomId:'study'}` 실행 | 모든 에이전트가 `game_event`(`event.type==='player_move'`) 수신, `event.data.playerName` 존재, `event.data.from` 존재, `event.data.to` 존재, `event.visibility.scope==='all'` | FR-032 AC2 |
| L1-MP-005 | 이동 후 방 정보 | Agent A가 NPC 1명 이상, 아이템 1개 이상 있는 방으로 이동 | 이동 완료 직후 서버 응답 대기 | `room_changed.room.description` 문자열 존재(방 분위기/묘사), `room_changed.room.items[]` 배열 존재, `room_changed.room.npcs[]` 배열 존재, `room_changed.room.players[]` 배열 존재 | FR-034 AC1, AC2, AC3 |
| L1-MP-006 | /look 명령 | Agent A 현재 방에 체류 중 | `{type:'request_look'}` 전송 | `room_changed.room.id===player_state.currentRoomId` (서버가 RoomChangedMessage로 응답), `room_changed.room.description` 문자열 존재, `room_changed.room.items[]` 배열 존재, `room_changed.room.npcs[]` 배열 존재, `room_changed.room.players[]` 배열 존재 | FR-034 AC1, AC2, AC3 |
| L1-MP-007 | 맵 크기 검증 (4인) | 플레이어 수 4명으로 새 세션 생성 | `{type:'request_map'}` 전송 | 접속 플레이어 수 4, `map_info.map.rooms.length>=6`, `map_info.map.rooms.length>=playerCount+2`, `map_info.map.rooms.filter(r=>r.type==='private').length>=2` | FR-035 AC1, AC2 |
| L1-MP-008 | 2인 게임 맵 크기 | 플레이어 수 2명으로 새 세션 생성 | `{type:'request_map'}` 전송 | 접속 플레이어 수 2, `map_info.map.rooms.length>=4`, `map_info.map.rooms.length>=playerCount+2` | FR-035 AC1 |
| L1-MP-009 | 6인 게임 맵 크기 | 플레이어 수 6명으로 새 세션 생성 | `{type:'request_map'}` 전송 | 접속 플레이어 수 6, `map_info.map.rooms.length>=8`, `map_info.map.rooms.length>=playerCount+2` | FR-035 AC1 |
| L1-MP-010 | 8인 게임 맵 크기 | 플레이어 수 8명으로 새 세션 생성 | `{type:'request_map'}` 전송 | 접속 플레이어 수 8, `map_info.map.rooms.length>=10`, `map_info.map.rooms.length>=playerCount+2` | FR-035 AC1 |
| L1-MP-011 | 밀담 가능 방 수 | 플레이어 수 6명 이상 세션 생성 | `{type:'request_map'}` 전송 후 `type==='private'` 방 필터링 | `map_info.map.rooms.filter(r=>r.type==='private')` 배열 존재, private 방 수 `>=ceil(playerCount/2)`, private 방의 `id`가 모두 고유, 모든 private 방 `id`가 `map_info.map.rooms[].id`에 존재 | FR-035 AC2 |
| L1-MP-013 | 단방향 연결 이동 제한 | `Bidirectional=false` 연결이 있는 fixture (`roomA -> roomB` 단방향) | Agent A가 roomA에서 roomB로 이동 후, roomB에서 roomA로 이동 시도 | roomA→roomB 이동: `room_changed.room.id==='roomB'` 성공; roomB→roomA 이동: `error.code==='INVALID_MOVE'` 반환; 단방향 연결 동작 보장 | FR-010 AC2 |
| L1-MP-012 | 맵 연결 그래프 전체 도달성 (BFS) | 4인 세션 시작 완료, `map_info` 수신 가능 | `{type:'request_map'}` 전송 후 `map_info.map.rooms[]`와 `map_info.map.connections[]`를 이용해 임의 방에서 BFS 수행 | BFS 방문 room id 집합 크기가 `map_info.map.rooms.length`와 동일, 방문 불가 room id 목록이 빈 배열, 모든 방 쌍 간 경로가 존재(고립된 방 0개) | FR-010 AC4 |

---

## L2 Phase 참조: Phase 2D (Exploration & Actions)

1. **초기 맵 스냅샷 검증**
   - Explorer가 첫 턴에 `{type:'request_map'}` 실행.
   - 검증: `map_info.map.rooms[].name`, `map_info.map.rooms[].playerNames[]`, `map_info.map.myRoomId`.

2. **이동 성공/실패 분기 검증**
   - Explorer가 인접 이동 10회 실행.
   - Chaotic가 비인접 이동 5회 실행.
   - 검증: 성공 이동마다 `room_changed` + `game_event(player_move)` 발생, 실패 이동마다 `error.code==='INVALID_MOVE'`.

3. **이동 후 방 정보 완전성 검증**
   - 모든 성공 이동 응답에서 `room_changed.room.description`, `room_changed.room.players[]`, `room_changed.room.npcs[]` 확인.
   - `/look` 실행 시 `room_changed.room.description`, `room_changed.room.items[]`, `room_changed.room.npcs[]`, `room_changed.room.players[]` 재확인.

4. **맵 규모/밀담 수용력 검증**
   - 2인/4인/6인/8인 세션 각각 1회 생성 후 `request_map` 실행.
   - 검증: `rooms.length >= playerCount + 2`, `privateRooms.length >= ceil(playerCount/2)`.

---

## L3 참조

- FG-001~FG-010: 모든 풀 게임 시나리오에서 맵 이동 검증
- FG-002: 최대 인원(8명) 맵 크기 적절성
- FG-003: 최소 인원(2명) 맵 크기 적절성

---

## L4 체크리스트

> L4 subagent가 TUI 플레이 중 아래 항목에 해당하는 상황을 만나면 **반드시 검증하고** PlayExperienceReport의 `checklist` 필드에 기록한다.
> 자유 플레이를 유지하되, 체크리스트 항목을 의도적으로 커버하는 행동을 포함한다.

| 원본 ID | 체크 항목 | TUI 검증 방법 | Pass 기준 |
|---------|---------|--------------|----------|
| L1-MP-001 | 맵 조회 결과 표시 | `/map` 명령 실행 | 방 이름, 현재 위치 강조, 연결 정보가 화면에 표시됨 |
| L1-MP-002 | 인접 방 이동 성공 | `/move <방이름>` 실행 | 이동 후 새 방 설명과 NPC/플레이어 목록이 표시됨 |
| L1-MP-003 | 비인접 방 이동 실패 | 연결되지 않은 방으로 이동 시도 | 이동 불가 오류 메시지가 표시됨 |
| L1-MP-004 | 이동 알림 메시지 | 다른 플레이어 이동 시 관찰 | `[닉네임]이(가) X에서 Y으로 이동했습니다` 메시지 표시 |
| L1-MP-006 | /look 명령 결과 | `/look` 실행 | 현재 방의 분위기, 아이템, NPC, 플레이어 정보가 표시됨 |
