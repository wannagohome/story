# L2: Story Verification (스토리 검증)

**Traces**: FR-020, FR-021, FR-022, FR-023, FR-025, FR-093
**Layer**: L2 Phase Agent (Phase 2B — 검증 에이전트)
**인터랙션**: Protocol level
**AI 사용**: 실제 AI (세계 생성) + 검증 에이전트 (LLM 판정)
**Note**: PRD 4.3에는 FR-024 항목이 정의되어 있지 않아 본 문서는 FR-020/021/022/023/025/093 기준으로 검증한다.

---

## Phase 2B: Story Verification

세계 생성 완료 후 검증 에이전트가 생성된 스토리의 논리적 정합성을 평가한다.

### 스토리 검증 Assertions

| ID | 시나리오 | 사전조건 | 에이전트 행동 | 검증 (필드 레벨) | 관련 FR AC |
|----|---------|---------|-------------|------------------|------------|
| L2-SV-001 | 구조적 모순 기본 통과 케이스 | 정상 생성된 world fixture, 결함 주입 없음 | 서버 내부 `storyValidator.Verify(worldGeneration)` 실행 | `verification_report.status==='passed'`, `verification_report.contradictions`가 배열이며 길이 `0`, `verification_report.checkedRules`에 `timeline`/`relationships` 포함 | FR-020 AC1, AC2, AC3 |
| L2-SV-002 | 시간선 모순 검출 (NEGATIVE) | 결함 world fixture: 동일 인물의 사건 시간이 역전되도록 주입 | 검증 실행 | `verification_report.status==='failed'`, `contradictions.length >= 1`, 최소 1개 항목이 `type==='timeline'`, 해당 항목 `entities.length >= 1`, `evidencePaths.length >= 1` | FR-020 AC1, AC3 |
| L2-SV-003 | 관계 설정 모순 검출 (NEGATIVE) | 결함 world fixture: A-B 관계가 동시에 `절친`과 `서로 모르는 사이`로 주입 | 검증 실행 | `verification_report.status==='failed'`, `contradictions`에 `type==='relationship'` 항목 존재, `contradictions[*].entities`에 충돌 캐릭터 2명 이상 포함 | FR-020 AC2, AC3 |
| L2-SV-004 | 복선 회수 경로 존재 확인 | `clues[].relatedClueIds`와 `gameStructure.endConditions`가 정상 연결된 world fixture | 검증 실행 | `verification_report.foreshadowing.unresolvedIds.length===0`, `foreshadowing.paths`의 각 엔트리 `steps.length >= 1` | FR-021 AC1 |
| L2-SV-005 | 회수 불가능 복선 보고 (NEGATIVE) | 결함 world fixture: 단서 `clue_dead_end`를 어떤 경로로도 회수 불가하게 주입 | 검증 실행 | `verification_report.status==='failed'`, `foreshadowing.unresolvedIds`에 `clue_dead_end` 포함, `issues`에 `code==='UNRESOLVABLE_FORESHADOWING'` 존재 | FR-021 AC2 |
| L2-SV-006 | 단서 roomId 필수값 확인 | 정상 world fixture, `clues.length >= 1` | `clues[]` 필드 검사 + 검증 실행 | 모든 `clues[*].roomId`가 non-empty 문자열, `verification_report.cluePlacement.missingRoomIdClueIds.length===0` | FR-022 AC1 |
| L2-SV-007 | `clue.roomId` ↔ `map.rooms[].id` 교차 참조 (NEGATIVE 포함) | 결함 world fixture: `clues[2].roomId='room_missing'` 강제 주입 | 검증 실행 | `verification_report.status==='failed'`, `issues`에 `code==='CLUE_ROOM_NOT_FOUND'` 존재, `issues[*].path==='clues[2].roomId'`, `issues[*].expectedRoomIds` 배열 길이 `>= 1` | FR-022 AC2 |
| L2-SV-008 | NPC 보유 정보 정합성 | 정상 world fixture, NPC가 `knownInfo[]` 보유 | 검증 실행 | `verification_report.npcConsistency.infoMismatches.length===0`, 각 NPC `knownInfo[]` 항목이 `worldFacts[]`에 매핑됨(`unmappedKnownInfoCount===0`) | FR-023 AC1 |
| L2-SV-009 | NPC 위치 유효성 | 정상 world fixture, NPC 1명 이상 존재 | 검증 실행 | `verification_report.npcConsistency.invalidRoomNpcIds.length===0`, 모든 `characters.npcs[*].currentRoomId`가 `map.rooms[].id`에 포함 | FR-023 AC2 |
| L2-SV-010 | NPC 기믹 트리거 달성 가능성/엔티티 참조 유효성 | 정상 world fixture, 모든 NPC에 `gimmick` 존재 | 트리거 표현식 파서와 검증 실행 | `verification_report.npcConsistency.unresolvableTriggerIds.length===0`, `triggerEntityRefs.missingEntityRefs.length===0`, 각 트리거의 참조 대상이 실제 `roomId`/`npcId`/`clueId`/`itemId` 집합에 존재 | FR-023 AC3 |
| L2-SV-011 | 종료 조건 달성 가능 경로 검증 | 정상 world fixture, `endConditions.length >= 1` | 서버 내부 `storyValidator.Validate()` 결과의 `gameStructureValidation` 확인 | `gameStructureValidation.feasible===true`, `reachableEndConditionIds.length === gameStructure.endConditions.length`, 각 endCondition에 대응하는 `pathsByEndCondition[id].steps.length >= 1` | FR-093 AC1 |
| L2-SV-012 | 모든 플레이어 의미 있는 행동 경로 검증 | 정상 world fixture, playerCount=4 | 검증 실행 | `playerActionPaths` 키 수가 `playerCount`와 동일, 각 플레이어의 경로 길이 `>= 3`, 각 경로에 `personal_goal`/`info_gathering`/`dialogue` 액션이 각각 최소 1개 포함 | FR-093 AC2 |
| L2-SV-013 | 종료 판정 기준 자동 평가 가능성 검증 | 정상 world fixture | 검증 실행 | `gameStructureValidation.autoEvaluable===true`, `nonEvaluableEndConditionIds.length===0`, 모든 `endConditions[*].triggerCriteria`가 key 개수 `>= 1`인 객체 | FR-093 AC3 |
| L2-SV-014 | 종료 조건 필요 시스템 명시 검증 | 정상 world fixture | `endConditions.triggerType`와 `gameStructure.requiredSystems` 비교 검증 실행 | `requiredSystemsCoverage.missing.length===0`, `requiredSystemsCoverage.extra.length===0`, `triggerType==='vote'` 조건 존재 시 `requiredSystems`에 `vote` 포함, `consensus`/`ai_judgment`도 동일 규칙 충족 | FR-093 AC4 |
| L2-SV-015 | 검증 실패 후 부분 재생성 성공 플로우 (Step 1~4) | Step1: 의도적 결함 world 생성(`clues`만 실패하도록 주입) | Step2 서버 내부 검증 실행 -> Step3 서버 내부 부분 재생성(`components:['clues']`) 자동 트리거 -> Step4 재검증 실행 | Step2에서 `verification_report.status==='failed'` + `failedComponents===['clues']`; Step3에서 재생성 대상이 `['clues']`로 정확히 1개; 재생성 전후 `hash(map,gameStructure,characters,information)` 동일 + `hash(clues)`만 변경; Step4에서 `verification_report.status==='passed'` | FR-025 AC1, AC2 |
| L2-SV-016 | 부분 재생성 3회 실패 후 전체 재생성/세션 종료 (Step 5) | 의도적으로 해결 불가능한 결함 world fixture, 부분 재생성으로 해결 불가 | 검증 실패 -> 부분 재생성/재검증 루프 3회 -> 전체 재생성 1회 -> 재검증 실패 | `partialRegenerationAttempts===3`, 3회 실패 직후 서버 내부 전체 재생성 자동 트리거(`reason==='PARTIAL_RETRY_EXHAUSTED'`), 전체 재생성 후에도 `verification_report.status==='failed'`, 최종 `error.code==='WORLD_VERIFICATION_RECOVERY_FAILED'`, `game_cancelled.reason`에 `STORY_VERIFICATION_FAILED` 포함 | FR-025 AC3, AC4 |

