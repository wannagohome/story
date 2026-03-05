# AIProvider (`internal/ai/provider/`)

## 책임

AI 프로바이더별 구현을 추상화. **교체 가능한 어댑터 패턴.** 멀티 프로바이더 레지스트리를 통해 역할별 최적 모델 배정.

## 인터페이스

```go
// internal/ai/provider/provider.go

// AIProvider는 AI 프로바이더 추상화 인터페이스.
// 구조화 출력과 자유 텍스트 생성 두 가지 모드를 지원.
type AIProvider interface {
    // GenerateStructured는 JSON 모드로 구조화된 출력을 반환.
    // 호출자가 json.Unmarshal + Validate()로 처리.
    GenerateStructured(ctx context.Context, req StructuredRequest) (json.RawMessage, error)

    // GenerateText는 엔딩 서술 등 자유 형식 텍스트를 반환.
    // 내부 중간 처리용. UI에 직접 노출되지 않으며, 최종 출력은 항상 구조화된 JSON(GenerateStructured)을 통해 전달됨. concept.md/PRD의 '자유 형식 텍스트 출력 금지' 원칙은 클라이언트 전달 기준.
    GenerateText(ctx context.Context, req TextRequest) (string, error)
}

type StructuredRequest struct {
    SystemPrompt string
    UserPrompt   string
    MaxTokens    int
    Temperature  float64
}

type TextRequest struct {
    SystemPrompt string
    UserPrompt   string
    MaxTokens    int
    Temperature  float64
}
```

## 구현체

### OpenAI Provider

```go
// internal/ai/provider/openai.go
// 패키지 import: github.com/openai/openai-go/v3
//                github.com/openai/openai-go/v3/option
//                github.com/openai/openai-go/v3/shared

type OpenAIProvider struct {
    client *openai.Client
    model  openai.ChatModel
}

func NewOpenAIProvider(apiKey string) *OpenAIProvider {
    return &OpenAIProvider{
        client: openai.NewClient(option.WithAPIKey(apiKey)),
        model:  openai.ChatModel("gpt-5-mini"),
    }
}

func (p *OpenAIProvider) GenerateStructured(ctx context.Context, req StructuredRequest) (json.RawMessage, error) {
    // OpenAI JSON object mode로 structured output 보장
    resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
        Model: p.model,
        Messages: []openai.ChatCompletionMessageParamUnion{
            openai.SystemMessage(req.SystemPrompt),
            openai.UserMessage(req.UserPrompt),
        },
        ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
            OfJSONObject: &shared.ResponseFormatJSONObjectParam{},
        },
        MaxCompletionTokens: openai.Int(int64(req.MaxTokens)),
        Temperature: openai.Float(req.Temperature),
    })
    if err != nil {
        return nil, err
    }
    return json.RawMessage(resp.Choices[0].Message.Content), nil
}

func (p *OpenAIProvider) GenerateText(ctx context.Context, req TextRequest) (string, error) {
    resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
        Model: p.model,
        Messages: []openai.ChatCompletionMessageParamUnion{
            openai.SystemMessage(req.SystemPrompt),
            openai.UserMessage(req.UserPrompt),
        },
        MaxCompletionTokens: openai.Int(int64(req.MaxTokens)),
        Temperature: openai.Float(req.Temperature),
    })
    if err != nil {
        return "", err
    }
    return resp.Choices[0].Message.Content, nil
}
```

### Anthropic Provider

> **주의:** Anthropic은 OpenAI의 JSON mode에 해당하는 구조화 출력을 네이티브로 보장하지 않습니다.
> MVP에서 1개 프로바이더만 지원한다면 **OpenAI를 기본으로 권장**합니다.
> Anthropic을 사용할 경우 아래의 `extractJSON` fallback이 필수입니다.
> 대안으로 Anthropic의 tool use(function calling) API를 사용하면 구조화 출력을 더 안정적으로 보장할 수 있습니다.
>
> **참고 (Claude 4.6 breaking change):** Claude 4.6 모델(`claude-opus-4-6`, `claude-sonnet-4-6`)은
> assistant message prefill을 지원하지 않습니다 (400 에러 반환).
> JSON 출력 유도 시 prefill 대신 system prompt 지시 또는 tool use를 사용해야 합니다.

