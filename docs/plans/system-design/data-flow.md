# Data Flow (주요 시나리오별 데이터 흐름)

---

## 1. 게임 세션 전체 라이프사이클

```
Phase 1: 세션 생성
──────────────────
Host: story host
  → CLI가 ServerConfig 구성 (API 키 설정 포함)
  → StoryServer 생성 및 Start()
  → SessionManager.createSession() → 룸 코드 발급
  → NetworkServer.Start()
    → WebSocket 서버 시작 (ws://localhost:PORT)
  → Host도 Client로 로컬 WebSocket 연결 (또는 직접 메서드 호출)

Phase 2: 참가자 합류
────────────────────
Joiner: story join WOLF-7423
  → NetworkClient.connect(serverUrl)
    → WebSocket 연결 수립 (ws://host:port)
  → Client → Server (WebSocket): { type: 'join', nickname: 'Alice' }
  → SessionManager.addPlayer() → Player 생성
  → Server → All (WebSocket): { type: 'lobby_update', players: [...] }

Phase 3: 세계 생성
──────────────────
Host: 게임 시작
  → Client → Server: { type: 'start_game' }
  → SessionManager.startGame() → status: 'generating'
  → AILayer.generateWorld(playerCount)
      → WorldGenerator.generate()
      → StoryValidator.validateStructure()
      → (실패 시 부분 재생성)
  → 진행 중: Server → All: { type: 'generation_progress', ... }
  → 완료: GameStateManager.initializeWorld(world)
  → SessionManager.onWorldGenerated() → status: 'briefing'

Phase 4: 브리핑
──────────────
  → Server → All: { type: 'briefing_public', info: PublicInfo }
  → Server → Each: { type: 'briefing_private', role: PlayerRole, secrets: [...], semiPublicInfo: [...] }
    (FR-051: 반공개 정보 중 해당 플레이어 대상분을 semiPublicInfo로 포함)
  → Client → Server: { type: 'ready', phase: 'briefing_read' } (각 플레이어, 공개 브리핑 읽음 확인)
  → Client → Server: { type: 'ready', phase: 'game_ready' } (각 플레이어, 게임 시작 준비 완료)
  → SessionManager.onAllPlayersReady() → status: 'playing'
  → Server → All: { type: 'game_started', initialRoom: RoomView }
  → Server → Each: { type: 'map_info', map: MapView }  ← 초기 MapOverview 설정용 (클라이언트 HeaderBar 표시)
  → EndConditionEngine.startMonitoring()
  → GMEngine.generateOpening() (GM 있는 경우)

Phase 5: 게임 진행
──────────────────
  (이하 시나리오 2~6 참조)

Phase 6: 게임 종료
──────────────────
  → EndConditionEngine이 종료 조건 감지
  → SessionManager.startEnding() → status: 'ending'
  → AILayer.generateEndings()
  → Server → All: { type: 'game_ending', ... }
  → SessionManager.finishGame() → status: 'finished'
```

---

## 2. 같은 방 채팅

```
Alice (1층 거실)
    │
    ▼
Client → Server: { type: 'chat', content: '누가 서재에 갔어?' }
    │
    ▼
ActionProcessor.handleChat()
    │
    ▼
EventBus.emit('chat', {
  senderId: 'alice-id',
  senderName: 'Alice',
  roomId: 'living-room',
  content: '누가 서재에 갔어?',
  scope: 'room',
})
    │
    ▼
MessageRouter.routeChat()
    │
    ▼
GameStateManager.getPlayersInRoom('living-room')
→ [Alice, Bob]    ← Carol은 부엌에 있으므로 제외
    │
    ▼
NetworkServer.sendToMany(['alice-id', 'bob-id'], {
  type: 'chat_message',
  senderId: 'alice-id',
  senderName: 'Alice',
  content: '누가 서재에 갔어?',
  scope: 'room',
  timestamp: 1234567890,
})
    │
    ▼
Alice의 Client: ChatLog에 "Alice: 누가 서재에 갔어?" 표시
Bob의 Client:   ChatLog에 "Alice: 누가 서재에 갔어?" 표시
Carol의 Client: (아무것도 안 보임)
```