---

### 멀티 모델 품질 검증 Assertions

| ID | 시나리오 | 사전조건 | 에이전트 행동 | 검증 (필드 레벨) | 관련 FR AC |
|----|---------|---------|-------------|------------------|------------|
| L2-SV-017 | 멀티 모델 생성 결과의 톤 일관성 | `qualityMode=premium`, playerCount=4로 세계 생성 완료, Orchestrator 내부 중간 산출물(SeedProposal, ConflictDesign, CastDesign, MapClueDesign) 및 최종 WorldGeneration 캡처 | LLM evaluator에 중간 산출물 + 최종 WorldGeneration 전달 | LLM evaluator 평가: `toneConsistency >= 4` (1-5 스케일), `worldviewContradictions.length === 0`, 각 중간 산출물의 `genre`/`setting`/`atmosphere` 키워드가 최종 WorldGeneration의 대응 필드와 의미적으로 일치 (LLM 판정) | FR-009 AC2, FR-020 AC1 |
| L2-SV-018 | 품질 모드별 스토리 품질 비교 | playerCount=4로 `qualityMode=fast` 5회 + `qualityMode=premium` 5회 세계 생성 | LLM evaluator가 각 WorldGeneration을 engagement/coherence/creativity 각 1-5로 평가 | `premium` 5회 평균 engagement ≥ `fast` 5회 평균 engagement, `premium` 5회 평균 coherence ≥ `fast` 5회 평균 coherence, 두 모드 모두 `coherence` 평균 ≥ 3.0 | FR-009 AC1 |

