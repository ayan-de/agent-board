package config

type GeneralConfig struct {
	Log  string `toml:"log"`
	Addr string `toml:"addr"`
	Mode string `toml:"mode"`
	Tmux string `toml:"tmux"`
}