---

## 3. 방 이동

```
Alice → Client: /move 부엌
    │
    ▼
CommandParser: { type: 'command', command: 'move', args: '부엌' }
    │
    ▼
Client → Server: { type: 'move', targetRoomId: '부엌' }
    │
    ▼
ActionProcessor.handleMove()
    │
    ▼
MapEngine.movePlayer('alice-id', '부엌')
    ├── MapEngine.getRoomByName('부엌') → kitchen room
    ├── MapEngine.isAdjacent('living-room', 'kitchen') → true
    └── GameStateManager.movePlayer('alice-id', 'kitchen')
    │
    ├── EventBus.emit('game_event', {
    │     type: 'player_move',
    │     visibility: { scope: 'all' },
    │     data: { playerId: 'alice', from: '1층 거실', to: '부엌' }
    │   })
    │
    └── MessageRouter → 전체 플레이어에게:
          "--- Alice이(가) 1층 거실에서 부엌으로 이동했습니다 ---"

    + Alice에게 추가 전송:
      { type: 'room_changed', room: kitchenRoomView }

    + 거실 (Bob에게):
      { type: 'player_left_room', nickname: 'Alice', destination: '부엌' }

    + 부엌 (Carol에게):
      { type: 'player_joined_room', nickname: 'Alice' }

    + 전체 플레이어에게 맵 갱신:
      { type: 'map_info', map: updatedMapView }
      (HeaderBar의 실시간 플레이어 위치 표시에 사용)
```

---

## 4. 방 조사 (/examine)

```
Alice → Client: /examine 책상
    │
    ▼
Client → Server: { type: 'examine', target: '책상' }
    │
    ▼
ActionProcessor.handleExamine()
    │
    ├── player = GameStateManager.getPlayer('alice-id')
    ├── room = GameStateManager.getPlayerRoom('alice-id')
    ├── context = buildGameContext('alice-id')
    │
    ▼
AILayer.evaluateExamine(context, room, '책상')
    │
    ▼
ActionEvaluator → AI API 호출
    │
    ▼
AI 응답 (구조화된 JSON):
{
  events: [
    { type: 'examine_result', data: { description: '오래된 편지가 있다...' } },
    { type: 'clue_found', data: { clue: { id: 'clue-3', name: '오래된 편지' } } }
  ],
  stateChanges: [
    { type: 'discover_clue', playerId: 'alice-id', clueId: 'clue-3' }
  ]
}
    │
    ├── GameStateManager.discoverClue('alice-id', 'clue-3')
    │
    └── EventBus.emit('game_event', examineResultEvent)
        EventBus.emit('game_event', clueFoundEvent)
        (visibility: { scope: 'room', roomId: 'kitchen' })
    │
    ▼
MessageRouter → 부엌에 있는 플레이어들(Alice, Carol)에게만:
  "[조사] 책상 위에 오래된 편지가 놓여 있다..."
  "* Alice이(가) [오래된 편지]를 발견했습니다!"
```

---

## 5. NPC 대화

