package mcp

import (
	"context"
	"fmt"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
)

type HandoffInput struct {
	TicketID     string
	Title        string
	Description  string
	ContextCarry string
}

type ContextCarryAdapter struct {
	manager     *Manager
	projectName string
}

func NewContextCarryAdapter(manager *Manager, projectName string) *ContextCarryAdapter {
	return &ContextCarryAdapter{
		manager:     manager,
		projectName: projectName,
	}
}


// Build currently just formats a string.
func (a *ContextCarryAdapter) Build(input HandoffInput) string {
	return fmt.Sprintf(
		"ticket=%s\ntitle=%s\ncontext_carry=%s\n",
		input.TicketID,
		input.Title,
		input.ContextCarry,
	)
}

// Load loads context for a given project and branch.
func (a *ContextCarryAdapter) Load(ctx context.Context, project, branch string) (string, error) {
	if a.manager == nil {
		return "", fmt.Errorf("contextcarry.load: manager not configured")
	}

	client, err := a.manager.GetClient(ctx, "contextcarry")
	if err != nil {
		return "", err
	}

	result, err := client.CallTool(ctx, "load_context", map[string]interface{}{
		"project": project,
		"branch":  branch,
	})
	if err != nil {
		return "", fmt.Errorf("contextcarry.load: %w", err)
	}

	if result.IsError {
		return "", fmt.Errorf("contextcarry.load: server returned error")
	}

	if len(result.Content) == 0 {
		return "", nil
	}

	for _, content := range result.Content {
		if textContent, ok := mcpgo.AsTextContent(content); ok {
			return textContent.Text, nil
		}
	}

	return "", fmt.Errorf("contextcarry.load: no text content found in result")
}

// Save saves context for a given project and branch.
func (a *ContextCarryAdapter) Save(ctx context.Context, project, branch, contextStr string) error {
	if a.manager == nil {
		return fmt.Errorf("contextcarry.save: manager not configured")
	}

	client, err := a.manager.GetClient(ctx, "contextcarry")
	if err != nil {
		return err
	}

	result, err := client.CallTool(ctx, "save_context", map[string]interface{}{
		"project": project,
		"branch":  branch,
		"context": contextStr,
	})
	if err != nil {
		return fmt.Errorf("contextcarry.save: %w", err)
	}

	if result.IsError {
		return fmt.Errorf("contextcarry.save: server returned error")
	}

	return nil
}
// LoadContext implements orchestrator.ContextCarryProvider.
func (a *ContextCarryAdapter) LoadContext(ctx context.Context, ticketID string) (string, error) {
	// In a real implementation, we'd fetch the ticket to get the branch.
	// For now, let's assume branch = ticketID or project-branch.
	return a.Load(ctx, a.projectName, ticketID)
}

// SaveContext implements orchestrator.ContextCarryProvider.
func (a *ContextCarryAdapter) SaveContext(ctx context.Context, ticketID, outcome string) error {
	return a.Save(ctx, a.projectName, ticketID, outcome)
}

