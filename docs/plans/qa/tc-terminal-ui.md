# L3: Terminal UI (TUI 렌더링/UX)

**Traces**: FR-073, FR-074, FR-075, FR-076, FR-077, FR-078, FR-079, FR-080
**Layer**: L3 Full Game Agent (TUIAgent)
**인터랙션**: TUI (stdin/stdout via child_process)
**AI 사용**: LLM 의사결정 (TUIAgent가 화면을 읽고 판단)

---

## L3 시나리오 (실행형)

TUIAgent는 `child_process.spawn`으로 실제 CLI를 스폰하고, stdout ANSI strip 결과 + raw ANSI 토큰을 모두 캡처해 아래 시나리오를 검증한다.

| ID | 시나리오 | 사전조건 | 에이전트 행동 | 검증 (필드 레벨) | 관련 FR AC |
|----|---------|---------|-------------|------------------|------------|
| L3-TU-001 | 입력 영역 고정 위치 | 2인 이상 세션, 터미널 120x32, 채팅 로그 20줄 이상 | 게임 화면 3프레임 연속 캡처 후 레이아웃 파싱 | `layout.input.y + layout.input.height == terminal.height`, `layout.chat.y < layout.input.y`, `layout.chat.y + layout.chat.height == layout.input.y`, 입력창이 항상 최하단에 유지 | FR-073 AC3 |
| L3-TU-002 | 명령어 자동완성 *(Priority: P1)* | 명령어 목록(`/help`,`/map`,`/who`) 로드, 입력 포커스 활성 | 입력창에 `/` 입력 후 `Tab` 1회 | `autocomplete.visible===true`, `autocomplete.items.length>=1`, `autocomplete.items.every(v => v.startsWith('/'))===true`, 첫 후보가 입력창에 프리뷰 표시 | FR-076 AC1 |
| L3-TU-003 | 입력 중 채팅 수신 | Agent A/B 같은 방, A 입력 버퍼에 10자 이상 미전송 텍스트 존재 | A가 입력 유지 중 B가 채팅 1회 전송 | A에서 `input.buffer` 값/커서 위치 보존, 신규 메시지 `chatLog.last.messageId` 갱신, 입력 상태 `mode==='typing'` 유지 | FR-073 AC2 |
| L3-TU-004 | 메시지 구분 표시 | 플레이어/GM/NPC/시스템 메시지 fixture 준비 | 메시지 타입별로 1개씩 발생시켜 렌더링 비교 | 각 메시지에 `senderName` + `timestamp` 텍스트 존재, `render.roleStyle` 값이 타입별로 상이, 동일 로그에서 구분 가능한 접두사/색상 동시 존재 | FR-074 AC1, AC3, AC4, AC5 |
| L3-TU-004a | 글로벌 메시지 구분 표시 | 글로벌 메시지(전체 공지) + 방 메시지 fixture 준비 | 글로벌 메시지 1건 + 방 메시지 1건 순차 렌더링 | 글로벌 메시지에 `[전체]` 또는 동등 접두사 텍스트 존재, raw ANSI 기준 글로벌 메시지 색상이 방 메시지와 상이, strip 텍스트에서도 접두사로 구분 가능 | FR-074 AC2 |
| L3-TU-005 | 200개 메시지 스크롤 *(Priority: P1)* | 동일 방 채팅 200개 이상 누적 | `PageUp`으로 최상단 이동 후 `PageDown`으로 최하단 복귀 | 스크롤 상단에서 `chatViewport.first.sequence===1`, 하단 복귀 시 `chatViewport.last.sequence===200`, 스크롤 중 크래시/입력 손실 없음 | FR-039 AC1 |
| L3-TU-006 | 새 메시지 자동 스크롤 *(Priority: P1)* | 뷰포트가 최하단(`autoScroll=true`) | 다른 에이전트가 새 메시지 전송 | 신규 메시지 수신 후 200ms 내 `chatViewport.last.messageId===incoming.messageId`, `autoScroll` 상태 유지 | FR-039 AC3 |
| L3-TU-007 | 현재 위치 표시 | 플레이어가 `living_room`에서 시작 | `/move kitchen` 실행 후 헤더/패널 재렌더 확인 | `infoPanel.currentRoom==='kitchen'`, 헤더 텍스트에 `kitchen` 포함, 이전 방(`living_room`) 문자열은 현재 위치 필드에서 제거 | FR-073 AC1 |
| L3-TU-008 | 같은 방 플레이어 목록 | A/B는 `library`, C는 `kitchen` | A 화면의 정보 패널 스냅샷 캡처 | `infoPanel.roomPlayers` 집합이 `{A,B}`와 일치, `C` 미포함, 각 원소에 `nickname` 존재 | FR-073 AC1 |
| L3-TU-009 | 시스템 정보 패널 실시간 갱신 | A 혼자 `hall`, B가 인접 방 대기, 타임스탬프 기록 가능 | B가 `hall`로 이동 | `player_joined_room` 이벤트 시각 대비 `infoPanel.roomPlayers` 반영 지연 `<=1000ms`, 화면 재렌더 횟수 1회 이상 확인 | FR-073 AC1 |
| L3-TU-010 | 방 목록 시각화 *(Priority: P1)* | 맵 데이터(방/연결) 로드 완료 | `/map` 실행 후 미니맵 영역 파싱 | `mapView.rooms.length===world.rooms.length`, `mapView.edges.length===world.connections.length`, 방 이름이 텍스트로 렌더링 | FR-080 AC3 |
| L3-TU-011 | 현재 위치 강조 *(Priority: P1)* | 맵 표시 상태, 현재 방 식별 가능 | 플레이어가 임의 방으로 이동 후 `/map` 재호출 | `mapView.currentRoom.id===player.currentRoomId`, 현재 방 마커(`>`,`*`,inverse 등) 존재, 비현재 방과 스타일 값 상이 | FR-080 AC3 |
| L3-TU-012 | 이동 가능 방향 표시 *(Priority: P1)* | 현재 방에 2개 이상 인접 방 존재 | 맵 화면에서 인접/비인접 방 스타일 비교 | 인접 방 `mapView.reachable===true` 표시, 비인접 방은 `reachable===false`, 색상/기호 최소 1개 차등 적용 | FR-080 AC3 |
| L3-TU-013 | 역할별 색상 | GM/NPC/Player 발화 이벤트 발생 가능 | GM 서술, NPC 대화, Player 채팅 순차 발생 | raw ANSI 기준 `style.gm.color`, `style.npc.color`, `style.player.color`가 서로 다름, strip 텍스트에도 `[GM]`, `[NPC]`, `nickname:` 접두사 유지 | FR-074 AC1, AC3, AC5 |
| L3-TU-014 | 시스템 메시지 구분 | 시스템 메시지(입장/오류/알림) 3종 준비 | 각 시스템 메시지 렌더 후 스타일 비교 | 시스템 메시지에 `style.variant==='system'`, dim/gray ANSI 코드 포함, 일반 채팅과 prefix(`SYSTEM`/아이콘)로 구분 | FR-074 AC4 |
| L3-TU-015 | 중요 이벤트 하이라이트 | `story_event` 발생 가능한 fixture | 스토리 분기 트리거 1회 실행 | `story_event` 라인에 강조 스타일(`bold`) 적용, `event.type==='story_event'`와 렌더 결과 1:1 매핑, 로그에서 즉시 식별 가능 | FR-077 AC1 |
| L3-TU-015a | 미지원 이벤트 타입 폴백 렌더링 | 미지원 이벤트 타입(`unknown_event`) 주입 가능한 fixture | 미지원 이벤트 타입 1건 수신 후 렌더링 | 앱 크래시 없음, 기본 렌더러로 폴백되어 `event.type` 텍스트 표시, 로그 라인이 정상 출력됨 | FR-077 AC2 |
| L3-TU-016 | 최소 터미널 크기(80x24) 지원 | 터미널 크기 80x24로 고정 | 접속 후 `/help`, `/map`, 일반 채팅 1회씩 수행 | 명령 3종 모두 정상 처리(`error` 없음), 입력창/헤더/채팅 영역이 동시에 보임, 잘림으로 인한 블로킹 없음 | FR-080 AC1 |
| L3-TU-017 | 크기 변경 대응 *(Priority: P1)* | 시작 크기 120x36, 게임 진행 중 리사이즈 가능 | `120x36 -> 90x28 -> 140x40` 순으로 리사이즈 | 각 리사이즈 후 300ms 내 `layout.recomputed===true`, 영역 겹침(`overlapCount===0`), 입력 포커스 유지 | FR-080 AC3 |
| L3-TU-018 | 작은 화면 graceful degradation *(Priority: P1)* | 터미널 크기 60x20 | 접속 후 채팅/명령 3회 실행 | `ui.warning.code==='TERMINAL_TOO_SMALL'` 표시, 핵심 기능(입력/메시지 수신/명령 실행) 유지, 앱 비정상 종료 없음 | FR-080 AC2 |
| L3-TU-019 | 스크린 리더 호환 | ANSI strip 텍스트 캡처 활성, 색상 비활성 모드 가능 | 헤더/패널/채팅/입력 영역 각각 1회 갱신 | 모든 핵심 정보(룸코드, 현재방, 플레이어, 메시지 타입)가 순수 텍스트로 존재, 색상 제거 시에도 의미 손실 없음 | FR-074 AC1, AC3, AC4, AC5 |
| L3-TU-020 | 고대비 모드 | 테마 전환 명령 또는 환경변수 지원 | 고대비 테마 적용 후 동일 화면 비교 | `theme.current==='high-contrast'`, 텍스트/배경 대비 팔레트로 전환, 메시지 타입 구분 표식 유지 | FR-074 AC1, AC3, AC4, AC5 |
| L3-TU-021 | 키보드 전용 네비게이션 | 마우스 이벤트 비활성, 키보드 입력만 허용 | `Tab`, `Shift+Tab`, `Enter`, 방향키로 UI 이동/실행 | 모든 핵심 동작(입력, 자동완성 선택, 스크롤, 명령 실행)이 키보드만으로 완료, mouse dependency 0건 | FR-076 AC3 |
| L3-TU-022 | 한국어 표시 | UTF-8 로케일(`LANG=ko_KR.UTF-8`) | 한국어 헤더/시스템 메시지 렌더 | 문자열 깨짐(`�`) 0건, 한글 조사 포함 문장 정상 출력, 글자폭 계산 오류로 인한 컬럼 깨짐 없음 | NFR-029 |
| L3-TU-023 | UTF-8 입력 | UTF-8 입력 가능한 터미널, 같은 방 수신자 1명 이상 | 입력창에 `안녕하세요 단서 찾음` 전송 | 송신 입력 버퍼와 수신 payload `chat_message.content` 완전 일치, 로그 저장본에서도 동일 바이트 시퀀스 | FR-075 AC2 |
| L3-TU-024 | 다국어 준비(i18n 키 구조) | 소스 접근 가능한 CI 워크스페이스 | i18n 정적 검사 스크립트 실행(`npm run qa:i18n-scan`) | UI 텍스트가 `i18n/*.json` 키를 통해 참조, 하드코딩 문자열 위반 0건, 신규 locale 파일 추가 시 fallback 동작 확인 | NFR-012 |
| L3-TU-025 | `/` 입력 명령어 처리 | 동일 세션에 명령어 라우터 활성, 플레이어 현재 방 식별 가능 | 입력창에 `/who` 입력 후 Enter | `input.raw.startsWith('/')===true`, `commandRouter.lastCommand.name==='who'`, `chat.send` 호출 0회, 명령 결과 패널 렌더 | FR-075 AC1 |
| L3-TU-026 | 일반 텍스트 채팅 처리 | Agent A/B가 같은 방(`hall`)에 연결, 입력창 포커스 활성 | A가 `단서 공유해요` 입력 후 Enter | `input.raw.startsWith('/')===false`, `chat.send.roomId===A.currentRoomId`, B의 `chatLog.last.content==='단서 공유해요'`, commandRouter 호출 0회 | FR-075 AC2 |
| L3-TU-027 | 유효하지 않은 명령어 오류 메시지 | 알 수 없는 명령어 `/foobar`가 등록되지 않음 | 입력창에 `/foobar` 입력 후 Enter | `commandRouter.error.code==='UNKNOWN_COMMAND'`, `ui.toast.variant==='error'`, 오류 라인에 해결 힌트(`/help`) 포함, 앱 종료 없음 | FR-075 AC3 |
| L3-TU-028 | NPC 대화 별도 스타일 | NPC 발화 이벤트(`npc_dialogue`)와 일반 플레이어 채팅 fixture 준비 | NPC 발화 1건 + 플레이어 발화 1건 순차 렌더 | NPC 라인에 `speaker.type==='npc'`, `render.variant==='npc_dialogue'`, 플레이어 라인과 ANSI 스타일/접두사 값이 상이 | FR-074 AC5 |
| L3-TU-029 | 대상 자동완성(NPC/방/플레이어) *(Priority: P1)* | 현재 맵에 `kitchen`, NPC `Eve`, 플레이어 `Bob` 존재 | `/move k` + Tab, `/ask E` + Tab, `/examine B` + Tab 순차 입력 | 각 명령에서 `autocomplete.targetType`이 `room/npc/player`로 분기, 입력 버퍼가 `kitchen`/`Eve`/`Bob`로 보정, 잘못된 타입 후보 0건 | FR-076 AC2 |
| L3-TU-030 | 자동완성 복수 후보 목록 표시 *(Priority: P1)* | 동일 접두사를 공유하는 후보 2개 이상(`library`,`lobby`) 존재 | `/move l` 입력 후 Tab 1회 | `autocomplete.items.length>=2`, 목록 UI가 입력창 근처에 표시, 후보 선택 전 입력 버퍼는 prefix(`l`) 유지, 방향키+Enter로 후보 선택 가능 | FR-076 AC3 |
| L3-TU-031 | 넓은 터미널 사이드바 표시 *(Priority: P1)* | 터미널 크기 160x40, 맵/플레이어 정보 렌더 가능 | 게임 화면 2프레임 캡처 후 레이아웃 파싱 | `layout.sidebar.visible===true`, `layout.sidebar.width>=24`, `layout.sidebar.sections`에 `map`과 `roomPlayers` 포함, 메인 채팅 영역과 겹침 없음 | FR-080 AC3 |
| L3-TU-032 | 브리핑 공개정보 표시 | 게임 시작 직후, 공개 브리핑 payload(`briefing_public.info`) 수신 가능 | 세션 시작 후 브리핑 화면 렌더 완료까지 대기 | `briefing.phase==='public'`, `briefing_public.info.synopsis.length>0`, `briefing_public.info.characterList.length>=1`, `briefing_public.info.npcList.length>=0` | FR-078 AC1 |
| L3-TU-033 | 브리핑 읽음 확인 | 공개 브리핑 화면 노출 중, 확인 액션 키 바인딩(`Enter`) 존재 | 읽음 확인 키 입력 1회 | `briefing.public.readAck===true`, 서버로 `ready` 메시지 1회 전송 (protocol.md ReadyMessage), 화면 상태가 `public -> private_pending`으로 전이 | FR-078 AC2 |
| L3-TU-034 | 개인 역할/목표 표시 | 읽음 확인 완료, 개인 브리핑 payload(`role`,`secrets`,`semiPublicInfo`) 수신 | 개인 브리핑 화면 렌더 후 본인/타인 클라이언트 비교 | 본인 화면 `briefing.private.role`(personalGoals 포함)/`secrets`/`semiPublicInfo` 모두 표시, 타인 화면에는 해당 플레이어 `secret` 문자열 미노출 | FR-078 AC3 |
| L3-TU-035 | ready 후 게임 시작 | 모든 플레이어가 개인 브리핑 단계, 각자 `ready=false` | 모든 플레이어가 ready 입력(예: `r`) | `players.ready.count===players.total`, `screen.mode`가 `briefing`에서 `gameplay`로 전환, 전환 후 입력 프롬프트(`>`) 활성 | FR-078 AC4 |
| L3-TU-035a | 읽음 확인 전 진행 시도 차단 | 공개 브리핑 화면 노출 중, 읽음 확인 키 미입력 | 플레이어가 다른 키(예: 스페이스, 엔터 외 임의 키) 입력 시도 | 화면이 공개 브리핑 상태 유지, `briefing.public.readAck===false` 유지, 진행 불가 메시지 표시(예: "먼저 읽음을 확인하세요"), 상태 전이 없음 | FR-078 AC2 |
| L3-TU-035b | ready 전 게임 시작 차단 | 개인 브리핑 단계, 일부 플레이어만 ready 상태 | 모든 플레이어가 ready하지 않은 상태에서 게임 시작 시도 | 게임 시작 불가, `screen.mode`가 `briefing` 유지, 오류 메시지 표시(예: "모든 플레이어가 준비할 때까지 대기"), 상태 전이 없음 | FR-078 AC4 |
| L3-TU-036 | 공통 결과 표시 | `game_ending` 이벤트 payload에 `commonResult` 포함 | 게임 종료 이벤트 수신 후 엔딩 화면 렌더 | `ending.commonResult.length>0` (commonResult는 문자열), 종료 사유는 `game_end` event의 `event.data.reason`에서 표시, 모든 클라이언트에서 동일 텍스트 해시 일치 | FR-079 AC1 |
| L3-TU-037 | 개인화 엔딩 표시 | 엔딩 payload에 플레이어별 `personalEnding` 포함 | 각 플레이어 클라이언트에서 엔딩 화면 확인 | 각 클라이언트의 `ending.personalEnding.playerId===self.id`, `ending.personalEnding.narrative.length>0`, 타 플레이어 개인 엔딩 전문 미노출 | FR-079 AC2 |
| L3-TU-038 | 비밀 공개 표시 | 엔딩 payload에 `secretReveal` 구조체 포함 | 엔딩 화면의 비밀 공개 섹션 렌더 | `ending.secretReveal.playerSecrets.length>=1`, 각 `playerSecrets[]` 항목에 `playerId`/`characterName`/`secret` 필드 표시; `secretReveal.npcSecrets[]`, `secretReveal.undiscoveredClues[]`, `secretReveal.semiPublicReveal[]` 각 섹션 렌더; 누락 항목 0건 | FR-079 AC3 |
| L3-TU-039 | 피드백 UI 전환 | 엔딩 공통/개인/비밀 섹션 렌더 완료 | 계속 진행 키 입력 후 다음 화면 진입 | `screen.mode==='feedback'`, `feedback.form.fields`에 `funRating`/`immersionRating`/`comment` 존재, 첫 입력 필드 포커스 활성 | FR-079 AC4 |

