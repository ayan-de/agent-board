package config

import (
	"path/filepath"
	"regexp"
	"strings"
)

func ExtractProjectName(remote, fallback string) string {
	if remote == "" {
		return slugify(fallback)
	}

	var name string

	if strings.HasPrefix(remote, "git@") {
		parts := strings.SplitN(remote, ":", 2)
		if len(parts) == 2 {
			name = parts[1]
		}
	} else {
		name = remote
	}

	name = filepath.Base(name)
	name = strings.TrimSuffix(name, ".git")

	return slugify(name)
}

func slugify(s string) string {
	s = strings.ToLower(s)
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	s = reg.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}
