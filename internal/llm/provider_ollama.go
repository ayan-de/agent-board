package llm

import (
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

func init() {
	RegisterProvider("ollama", func(model, apiKey, baseURL string) (llms.Model, error) {
		opts := []ollama.Option{}
		if model != "" {
			opts = append(opts, ollama.WithModel(model))
		}
		if baseURL != "" {
			opts = append(opts, ollama.WithServerURL(baseURL))
		}
		return ollama.New(opts...)
	}, ProviderInfo{
		DisplayName:    "Ollama",
		Description:    "Local models via Ollama",
		RequiresKey:    false,
		DefaultBaseURL: "http://127.0.0.1:11434",
		Models: []ModelInfo{
			{ID: "qwen2.5-coder", DisplayName: "Qwen 2.5 Coder"},
			{ID: "qwen3", DisplayName: "Qwen 3"},
			{ID: "llama3.1", DisplayName: "Llama 3.1"},
			{ID: "mistral", DisplayName: "Mistral"},
			{ID: "codellama", DisplayName: "Code Llama"},
			{ID: "deepseek-coder-v2", DisplayName: "DeepSeek Coder V2"},
		},
	})
}