| L3-TU-040 | `clue_found` 렌더러 | `clue_found` 이벤트 발생 가능한 fixture, 플레이어가 단서 소재 방에 위치 | 단서가 있는 오브젝트 `/examine` 실행 → `clue_found` 이벤트 수신 | `clue_found` 라인에 전용 스타일(아이콘/접두사/색상) 적용; `event.type==='clue_found'`; 단서 이름(`event.data.clue.name`) 텍스트 표시; 일반 `examine_result`와 시각적으로 구분 가능; strip 텍스트에서도 단서 발견 표식 존재 | FR-077 AC1 |
| L3-TU-041 | `npc_give_item` 렌더러 | NPC가 아이템 전달 가능한 기믹/조건 fixture | NPC에게 특정 대화 후 아이템 수령 이벤트 발생 | `npc_give_item` 라인에 전용 스타일 적용; `event.type==='npc_give_item'`; `event.data.item.name` 및 `event.data.npcName` 텍스트 표시; 일반 `npc_dialogue`와 시각적으로 구분; strip 텍스트에 아이템명/NPC명 포함 | FR-077 AC1 |
| L3-TU-042 | `npc_receive_item` 렌더러 | 플레이어 인벤토리에 전달 가능 아이템 존재, 대상 NPC 같은 방 | `/give` 명령으로 NPC에게 아이템 전달 후 이벤트 수신 | `npc_receive_item` 라인에 전용 스타일 적용; `event.type==='npc_receive_item'`; `event.data.item.name` 및 `event.data.npcName` 텍스트 표시; strip 텍스트에 아이템명 포함 | FR-077 AC1 |
| L3-TU-043 | `npc_reveal` 렌더러 | NPC 정보 공개 조건 충족 가능한 fixture | NPC 기믹 트리거 → `npc_reveal` 이벤트 수신 | `npc_reveal` 라인에 강조 스타일(highlight/bold) 적용; `event.type==='npc_reveal'`; `event.data.npcName` 및 공개 정보 텍스트 표시; 일반 NPC 대화와 시각적으로 구분(색상/접두사 차이); strip 텍스트에서도 공개 표식 존재 | FR-077 AC1 |
| L3-TU-044 | `game_end` 렌더러 | 종료 조건 충족 직전 상태 | 종료 조건 달성 → `game_end`(`game_ending`) 이벤트 수신 | `game_end` 이벤트에 전용 풀스크린/분리형 렌더링 적용; 일반 채팅/이벤트 로그와 명확히 분리된 화면 전환; `ending.reason` 텍스트 표시; strip 텍스트에 종료 안내 문구 존재; 렌더 후 입력 모드 전환(게임플레이 → 엔딩) | FR-077 AC1 |
| L3-TU-045 | `examine_result` 렌더러 | 조사 가능한 오브젝트가 있는 방에 플레이어 위치 | `/examine desk` 실행 → `examine_result` 이벤트 수신 | `examine_result` 라인에 서술형 블록 스타일 적용; `event.type==='examine_result'`; `event.data.target` 및 `event.data.description` 텍스트 표시; 일반 채팅과 레이아웃/들여쓰기로 구분; strip 텍스트에 대상 이름 포함 | FR-077 AC1 |
| L3-TU-046 | `action_result` 렌더러 | 자유 행동(`/do`) 가능한 게임 진행 중 상태 | `/do 창문을 열어본다` 실행 → `action_result` 이벤트 수신 | `action_result` 라인에 서술형 블록 스타일 적용; `event.type==='action_result'`; 행동 판정 결과(`event.data.result`) 텍스트 표시; 성공/실패에 따른 시각적 차이(색상 또는 접두사); strip 텍스트에 판정 결과 포함 | FR-077 AC1 |
| L3-TU-047 | `time_warning` 렌더러 | 강제 종료 타이머 활성 세션, 경고 시점 도달 가능 | 남은 시간 5분 전 → `time_warning` 이벤트 수신 | `time_warning` 라인에 긴급 스타일(bold + 경고색) 적용; `event.type==='time_warning'`; `event.data.remainingMinutes` 또는 남은 시간 텍스트 표시; 일반 시스템 메시지보다 시각적 강조도 높음; strip 텍스트에 시간 정보 및 경고 표식 존재 | FR-077 AC1 |
| L3-TU-048 | 하단 커맨드 힌트 표시 | 게임 진행 중 상태, 입력창 활성, 터미널 120x32 | 게임 화면 2프레임 캡처 후 입력 영역 파싱 | 입력창 근처(하단)에 주요 커맨드 힌트 텍스트 존재(예: `/help`, `/map`, `/examine` 등 최소 2개 이상); strip 텍스트에서도 힌트 문자열 식별 가능; 힌트가 입력 버퍼와 겹치지 않음; 힌트 텍스트가 실제 등록된 명령어와 일치 | FR-073 AC3 |
| L3-TU-049 | 각 방 플레이어 현황 표시 | 4인 세션, Agent A/B는 `library`, Agent C는 `kitchen`, Agent D는 `hall` | Agent A 화면의 상단(헤더) 영역 스냅샷 캡처 | 헤더 또는 정보 패널에 **전체 방별 플레이어 현황** 표시; `library`에 2명, `kitchen`에 1명, `hall`에 1명 정보가 확인 가능; 현재 방(`library`)뿐 아니라 다른 방(`kitchen`, `hall`)의 인원/목록도 요약 표시; 플레이어 이동 시 해당 정보 갱신 반영 | FR-073 AC1 |
| L3-TU-050 | 헤더에 룸 코드 표시 | 게임 진행 중(`playing`) 상태, 세션 룸 코드 알려진 상태 | 게임플레이 화면 1프레임 캡처 후 헤더 영역 파싱 | 헤더 텍스트에 현재 세션의 룸 코드(`^[A-Z]+-\d{4}$` 형식) 문자열이 포함됨; strip 텍스트에서도 룸 코드 식별 가능; 로비 단계에서 이미 표시되었던 것과 동일한 코드값이 게임 진행 중에도 유지됨 | FR-001 AC2, FR-073 AC1 |

