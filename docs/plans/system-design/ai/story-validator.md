# StoryValidator (`internal/ai/validator/`)

## 책임

세계 생성 결과의 **구조적 결함만 검사**. 창의성이나 개연성은 판단하지 않음. AI 호출 없이 규칙 기반으로 동작.

## 의존하는 모듈

없음 (순수 로직)

## 인터페이스

```go
// internal/ai/validator/story_validator.go

type StoryValidator struct{}

func NewStoryValidator() *StoryValidator {
    return &StoryValidator{}
}

func (sv *StoryValidator) ValidateStructure(world *World) ValidationResult

type ValidationResult struct {
    Valid  bool
    Errors []ValidationError
}

type ValidationError struct {
    Category    string   // "map" | "clue" | "npc" | "end_condition" | "player_path" | "information"
    Severity    string   // "critical" | "warning"
    Message     string
    AffectedIDs []string // 부분 재생성 시 어디를 고칠지
}
```

## 검증 규칙

### 1. 맵 정합성 (`checkMapConsistency`)

| 검사 항목 | severity | 설명 |
|-----------|----------|------|
| 모든 방이 연결 그래프에서 도달 가능 | critical | 고립된 방이 없어야 함 |
| 최소 방 수 = 플레이어 수 + 2 | critical | 밀담을 위한 충분한 방 |
| Connection에 참조된 roomId가 모두 유효 | critical | 존재하지 않는 방 참조 방지 |
| 공개 공간과 밀실이 각각 1개 이상 | warning | 게임 메커닉 다양성 |

```go
func (sv *StoryValidator) checkMapConsistency(world *World) []ValidationError {
    var errors []ValidationError

    roomIDs := make(map[string]bool)
    for _, r := range world.Map.Rooms {
        roomIDs[r.ID] = true
    }

    // 연결 그래프 도달 가능성 (BFS)
    visited := bfs(world.Map.Rooms[0].ID, world.Map.Connections)
    var unreachable []string
    var unreachableNames []string
    for _, r := range world.Map.Rooms {
        if !visited[r.ID] {
            unreachable = append(unreachable, r.ID)
            unreachableNames = append(unreachableNames, r.Name)
        }
    }
    if len(unreachable) > 0 {
        errors = append(errors, ValidationError{
            Category:    "map",
            Severity:    "critical",
            Message:     fmt.Sprintf("고립된 방: %s", strings.Join(unreachableNames, ", ")),
            AffectedIDs: unreachable,
        })
    }

    // 최소 방 수
    // Connection의 roomId 유효성
    // 공개/밀실 구분
    return errors
}
```

### 2. 단서 배치 (`checkCluePlacement`)

| 검사 항목 | severity |
|-----------|----------|
| 모든 단서가 존재하는 방에 배치 | critical |
| 최소 단서 수 >= 플레이어 수 * 2 | critical |
| 단서의 relatedClueIds가 유효한 clue ID를 참조 | critical |

### 3. NPC 정합성 (`checkNPCConsistency`)

| 검사 항목 | severity |
|-----------|----------|
| NPC 위치가 유효한 방 | critical |
| NPC knownInfo가 비어있지 않음 | warning |
| NPC 기믹의 트리거 조건이 논리적으로 존재 | warning |

### 4. 종료 조건 도달 가능성 (`checkEndConditionReachability`)

| 검사 항목 | severity |
|-----------|----------|
| 최소 1개의 종료 조건 존재 | critical |
| timeout fallback 종료 조건 존재 | critical |
| requiredSystems에 맞는 종료 조건 타입 존재 | critical |
| 종료 판정 기준이 명확하고 자동 평가 가능 (evaluationCriteria 필드 비어있지 않음) | critical |

### 5. 플레이어 행동 경로 (`checkPlayerActionPaths`)

