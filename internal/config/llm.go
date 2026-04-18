package config

type LLMConfig struct {
	Provider string `toml:"provider"`
	Model    string `toml:"model"`
	APIKey   string `toml:"api_key"`
	BaseURL  string `toml:"base_url"`

	CoordinatorModel string `toml:"coordinator_model"`
	SummarizerModel  string `toml:"summarizer_model"`
	RequireApproval  bool   `toml:"require_approval"`
}
