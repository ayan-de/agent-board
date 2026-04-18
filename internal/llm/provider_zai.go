package llm

import (
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

func init() {
	RegisterProvider("zai", func(model, apiKey, baseURL string) (llms.Model, error) {
		opts := []openai.Option{}
		if apiKey != "" {
			opts = append(opts, openai.WithToken(apiKey))
		}
		if baseURL == "" {
			baseURL = "https://api.z.ai/v1"
		}
		opts = append(opts, openai.WithBaseURL(baseURL))
		if model != "" {
			opts = append(opts, openai.WithModel(model))
		}
		return openai.New(opts...)
	}, ProviderInfo{
		DisplayName:    "z.ai",
		Description:    "Models via z.ai API (OpenAI-compatible)",
		RequiresKey:    true,
		DefaultBaseURL: "https://api.z.ai/v1",
		Models: []ModelInfo{
			{ID: "default", DisplayName: "Default"},
		},
	})
}
