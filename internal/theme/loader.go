package theme

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type themeJSON struct {
	Name  string            `json:"name"`
	Defs  map[string]string `json:"defs"`
	Theme map[string]any    `json:"theme"`
}

func parseThemeJSON(data []byte, mode string, source string) (*Theme, error) {
	var tj themeJSON
	if err := json.Unmarshal(data, &tj); err != nil {
		return nil, fmt.Errorf("theme.parseThemeJSON: %w", err)
	}

	resolved := make(map[string]string)
	for k, v := range tj.Defs {
		resolved[k] = v
	}

	th := &Theme{
		Name:   tj.Name,
		Source: source,
	}

	colorKeys := map[string]*lipgloss.Color{
		"primary":           &th.Primary,
		"secondary":         &th.Secondary,
		"accent":            &th.Accent,
		"error":             &th.Error,
		"warning":           &th.Warning,
		"success":           &th.Success,
		"info":              &th.Info,
		"text":              &th.Text,
		"textMuted":         &th.TextMuted,
		"background":        &th.Background,
		"backgroundPanel":   &th.BackgroundPanel,
		"backgroundElement": &th.BackgroundElement,
		"border":            &th.Border,
		"borderActive":      &th.BorderActive,
	}

	for key, ptr := range colorKeys {
		val, ok := tj.Theme[key]
		if !ok {
			continue
		}
		color, err := resolveColor(val, mode, resolved)
		if err != nil {
			return nil, fmt.Errorf("theme.parseThemeJSON: key %q: %w", key, err)
		}
		*ptr = lipgloss.Color(color)
	}

	return th, nil
}

func resolveColor(val any, mode string, resolved map[string]string) (string, error) {
	switch v := val.(type) {
	case string:
		if strings.HasPrefix(v, "#") {
			return v, nil
		}
		if r, ok := resolved[v]; ok {
			return r, nil
		}
		return v, nil
	case map[string]any:
		dark, _ := v["dark"].(string)
		light, _ := v["light"].(string)
		if mode == "light" && light != "" {
			return resolveColor(light, mode, resolved)
		}
		if dark != "" {
			return resolveColor(dark, mode, resolved)
		}
		return "", fmt.Errorf("variant has no color for mode %q", mode)
	default:
		return "", fmt.Errorf("unexpected color type %T", val)
	}
}

func loadFromFS(dir string, mode string, source string) []*Theme {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var themes []*Theme
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") || strings.HasPrefix(name, ".") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		th, err := parseThemeJSON(data, mode, source)
		if err != nil {
			continue
		}

		if th.Name == "" {
			th.Name = strings.TrimSuffix(entry.Name(), ".json")
		}

		themes = append(themes, th)
	}
	return themes
}
