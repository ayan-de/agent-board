package mcp_test

import (
	"strings"
	"testing"

	"github.com/ayan-de/agent-board/internal/mcp"
)

func TestContextCarryAdapterBuildsHandoffPayload(t *testing.T) {
	// NewContextCarryAdapter(nil) is fine for testing Build which doesn't use the manager.
	adapter := mcp.NewContextCarryAdapter(nil)

	payload := adapter.Build(mcp.HandoffInput{
		TicketID:     "AGE-01",
		Title:        "Add orchestration",
		ContextCarry: "prior summary",
	})

	if !strings.Contains(payload, "prior summary") {
		t.Fatalf("payload %q does not include prior summary", payload)
	}
}
