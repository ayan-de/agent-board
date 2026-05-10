package orchestrator

import "testing"

func TestExtractResumeCommand(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "opencode resume",
			output: "Session   New session - 2026-05-10T07:36:46.777Z\nContinue  opencode -s ses_1ef2eca46ffeKTXRokicTzd5iI",
			want:   "opencode -s ses_1ef2eca46ffeKTXRokicTzd5iI",
		},
		{
			name:   "claude resume",
			output: "Resume this session with:\nclaude --resume 31a136eb-7bf4-496d-b00b-73c3ac8158de",
			want:   "claude --resume 31a136eb-7bf4-496d-b00b-73c3ac8158de",
		},
		{
			name:   "no resume command",
			output: "Agent completed successfully",
			want:   "",
		},
		{
			name:   "empty output",
			output: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractResumeCommand(tt.output)
			if got != tt.want {
				t.Errorf("ExtractResumeCommand() = %q, want %q", got, tt.want)
			}
		})
	}
}