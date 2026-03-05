package provider

import (
	"context"
	"encoding/json"
	"testing"
)

func TestExtractJSON_DirectJSON(t *testing.T) {
	input := `{"key": "value"}`
	result, err := ExtractJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != input {
		t.Errorf("expected %q, got %q", input, result)
	}
}

func TestExtractJSON_MarkdownJSONBlock(t *testing.T) {
	input := "some text\n```json\n{\"key\": \"value\"}\n```\nmore text"
	result, err := ExtractJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"key": "value"}`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestExtractJSON_MarkdownBlock(t *testing.T) {
	input := "text\n```\n{\"a\": 1}\n```\n"
	result, err := ExtractJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"a": 1}`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestExtractJSON_BraceExtraction(t *testing.T) {
	input := "Here is the result: {\"foo\": \"bar\"} end"
	result, err := ExtractJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"foo": "bar"}`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestExtractJSON_NoJSON(t *testing.T) {
	input := "no json here at all"
	_, err := ExtractJSON(input)
	if err == nil {
		t.Fatal("expected error for non-JSON input")
	}
}

func TestProviderConfig_Factory(t *testing.T) {
	// Test unknown provider type
	_, err := NewProviderFromConfig(ProviderConfig{Type: "unknown", APIKey: "key"})
	if err == nil {
		t.Fatal("expected error for unknown provider type")
	}

	// Test empty API key
	_, err = NewProviderFromConfig(ProviderConfig{Type: "openai", APIKey: ""})
	if err == nil {
		t.Fatal("expected error for empty API key")
	}
}

func TestNewAIProvider(t *testing.T) {
	_, err := NewAIProvider("unknown", "key")
	if err == nil {
		t.Fatal("expected error for unknown provider type")
	}
}

func TestProviderRegistry(t *testing.T) {
	configs := map[string]ProviderConfig{
		"main": {Type: "openai", APIKey: "test-key", Model: "gpt-4o-mini"},
	}
	registry, err := NewProviderRegistry(configs)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Test Get
	p, err := registry.Get("main")
	if err != nil {
		t.Fatalf("failed to get provider: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}

	// Test Get unknown
	_, err = registry.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent provider")
	}

	// Test Available
	names := registry.Available()
	if len(names) != 1 {
		t.Fatalf("expected 1 available provider, got %d", len(names))
	}

	// Test MustGet panics for unknown
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic from MustGet with unknown provider")
		}
	}()
	registry.MustGet("nonexistent")
}

func TestRetryOptions(t *testing.T) {
	attempts := 0
	err := WithRetry(context.Background(), func() error {
		attempts++
		if attempts < 3 {
			return context.DeadlineExceeded
		}
		return nil
	}, RetryOptions{MaxRetries: 3, BackoffMs: 1})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryOptions_AllFail(t *testing.T) {
	err := WithRetry(context.Background(), func() error {
		return context.DeadlineExceeded
	}, RetryOptions{MaxRetries: 2, BackoffMs: 1})
	if err == nil {
		t.Fatal("expected error after all retries exhausted")
	}
}

func TestRetryOptions_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := WithRetry(ctx, func() error {
		return context.DeadlineExceeded
	}, RetryOptions{MaxRetries: 5, BackoffMs: 100})
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// MockProvider for testing
type MockProvider struct {
	structuredResp json.RawMessage
	textResp       string
	err            error
}

func (m *MockProvider) GenerateStructured(_ context.Context, _ StructuredRequest) (json.RawMessage, error) {
	return m.structuredResp, m.err
}

func (m *MockProvider) GenerateText(_ context.Context, _ TextRequest) (string, error) {
	return m.textResp, m.err
}

func TestMockProvider(t *testing.T) {
	mock := &MockProvider{
		structuredResp: json.RawMessage(`{"test": true}`),
		textResp:       "hello",
	}

	resp, err := mock.GenerateStructured(context.Background(), StructuredRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(resp) != `{"test": true}` {
		t.Errorf("unexpected response: %s", resp)
	}

	text, err := mock.GenerateText(context.Background(), TextRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "hello" {
		t.Errorf("unexpected response: %s", text)
	}
}