```go
// internal/ai/provider/anthropic.go
// 패키지 import: github.com/anthropics/anthropic-sdk-go
//                github.com/anthropics/anthropic-sdk-go/option

type AnthropicProvider struct {
    client *anthropic.Client
    model  anthropic.Model
}

func NewAnthropicProvider(apiKey string) *AnthropicProvider {
    return &AnthropicProvider{
        client: anthropic.NewClient(option.WithAPIKey(apiKey)),
        model:  anthropic.ModelClaudeSonnet4_20250514,
    }
}

func (p *AnthropicProvider) GenerateStructured(ctx context.Context, req StructuredRequest) (json.RawMessage, error) {
    // Anthropic은 JSON mode 미지원. system prompt에 JSON 출력 지시를 추가하고
    // 응답 텍스트를 extractJSON()으로 파싱한 후 반환.
    augmentedSystem := req.SystemPrompt + "\n\n[출력 형식] 반드시 유효한 JSON 객체만 출력하세요. 설명 텍스트나 마크다운 코드 블록 없이 JSON만 반환하세요."
    resp, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
        Model:     p.model,
        MaxTokens: int64(req.MaxTokens),
        System: []anthropic.TextBlockParam{
            {Text: augmentedSystem},
        },
        Messages: []anthropic.MessageParam{
            anthropic.NewUserMessage(anthropic.NewTextBlock(req.UserPrompt)),
        },
    })
    if err != nil {
        return nil, err
    }
    jsonStr, err := extractJSON(resp.Content[0].Text)
    if err != nil {
        return nil, fmt.Errorf("anthropic 응답에서 JSON 추출 실패: %w", err)
    }
    return json.RawMessage(jsonStr), nil
}

func (p *AnthropicProvider) GenerateText(ctx context.Context, req TextRequest) (string, error) {
    // GenerateStructured와 달리 JSON 강제 지시 없이 자유 텍스트 반환.
    resp, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
        Model:     p.model,
        MaxTokens: int64(req.MaxTokens),
        System: []anthropic.TextBlockParam{
            {Text: req.SystemPrompt},
        },
        Messages: []anthropic.MessageParam{
            anthropic.NewUserMessage(anthropic.NewTextBlock(req.UserPrompt)),
        },
    })
    if err != nil {
        return "", err
    }
    return resp.Content[0].Text, nil
}

// extractJSON은 원시 문자열에서 JSON을 추출하는 fallback 헬퍼.
// 시도 순서:
//  1. 직접 파싱 (json.Valid)
//  2. ```json ... ``` 마크다운 코드 블록에서 추출
//  3. 첫 번째 { ... } 블록 추출
func extractJSON(raw string) (string, error) {
    // 1. 직접 파싱
    trimmed := strings.TrimSpace(raw)
    if json.Valid([]byte(trimmed)) {
        return trimmed, nil
    }

    // 2. ```json ... ``` 블록 추출
    if idx := strings.Index(trimmed, "```json"); idx != -1 {
        start := idx + len("```json")
        end := strings.Index(trimmed[start:], "```")
        if end != -1 {
            candidate := strings.TrimSpace(trimmed[start : start+end])
            if json.Valid([]byte(candidate)) {
                return candidate, nil
            }
        }
    }
    // ``` 블록 (언어 태그 없음)
    if idx := strings.Index(trimmed, "```"); idx != -1 {
        start := idx + len("```")
        end := strings.Index(trimmed[start:], "```")
        if end != -1 {
            candidate := strings.TrimSpace(trimmed[start : start+end])
            if json.Valid([]byte(candidate)) {
                return candidate, nil
            }
        }
    }

    // 3. 첫 번째 { ... } 블록 추출
    start := strings.Index(trimmed, "{")
    end := strings.LastIndex(trimmed, "}")
    if start != -1 && end != -1 && end > start {
        candidate := trimmed[start : end+1]
        if json.Valid([]byte(candidate)) {
            return candidate, nil
        }
    }

    return "", fmt.Errorf("유효한 JSON을 찾을 수 없음 (raw 길이: %d)", len(raw))
}
```

## 추가 프로바이더

### Gemini Provider

```go
// internal/ai/provider/gemini.go
// 패키지 import: google.golang.org/genai v1.49.0 (Google AI Go SDK)

type GeminiProvider struct {
    client *genai.Client
    model  string
}

func NewGeminiProvider(apiKey string, model string) (*GeminiProvider, error) {
    client, err := genai.NewClient(ctx, &genai.ClientConfig{
        APIKey:  apiKey,
        Backend: genai.BackendGeminiAPI,
    })
    if err != nil {
        return nil, err
    }
    if model == "" {
        model = "gemini-2.5-flash"
    }
    return &GeminiProvider{client: client, model: model}, nil
}

