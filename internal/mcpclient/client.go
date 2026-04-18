package mcpclient

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// Client is a wrapper around the mcp-go client that manages a stdio connection.
type Client struct {
	mcpClient *client.Client
}

// NewClient creates a new Client by starting the specified command.
func NewClient(ctx context.Context, command string, args ...string) (*Client, error) {
	// NewStdioMCPClient(command, env, args...)
	// Uses current env by passing nil for env.
	mcpClient, err := client.NewStdioMCPClient(command, nil, args...)
	if err != nil {
		return nil, fmt.Errorf("mcpclient: failed to create stdio client: %w", err)
	}

	// Initialize the client
	initRequest := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			Capabilities: mcp.ClientCapabilities{
				Sampling: &struct{}{},
			},
			ClientInfo: mcp.Implementation{
				Name:    "agent-board-client",
				Version: "0.1.0",
			},
		},
	}

	_, err = mcpClient.Initialize(ctx, initRequest)
	if err != nil {
		mcpClient.Close()
		return nil, fmt.Errorf("mcpclient: failed to initialize client: %w", err)
	}

	return &Client{
		mcpClient: mcpClient,
	}, nil
}

// Close closes the client and the underlying process.
func (c *Client) Close() error {
	return c.mcpClient.Close()
}

// CallTool calls a tool on the MCP server.
func (c *Client) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: arguments,
		},
	}

	return c.mcpClient.CallTool(ctx, req)
}
