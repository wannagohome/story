# L1: NPC System (NPC 시스템)

**Traces**: FR-055, FR-056, FR-057, FR-058, FR-043 (NPC 대화 공개 범위), FR-016 (NPC 없는 게임 진행)
**Layer**: L1 Command Agent
**인터랙션**: Protocol level
**AI 사용**: 없음 (fixture AI)

---

## L1 시나리오

| ID | 시나리오 | 사전조건 | 에이전트 행동 | 검증 (필드 레벨) | 관련 FR AC |
|----|---------|---------|-------------|------------------|------------|
| L1-NP-001 | NPC 퍼소나 일관성 | `hasNPC=true`, `raymond.persona='충직한 집사'`, Agent A/B는 같은 방(거실) | Agent A가 `talk` 3회 전송 (`인사` -> `사건 질문` -> `확인 질문`) | `npc_dialogue` 3건 수신, 각 이벤트 `data.npcId='raymond'`, `data.text` 비어있지 않음, 3개 응답 모두 반말/무례 토큰(`야`,`너`,`닥쳐`,`ㅋㅋ`) 미포함, 3개 응답 모두 집사 톤 키워드(`주인님`,`알겠습니다`,`실례`) 중 1개 이상 포함 | FR-055 AC1 |
| L1-NP-002 | NPC 정보 비공개 유지 | `raymond.hiddenInfo=['본인이 현장에 있었다']`, 첫 대화 단계, 신뢰도 `trustLevel=0.3` | Agent A가 `{type:'talk', npcId:'raymond', message:'당신이 현장에 있었나요?'}` 전송 | `npc_dialogue.data.text`에 `본인이 현장` 문구 미포함, `npc_dialogue.data.text`에 회피 응답 문구 포함, 같은 턴에 `npc_reveal` 0건, `npc_give_item` 0건 | FR-056 AC2 |
| L1-NP-003 | NPC 기믹 트리거 *(Priority: P1 — MVP 이후)* | `raymond.gimmick.triggerCondition='trustLevel>=0.7 && dialogueProgress>=2'`, Agent A 인벤토리에 `letter` 존재 | 1) Agent A가 신뢰 대화 2회 수행 2) Agent A가 `{type:'give', npcId:'raymond', itemId:'letter'}` 전송 | 조건 미충족 구간에서 `npc_give_item` 0건, 조건 충족 직후 `npc_give_item` 1건, `npc_give_item.data.npcId='raymond'`, `npc_give_item.data.item.id='master_key'`, 같은 세션 로그에 구조화 이벤트 타입 문자열이 `npc_give_item`으로 고정 | FR-057 AC1, AC2 |
| L1-NP-004 | NPC 대화 같은 방 공개 | Agent A/B는 거실, Agent C는 부엌, `raymond`는 거실 | Agent A가 `{type:'talk', npcId:'raymond', message:'어젯밤 무슨 일 있었나요?'}` 전송 | Agent A/B가 동일 `npc_dialogue.id` 수신, Agent C 수신 로그의 `npc_dialogue` 건수 `0`, 수신 이벤트 `visibility.scope='room'`, `visibility.roomId='living_room'` | FR-043 AC3 |
| L1-NP-005 | 동시 NPC 대화 | Agent A/B가 같은 틱에서 `raymond`에게 대화 가능, 이벤트 시퀀스 기록 활성화 | Agent A/B가 같은 tick에 각각 `talk` 1회 전송 | `game_event`(`event.type==='npc_dialogue'`) 2건 모두 생성, 각 이벤트 `event.data.playerId`가 요청자와 일치, `event.data.playerName`이 요청자 닉네임과 일치, 서버 이벤트 시퀀스 번호가 단조 증가, 두 응답 모두 상대 요청 텍스트를 섞지 않음 | FR-043 AC2, AC4 |
| L1-NP-006 | NPC 대화 이력 유지 (연속 대화) | Agent A와 `raymond` 같은 방, 대화 히스토리 저장 디버그 훅 활성화 | Agent A가 `talk` 2회 전송 (`열쇠를 본 적 있나요?` -> `그 열쇠 색이 뭐였죠?`) | 두 번째 `npc_dialogue.data.text`에 첫 질문의 핵심 엔터티 `열쇠` 재참조, `debug.npcConversationHistory['raymond:AgentA'].length`가 4(유저2+NPC2), 대화 히스토리 타임스탬프가 순차 증가 — **참고: `debug.npcConversationHistory` 접근은 테스트 인프라에서 debug hook 제공 필요. 프로덕션 빌드에서는 미포함.** | FR-055 AC2 |
| L1-NP-007 | NPC 위치 고정 | 월드 시작 직후, `raymond.currentRoomId='living_room'`, 이동 이벤트 미발생 상태 | Agent A가 `request_map`, `request_who`, `talk` 순서로 실행 | `room_changed.room.npcs[]`에 `raymond` 존재(거실 입장 시), 첫 `npc_dialogue.visibility.roomId='living_room'`, 게임 시작 후 5분 관찰 동안 `raymond` 위치 변경 로그(`npc_moved`) 0건 | FR-058 AC1 |
| L1-NP-008 | 퍼소나 이탈 금지 체크리스트 (금지 응답 fixture) | `raymond.persona='충직한 집사'`, `fixture.forbiddenResponses` 정의됨 | Agent A가 도발형 질문 3종 전송 (`명령조`, `비밀 유도`, `반말 유도`) | `fixture.forbiddenResponses.length>=3`, `fixture.forbiddenResponses`에 아래 문자열 3개 포함: `야, 내가 왜 그걸 말해`, `아무도 안 물었지만 내가 현장에 있었다`, `됐고 네가 범인이잖아 ㅋㅋ`, 실제 `npc_dialogue.data.text` 3건이 금지 응답 목록과 일치하는 항목 0건, 숨김정보 키워드 무질문 노출 0건 | FR-055 AC3 |
| L1-NP-009 | NPC 데이터 계층 분리 (`knownInfo[]`, `hiddenInfo[]`) | 세계 생성 JSON 캡처 완료, `world.npcs` 길이 >= 1 | QA 훅이 `world.npcs[*]` 스키마 검사 실행 | 각 NPC에 `knownInfo` 배열 존재, `hiddenInfo` 배열 존재, 두 배열 길이 모두 >= 1, 동일 NPC 기준 교집합 크기 `0`, `knownInfo`만 브리핑 공개 로그에 포함되고 `hiddenInfo`는 비공개 저장소에만 존재 | FR-056 AC1 |
| L1-NP-010 | 조건부 단계 공개 | `raymond.knownInfo`에 단계형 정보 2단계 설정, `trustLevel=0.5` | 1) Agent A가 일반 질문 `피해자가 다툰 사실을 아나요?` 2) Agent A가 후속 질문 `누구와 다퉜는지 단서가 있나요?` | 1단계 응답 `npc_dialogue#1.data.text`에 `다툼이 있었다` 포함, 1단계 응답에 `다툰 상대 식별 정보` 미포함, 2단계 응답 `npc_dialogue#2.data.text`에 추가 단서 `남성 목소리` 포함, 2단계 공개 정보 수가 1단계 대비 증가 (`debug.revealedInfoCount` 증가) | FR-056 AC2 |
| L1-NP-011 | 행동 원칙 준수: 먼저 말하지 않음 | `raymond.behaviorPrinciple='직접 묻지 않으면 먼저 말하지 않는다'`, `knownInfo` 키워드 목록 준비 | Agent A가 비관련 잡담 3회 전송 (`날씨`, `저택 분위기`, `식사`) | 3개 `npc_dialogue.data.text` 모두 `knownInfo` 키워드 0회 출현, 3개 응답 모두 정보 제공 트리거 이벤트(`npc_reveal`,`story_event`) 0건, 동일 세션에서 직접 질문 전에는 `debug.revealedInfoIds` 빈 배열 유지 | FR-056 AC3 |
| L1-NP-012 | 기믹 실행으로 스토리 분기 발생 *(Priority: P1 — MVP 이후)* | `raymond` 기믹 효과가 `storyFlag='vault_unlocked'` 설정으로 연결됨, 분기 전 `gameState.gimmickStates['vault']` 미존재 또는 `isTriggered===false` | Agent A가 기믹 트리거 조건 충족 후 트리거 질문 전송 | 트리거 직후 `story_event` 1건 발생, `story_event.data.title` 비어있지 않음, `story_event.data.description` 비어있지 않음, `story_event.data.consequences.length>=1`, 적용 후 `gameState.gimmickStates['vault'].isTriggered===true`, 후속 `request_map`에서 잠긴 방 상태 필드가 해제값으로 변경 | FR-057 AC3 |
| L1-NP-013 | GM/스토리 이벤트에 의한 NPC 이동 *(Priority: P1 — MVP 이후)* | 초기 `raymond.currentRoomId='living_room'`, GM 이벤트 `move_npc` 사용 가능 | GM 테스트 훅이 `move_npc(raymond, library)` 실행 | 이동 전후 비교에서 `from='living_room'`, `to='library'`, 이동 후 `library` 방에서 `room_changed.room.npcs[]`에 `raymond` 존재, 이동 후 `talk raymond` 실행 시 `visibility.roomId='library'` | FR-058 AC2 |
| L1-NP-014 | NPC 이동 시 전체 알림 (`npc_moved`) *(Priority: P1 — MVP 이후)* | Agent A/B/C가 서로 다른 방에 위치, `raymond` 이동 트리거 가능 | 스토리 이벤트로 `raymond`를 `living_room -> library` 이동 | Agent A/B/C 전원 수신 로그에 `game_event`(`event.type==='npc_moved'`) 1건씩 존재, 각 페이로드 `event.data.npcId='raymond'`, `event.data.from`(이전 방 이름), `event.data.to`(이동한 방 이름), `event.visibility.scope==='all'`, 3명 수신 `event.id` 값이 동일 | FR-058 AC3 |
| L1-NP-015 | NPC 없는 게임에서 `/talk` 거부 | 세계 생성 결과 `world.npcs.length===0`, 게임 상태 `playing` | Agent A가 `{type:'talk', message:'누구 있나요?'}` 전송 | Agent A가 `error` 1건 수신, `error.code='NPC_NOT_IN_ROOM'`, `error.message`에 `NPC가 없는 게임` 문구 포함, 같은 턴 `npc_dialogue` 0건 | FR-016 AC3 |
| L1-NP-016 | NPC 대화 이력 persistent (방 이탈 후 복귀) | Agent A가 `raymond`와 초기 대화 1회 완료, 이동 가능한 인접 방 존재 | 1) Agent A가 `talk` 1회 2) `move`로 다른 방 이동 3) 원래 방 복귀 4) 같은 주제로 `talk` 재시도 | 두 번째 대화 응답 `npc_dialogue.data.text`에 첫 대화 참조 문구 포함(`아까 말씀드린`), `debug.npcConversationHistory['raymond:AgentA']`가 방 이동 후에도 유지, 복귀 후 대화가 새 스레드로 초기화되지 않음(`debug.npcConversationHistory` 연속 유지) — **참고: `debug.npcConversationHistory` 접근은 테스트 인프라에서 debug hook 제공 필요. 프로덕션 빌드에서는 미포함.** | FR-055 AC2, FR-043 AC2 |
| L1-NP-018 | NPC 자동 타겟 (방에 NPC 1명) | Agent A와 `raymond` 단둘이 같은 방, npcId 미지정 | Agent A가 `{type:'talk', message:'어젯밤 무슨 일이 있었죠?'}` 전송 (npcId 빈 문자열) | `npc_dialogue` 1건 수신; `npc_dialogue.data.npcId='raymond'`; `error` 미수신; 방에 NPC가 1명일 때 자동 타겟 적용 | FR-043 AC1 |
| L1-NP-017 | NPC 직접 질문 시 knownInfo 공개 | `raymond.knownInfo=['피해자가 어젯밤 누군가와 다퉜다']`, `raymond.behaviorPrinciple='직접 묻지 않으면 먼저 말하지 않는다'`, Agent A와 `raymond` 같은 방 | Agent A가 `{type:'talk', npcId:'raymond', message:'피해자가 어젯밤 누군가와 다퉜다는 게 사실인가요?'}` 전송 | `npc_dialogue.data.text`에 `다퉜` 또는 `다툼` 키워드 포함, 응답이 knownInfo 내용을 확인/전달하는 형태, `debug.revealedInfoIds`에 해당 knownInfo id 추가, 같은 턴 `hiddenInfo` 키워드 미포함 | FR-056 AC2 |

