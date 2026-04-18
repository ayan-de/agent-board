package llm

import (
	"fmt"
	"sort"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/tmc/langchaingo/llms"
)

type ProviderFactory func(model, apiKey, baseURL string) (llms.Model, error)

type ProviderInfo struct {
	Name           string
	DisplayName    string
	Description    string
	Models         []ModelInfo
	RequiresKey    bool
	DefaultBaseURL string
}

type ModelInfo struct {
	ID          string
	DisplayName string
}

var (
	providerRegistry = map[string]ProviderFactory{}
	providerInfo     = map[string]ProviderInfo{}
)

func RegisterProvider(name string, factory ProviderFactory, info ProviderInfo) {
	providerRegistry[name] = factory
	info.Name = name
	providerInfo[name] = info
}

func Providers() []ProviderInfo {
	result := make([]ProviderInfo, 0, len(providerInfo))
	for _, info := range providerInfo {
		result = append(result, info)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

func GetProvider(name string) (ProviderInfo, bool) {
	info, ok := providerInfo[name]
	return info, ok
}

func NewFromConfig(cfg config.LLMConfig) (Client, error) {
	if cfg.Provider == "" {
		return nil, fmt.Errorf("llm.newFromConfig: no provider configured")
	}

	coordinatorModel := cfg.CoordinatorModel
	if coordinatorModel == "" {
		coordinatorModel = cfg.Model
	}

	summarizerModel := cfg.SummarizerModel
	if summarizerModel == "" {
		summarizerModel = cfg.Model
	}

	coordinator, err := newProviderModel(cfg.Provider, coordinatorModel, cfg.APIKey, cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("llm.newFromConfig.coordinator: %w", err)
	}

	summarizer, err := newProviderModel(cfg.Provider, summarizerModel, cfg.APIKey, cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("llm.newFromConfig.summarizer: %w", err)
	}

	return &LangChainClient{
		Coordinator: coordinator,
		Summarizer:  summarizer,
	}, nil
}

func newProviderModel(provider, model, apiKey, baseURL string) (llms.Model, error) {
	factory, ok := providerRegistry[provider]
	if !ok {
		return nil, fmt.Errorf("llm: unsupported provider %q", provider)
	}
	return factory(model, apiKey, baseURL)
}
