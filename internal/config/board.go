package config

import (
	"fmt"
	"regexp"
	"strings"
)

type Column struct {
	Status string `toml:"status"`
	Name   string `toml:"name"`
}

type BoardConfig struct {
	Columns []Column `toml:"columns"`
	Prefix          string `toml:"prefix"`
	ProjectInitDate string `toml:"-"` // format: "2006-01-02" — read from dir mtime, not user-editable
}

func DefaultColumns() []Column {
	return []Column{
		{Status: "backlog", Name: "Backlog"},
		{Status: "in_progress", Name: "In Progress"},
		{Status: "review", Name: "Review"},
		{Status: "done", Name: "Done"},
	}
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
