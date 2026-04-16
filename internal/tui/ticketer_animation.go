package tui

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const AnimFrames = 8

var animPatterns = [AnimFrames]string{
	"░░▒▒▓▓██▓▓▒▒░░",
	"░▒▒▓▓██▓▓▒▒░░░",
	"▒▒▓▓██▓▓▒▒░░░░",
	"▒▓▓██▓▓▒▒░░░░▒",
	"▓▓██▓▓▒▒░░░░▒▒",
	"▓██▓▓▒▒░░░░▒▒▓",
	"██▓▓▒▒░░░░▒▒▓▓",
	"▓▓▒▒░░░░▒▒▓▓██",
}

type tickMsg struct{}

func ActivityBar(frame int, width int, t *theme.Theme) string {
	if width < 4 {
		width = 4
	}

	pattern := animPatterns[frame%AnimFrames]
	patternRunes := []rune(pattern)
	patternLen := utf8.RuneCountInString(pattern)

	filledColor := lipgloss.Color("213")
	emptyColor := lipgloss.Color("240")
	if t != nil {
		filledColor = t.Accent
		emptyColor = t.TextMuted
	}

	var b strings.Builder
	for i := 0; i < width; i++ {
		r := patternRunes[i%patternLen]
		switch r {
		case '█', '▓', '▒':
			b.WriteString(lipgloss.NewStyle().Foreground(filledColor).Render(string(r)))
		default:
			b.WriteString(lipgloss.NewStyle().Foreground(emptyColor).Render(string(r)))
		}
	}

	return b.String()
}

func agentDot(agent string, active bool) string {
	if agent == "" {
		return ""
	}

	if active {
		color := config.AgentColor(agent)
		return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render("●")
	}

	return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("○")
}

func animationTick() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}
