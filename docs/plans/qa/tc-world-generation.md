# L2: World Generation (세계 생성 구조 검증)

**Traces**: FR-009, FR-010, FR-011, FR-012, FR-013, FR-014, FR-015, FR-016, FR-017, FR-018, FR-019, FR-035, FR-092, FR-093
**Layer**: L2 Phase Agent (Phase 2B)
**인터랙션**: Protocol level
**AI 사용**: 실제 AI (세계 생성)
> 프로토콜 메시지 타입과 필드명은 /docs/plans/system-design/shared/protocol.md 및 /docs/plans/system-design/shared/events.md 기준

---

## Phase 2B: World Generation & Verification

```
게임 시작
  -> generation_progress 동기 수신
  -> 세계 생성 완료
  -> 생성 JSON 캡처
  -> 스키마/구조 검증
```

### 구조 검증 Assertions

| ID | 시나리오 | 사전조건 | 에이전트 행동 | 검증 (필드 레벨) | 관련 FR AC |
|----|---------|---------|-------------|------------------|------------|
| L2-WG-001 | 세계 자동 생성 기본 스키마 | Host + 3명(총 4명) 로비에서 `start_game` 직후, 세계 생성 완료 | 서버 내부 `worldGeneration` 결과 JSON 직접 검사 | `worldGeneration` 최상위 키 집합이 정확히 `{meta, world, gameStructure, map, characters, information, clues, gimmicks}`와 일치, `meta.theme`/`meta.setting` 문자열 non-empty, `meta.hasGM`/`meta.hasNPC` boolean | FR-009 AC1, AC4 |
| L2-WG-002 | 세계관/사건/분위기 필드 생성 | L2-WG-001 통과 | `worldGeneration.world` 검사 | `world.title`, `world.synopsis`, `world.atmosphere`가 모두 `trim().length > 0` 문자열, `information.public.synopsis`도 non-empty | FR-009 AC2 |
| L2-WG-003 | 게임 구조 동시 설계 | L2-WG-001 통과 | `worldGeneration.gameStructure` 검사 | `gameStructure.concept`, `coreConflict`, `progressionStyle`, `briefingText` non-empty, `endConditions.length >= 1`, `winConditions.length >= 1` | FR-009 AC3, FR-014 AC2 |
| L2-WG-004 | 플레이어 수 기반 맵 크기 동적 결정 | 동일 테스트를 2인/4인/6인/8인 세션으로 반복 실행 | 각 세션에서 서버 내부 `worldGeneration` 결과 검사 | 각 세션마다 `map.rooms.length >= playerCount + 2`, 2인 세션 `>=4`, 4인 세션 `>=6`, 6인 세션 `>=8`, 8인 세션 `>=10` | FR-010 AC1, FR-035 AC1 |
| L2-WG-005 | 각 방 필수 필드 + 연결성 보장 | L2-WG-001 통과 | `map.rooms[]`, `map.connections[]` 검증 | 모든 방에 `id`, `name`, `description` non-empty, `map.connections[]`에서 각 방의 연결 수를 계산하여 `>= 1`, `map.connections[]`의 양쪽 방 id가 `map.rooms[].id`에 존재 | FR-010 AC2 |
| L2-WG-005a | 방 이름 고유성 검증 | L2-WG-005 통과 | `map.rooms[]` 순회하여 `name` 필드 중복 검사 | 모든 `rooms[].name` 값이 고유함, 중복 이름 0건, `uniqueNames.length === map.rooms.length` | FR-010 AC2 |
| L2-WG-006 | 공개 공간/밀실 구분 + 동시 밀담 가능 수 | playerCount=6 세션 생성 완료 | `map.rooms[]`의 `type` 분포 검사 | `map.rooms`에 `type==='public'` 최소 1개, `type==='private'` 최소 1개, `private` 방 수가 `ceil(playerCount/2)` 이상 | FR-010 AC3, FR-035 AC2 |
| L2-WG-007 | 맵 연결 그래프 고립 방 없음 | L2-WG-001 통과 | 첫 방에서 BFS 수행 | BFS 방문한 room id 집합 크기가 `map.rooms.length`와 동일, 방문 불가 room id 목록이 빈 배열 | FR-010 AC4 |
| L2-WG-008 | 플레이어 역할 기본 필드 완전성 | L2-WG-001 통과, playerCount=4 | `characters.playerRoles[]` 순회 검사 | `playerRoles.length === 4`, 각 role에 `characterName`, `background`, `secret` non-empty, `personalGoals.length >= 1`, 각 `personalGoals[*].id`/`description` non-empty | FR-011 AC1 |
| L2-WG-009 | 역할 간 충돌/협력 관계 포함 | L2-WG-008 통과 | role별 `relationships[]` 집계 | 전체 role의 `relationships[]` 총합 `>= playerCount`, 최소 1개 관계 설명에 협력 의도 키워드(예: `협력`, `도움`) 존재, 최소 1개 관계 설명에 갈등 키워드(예: `의심`, `경쟁`) 존재 | FR-011 AC2 |
| L2-WG-010 | 게임 구조별 역할 성격 다양성 (다중 실행) | 동일 playerCount=4로 world generation 10회 수행 가능 | 동일 입력 조건으로 10회 생성 후 role 프로필 해시 비교 | 10회 중 최소 2회 이상에서 동일 슬롯 role의 `background + personalGoals` 조합 해시가 서로 다름, 단일 고정 템플릿 반복률 100%가 아님 | FR-011 AC3 |
| L2-WG-011 | 개인 목표 독립성/고유 동기 | L2-WG-008 통과 | `gameStructure.commonGoal`, `playerRoles[*].personalGoals[]` 비교 | 모든 플레이어의 개인 목표 문자열이 `commonGoal`과 완전 동일하지 않음, 플레이어 간 개인 목표 설명 완전 중복률 100% 케이스 없음 | FR-011 AC4, FR-092 AC1, AC2 |
| L2-WG-012 | 개인 목표 충돌 관계 탐지 | L2-WG-008 통과 | goal conflict detector 실행 | `goalConflictPairs.length >= 1`, 각 pair는 서로 다른 `playerId`를 가지며 `conflictReason` non-empty | FR-092 AC3, AC4 |
| L2-WG-013 | 엔딩 개인 목표 평가 가능 데이터 포함 | L2-WG-008 통과 | ending preflight validator 실행 | 각 `playerRoles[*].personalGoals[*].id`가 고유, `gameStructure.winConditions[*].evaluationCriteria` non-empty, `ending_precheck.missingGoalIds.length === 0` | FR-092 AC5 |
| L2-WG-014 | 단서 최소 개수 보장 | playerCount=4 세션 생성 완료 | `clues[]` 길이 검사 | `clues.length >= playerCount * 2` | FR-012 AC4 |
| L2-WG-015 | 단서 방 배치 + 발견 조건 명시 | L2-WG-014 통과 | `clues[]` 순회 검사 | 각 `clue.id`, `clue.roomId`, `clue.discoverCondition` non-empty, 모든 `clue.roomId`가 `map.rooms[].id`에 존재 | FR-012 AC1, AC2 |
| L2-WG-016 | 복선 회수 경로 존재 | L2-WG-014 통과 | 각 단서의 `relatedClueIds` 그래프 경로 확인 | `relatedClueIds`가 설정된 각 단서에서 시작한 경로의 최종 노드가 `gameStructure.endConditions[].id` 중 하나와 일치, `unrecoverableForeshadowingIds.length === 0` | FR-012 AC3 |
| L2-WG-017 | 공개 정보 필드 완전성 (관계도 포함) | L2-WG-001 통과 | `information.public` 검사 | `public.title`, `public.synopsis`, `public.mapOverview`, `public.gameRules` non-empty, `public.characterList.length >= playerCount`, `public.relationships` non-empty 문자열 | FR-013 AC1 |
| L2-WG-018 | 반공개 정보 분배 + 2인 이상 공유쌍 | L2-WG-001 통과 | `information.semiPublic[]` 검사 | `semiPublic.length >= 1`, 최소 1개 엔트리에서 `targetPlayerIds.length >= 2`, 모든 `targetPlayerIds`가 실제 player id 집합에 포함, `content` non-empty | FR-013 AC2, AC4 |
| L2-WG-019 | 비공개 정보 1:1 전달 구조 | L2-WG-001 통과 | `information.private[]` 검사 | `private.length === playerCount`, 각 엔트리의 `playerId`가 유일, 각 `additionalSecrets`는 배열 타입(빈 배열 허용) | FR-013 AC3 |
| L2-WG-020 | 게임 구조 자유 설계 + 명확한 종료/승리 판정 | L2-WG-003 통과 | `gameStructure` 텍스트/스키마 검사 | `gameStructure.concept`가 고정 모드 문자열 하나로만 구성되지 않음, `endConditions[*].description` non-empty, `winConditions[*].description`/`evaluationCriteria` non-empty | FR-014 AC1, AC3 |
| L2-WG-021 | GM 필요 여부 + 행동 원칙 필드 | 두 세션 준비: A(`hasGM=true`) / B(`hasGM=false`) | 각 세션 `worldGeneration` 결과 비교 | A 세션에서 `meta.hasGM===true`이고 `meta.gmProfile.persona`/`meta.gmProfile.behaviorPrinciple` non-empty, B 세션에서 `meta.hasGM===false`이고 `meta.gmProfile` 미포함 | FR-015 AC1, AC2, AC3 |
| L2-WG-022 | NPC 필요 시 NPC 필드 생성 | `meta.hasNPC===true` 세션 | `characters.npcs[]` 순회 검사 | `npcs.length >= 1`, 각 NPC에 `id`, `name`, `currentRoomId`, `persona`, `behaviorPrinciple` non-empty, `knownInfo`/`hiddenInfo` 배열 존재, 각 NPC의 `gimmick`은 선택 필드(null/미존재 허용) — 존재할 경우 `gimmick.description`/`gimmick.triggerCondition`/`gimmick.effect` non-empty | FR-016 AC1, AC2 |
| L2-WG-023 | NPC 없는 세계 정상 진행 | `meta.hasNPC===false` 세션 | 생성 완료 후 브리핑 시작까지 진행 | `characters.npcs.length === 0`, `information.public.npcList.length === 0`, 세션 상태가 `generating -> briefing -> playing`으로 정상 전이, `error` 메시지 0건 | FR-016 AC3 |
| L2-WG-024 | 세계 생성 중 generation_progress 메시지 수신 | Host + Joiner 2명 접속, `start_game` 직후 | 모든 에이전트가 최초 5개 생성 이벤트 캡처 | 수신 메시지 타입이 `generation_progress`, 각 메시지에 `step` non-empty 문자열과 `progress` 숫자 필드 존재, `progress`가 역행하지 않음 | FR-017 AC1 |
| L2-WG-025 | 진행률 표시 | L2-WG-024 통과 | Host/Joiner TUI 스냅샷 캡처 | 각 클라이언트 UI에 `progressPercent` 정수(0~100) **또는 `estimatedRemainingTime`** 이 표시됨 — 두 필드 중 하나 이상이 non-empty면 진행 상황을 나타내는 것으로 유효; 마지막 생성 메시지에서 `progress===1.0` 수신 후 로딩 UI 종료 | FR-017 AC2 |
| L2-WG-026 | 모든 참가자 동일 진행 정보 | Host + Joiner 3명 동시 수신 로그 저장 | generation 단계 전체 로그 해시 비교 | 참가자별 `generation_progress` 시퀀스의 `(step, progress)` 튜플 해시가 전원 동일, 누락/추가 프레임 없는지 `diffCount===0` | FR-017 AC3 |
| L2-WG-027 | 스키마 외 필드 포함 시 거부 | AI 응답에 `world.unexpectedField` 강제 주입 가능한 fixture 준비 | 변조된 JSON으로 validator 실행 | `schema_validation_failed` 발생, `error.code==='WORLD_SCHEMA_UNKNOWN_FIELD'`, `error.details.path`에 `world.unexpectedField` 포함, 해당 결과 채택되지 않음 | FR-018 AC1 |
| L2-WG-028 | 필수 필드 누락 시 재생성 요청 | AI 응답에서 `world.synopsis` 제거한 fixture 준비 | validator 실행 후 재생성 사이클 관찰 | 1차 검증 실패 시 `regeneration_requested.reason==='MISSING_REQUIRED_FIELD'`, `reason.path==='world.synopsis'`, 2차 생성 요청이 즉시 발행 | FR-018 AC2 |
| L2-WG-029 | 파싱 실패 시 최대 3회 재시도 | AI 응답을 연속 3회 malformed JSON으로 주입 | 생성 파이프라인 실행 | `parse_retry` 이벤트가 정확히 3회(`attempt` 값 1,2,3) 기록, 4번째 파싱 시도는 발생하지 않음 | FR-018 AC3 |
| L2-WG-030 | 재시도 실패 시 오류 후 세션 종료 | L2-WG-029와 동일(3회 모두 실패) | 3회 실패 후 세션 상태 관찰 | Host/참가자 모두 `error.code==='WORLD_GENERATION_PARSE_FAILED'` 수신, 서버가 `game_cancelled.reason`에 `WORLD_GENERATION_FAILED` 포함하여 전송, 상태가 `generating -> finished` 전이 | FR-018 AC4 |
| L2-WG-031 | 종료 조건 스키마 + timeout fallback 포함 | L2-WG-003 통과 | `gameStructure.endConditions[]` 검사 | 각 endCondition에 `id`, `description`, `triggerType`, `triggerCriteria`, `isFallback` 존재, `isFallback===true`인 항목이 정확히 1개 이상이고 해당 `triggerType==='timeout'` | FR-019 AC1, AC2 |
| L2-WG-032 | requiredSystems와 종료 조건의 시스템 요구 일치 | L2-WG-031 통과 | `endConditions.triggerType`에서 필요 시스템 집합 계산 후 비교 | `triggerType==='vote'`가 있으면 `requiredSystems`에 `vote` 포함, `consensus`면 `consensus` 포함, `ai_judgment`면 `ai_judge` 포함, `missingRequiredSystems.length===0` | FR-019 AC3, FR-093 AC4 |
| L2-WG-033 | timeout fallback 제거 불가 검증 | L2-WG-001 통과, AI 응답에서 `isFallback===true` 항목을 제거한 fixture 준비 | 변조된 JSON(timeout fallback 없음)으로 validator 실행 | `schema_validation_failed` 발생, `error.code==='MISSING_TIMEOUT_FALLBACK'`, 해당 결과 채택되지 않음, `regeneration_requested` 발행, 정상 재생성 후 `endConditions`에 `isFallback===true && triggerType==='timeout'` 항목이 복원됨 | FR-019 AC2, FR-027 AC4 |

