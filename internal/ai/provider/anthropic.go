package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type AnthropicProvider struct {
	client anthropic.Client
	model  anthropic.Model
}

func NewAnthropicProvider(apiKey string, model string) *AnthropicProvider {
	if model == "" {
		model = string(anthropic.ModelClaudeSonnet4_20250514)
	}
	return &AnthropicProvider{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:  anthropic.Model(model),
	}
}

func (p *AnthropicProvider) GenerateStructured(ctx context.Context, req StructuredRequest) (json.RawMessage, error) {
	augmentedSystem := req.SystemPrompt + "\n\n[Output Format] You MUST output only a valid JSON object. No explanatory text, no markdown code blocks, just JSON."
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
	jsonStr, err := ExtractJSON(resp.Content[0].Text)
	if err != nil {
		return nil, fmt.Errorf("failed to extract JSON from Anthropic response: %w", err)
	}
	return json.RawMessage(jsonStr), nil
}

func (p *AnthropicProvider) GenerateText(ctx context.Context, req TextRequest) (string, error) {
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
