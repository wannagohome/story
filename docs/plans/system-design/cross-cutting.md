# Cross-Cutting Concerns (횡단 관심사)

---

## 1. 에러 처리

### Result 타입

Go는 별도의 Result 타입 없이 다중 반환값으로 에러를 처리한다.

> **참고**: types.md의 Result[T, E] 제네릭 타입은 AI 응답 파싱 등 구조화된 결과 반환에 사용. 일반 함수의 에러 처리는 Go 관용적 다중 반환값 패턴을 따른다.

```go
// Go 관용적 에러 반환 패턴
func doSomething() (Value, error) {
    // 성공 시
    return value, nil
    // 실패 시
    return zero, fmt.Errorf("실패 원인: %w", err)
}
```

### 레이어별 전략

| 레이어 | 에러 유형 | 처리 |
|--------|----------|------|
| **Network** | WebSocket 연결 끊김 | 클라이언트: WebSocket 재연결 시도 (최대 3회, 지수 백오프 1s→2s→4s, 총 ~7초). 서버: FR-007에 따라 30초 유예 후 다른 플레이어에 알림 |
| **Network** | WebSocket 연결 실패 | "서버에 연결할 수 없습니다. 서버 URL과 포트를 확인하거나, 같은 네트워크에서 시도해주세요." |
| **Network** | 메시지 파싱 실패 | 무시 + 로깅 |
| **AI Provider** | API 오류 (rate limit, timeout) | 지수 백오프 재시도 (1s→2s→4s, 최대 3회) |
| **AI Provider** | 프로바이더 불가 (세션 시작 시 HealthCheck) | 해당 프로바이더 제외, 대체 편성 적용. 모든 외부 프로바이더 불가 시 OpenAI 단독 fallback |
| **AI Output** | 스키마 검증 실패 | Orchestrator 내부에서 SchemaEditor 패치 (최대 1회) → 실패 시 전체 재생성 (1회) → 실패 시 세션 종료 |
| **Game Logic** | 잘못된 행동 (이동 불가, 존재하지 않는 NPC) | 에러 메시지 반환, 상태 변경 없음 |
| **Session** | 중복 닉네임, 방 가득 참 | 명확한 에러 코드 + 메시지 |

### 에러 메시지 형식

```go
type ErrorMessage struct {
    Type    string    `json:"type"`    // "error"
    Code    ErrorCode `json:"code"`
    Message string    `json:"message"`
}
```

모든 에러 메시지는 문제와 해결 방법을 명시 (NFR-021).

```
예: "이동할 수 없습니다: '부엌'은(는) 현재 방에서 인접하지 않습니다. /map으로 이동 가능한 방을 확인하세요."
```

---

## 2. 로깅

### 서버 측

Go 1.26 표준 `log/slog` 패키지 사용. 구조화된 JSON 로깅 지원.

```go
// log/slog 기반 구조화 로깅
import "log/slog"

// 모듈별 로거 생성
var logger = slog.Default().With("module", "session")

// 사용 예
logger.Info("플레이어 입장", "playerID", playerID, "nickname", nickname)
logger.Error("AI 호출 실패", "err", err, "attempt", attempt)
```

| 모듈 | 로깅 대상 |
|------|----------|
| NetworkServer | 연결/해제, 메시지 수신 (내용 제외) |
| SessionManager | 상태 전이, 플레이어 입퇴장 |
| GameStateManager | 상태 변이 (변경 유형만) |
| AILayer | AI 호출 시작/완료, 토큰 사용량, 재시도 |
| EndConditionEngine | 종료 조건 평가 결과 |

### 민감 정보 마스킹

- API 키: 절대 로그에 출력하지 않음
- 비공개 정보: 로그에 "[PRIVATE]"로 마스킹
- 플레이어 비밀: 로그에 포함하지 않음

### 클라이언트 측

에러만 로깅. 게임 이벤트 자체가 표시되므로 별도 로깅 불필요.

---

## 3. Graceful Shutdown

```
Ctrl+C (SIGINT) 또는 SIGTERM 수신
    │
    ├── 호스트인 경우:
     │     1. 모든 플레이어에게 종료 알림 전송 (WebSocket)
     │     2. 게임 중이면: 현재 상태 로깅 (데이터 저장은 P1)
     │     3. EndConditionEngine 정리 (타이머 해제)
     │     4. 모든 WebSocket 연결 정리 (close) 및 WebSocket 서버 종료
     │     5. 프로세스 종료
    │
    └── 참가자인 경우:
          1. 서버에 연결 해제 알림 (WebSocket, 가능하면)
          2. WebSocket 연결 종료
          3. TUI 정리 (터미널 상태 복원)
          4. 프로세스 종료
```

### 구현

```go
func setupGracefulShutdown(server *StoryServer) {
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

    go func() {
        <-sigCh
        fmt.Println("종료 중...")
        server.Stop()
        os.Exit(0)
    }()
}
```

---

## 4. AI 토큰 관리

### 세계 생성 (Orchestrator 멀티 모델 파이프라인)

세계 생성은 역할별로 다른 모델을 사용하는 멀티 모델 파이프라인으로 실행됨.
자세한 에이전트 편성은 [orchestrator.md](./ai/orchestrator.md) 참조.

**빠른 제작 모드 예상 비용 (6인 기준):**

