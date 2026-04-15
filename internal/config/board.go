package config

import (
	"fmt"
	"regexp"
	"strings"
)

type BoardConfig struct {
	Statuses []string `toml:"statuses"`
	Prefix   string   `toml:"prefix"`
}

func DefaultPrefix(projectName string) string {
	s := strings.ToUpper(projectName)
	s = regexp.MustCompile(`[^A-Z0-9]`).ReplaceAllString(s, "")
	if len(s) > 3 {
		s = s[:3]
	}
	if len(s) == 0 {
		s = "AGT"
	}
	return fmt.Sprintf("%s-", s)
}