```
Alice → Client: /talk 레이몬드 어젯밤 무슨 일이 있었죠?
    │
    ▼
Client → Server: { type: 'talk', npcId: '레이몬드', message: '어젯밤 무슨 일이 있었죠?' }
    // 클라이언트가 NPC 이름을 입력하면, 서버의 MapEngine.GetNPCByName()이 ID로 변환
    │
    ▼
ActionProcessor.handleTalk()
    │
    ├── NPC '레이몬드'가 Alice와 같은 방에 있는지 확인
    │     → 같은 방 (1층 거실) ✓
    │
    ▼
AILayer.chatWithNPC(npc, 'alice-id', message, context)
    │
    ▼
NPCEngine.chat()
    ├── 대화 이력 조회
    ├── 프롬프트 구성 (퍼소나 + 보유 정보 + 신뢰도 + 이력)
    └── AI API 호출
    │
    ▼
NPCResponse {
  dialogue: "어젯밤이요? 별 일 없었습니다만...",
  internalThought: "아직 이 플레이어를 믿을 수 없다",
  infoRevealed: [],
  trustChange: 0.1,
  triggeredGimmick: false,
  events: [{
    type: 'npc_dialogue',
    data: { npcId: 'raymond', npcName: '레이몬드', playerId: 'alice-id', playerName: 'Alice', text: '어젯밤이요? 별 일 없었습니다만...', emotion: 'cautious' }
  }]
}
    │
    ├── GameStateManager.updateNPCTrust('raymond', 'alice-id', 0.1)
    │
    └── EventBus.emit('game_event', npcDialogueEvent)
        (visibility: { scope: 'room', roomId: 'living-room' })
    │
    ▼
MessageRouter → 1층 거실의 플레이어들에게:
  "[레이몬드] 어젯밤이요? 별 일 없었습니다만..."
```

---

## 6. 종료 흐름 (투표 기반)

```
[종료 조건: "과반수가 범인을 지목하면 게임 종료"]

(1) 투표 시작
EndConditionEngine.startVote('범인을 지목하세요', candidates, 120)
    │
    ▼
Server → All: {
  type: 'vote_started',
  reason: '범인을 지목하세요',
  candidates: ['Alice', 'Bob', 'Carol', 'Dave'],
  timeoutSeconds: 120,
}

(2) 투표 진행
각 플레이어 → Server: { type: 'vote', targetId: 'bob-id' }
    │
    ▼
EndConditionEngine.castVote()
    │
    ▼
Server → All: { type: 'vote_progress', votedCount: 3, totalVoters: 4 }

(3) 투표 완료 + 종료 조건 평가
모든 플레이어 투표 완료
    │
    ▼
Server → All: {
  type: 'vote_ended',
  results: [{ candidateId: 'bob-id', candidateName: 'Bob', votes: 3 }, ...],
  outcome: 'Bob이 가장 많은 표를 받았습니다',
}
    │
    ▼
EndConditionEngine → 종료 조건과 대조 → 조건 충족
    │
    ▼
triggerEnding('vote_result')

(4) 엔딩 생성
AILayer.generateEndings(context)
    │
    ▼
EndingGenerator → AI API 호출
    │
    ▼
GameEndData {
  commonResult: "범인은 Bob이었습니다...",
  playerEndings: [...],
  secretReveal: { ... },
}

(5) 엔딩 전달
Server → Each Player: {
  type: 'game_ending',
  commonResult: "범인은 Bob이었습니다...",
  personalEnding: "(해당 플레이어의 개인화된 엔딩)",
  secretReveal: { ... },
}
    │
    ▼
각 Client: EndingScreen 렌더링

(6) 게임 종료
Server → All: { type: 'game_finished' }
```

---

## 7. 글로벌 채팅 (/shout)

```
Alice (부엌) → Client: /shout 범인은 Bob이야!
    │
    ▼
Client → Server: { type: 'shout', content: '범인은 Bob이야!' }
    │
    ▼
ActionProcessor.handleShout()
    │
    ▼
MessageRouter.routeChat(ChatData{..., Scope: "global"})
    │
    ▼
GameStateManager.getAllPlayers() → [Alice, Bob, Carol, Dave]
    │
    ▼
NetworkServer.sendToAll({
  type: 'chat_message',
  senderId: 'alice-id',
  senderName: 'Alice',
  content: '범인은 Bob이야!',
  scope: 'global',
  senderLocation: '부엌',   // FR-037 AC3: 발신자 위치 포함
  timestamp: 1234567890,
})
    │
    ▼
모든 플레이어의 Client: ChatLog에 "[전체] Alice (부엌): 범인은 Bob이야!" 표시
```

---

## 8. 행동 묘사 (/do)

