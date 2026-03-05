package provider

import (
	"context"
	"encoding/json"

	"google.golang.org/genai"
)

type GeminiProvider struct {
	client *genai.Client
	model  string
}

func NewGeminiProvider(apiKey string, model string) (*GeminiProvider, error) {
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
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
	contents := []*genai.Content{
		{Parts: []*genai.Part{{Text: req.UserPrompt}}},
	}
	temp := float32(req.Temperature)
	result, err := p.client.Models.GenerateContent(ctx, p.model, contents, &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: req.SystemPrompt}},
		},
		ResponseMIMEType: "application/json",
		Temperature:      &temp,
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
	temp := float32(req.Temperature)
	result, err := p.client.Models.GenerateContent(ctx, p.model, contents, &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: req.SystemPrompt}},
		},
		Temperature:     &temp,
		MaxOutputTokens: int32(req.MaxTokens),
	})
	if err != nil {
		return "", err
	}
	return result.Text(), nil
}