---

## L1 참조

없음 — 스토리 검증은 생성된 world JSON에 대한 L2 레벨 검증으로, L1 단위 명령 테스트 없음.

---

## L3 참조

- FG-*: 모든 풀 게임에서 실제 AI가 생성한 스토리로 플레이하며 간접 검증
- 스토리 품질 평가는 L3 `AgentQAReport.storyEvaluation`(coherence, engagement, pacing)에서 정량 확인

---

## L4 체크리스트

> L4 subagent가 TUI 플레이 중 아래 항목에 해당하는 상황을 만나면 **반드시 검증하고** PlayExperienceReport의 `checklist` 필드에 기록한다.
> 자유 플레이를 유지하되, 체크리스트 항목을 의도적으로 커버하는 행동을 포함한다.

| 원본 ID | 체크 항목 | TUI 검증 방법 | Pass 기준 |
|---------|---------|--------------|----------|
| L2-SV-001 | 스토리 논리적 정합성 체감 | 플레이 중 스토리 모순 관찰 | 플레이 중 명백한 시간선/관계 모순이 느껴지지 않음 |
| L2-SV-004 | 복선 회수 체감 | 게임 후반부 관찰 | 초반 단서/복선이 후반에 의미 있게 연결됨 |

### 정성 평가 (기존 L4 참조)

- L4-001~004: subagent가 실제 플레이 후 `PlayExperienceReport.story` 항목으로 정성 평가
- L4는 코딩된 assertion이 아닌 체험 기반 판단이므로, 구조적으로 통과했지만 플레이 감각이 낮은 케이스를 발견 가능

---

## AI Quality Eval 참조

- **WorldGen Eval**: 세계 생성을 N회 실행하여 구조적 품질 독립 평가 (장르 다양성, 역할 긴장 관계, 정보 레이어 균형)
- Phase 2B 구조 검증과 달리 WorldGen Eval은 **품질/흥미도**를 평가 (모순 검사가 아닌 플레이 재미 중심)