---

## L2 Phase 참조: Phase 2D (Exploration & Actions)

**Diplomat 에이전트:**
- NPC 있는 방으로 이동
- NPC와 3회 이상 대화
- 신뢰도 조건 충족 후 기믹 트리거

**교차 검증:**
- Agent A의 NPC 대화는 같은 방 에이전트만 수신
- NPC 이동 발생 시 `npc_moved`는 전체 에이전트 수신

---

## L3 참조

- FG-008: NPC 집중 상호작용 (Diplomat x2, Explorer, Strategist)에서 퍼소나 일관성, 정보 단계 공개, 기믹 분기, 이동 알림 재검증

---

## L4 체크리스트

> L4 subagent가 TUI 플레이 중 아래 항목에 해당하는 상황을 만나면 **반드시 검증하고** PlayExperienceReport의 `checklist` 필드에 기록한다.
> 자유 플레이를 유지하되, 체크리스트 항목을 의도적으로 커버하는 행동을 포함한다.

| 원본 ID | 체크 항목 | TUI 검증 방법 | Pass 기준 |
|---------|---------|--------------|----------|
| L1-NP-001 | NPC 퍼소나 일관성 | NPC와 3회 이상 대화 후 톤 확인 | 설정된 페르소나(예: 집사)에 맞는 톤 유지, 반말/무례 표현 없음 |
| L1-NP-002 | NPC 정보 비공개 유지 | 직접적 질문으로 숨김 정보 유도 시도 | 숨김 정보가 직접 노출되지 않고 회피 응답이 표시됨 |
| L1-NP-004 | NPC 대화 같은 방 공개 | NPC 대화 후 같은 방/다른 방 화면 비교 | 같은 방 플레이어만 NPC 대화 내용을 볼 수 있음 |
| L1-NP-006 | NPC 대화 이력 유지 | 연속 대화에서 이전 화제 참조 확인 | 두 번째 대화에서 첫 대화 내용 참조 문구가 NPC 응답에 포함됨 |
| L1-NP-007 | NPC 위치 고정 확인 | `/map`과 NPC 대화로 위치 확인 | NPC가 설정된 방에 계속 존재함 |
| L1-NP-014 | NPC 이동 전체 알림 | 스토리 이벤트로 NPC 이동 발생 시 | 모든 플레이어에게 NPC 이동 알림 메시지가 표시됨 |
