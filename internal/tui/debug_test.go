package tui

import (
    "path/filepath"
    "strings"
    "testing"
    "github.com/ayan-de/agent-board/internal/config"
    "github.com/ayan-de/agent-board/internal/keybinding"
    "github.com/ayan-de/agent-board/internal/store"
)

func TestDebugDashboardNotFound(t *testing.T) {
    dir := t.TempDir()
    dbPath := filepath.Join(dir, "test.db")
    s, err := store.Open(dbPath, []string{"backlog", "in_progress", "review", "done"}, "AGT-")
    if err != nil {
        t.Fatalf("open store: %v", err)
    }
    defer s.Close()

    km := keybinding.DefaultKeyMap()
    resolver := keybinding.NewResolver(km)
    agents := []config.DetectedAgent{
        {Name: "claude-code", Binary: "claude", Found: true},
        {Name: "opencode", Binary: "opencode", Found: true},
        {Name: "codex", Binary: "codex", Found: false},
        {Name: "cursor", Binary: "cursor", Found: false},
    }

    fo := newFakeOrchestrator(s)
    m := NewDashboardModel(s, fo, resolver, agents, testDashboardTheme())
    m.width = 120
    m.height = 40

    view := m.View()
    
    // Find all lines containing codex or cursor
    lines := strings.Split(view, "\n")
    for i, line := range lines {
        if strings.Contains(strings.ToLower(line), "codex") || strings.Contains(strings.ToLower(line), "cursor") {
            t.Logf("line %d: %s", i, line)
        }
    }
    
    // Check how many agents are marked as found
    foundCount := 0
    for _, a := range m.Agents {
        if a.Found {
            foundCount++
        }
    }
    t.Logf("Agents with Found=true: %d", foundCount)
    t.Logf("ActiveSessions: %+v", m.ActiveSessions)
}
