package tui

import (
    "github.com/ayan-de/agent-board/internal/board"
    tea "github.com/charmbracelet/bubbletea"
)

type boardIntentMsg struct {
    intent board.Intent
}

func BoardIntent(i board.Intent) tea.Msg {
    return boardIntentMsg{intent: i}
}

func extractIntent(msg tea.Msg) board.Intent {
    switch m := msg.(type) {
    case boardIntentMsg:
        return m.intent
    }
    return nil
}