func (p *GeminiProvider) GenerateStructured(ctx context.Context, req StructuredRequest) (json.RawMessage, error) {
    // Gemini는 responseMimeType: "application/json"으로 JSON 출력 보장
    contents := []*genai.Content{
        {Parts: []*genai.Part{{Text: req.UserPrompt}}},
    }
    result, err := p.client.Models.GenerateContent(ctx, p.model, contents, &genai.GenerateContentConfig{
        SystemInstruction: &genai.Content{
            Parts: []*genai.Part{{Text: req.SystemPrompt}},
        },
        ResponseMIMEType: "application/json",
        Temperature:      genai.Ptr(float32(req.Temperature)),
        MaxOutputTokens:  int32(req.MaxTokens),
    })
    if err != nil {
        return nil, err
    }
    text := result.Text()
    return json.RawMessage(text), nil
}

func (p *GeminiProvider) GenerateText(ctx context.Context, req TextRequest) (string, error) {
    contents := []*genai.Content{
        {Parts: []*genai.Part{{Text: req.UserPrompt}}},
    }
    result, err := p.client.Models.GenerateContent(ctx, p.model, contents, &genai.GenerateContentConfig{
        SystemInstruction: &genai.Content{
            Parts: []*genai.Part{{Text: req.SystemPrompt}},
        },
        Temperature:     genai.Ptr(float32(req.Temperature)),
        MaxOutputTokens: int32(req.MaxTokens),
    })
    if err != nil {
        return "", err
    }
    return result.Text(), nil
}
```

### OpenAI-Compatible Provider (Grok / DeepSeek)

Grok과 DeepSeek는 OpenAI-compatible API를 제공하므로, `openai-go/v3` SDK에 `WithBaseURL()` 옵션으로 구현. 별도 SDK 불필요.

```go
// internal/ai/provider/openai_compatible.go
// OpenAIProvider를 재활용하되 BaseURL과 모델만 변경

func NewGrokProvider(apiKey string, model string) *OpenAIProvider {
    if model == "" {
        model = "grok-4-fast"
    }
    return &OpenAIProvider{
        client: openai.NewClient(
            option.WithAPIKey(apiKey),
            option.WithBaseURL("https://api.x.ai/v1"),
        ),
        model: openai.ChatModel(model),
    }
}

func NewDeepSeekProvider(apiKey string, model string) *OpenAIProvider {
    if model == "" {
        model = "deepseek-chat"
    }
    return &OpenAIProvider{
        client: openai.NewClient(
            option.WithAPIKey(apiKey),
            option.WithBaseURL("https://api.deepseek.com"),
        ),
        model: openai.ChatModel(model),
    }
}
```

> **참고:** Grok과 DeepSeek는 OpenAI의 JSON mode(`response_format: {type: "json_object"}`)를 지원하므로,
> 기존 `OpenAIProvider.GenerateStructured()`의 JSON object mode가 그대로 동작합니다.

## Provider Registry

멀티 프로바이더 환경에서 이름으로 프로바이더를 조회하는 레지스트리.

```go
// internal/ai/provider/registry.go

type ProviderConfig struct {
    Type    string `json:"type"`    // "openai" | "anthropic" | "gemini" | "grok" | "deepseek"
    APIKey  string `json:"apiKey"`
    Model   string `json:"model"`   // 모델 ID override (선택)
    BaseURL string `json:"baseURL"` // 커스텀 base URL (선택)
}

type ProviderRegistry struct {
    providers map[string]AIProvider // name → provider
}

func NewProviderRegistry(configs map[string]ProviderConfig) (*ProviderRegistry, error) {
    registry := &ProviderRegistry{
        providers: make(map[string]AIProvider),
    }
    for name, cfg := range configs {
        provider, err := newProviderFromConfig(cfg)
        if err != nil {
            return nil, fmt.Errorf("프로바이더 '%s' 초기화 실패: %w", name, err)
        }
        registry.providers[name] = provider
    }
    return registry, nil
}

func (r *ProviderRegistry) Get(name string) (AIProvider, error) {
    p, ok := r.providers[name]
    if !ok {
        return nil, fmt.Errorf("등록되지 않은 프로바이더: %s", name)
    }
    return p, nil
}

func (r *ProviderRegistry) MustGet(name string) AIProvider {
    p, err := r.Get(name)
    if err != nil {
        panic(err)
    }
    return p
}

