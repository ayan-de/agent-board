package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AnimFrames is the number of distinct frames in the active animation cycle.
// Tuned for an 80ms tick interval — one full pass is ~1.28s, fast enough to
// feel alive but slow enough that the eye can resolve individual transitions.
const AnimFrames = 16

// AnimIntervalMs is the milliseconds between animation ticks when an agent
// is active. Matches opencode's spinner interval (80ms).
const AnimIntervalMs = 80

// AnimIdleIntervalMs is the milliseconds between ticks when nothing is
// active. The kanban gates animation on anyAgentActive(), so this rarely
// fires — kept here for any future idle-only animation.
const AnimIdleIntervalMs = 200

type AnimationType int

const (
	AnimationPlasma AnimationType = iota
	AnimationDual
	AnimationSpark
	AnimationDefault = AnimationPlasma
)

// animDensity is the per-character density at a given frame slot.
// Index 0 = dimmest, 4 = brightest. Five levels give the eye enough depth
// to read motion without going beyond the standard terminal block ramp.
var animDensity = [5]rune{'·', '░', '▒', '▓', '█'}

var (
	plasmaPatterns [AnimFrames]string
	dualPatterns   [AnimFrames]string
	sparkPatterns  [AnimFrames]string
)

func init() {
	const width = 16

	// Plasma: a "wave" of density that scrolls left-to-right. Each frame
	// shifts the peak one slot, producing a smooth roll across the bar.
	// The density ramp is wider than the bar, so the leading edge fades in
	// instead of popping in.
	plasmaWave := []int{0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 3, 3, 2, 2, 1, 1, 0, 0}
	for f := 0; f < AnimFrames; f++ {
		arr := make([]rune, width)
		for i := range arr {
			arr[i] = animDensity[plasmaWave[(i+f)%len(plasmaWave)]]
		}
		plasmaPatterns[f] = string(arr)
	}

	// Dual: knight-rider style — two peaks sliding in from opposite ends,
	// meeting in the middle. Opencode's createFrames uses the same idea but
	// we render the full pattern so a single bar can show the bounce.
	for f := 0; f < AnimFrames; f++ {
		arr := make([]rune, width)
		for i := range arr {
			arr[i] = animDensity[0]
		}
		// Forward sweep: a peak rises from index 0 outward.
		// Backward sweep: a peak rises from the last index inward.
		// Phase is AnimFrames/2 (8) — the first half sweeps out, the second
		// half sweeps back.
		phase := f
		if phase >= AnimFrames/2 {
			phase = AnimFrames - 1 - phase
		}
		for i := 0; i <= phase; i++ {
			d := 4 - (phase - i)
			if d < 0 {
				d = 0
			}
			if d > 4 {
				d = 4
			}
			arr[i] = animDensity[d]
		}
		for i := width - 1 - phase; i < width; i++ {
			d := 4 - (i - (width - 1 - phase))
			if d < 0 {
				d = 0
			}
			if d > 4 {
				d = 4
			}
			arr[i] = animDensity[d]
		}
		dualPatterns[f] = string(arr)
	}

	// Spark: a single bright pixel with a fading trail.
	trailLen := 4
	totalFrames := width + trailLen
	for f := 0; f < AnimFrames; f++ {
		arr := make([]rune, width)
		for i := range arr {
			arr[i] = animDensity[0]
		}
		pos := f % totalFrames
		for t := 0; t < trailLen; t++ {
			p := pos - t
			if p >= 0 && p < width {
				density := trailLen - t
				if density > 4 {
					density = 4
				}
				arr[p] = animDensity[density]
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

// gradientLerp returns an interpolated color string between `from` and `to`
// at position `t` in [0..steps]. Used to derive a 5-level color ramp
// between the theme's TextMuted (low) and Accent (high). When the theme
// doesn't expose RGB info, falls back to a fixed ANSI ramp that looks
// reasonable on dark and light backgrounds alike.
func gradientLerp(from, to lipgloss.Color, level int) lipgloss.Color {
	if level <= 0 {
		return from
	}
	if level >= 4 {
		return to
	}
	fr, fg, fb, ok := parseHexColor(string(from))
	if !ok {
		return ansiLevel(level)
	}
	tr, tg, tb, ok := parseHexColor(string(to))
	if !ok {
		return ansiLevel(level)
	}
	r := fr + (tr-fr)*level/4
	g := fg + (tg-fg)*level/4
	b := fb + (tb-fb)*level/4
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}

// ansiLevel returns a fixed-color level for the gradient when the theme
// colors can't be parsed as hex. The 240/244/245/246 ramp is a smooth
// grayscale that works on both dark and light backgrounds.
func ansiLevel(level int) lipgloss.Color {
	switch level {
	case 0:
		return lipgloss.Color("240")
	case 1:
		return lipgloss.Color("244")
	case 2:
		return lipgloss.Color("245")
	case 3:
		return lipgloss.Color("246")
	default:
		return lipgloss.Color("213")
	}
}

// parseHexColor accepts #rgb or #rrggbb and returns the components.
// Returns ok=false for ANSI index strings (e.g. "240") so the caller can
// fall back to a fixed ramp.
func parseHexColor(s string) (int, int, int, bool) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "#") {
		return 0, 0, 0, false
	}
	s = s[1:]
	if len(s) == 3 {
		// Expand #abc → #aabbcc
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 {
		return 0, 0, 0, false
	}
	r, err1 := strconv.ParseUint(s[0:2], 16, 8)
	g, err2 := strconv.ParseUint(s[2:4], 16, 8)
	b, err3 := strconv.ParseUint(s[4:6], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, false
	}
	return int(r), int(g), int(b), true
}

func ActivityBar(frame int, width int, t *theme.Theme) string {
	return ActivityBarWithType(frame, width, t, AnimationDefault)
}

// ActivityBarWithType renders the activity bar for the given frame. Each
// character slot uses one of 5 density levels (·░▒▓█) mapped to 5
// interpolated colors between TextMuted and Accent, producing a smooth
// gradient that scrolls per frame. Designed for an 80ms tick.
func ActivityBarWithType(frame int, width int, t *theme.Theme, animType AnimationType) string {
	if width < 4 {
		width = 4
	}

	patterns := patternFor(animType)
	pattern := patterns[frame%AnimFrames]
	patternRunes := []rune(pattern)
	patternLen := utf8.RuneCountInString(pattern)

	muted := lipgloss.Color("240")
	accent := lipgloss.Color("213")
	if t != nil {
		muted = t.TextMuted
		accent = t.Accent
	}

	// Pre-compute the 5-step color ramp once per render — interpolation
	// is cheap but not free, and a kanban page can have many cards.
	colors := [5]lipgloss.Color{
		gradientLerp(muted, accent, 0),
		gradientLerp(muted, accent, 1),
		gradientLerp(muted, accent, 2),
		gradientLerp(muted, accent, 3),
		gradientLerp(muted, accent, 4),
	}

	var b strings.Builder
	for i := 0; i < width; i++ {
		r := patternRunes[i%patternLen]
		level := densityToLevel(r)
		b.WriteString(lipgloss.NewStyle().Foreground(colors[level]).Render(string(r)))
	}

	return b.String()
}

// densityToLevel maps one of the 5 density runes to a level index 0..4.
// Anything unrecognised falls back to 0 (dimmest).
func densityToLevel(r rune) int {
	switch r {
	case '█':
		return 4
	case '▓':
		return 3
	case '▒':
		return 2
	case '░':
		return 1
	default:
		return 0
	}
}

func agentDot(agent string, selected bool, active bool) string {
	if agent == "" {
		return ""
	}

	color := config.AgentColor(agent)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))

	if active || selected {
		return style.Render("●")
	}

	return style.Render("○")
}

func agentNameStyled(agent string) string {
	if agent == "" {
		return ""
	}
	color := config.AgentColor(agent)
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(agent)
}

// animationTick schedules the next animation tick at the active rate.
// Used when an agent is active and we want the kanban to animate.
func animationTick() tea.Cmd {
	return tea.Tick(AnimIntervalMs*time.Millisecond, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}