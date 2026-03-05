package provider

import (
	"context"
	"encoding/json"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
)

type OpenAIProvider struct {
	client openai.Client
	model  openai.ChatModel
}

func NewOpenAIProvider(apiKey string, model string) *OpenAIProvider {
	if model == "" {
		model = "gpt-5-mini"
	}
	return &OpenAIProvider{
		client: openai.NewClient(option.WithAPIKey(apiKey)),
		model:  openai.ChatModel(model),
	}
}

func (p *OpenAIProvider) GenerateStructured(ctx context.Context, req StructuredRequest) (json.RawMessage, error) {
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
		Temperature:         openai.Float(req.Temperature),
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
		Temperature:         openai.Float(req.Temperature),
	})
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}
