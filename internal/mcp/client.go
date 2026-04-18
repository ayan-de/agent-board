package mcp

import (
	"context"
	"fmt"
	"sync"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/mcpclient"
)

// Manager manages MCP server connections.
type Manager struct {
	cfg     config.MCPConfig
	clients map[string]*mcpclient.Client
	mu      sync.Mutex
}

// NewManager creates a new Manager.
func NewManager(cfg config.MCPConfig) *Manager {
	return &Manager{
		cfg:     cfg,
		clients: make(map[string]*mcpclient.Client),
	}
}

// GetClient returns an MCP client for the specified server name.
// It starts the server if it's not already running.
func (m *Manager) GetClient(ctx context.Context, name string) (*mcpclient.Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if client, ok := m.clients[name]; ok {
		return client, nil
	}

	serverCfg, ok := m.cfg.Servers[name]
	if !ok {
		return nil, fmt.Errorf("mcp: server %q not configured", name)
	}

	if !serverCfg.Enabled {
		return nil, fmt.Errorf("mcp: server %q is disabled", name)
	}

	client, err := mcpclient.NewClient(ctx, serverCfg.Command, serverCfg.Args...)
	if err != nil {
		return nil, fmt.Errorf("mcp: failed to start server %q: %w", name, err)
	}

	m.clients[name] = client
	return client, nil
}

// Close closes all active MCP client connections.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, client := range m.clients {
		client.Close()
		delete(m.clients, name)
	}
}
