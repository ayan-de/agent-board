package llm_test

import (
	"testing"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/llm"
)

func TestFactoryReturnsClientWithOpenAI(t *testing.T) {
	cfg := config.LLMConfig{
		Provider: "openai",
		Model:    "gpt-4o-mini",
		APIKey:   "test-key",
	}

	client, err := llm.NewFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("expected client")
	}
}

func TestFactoryReturnsClientWithOllama(t *testing.T) {
	cfg := config.LLMConfig{
		Provider: "ollama",
		Model:    "qwen2.5-coder",
		BaseURL:  "http://127.0.0.1:11434",
	}

	client, err := llm.NewFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("expected client")
	}
}

func TestFactoryUsesSeparateModelNames(t *testing.T) {
	cfg := config.LLMConfig{
		Provider:         "openai",
		APIKey:           "test-key",
		CoordinatorModel: "gpt-4o-mini",
		SummarizerModel:  "gpt-4o",
	}

	client, err := llm.NewFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("expected client")
	}
}

func TestFactoryReturnsClientWithClaude(t *testing.T) {
	cfg := config.LLMConfig{
		Provider: "claude",
		Model:    "claude-sonnet-4-20250514",
		APIKey:   "test-key",
	}

	client, err := llm.NewFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("expected client")
	}
}

func TestFactoryReturnsClientWithZAI(t *testing.T) {
	cfg := config.LLMConfig{
		Provider: "zai",
		Model:    "default",
		APIKey:   "test-key",
	}

	client, err := llm.NewFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("expected client")
	}
}

func TestFactoryReturnsErrorForUnsupportedProvider(t *testing.T) {
	cfg := config.LLMConfig{
		Provider: "unsupported_provider",
	}

	_, err := llm.NewFromConfig(cfg)
	if err == nil {
		t.Fatal("expected error for unsupported provider")
	}
}

func TestProvidersReturnsAllRegistered(t *testing.T) {
	providers := llm.Providers()

	names := make(map[string]bool)
	for _, p := range providers {
		names[p.Name] = true
	}

	for _, want := range []string{"openai", "ollama", "claude", "zai"} {
		if !names[want] {
			t.Errorf("missing provider %q", want)
		}
	}
}

func TestProviderInfoHasMetadata(t *testing.T) {
	info, ok := llm.GetProvider("openai")
	if !ok {
		t.Fatal("expected openai provider")
	}
	if info.DisplayName != "OpenAI" {
		t.Fatalf("DisplayName = %q, want OpenAI", info.DisplayName)
	}
	if !info.RequiresKey {
		t.Fatal("OpenAI should require API key")
	}
	if len(info.Models) == 0 {
		t.Fatal("expected models list")
	}
}

func TestOllamaDoesNotRequireKey(t *testing.T) {
	info, ok := llm.GetProvider("ollama")
	if !ok {
		t.Fatal("expected ollama provider")
	}
	if info.RequiresKey {
		t.Fatal("Ollama should not require API key")
	}
}

func TestFactoryReturnsNilClientWhenNoProvider(t *testing.T) {
	cfg := config.LLMConfig{}

	client, err := llm.NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client != nil {
		t.Fatal("expected nil client when no provider configured")
	}
}
