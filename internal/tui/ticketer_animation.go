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

const AnimFrames = 16

type AnimationType int

const (
	AnimationPlasma AnimationType = iota
	AnimationDual
	AnimationSpark
	AnimationDefault = AnimationPlasma
)

var (
	plasmaPatterns [AnimFrames]string
	dualPatterns   [AnimFrames]string
	sparkPatterns  [AnimFrames]string
)

func init() {
	const width = 16

	plasma := []rune{'░', '░', '▒', '▒', '▓', '▓', '█', '█', '▓', '▓', '▒', '▒', '░', '░', '·', '·'}
	for f := 0; f < AnimFrames; f++ {
		arr := make([]rune, width)
		for i := range arr {
			arr[i] = plasma[(i+f)%len(plasma)]
		}
		plasmaPatterns[f] = string(arr)
	}

	levels := []rune{'░', '▒', '▓', '█'}
	for f := 0; f < AnimFrames; f++ {
		arr := make([]rune, width)
		for i := range arr {
			arr[i] = '░'
		}
		a := f % (width / 2)
		b := width - 1 - a
		for i := 0; i <= a; i++ {
			d := a - i
			if d < len(levels) {
				arr[i] = levels[len(levels)-1-d]
			}
		}
		for i := b; i < width; i++ {
			d := i - b
			if d < len(levels) {
				arr[i] = levels[len(levels)-1-d]
			}
		}
		dualPatterns[f] = string(arr)
	}

	bg := '·'
	trail := []rune{'░', '▒', '▓', '█'}
	totalFrames := width + len(trail)
	for f := 0; f < AnimFrames; f++ {
		arr := make([]rune, width)
		for i := range arr {
			arr[i] = bg
		}
		pos := f % totalFrames
		for t, ch := range trail {
			p := pos - t
			if p >= 0 && p < width {
				arr[p] = ch
			}
		}
		sparkPatterns[f] = string(arr)
	}
}

func patternFor(t AnimationType) *[AnimFrames]string {
	switch t {
	case AnimationDual:
		return &dualPatterns
	case AnimationSpark:
		return &sparkPatterns
	default:
		return &plasmaPatterns
	}
}

type tickMsg struct{}

func ActivityBar(frame int, width int, t *theme.Theme) string {
	return ActivityBarWithType(frame, width, t, AnimationDefault)
}

func ActivityBarWithType(frame int, width int, t *theme.Theme, animType AnimationType) string {
	if width < 4 {
		width = 4
	}

	patterns := patternFor(animType)
	pattern := patterns[frame%AnimFrames]
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

func agentNameStyled(agent string) string {
	if agent == "" {
		return ""
	}
	color := config.AgentColor(agent)
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(agent)
}

func animationTick() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}
