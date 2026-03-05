package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"
)

// AIProvider abstracts AI provider implementations.
type AIProvider interface {
	GenerateStructured(ctx context.Context, req StructuredRequest) (json.RawMessage, error)
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

// ProviderConfig holds configuration for creating a provider.
type ProviderConfig struct {
	Type    string `json:"type"`    // "openai" | "anthropic" | "gemini" | "grok" | "deepseek"
	APIKey  string `json:"apiKey"`
	Model   string `json:"model"`
	BaseURL string `json:"baseURL"`
}

// ProviderRegistry manages named AI providers.
type ProviderRegistry struct {
	providers map[string]AIProvider
}

func NewProviderRegistry(configs map[string]ProviderConfig) (*ProviderRegistry, error) {
	registry := &ProviderRegistry{
		providers: make(map[string]AIProvider),
	}
	for name, cfg := range configs {
		p, err := NewProviderFromConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("provider '%s' init failed: %w", name, err)
		}
		registry.providers[name] = p
	}
	return registry, nil
}

func (r *ProviderRegistry) Get(name string) (AIProvider, error) {
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("unregistered provider: %s", name)
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

func (r *ProviderRegistry) HealthCheck(ctx context.Context) map[string]bool {
	results := make(map[string]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for name, p := range r.providers {
		wg.Add(1)
		go func(n string, provider AIProvider) {
			defer wg.Done()
			pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			_, err := provider.GenerateText(pingCtx, TextRequest{
				SystemPrompt: "Reply OK",
				UserPrompt:   "ping",
				MaxTokens:    5,
				Temperature:  0,
			})
			mu.Lock()
			results[n] = (err == nil)
			mu.Unlock()
		}(name, p)
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

// NewProviderFromConfig creates a provider from configuration.
func NewProviderFromConfig(cfg ProviderConfig) (AIProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key required for provider type: %s", cfg.Type)
	}
	switch cfg.Type {
	case "openai":
		return NewOpenAIProvider(cfg.APIKey, cfg.Model), nil
	case "anthropic":
		return NewAnthropicProvider(cfg.APIKey, cfg.Model), nil
	case "gemini":
		return NewGeminiProvider(cfg.APIKey, cfg.Model)
	case "grok":
		return NewGrokProvider(cfg.APIKey, cfg.Model), nil
	case "deepseek":
		return NewDeepSeekProvider(cfg.APIKey, cfg.Model), nil
	default:
		return nil, fmt.Errorf("unknown provider type: %s", cfg.Type)
	}
}

// NewAIProvider creates a provider by type and API key.
func NewAIProvider(providerType string, apiKey string) (AIProvider, error) {
	return NewProviderFromConfig(ProviderConfig{Type: providerType, APIKey: apiKey})
}

// RetryOptions configures retry behavior.
type RetryOptions struct {
	MaxRetries int
	BackoffMs  time.Duration
}

var DefaultRetryOptions = RetryOptions{
	MaxRetries: 3,
	BackoffMs:  1000 * time.Millisecond,
}

// WithRetry retries a function with exponential backoff.
func WithRetry(ctx context.Context, fn func() error, opts RetryOptions) error {
	var lastErr error
	for attempt := 0; attempt <= opts.MaxRetries; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if attempt == opts.MaxRetries {
			return lastErr
		}
		slog.Warn("retrying AI call", "attempt", attempt+1, "error", lastErr)
		backoff := opts.BackoffMs * time.Duration(math.Pow(2, float64(attempt)))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
	}
	return errors.New("unreachable")
}

// extractJSON extracts JSON from raw text response.
func ExtractJSON(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if json.Valid([]byte(trimmed)) {
		return trimmed, nil
	}

	// Try ```json ... ``` block
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

	// Try ``` ... ``` block
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

	// Try first { ... } block
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start != -1 && end != -1 && end > start {
		candidate := trimmed[start : end+1]
		if json.Valid([]byte(candidate)) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("no valid JSON found (raw length: %d)", len(raw))
}
