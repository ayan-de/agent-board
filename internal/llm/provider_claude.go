package llm

import (
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

func init() {
	RegisterProvider("claude", func(model, apiKey, baseURL string) (llms.Model, error) {
		opts := []anthropic.Option{}
		if apiKey != "" {
			opts = append(opts, anthropic.WithToken(apiKey))
		}
		if baseURL != "" {
			opts = append(opts, anthropic.WithBaseURL(baseURL))
		}
		if model != "" {
			opts = append(opts, anthropic.WithModel(model))
		}
		return anthropic.New(opts...)
	}, ProviderInfo{
		DisplayName: "Claude (Anthropic)",
		Description: "Claude models via Anthropic API",
		RequiresKey: true,
		Models: []ModelInfo{
			{ID: "claude-sonnet-4-20250514", DisplayName: "Claude Sonnet 4"},
			{ID: "claude-opus-4-20250514", DisplayName: "Claude Opus 4"},
			{ID: "claude-3.5-haiku-20241022", DisplayName: "Claude 3.5 Haiku"},
			{ID: "claude-3.5-sonnet-20241022", DisplayName: "Claude 3.5 Sonnet"},
		},
	})
}