---

### 오케스트레이션 검증 Assertions

| ID | 시나리오 | 사전조건 | 에이전트 행동 | 검증 (필드 레벨) | 관련 FR AC |
|----|---------|---------|-------------|------------------|------------|
| L2-WG-034 | 멀티 모델 seed 병렬 생성 | Host + 3명(총 4명) 로비에서 `start_game` 직후, Orchestrator 로그 캡처 | Orchestrator 내부 로그에서 seed 생성 단계 확인 | seed 3개 이상 생성, 각 seed의 `provider` 필드가 서로 다른 프로바이더 이름, 각 seed에 `hook`/`genre`/`coreConflict` non-empty | FR-009 AC1 |
| L2-WG-035 | seed 선택 점수 함수 동작 | L2-WG-034 통과, seed 3개 이상 생성 완료 | Orchestrator 내부 `SeedScore` 로그 확인 | 각 seed에 `total` 점수 존재, `total === fun*0.45 + socialTension*0.2 + secretConflict*0.15 + movementDriver*0.1 + solvability*0.05 + schemaRepairCost*0.05` (허용 오차 0.01), 최고 점수 seed가 Showrunner에 전달됨 | FR-009 AC1 |
| L2-WG-036 | 개별 Muse 타임아웃 시 graceful degradation | Orchestrator에 1개 Muse 프로바이더를 10초 이상 지연하도록 mock 주입 | seed 생성 완료까지 대기 | 타임아웃 seed는 `status==='timeout'`으로 기록, 나머지 seed로 정상 진행, 최종 `worldGeneration` 결과가 유효한 스키마 통과 | FR-009 AC1, FR-018 AC3 |
| L2-WG-037 | 모든 Muse 타임아웃 시 단일 모델 fallback | 모든 Muse 프로바이더를 10초 이상 지연하도록 mock 주입 | 세계 생성 완료까지 대기 | `fallback_triggered` 이벤트 발생, `fallback.provider==='openai'`, 최종 `worldGeneration` 결과가 기존 스키마 검증(L2-WG-001~033) 전부 통과 | FR-009 AC1, FR-018 AC3 |
| L2-WG-038 | 품질 모드 전환 | 동일 playerCount=4로 `qualityMode=fast` / `qualityMode=premium` 각각 세계 생성 | Orchestrator 내부 파이프라인 로그 비교 | `fast` 모드에서 직렬 단계 수 ≤ 4, `premium` 모드에서 직렬 단계 수 ≥ 6, `premium`에서 ConflictEngineer/CastSecretMaster/MapClueSmith 단계가 독립 존재, `fast`에서 해당 3단계가 Showrunner에 통합 | FR-009 AC1 |
| L2-WG-039 | 프로바이더 health check | 등록된 5개 프로바이더 중 2개를 의도적으로 불가 상태로 설정 | `ProviderRegistry.HealthCheck()` 실행 후 Orchestrator 편성 확인 | `healthCheck` 결과에서 불가 프로바이더 2개가 `false`, 가용 프로바이더 3개가 `true`, Orchestrator 편성에서 불가 프로바이더가 어떤 역할에도 배정되지 않음, 대체 프로바이더로 편성 완료 | FR-009 AC1 |
| L2-WG-040 | Story Bible 캐시 적중 | 동일 concept.md + prd.md로 세계 생성 2회 실행 | 1회차/2회차 StoryBible 로그 비교 | 1회차에서 `bible_cache_miss` + `bible_generated` 이벤트 발생, 2회차에서 `bible_cache_hit` 이벤트 발생 + `bible_generated` 미발생, 1회차/2회차 Bible 내용 해시 동일 | — (비용 최적화) |

