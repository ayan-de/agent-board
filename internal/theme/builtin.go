package theme

import (
	"embed"
	"strings"
)

//go:embed themes/*.json
var themeFS embed.FS

func registerBuiltins(r *Registry) {
	entries, err := themeFS.ReadDir("themes")
	if err != nil {
		return
	}

	for _, entry := range entries {
		data, err := themeFS.ReadFile("themes/" + entry.Name())
		if err != nil {
			continue
		}

		th, err := parseThemeJSON(data, r.mode, "builtin")
		if err != nil {
			continue
		}

		if th.Name == "" {
			th.Name = strings.TrimSuffix(entry.Name(), ".json")
		}

		r.Register(th)
	}
}
