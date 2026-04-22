package llm

import (
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

func init() {
	RegisterProvider("minimax", func(model, apiKey, baseURL string) (llms.Model, error) {
		opts := []openai.Option{}
		if apiKey != "" {
			opts = append(opts, openai.WithToken(apiKey))
		}
		if baseURL == "" {
			baseURL = "https://api.minimax.io/v1"
		}
		opts = append(opts, openai.WithBaseURL(baseURL))
		if model != "" {
			opts = append(opts, openai.WithModel(model))
		}
		return openai.New(opts...)
	}, ProviderInfo{
		DisplayName:    "MiniMax",
		Description:    "MiniMax models via MiniMax API (OpenAI-compatible)",
		RequiresKey:    true,
		DefaultBaseURL: "https://api.minimax.io/v1",
		Models: []ModelInfo{
			{ID: "MiniMax-M2.7", DisplayName: "MiniMax M2.7"},
			{ID: "MiniMax-M2.7-highspeed", DisplayName: "MiniMax M2.7 Highspeed"},
			{ID: "MiniMax-M2.5", DisplayName: "MiniMax M2.5"},
			{ID: "MiniMax-M2.5-highspeed", DisplayName: "MiniMax M2.5 Highspeed"},
			{ID: "MiniMax-M2.1", DisplayName: "MiniMax M2.1"},
			{ID: "MiniMax-M2", DisplayName: "MiniMax M2"},
		},
	})
}