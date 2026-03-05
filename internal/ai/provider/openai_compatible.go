package provider

import (
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

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
