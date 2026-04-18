package prompt_test

import (
	"strings"
	"testing"

	"github.com/ayan-de/agent-board/internal/prompt"
)

func TestGenerateProposalContainsContext(t *testing.T) {
	got := prompt.GenerateProposal("AGT-01", "Add auth", "Build JWT flow", "opencode", "prior summary")

	if !strings.Contains(got, "AGT-01") {
		t.Error("missing ticket ID")
	}
	if !strings.Contains(got, "Add auth") {
		t.Error("missing title")
	}
	if !strings.Contains(got, "Build JWT flow") {
		t.Error("missing description")
	}
	if !strings.Contains(got, "opencode") {
		t.Error("missing agent")
	}
	if !strings.Contains(got, "prior summary") {
		t.Error("missing context carry")
	}
}

func TestSummarizeContextContainsOutcome(t *testing.T) {
	got := prompt.SummarizeContext("AGT-01", "completed", "did the work")

	if !strings.Contains(got, "AGT-01") {
		t.Error("missing ticket ID")
	}
	if !strings.Contains(got, "completed") {
		t.Error("missing outcome")
	}
	if !strings.Contains(got, "did the work") {
		t.Error("missing summary")
	}
}
