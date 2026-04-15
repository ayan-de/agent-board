package config

import (
	"testing"
)

func TestExtractProjectName(t *testing.T) {
	tests := []struct {
		name     string
		remote   string
		fallback string
		want     string
	}{
		{
			name:     "github HTTPS URL",
			remote:   "https://github.com/ayan-de/agent-board.git",
			fallback: "",
			want:     "agent-board",
		},
		{
			name:     "github SSH URL",
			remote:   "git@github.com:ayan-de/agent-board.git",
			fallback: "",
			want:     "agent-board",
		},
		{
			name:     "github HTTPS without .git",
			remote:   "https://github.com/user/my-project",
			fallback: "",
			want:     "my-project",
		},
		{
			name:     "gitlab URL",
			remote:   "https://gitlab.com/org/nested/repo.git",
			fallback: "",
			want:     "repo",
		},
		{
			name:     "empty remote uses fallback",
			remote:   "",
			fallback: "my-folder",
			want:     "my-folder",
		},
		{
			name:     "slugify uppercase and special chars",
			remote:   "",
			fallback: "My Cool Project!",
			want:     "my-cool-project",
		},
		{
			name:     "nested gitlab SSH",
			remote:   "git@gitlab.com:group/subgroup/project.git",
			fallback: "",
			want:     "project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractProjectName(tt.remote, tt.fallback)
			if got != tt.want {
				t.Errorf("ExtractProjectName(%q, %q) = %q, want %q", tt.remote, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestDefaultPrefix(t *testing.T) {
	tests := []struct {
		name    string
		project string
		want    string
	}{
		{name: "agent-board", project: "agent-board", want: "AGE-"},
		{name: "my project", project: "my-project", want: "MYP-"},
		{name: "short", project: "ab", want: "AB-"},
		{name: "single char", project: "x", want: "X-"},
		{name: "with numbers", project: "app2", want: "APP-"},
		{name: "empty falls back to AGT", project: "", want: "AGT-"},
		{name: "special chars only", project: "---", want: "AGT-"},
		{name: "long name truncated", project: "my-super-long-project-name", want: "MYS-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DefaultPrefix(tt.project)
			if got != tt.want {
				t.Errorf("DefaultPrefix(%q) = %q, want %q", tt.project, got, tt.want)
			}
		})
	}
}