---

## L1 참조

- L1-SM-008: `start_game` 직후 `generation_progress(step, progress)` 첫 메시지 수신
- L1-GS-001~L1-GS-004: `gameStructure`, `endConditions`, `requiredSystems` 필드 단위 확인

---

## L3 참조

- FG-*: 모든 풀 게임 시나리오에서 실제 AI 세계 생성 동작 검증
- FG-002: 8명 최대 인원 스트레스 테스트 시 맵 크기 동적 결정
- FG-003: 2인 최소 인원 시 세계 적절성

---

## L4 체크리스트

> L4 subagent가 TUI 플레이 중 아래 항목에 해당하는 상황을 만나면 **반드시 검증하고** PlayExperienceReport의 `checklist` 필드에 기록한다.
> 자유 플레이를 유지하되, 체크리스트 항목을 의도적으로 커버하는 행동을 포함한다.

| 원본 ID | 체크 항목 | TUI 검증 방법 | Pass 기준 |
|---------|---------|--------------|----------|
| L2-WG-024 | 세계 생성 진행률 표시 | 게임 시작 직후 화면 관찰 | 생성 진행률(%)이 화면에 표시되고 역행하지 않음 |
| L2-WG-025 | 생성 완료 후 로딩 종료 | 세계 생성 완료 시 | progress=1.0 도달 후 로딩 UI가 사라지고 브리핑 화면으로 진입 |