| 검사 항목 | severity |
|-----------|----------|
| 모든 플레이어에게 personalGoals가 1개 이상 | critical |
| 모든 플레이어에게 secret이 존재 | critical |
| playerRoles 수 == 플레이어 수 | critical |
| 모든 플레이어에게 의미 있는 행동 경로(개인 목표 추구, 정보 수집, 대화 등)가 존재 | critical |

"의미 있는 행동 경로" 검증은 다음 규칙 기반 근사치로 구현:

```go
func (sv *StoryValidator) checkPlayerActionPaths(world *World) []ValidationError {
    var errors []ValidationError

    // 유효한 엔티티 ID 집합 구성
    roomIDs := make(map[string]bool)
    for _, r := range world.Map.Rooms {
        roomIDs[r.ID] = true
    }
    clueIDs := make(map[string]bool)
    for _, c := range world.Clues {
        clueIDs[c.ID] = true
    }
    npcIDs := make(map[string]bool)
    for _, n := range world.NPCs {
        npcIDs[n.ID] = true
    }
    playerIDs := make(map[string]bool)
    for _, r := range world.PlayerRoles {
        playerIDs[r.ID] = true
    }

    // 맵에서 도달 가능한 방 집합 (BFS)
    reachableRooms := bfs(world.Map.Rooms[0].ID, world.Map.Connections)

    for i, role := range world.PlayerRoles {
        // 규칙 1: 각 플레이어는 personalGoal이 1개 이상이어야 함
        if len(role.PersonalGoals) == 0 {
            errors = append(errors, ValidationError{
                Category:    "player_path",
                Severity:    "critical",
                Message:     fmt.Sprintf("playerRoles[%d](%s)에 개인 목표가 없음", i, role.CharacterName),
                AffectedIDs: []string{role.ID},
            })
        }

        // 규칙 2: 각 플레이어는 비공개 또는 반공개 정보가 1개 이상이어야 함
        hasPrivateInfo := role.Secret != ""
        hasSemiPublicInfo := false
        for _, sp := range world.Information.SemiPublic {
            for _, pid := range sp.TargetPlayerIDs {
                if pid == role.ID {
                    hasSemiPublicInfo = true
                    break
                }
            }
        }
        if !hasPrivateInfo && !hasSemiPublicInfo {
            errors = append(errors, ValidationError{
                Category:    "player_path",
                Severity:    "critical",
                Message:     fmt.Sprintf("playerRoles[%d](%s)에 비공개/반공개 정보가 없음", i, role.CharacterName),
                AffectedIDs: []string{role.ID},
            })
        }

        // 규칙 3: personalGoals가 참조하는 엔티티(NPC, 방, 단서, 다른 플레이어)가 세계에 존재해야 함
        for _, goal := range role.PersonalGoals {
            for _, ref := range goal.EntityRefs {
                if !roomIDs[ref] && !clueIDs[ref] && !npcIDs[ref] && !playerIDs[ref] {
                    errors = append(errors, ValidationError{
                        Category:    "player_path",
                        Severity:    "critical",
                        Message:     fmt.Sprintf("playerRoles[%d](%s) 목표가 존재하지 않는 엔티티 '%s'를 참조", i, role.CharacterName, ref),
                        AffectedIDs: []string{role.ID, ref},
                    })
                }
            }
        }

        // 규칙 4: 각 플레이어가 접근 가능한 방에 단서 또는 NPC가 최소 1개 이상 있어야 함
        hasAccessibleEntity := false
        for _, clue := range world.Clues {
            if reachableRooms[clue.RoomID] {
                hasAccessibleEntity = true
                break
            }
        }
        if !hasAccessibleEntity {
            for _, npc := range world.NPCs {
                if reachableRooms[npc.CurrentRoomID] {
                    hasAccessibleEntity = true
                    break
                }
            }
        }
        if !hasAccessibleEntity {
            errors = append(errors, ValidationError{
                Category:    "player_path",
                Severity:    "critical",
                Message:     fmt.Sprintf("playerRoles[%d](%s)이 접근 가능한 방에 단서 또는 NPC가 없음", i, role.CharacterName),
                AffectedIDs: []string{role.ID},
            })
        }
    }

    return errors
}
```