```
Alice (1층 거실) → Client: /do 벽에 걸린 그림을 뒤집어 본다
    │
    ▼
Client → Server: { type: 'do', action: '벽에 걸린 그림을 뒤집어 본다' }
    │
    ▼
ActionProcessor.handleDo()
    │
    ├── context = buildGameContext('alice-id')
    │
    ▼
AILayer.evaluateAction(context, action)
    │
    ▼
ActionEvaluator → AI API 호출
    │
    ▼
AI 응답 (구조화된 JSON):
{
  events: [
    { type: 'action_result', data: {
        action: '벽에 걸린 그림을 뒤집어 본다',
        result: '그림 뒤에 숨겨진 금고를 발견했다!'
    }}
  ],
  stateChanges: []
}
    │
    ▼
EventBus.emit('game_event', actionResultEvent)
    (visibility: { scope: 'room', roomId: 'living-room' })
    │
    ▼
MessageRouter → 1층 거실의 플레이어들에게:
  "[행동] Alice이(가) 벽에 걸린 그림을 뒤집어 보았다."
  "→ 그림 뒤에 숨겨진 금고를 발견했다!"
```

---

## 9. 플레이어 종료 요청 (/end)

```
Alice → Client: /end
    │
    ▼
Client → Server: { type: 'propose_end' }
    │
    ▼
ActionProcessor.handleProposeEnd()
    │
    ├── 게임 진행 중 확인 (status == 'playing')
    │
    ▼
EndConditionEngine.proposeEnd('alice-id')
    │
    ▼
Server → All: {
  type: 'end_proposed',
  proposerId: 'alice-id',
  proposerName: 'Alice',
  timeoutSeconds: 60,
}

각 플레이어 → Client: 종료 투표 UI 표시
각 플레이어 → Server: { type: 'end_vote', agree: true/false }
    │
    ▼
EndConditionEngine.castEndVote()
    │
    ▼
Server → All: { type: 'end_vote_result', agreed: 4, disagreed: 1, passed: true }
    │  (과반 이상 동의 시 통과)
    │
    ├── passed == true:
    │     → triggerEnding('player_vote')
    │     → (이하 시나리오 6의 (4)~(6)과 동일)
    │
    └── passed == false:
          → 게임 계속 진행
```

---

## 10. 합의 종료 (/solve)

```
[종료 조건 유형이 "consensus"인 세션]

(1) 합의 시작
EndConditionEngine.startSolve('해결안을 제시하세요', 120)
    │
    ▼
Server → All: {
  type: 'solve_started',
  prompt: '해결안을 제시하세요',
  timeoutSeconds: 120,
}

(2) 답변 제출
각 플레이어 → Server: { type: 'solve', answer: '범인은 Bob이다' }
    │
    ▼
EndConditionEngine.submitSolve()
    │
    ▼
Server → All: { type: 'solve_progress', submittedCount: 3, totalPlayers: 4 }

(3) 전원 제출 완료 + AI 판정
    │
    ▼
AILayer.evaluateConsensus(answers, context)
    │
    ▼
Server → All: {
  type: 'solve_result',
  answers: [{ playerId: '...', playerName: 'Alice', answer: '...' }, ...],
  outcome: 'AI 판정 결과',
}
    │
    ▼
EndConditionEngine → 종료 조건 평가 → 충족 시 triggerEnding('consensus')
    │
    ▼
(이하 시나리오 6의 (4)~(6)과 동일)
```

---

## 11. 타임아웃 종료

```
EndConditionEngine.timeoutTimer 만료 (GameSettings.TimeoutMinutes 경과)
    │
    ├── 5분 전 경고:
    │   EventBus.emit('game_event', TimeWarningEvent{ RemainingMinutes: 5 })
    │   → Server → All: { type: 'game_event', event: { type: 'time_warning', ... } }
    │
    ├── 1분 전 경고:
    │   EventBus.emit('game_event', TimeWarningEvent{ RemainingMinutes: 1 })
    │
    ▼
타임아웃 도달
    │
    ▼
triggerEnding('timeout')
    │
    ▼
(이하 시나리오 6의 (4)~(6)과 동일, endReason: 'timeout')
```