// HealthCheck는 등록된 프로바이더에 최소 요청을 발송하여 가용성을 확인.
// 세션 시작 시 1회 호출. 실패한 프로바이더는 해당 세션에서 제외.
func (r *ProviderRegistry) HealthCheck(ctx context.Context) map[string]bool {
    results := make(map[string]bool)
    var mu sync.Mutex
    var wg sync.WaitGroup

    for name, provider := range r.providers {
        wg.Add(1)
        go func(n string, p AIProvider) {
            defer wg.Done()
            pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
            defer cancel()
            _, err := p.GenerateText(pingCtx, TextRequest{
                SystemPrompt: "Reply OK",
                UserPrompt:   "ping",
                MaxTokens:    5,
                Temperature:  0,
            })
            mu.Lock()
            results[n] = (err == nil)
            mu.Unlock()
        }(name, provider)
    }
    wg.Wait()
    return results
}

func (r *ProviderRegistry) Available() []string {
    names := make([]string, 0, len(r.providers))
    for name := range r.providers {
        names = append(names, name)
    }
    return names
}
```

## Provider Factory (확장)

```go
// internal/ai/provider/factory.go

func NewAIProvider(providerType string, apiKey string) (AIProvider, error) {
    return newProviderFromConfig(ProviderConfig{Type: providerType, APIKey: apiKey})
}

func newProviderFromConfig(cfg ProviderConfig) (AIProvider, error) {
    switch cfg.Type {
    case "openai":
        p := NewOpenAIProvider(cfg.APIKey)
        if cfg.Model != "" {
            p.model = openai.ChatModel(cfg.Model)
        }
        return p, nil
    case "anthropic":
        p := NewAnthropicProvider(cfg.APIKey)
        if cfg.Model != "" {
            p.model = anthropic.Model(cfg.Model)
        }
        return p, nil
    case "gemini":
        return NewGeminiProvider(cfg.APIKey, cfg.Model)
    case "grok":
        return NewGrokProvider(cfg.APIKey, cfg.Model), nil
    case "deepseek":
        return NewDeepSeekProvider(cfg.APIKey, cfg.Model), nil
    default:
        return nil, fmt.Errorf("알 수 없는 프로바이더: %s", cfg.Type)
    }
}
```

## 재시도 래퍼

모든 AI 호출에 공통 적용.

```go
// internal/ai/provider/retry.go

type RetryOptions struct {
    MaxRetries int
    BackoffMs  time.Duration
}

var defaultRetryOptions = RetryOptions{
    MaxRetries: 3,
    BackoffMs:  1000 * time.Millisecond,
}

func withRetry(ctx context.Context, fn func() error, opts RetryOptions) error {
    for attempt := 0; attempt <= opts.MaxRetries; attempt++ {
        if err := fn(); err == nil {
            return nil
        } else if attempt == opts.MaxRetries {
            return err
        }
        backoff := opts.BackoffMs * time.Duration(math.Pow(2, float64(attempt)))
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(backoff):
        }
    }
    return errors.New("unreachable")
}
```

**지수 백오프:** 1초 → 2초 → 4초. 최대 3회 재시도.

## Go SDK 선택 이유

| SDK | 버전 | 대상 프로바이더 | 비고 |
|-----|------|--------------|------|
| `github.com/openai/openai-go/v3` | v3.24.0 | OpenAI, Grok, DeepSeek | JSON object mode로 구조화 출력 보장. Grok/DeepSeek는 `WithBaseURL()`로 재사용 |
| `github.com/anthropics/anthropic-sdk-go` | v1.26.0 | Anthropic | `extractJSON` fallback 필수. v1.26.0에서 자동 캐싱(cache control), `BetaToolRunner` 추가 |
| `google.golang.org/genai` | v1.49.0 | Gemini | Google AI 공식 Go SDK. `responseMimeType: "application/json"`으로 JSON 보장. `Backend` 필드 필수 지정 (`genai.BackendGeminiAPI`) |

- 커스텀 `AIProvider` interface로 멀티 프로바이더를 동일 인터페이스로 사용
- `json.RawMessage` + `json.Unmarshal` + `Validate()` 패턴으로 타입 안전성 확보
- 스트리밍 지원: 향후 스트리밍 API 전환 시 구현체만 교체

## 프로바이더별 모델 기본값

| Provider | 기본 모델 | 용도 |
|----------|----------|------|
| OpenAI | `gpt-5-mini` | Showrunner, SchemaEditor, 런타임 AI |
| Anthropic | `claude-sonnet-4-6` | Prestige Writer, Critic, Polish |
| Gemini | `gemini-2.5-flash` | Conflict Engineer, Validator, Fast Seed |
| Grok | `grok-4-fast` | Chaos Muse |
| DeepSeek | `deepseek-chat` | Cheap Counter-Voice, Seed |

> **최소 요구:** OpenAI API 키 1개 (fallback용). 다른 프로바이더는 선택적이며, 없으면 해당 역할은 OpenAI가 대체.
