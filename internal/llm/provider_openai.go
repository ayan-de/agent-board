package llm

import (
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

func init() {
	RegisterProvider("openai", func(model, apiKey, baseURL string) (llms.Model, error) {
		opts := []openai.Option{}
		if apiKey != "" {
			opts = append(opts, openai.WithToken(apiKey))
		}
		if baseURL != "" {
			opts = append(opts, openai.WithBaseURL(baseURL))
		}
		if model != "" {
			opts = append(opts, openai.WithModel(model))
		}
		return openai.New(opts...)
	}, ProviderInfo{
		DisplayName: "OpenAI",
		Description: "GPT models via OpenAI API",
		RequiresKey: true,
		Models: []ModelInfo{
			{ID: "gpt-4o", DisplayName: "GPT-4o"},
			{ID: "gpt-4o-mini", DisplayName: "GPT-4o Mini"},
			{ID: "gpt-4.1", DisplayName: "GPT-4.1"},
			{ID: "gpt-4.1-mini", DisplayName: "GPT-4.1 Mini"},
			{ID: "gpt-4.1-nano", DisplayName: "GPT-4.1 Nano"},
			{ID: "o3", DisplayName: "o3"},
			{ID: "o4-mini", DisplayName: "o4 Mini"},
		},
	})
}