### TUI 평가 리포트 필드

L3 AgentQAReport의 `tuiEvaluation`에 아래 필드를 반드시 기록한다.

- `layoutReadable`, `headerInfoPresent`, `scrollWorking`, `commandsResponsive`
- `messageTypesDistinguishable`, `renderingGlitches`
- `inputAnchorStable`, `autocompleteVisible`, `autoScrollWorking`
- `mapReadable`, `accessibilityTextFallback`, `highContrastAvailable`
- `i18nReady`, `unicodeInputStable`

---

## L1 참조

없음 - TUI 렌더링은 프로토콜 레벨에서 직접 검증 불가. TUIAgent 화면 파싱으로만 검증.

---

## L2 Phase 참조

- Phase 2D: `/map`, 이동, 채팅 이벤트의 프로토콜 정합성 선검증 후 L3에서 렌더링 검증
- Phase 2A: `/help`/명령어 처리의 기본 응답 경로 선검증 후 L3에서 입력 UX 검증

---

## L3 참조

- **FG-007**: TUI 경험 검증 전용 시나리오 (4 TUI 에이전트). 본 문서 `L3-TU-001~049`를 체크리스트로 사용.
- Pass/Fail: TUI 렌더링 결함(`renderingGlitches`) 3건 초과 시 FAIL.