| Agent | 입력 토큰 | 출력 토큰 | 모델 | 비용 |
|-------|----------|----------|------|------|
| ChaosMuse ×3 | ~2400 | ~1500 | Gemini Flash-Lite, Grok fast, DeepSeek | ~$0.001 |
| Showrunner (통합) | ~3000 | ~6000 | GPT-5 mini | ~$0.013 |
| ContinuityCop | ~4000 | ~500 | Gemini Flash | ~$0.002 |
| Polish | ~2000 | ~1000 | Claude Haiku 4.5 | ~$0.007 |
| **합계** | | | | **~$0.02** |

**품질 위주 모드 예상 비용 (6인 기준):**

| Agent | 입력 토큰 | 출력 토큰 | 모델 | 비용 |
|-------|----------|----------|------|------|
| ChaosMuse ×2 | ~1600 | ~1000 | Grok 4, DeepSeek reasoner | ~$0.02 |
| ConflictEngineer | ~1500 | ~2000 | Gemini 2.5 Pro | ~$0.02 |
| CastSecretMaster | ~2000 | ~3000 | Claude Opus 4.6 | ~$0.09 |
| MapClueSmith | ~2500 | ~2000 | Gemini 2.5 Pro | ~$0.02 |
| Showrunner | ~5000 | ~6000 | GPT-5 | ~$0.07 |
| ContinuityCop | ~4000 | ~500 | Claude Sonnet 4.6 | ~$0.02 |
| Polish | ~2000 | ~1000 | Claude Opus 4.6 | ~$0.04 |
| **합계** | | | | **~$0.20** |

### 런타임 AI 호출 (단일 runtimeProvider)

| AI 호출 유형 | max_tokens | temperature | 호출 빈도 |
|-------------|-----------|-------------|-----------|
| GM 서술 | 500 | 0.9 | 필요 시 (0~20회/게임) |
| NPC 대화 | 500 | 0.8 | 플레이어 요청 시 |
| /examine | 500 | 0.7 | 플레이어 요청 시 |
| /do | 500 | 0.8 | 플레이어 요청 시 |
| 종료 판정 | 300 | 0.3 | 이벤트 발생 시 |
| 엔딩 생성 | 3000 | 0.9 | 게임당 1회 |

### 예상 토큰 사용량 (30분 게임, 6명, 런타임만)

concept.md 기준 최대 게임 시간은 30분. 횟수를 30분 기준으로 산정.

| 항목 | 입력 토큰 | 출력 토큰 | 횟수 | 소계 |
|------|----------|----------|------|------|
| NPC 대화 | ~1500 | ~300 | ~15 | ~27000 |
| /examine | ~1000 | ~300 | ~10 | ~13000 |
| /do | ~1000 | ~300 | ~8 | ~10400 |
| GM 서술 | ~1500 | ~300 | ~5 | ~9000 |
| 종료 판정 | ~1000 | ~200 | ~3 | ~3600 |
| 엔딩 생성 | ~3000 | ~2000 | 1 | ~5000 |
| **합계** | | | | **~68,000** |

### 비용 최적화

- **Story Bible 캐시**: concept/PRD를 3종 요약본으로 압축하여 매 프롬프트에 원본 삽입 방지 (~1800 토큰으로 축약)
- 방 설명 등 반복 조회 가능 데이터: 캐싱 (AI 재호출 방지)
- 게임 컨텍스트: 최근 20개 이벤트만 포함 (전체 이력 X)
- NPC 대화 이력: 최대 20턴 (오래된 대화 자름)
- 종료 판정: 규칙 기반 사전 필터 후 AI 호출 (불필요한 호출 방지)
- Orchestrator seed 병렬 생성: 가장 싼 모델들로 seed를 뽑아 비용 절감

---

## 5. 보안

### 정보 격리 (NFR-014)

- **서버 사이드 필터링이 유일한 방어선.** 클라이언트는 신뢰하지 않음.
- MessageRouter가 모든 아웃바운드 메시지의 수신자를 결정.
- 비공개 정보는 해당 플레이어에게만 전송.

### API 키 보호 (NFR-013, NFR-017)

```
저장: ~/.story/config.json (파일 권한 0600)
전달: 환경변수 STORY_OPENAI_KEY 또는 설정 파일
노출 방지: 로그 마스킹, 네트워크 전송 금지
```

### 입력 검증 (NFR-015)

- 닉네임: 1~20자, 제어문자 제거
- 채팅: 최대 500자
- 명령어 인자: 최대 200자
- 모든 문자열: trim 후 처리

### 네트워크 보안 (NFR-018)

- WebSocket: 프로덕션 환경에서 WSS(WebSocket Secure, TLS) 사용. 개발/로컬 환경에서는 WS(비암호화) 허용.
- 클라이언트-서버: Host가 모든 메시지를 필터링하므로 정보 비대칭 보장.

---

## 6. Go 1.26 기능 활용

| 기능 | 적용 위치 | 설명 |
|------|----------|------|
| **Green Tea GC** | 전체 | GC 오버헤드 10~40% 감소. 별도 설정 없이 자동 적용 |
| **`errors.AsType[T]()`** | 에러 처리 전역 | 제네릭 기반 에러 타입 단언. `var target *MyError; errors.As(err, &target)` → `if e, ok := errors.AsType[*MyError](err); ok { ... }` |
| **post-quantum TLS** | WSS 연결 | Go 1.26에서 post-quantum TLS 기본 활성화. 별도 설정 없이 적용 |
| **`log/slog`** | 서버 로깅 | 표준 라이브러리 구조화 로깅. 위 로깅 섹션 참조 |