### 6. 정보 레이어 (`checkInformationLayers`)

| 검사 항목 | severity |
|-----------|----------|
| 공개 정보가 존재 | critical |
| 반공개 정보가 최소 1쌍 | warning |
| 비공개 정보가 플레이어 수만큼 | critical |
| semiPublic의 targetPlayerIds가 유효한 플레이어 | critical |

### 7. 타임라인 일관성 (`checkTimelineConsistency`)

| 검사 항목 | severity |
|-----------|----------|
| 사건 발생 순서가 논리적으로 역행하지 않음 (before/after 관계 명시된 경우) | warning |
| 단서가 참조하는 사건이 해당 단서 발견 이전 시점에 발생했음 | warning |

```go
func (sv *StoryValidator) checkTimelineConsistency(world *World) []ValidationError {
    // 사건 순서 역행, 단서-사건 시간 관계 검사
    // world.Events의 timestamp/order 필드(있는 경우)와 단서의 timeContext 필드를 기반으로 검사
    // AI 생성 결과에 명시적 순서 정보가 없는 경우 이 검사는 경고 없이 통과
    var errors []ValidationError
    // ... 규칙 기반 구현
    return errors
}
```

### 8. 관계 일관성 (`checkRelationshipConsistency`)

| 검사 항목 | severity |
|-----------|----------|
| 플레이어/NPC 관계가 상호 참조 가능 (A가 B를 알면 B의 world 정의에도 A 존재) | warning |
| SemiPublic 정보에서 참조하는 플레이어 ID가 모두 유효 | critical |
| PlayerRole.RelatedNPCs가 실제 존재하는 NPC를 참조 | critical |

```go
func (sv *StoryValidator) checkRelationshipConsistency(world *World) []ValidationError {
    // 플레이어-NPC-플레이어 관계망의 참조 무결성 검사
    // SemiPublic targetPlayerIDs 유효성은 checkInformationLayers와 중복이므로 여기서는 관계 대칭성에 집중
    var errors []ValidationError
    // ... 규칙 기반 구현
    return errors
}
```

## ValidateStructure 전체 흐름

```go
func (sv *StoryValidator) ValidateStructure(world *World) ValidationResult {
    var allErrors []ValidationError

    allErrors = append(allErrors, sv.checkMapConsistency(world)...)
    allErrors = append(allErrors, sv.checkCluePlacement(world)...)
    allErrors = append(allErrors, sv.checkNPCConsistency(world)...)
    allErrors = append(allErrors, sv.checkEndConditionReachability(world)...)
    allErrors = append(allErrors, sv.checkPlayerActionPaths(world)...)
    allErrors = append(allErrors, sv.checkInformationLayers(world)...)
    allErrors = append(allErrors, sv.checkTimelineConsistency(world)...)
    allErrors = append(allErrors, sv.checkRelationshipConsistency(world)...)

    hasCritical := false
    for _, e := range allErrors {
        if e.Severity == "critical" {
            hasCritical = true
            break
        }
    }

    return ValidationResult{
        Valid:  !hasCritical,
        Errors: allErrors,
    }
}
```

## 검증 → 재생성 흐름

Orchestrator.GenerateWorld() 내부 Phase 5에서 관리. 검증 실패 시 SchemaEditor가 부분 패치를 수행한다.

```
WorldGenerator.Generate()
       │
       ▼
StoryValidator.ValidateStructure()
       │
       ├── errors 없음 → 통과
       │
       └── critical errors 있음
             │
             ├── 부분 재생성 시도 (최대 3회)
             │     error.AffectedIDs로 재생성 범위 결정
             │     → 재검증
             │
             └── 3회 실패 → 전체 재생성 (최대 1회)
                   │
                   └── 실패 → 세션 종료 (에러 메시지)
```