---

## L4 체크리스트

> L4 subagent가 TUI 플레이 중 아래 항목에 해당하는 상황을 만나면 **반드시 검증하고** PlayExperienceReport의 `checklist` 필드에 기록한다.
> 자유 플레이를 유지하되, 체크리스트 항목을 의도적으로 커버하는 행동을 포함한다.

| 원본 ID | 체크 항목 | TUI 검증 방법 | Pass 기준 |
|---------|---------|--------------|----------|
| L3-TU-001 | 입력 영역 최하단 고정 | 게임 중 입력창 위치 확인 | 입력창이 항상 화면 최하단에 위치함 |
| L3-TU-002 | 명령어 자동완성 | `/` 입력 후 Tab | 자동완성 후보 목록이 표시됨 |
| L3-TU-003 | 입력 중 채팅 수신 | 입력 도중 다른 플레이어 메시지 수신 | 입력 내용이 유지되면서 새 메시지가 표시됨 |
| L3-TU-004 | 메시지 구분 표시 | 다양한 메시지 타입 관찰 | 플레이어/GM/NPC/시스템 메시지가 시각적으로 구분됨 |
| L3-TU-005 | 스크롤 동작 | PageUp/PageDown 사용 | 채팅 로그 스크롤이 정상 동작함 |
| L3-TU-007 | 현재 위치 표시 | 이동 후 헤더/패널 확인 | 현재 방 이름이 헤더/패널에 정확히 표시됨 |
| L3-TU-008 | 같은 방 플레이어 목록 | 정보 패널 확인 | 같은 방 플레이어만 목록에 표시됨 |
| L3-TU-010 | 방 목록 시각화 | `/map` 실행 | 방 이름과 연결 정보가 시각적으로 표시됨 |
| L3-TU-011 | 현재 위치 강조 | `/map` 실행 후 현재 방 확인 | 현재 방이 다른 방과 다른 스타일로 강조됨 |
| L3-TU-015 | 중요 이벤트 하이라이트 | 스토리 이벤트 발생 시 | 중요 이벤트가 강조 스타일(bold)로 표시됨 |
| L3-TU-022 | 한국어 표시 | 한국어 UI 텍스트 확인 | 한글 깨짐(`□`, `?`) 없이 정상 표시 |
| L3-TU-025 | / 명령어 처리 | `/who` 등 명령어 입력 | 명령어가 채팅이 아닌 명령으로 처리되어 결과 표시됨 |
| L3-TU-027 | 잘못된 명령어 오류 | `/foobar` 등 미지원 명령 입력 | 오류 메시지와 `/help` 힌트 표시, 앱 크래시 없음 |
| L3-TU-032 | 브리핑 공개정보 표시 | 게임 시작 직후 | 세계 요약, 관계도, NPC 목록이 표시됨 |
| L3-TU-034 | 개인 역할/목표 표시 | 개인 브리핑 단계 | 자신의 역할, 비밀, 목표가 표시됨 |
| L3-TU-036 | 공통 결과 표시 | 게임 종료 시 | 공통 엔딩 텍스트가 표시됨 |
| L3-TU-037 | 개인화 엔딩 표시 | 게임 종료 시 | 개인 엔딩 텍스트가 표시됨 |
| L3-TU-038 | 비밀 공개 표시 | 게임 종료 시 | 모든 플레이어의 비밀이 공개되어 표시됨 |
| L3-TU-039 | 피드백 UI 전환 | 엔딩 후 계속 진행 | 피드백 입력 화면으로 전환됨 |

