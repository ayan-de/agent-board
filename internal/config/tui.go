package config

type TUIConfig struct {
	Theme       string            `toml:"theme"`
	Layout      string            `toml:"layout"`
	Keybindings map[string]string `toml:"keybindings"`
}