### 정성 평가 (기존 L4 참조)

L4 Subagent Playtest에서 subagent가 TUI를 직접 조작하므로 실제 사용자 관점의 교차검증 레이어로 사용한다.

- **L4-001~004**: PlayExperienceReport `ux` + `errors(type='render_broken')`로 L3 결과 재검증
- L3에서 PASS여도 L4에서 재현된 UX blocker는 결함으로 재오픈

---

## NFR 참조 (PRD §5 기준)

| NFR | 검증 내용 | 연결 시나리오 |
|-----|----------|--------------|
| NFR-023 | 색상만이 아닌 텍스트/기호로 메시지 의미를 전달 | L3-TU-014, L3-TU-019 |
| NFR-024 | Go 1.26 에서 TUI 시나리오 실행 가능 | L3-TU-016, L3-TU-017 (환경 매트릭스) |
| NFR-025 | macOS/Linux/Windows(WSL)에서 동일 TUI 동작 | L3-TU-016, L3-TU-017, L3-TU-031 |
| NFR-026 | 주요 터미널 에뮬레이터(iTerm2, Terminal.app, Windows Terminal 등) 호환 | L3-TU-016, L3-TU-017 |
| NFR-027 | 최소 80x24 환경에서 정상 동작 | L3-TU-016 |
| NFR-029 | UTF-8 다국어 표시/입력 지원 | L3-TU-022, L3-TU-023 |
