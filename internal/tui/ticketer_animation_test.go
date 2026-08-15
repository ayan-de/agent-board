package tui

import (
	"strings"
	"testing"

	"github.com/ayan-de/agent-board/internal/theme"
	"github.com/charmbracelet/lipgloss"
	ansi "github.com/charmbracelet/x/ansi"
)

func stripAnsi(s string) string {
	return ansi.Strip(s)
}

func TestActivityBarWidth(t *testing.T) {
	bar := ActivityBar(0, 20, nil)
	visualWidth := lipgloss.Width(bar)
	if visualWidth != 20 {
		t.Errorf("ActivityBar width = %d, want 20", visualWidth)
	}
}

func TestActivityBarAllFrames(t *testing.T) {
	for frame := 0; frame < AnimFrames; frame++ {
		bar := ActivityBar(frame, 20, nil)
		if bar == "" {
			t.Errorf("frame %d: bar is empty", frame)
		}
	}
}

func TestActivityBarContainsGradientBlocks(t *testing.T) {
	bar := ActivityBar(0, 20, nil)
	stripped := stripAnsi(bar)
	if !strings.Contains(stripped, "█") {
		t.Error("bar should contain peak blocks '█'")
	}
	if !strings.Contains(stripped, "░") {
		t.Error("bar should contain empty blocks '░'")
	}
}

func TestActivityBarScrolling(t *testing.T) {
	bar0 := stripAnsi(ActivityBar(0, 20, nil))
	bar1 := stripAnsi(ActivityBar(1, 20, nil))
	if bar0 == bar1 {
		t.Error("consecutive frames should produce different bars")
	}
}

func TestActivityBarMinimumWidth(t *testing.T) {
	bar := ActivityBar(0, 4, nil)
	visualWidth := lipgloss.Width(bar)
	if visualWidth != 4 {
		t.Errorf("minimum width bar = %d, want 4", visualWidth)
	}
}

func TestActivityBarWrapsTo16Frames(t *testing.T) {
	bar16 := stripAnsi(ActivityBar(16, 20, nil))
	bar0 := stripAnsi(ActivityBar(0, 20, nil))
	if bar16 != bar0 {
		t.Errorf("frame 16 stripped = %q, want frame 0 stripped = %q", bar16, bar0)
	}
}

// TestActivityBarUsesGradientBetweenThemeColors confirms the bar uses
// the 5-level density ramp (·░▒▓█) — the encoding of the gradient —
// rather than the old 2-level bright/dim mapping. With a single static
// frame we expect at least 3 distinct density levels to be visible.
func TestActivityBarUsesGradientBetweenThemeColors(t *testing.T) {
	th := &theme.Theme{
		TextMuted: lipgloss.Color("#202020"),
		Accent:    lipgloss.Color("#ff00ff"),
	}
	bar := stripAnsi(ActivityBar(0, 20, th))
	levels := map[rune]bool{}
	for _, r := range bar {
		switch r {
		case '·', '░', '▒', '▓', '█':
			levels[r] = true
		}
	}
	if len(levels) < 3 {
		t.Errorf("expected ≥3 distinct density levels in a single frame, got %d (bar=%q)", len(levels), bar)
	}
}

// TestActivityBarAllFiveLevelsAppearAcrossFrames confirms that across the
// full 16-frame cycle, every density level from `dim` (·) to `bright` (█)
// appears at least once. The old 2-level implementation only ever
// emitted `█`/`▓`/`▒` as bright and `░`/`·` as dim — but mapped to a single
// color each, so the visible distinction was binary.
func TestActivityBarAllFiveLevelsAppearAcrossFrames(t *testing.T) {
	seen := map[rune]bool{}
	for f := 0; f < AnimFrames; f++ {
		bar := stripAnsi(ActivityBar(f, 16, &theme.Theme{
			TextMuted: lipgloss.Color("#000000"),
			Accent:    lipgloss.Color("#ffffff"),
		}))
		for _, r := range bar {
			switch r {
			case '·', '░', '▒', '▓', '█':
				seen[r] = true
			}
		}
	}
	for _, want := range []rune{'·', '░', '▒', '▓', '█'} {
		if !seen[want] {
			t.Errorf("density level %q never appeared across the 16-frame cycle", want)
		}
	}
}

// TestActivityBarGradientDistinctAcrossFrames ensures consecutive frames
// produce visibly different bars at the ANSI level (not only at the rune
// level — which TestActivityBarScrolling already covers).
func TestActivityBarGradientDistinctAcrossFrames(t *testing.T) {
	th := &theme.Theme{
		TextMuted: lipgloss.Color("#202020"),
		Accent:    lipgloss.Color("#ff00ff"),
	}
	bar0 := ActivityBar(0, 20, th)
	bar3 := ActivityBar(3, 20, th)
	if bar0 == bar3 {
		t.Error("frames 0 and 3 should produce visually different bars")
	}
}

func TestParseHexColor(t *testing.T) {
	cases := []struct {
		in       string
		r, g, b  int
		ok       bool
	}{
		{"#9d7cd8", 0x9d, 0x7c, 0xd8, true},
		{"#fff", 0xff, 0xff, 0xff, true},
		{"#000000", 0, 0, 0, true},
		{"240", 0, 0, 0, false},
		{"not-a-color", 0, 0, 0, false},
		{"#abcd", 0, 0, 0, false},
		{"  #abcdef  ", 0xab, 0xcd, 0xef, true},
	}
	for _, c := range cases {
		r, g, b, ok := parseHexColor(c.in)
		if ok != c.ok || r != c.r || g != c.g || b != c.b {
			t.Errorf("parseHexColor(%q) = (%d,%d,%d,%v), want (%d,%d,%d,%v)", c.in, r, g, b, ok, c.r, c.g, c.b, c.ok)
		}
	}
}

func TestGradientLerpClampsAtEnds(t *testing.T) {
	from := lipgloss.Color("#000000")
	to := lipgloss.Color("#ffffff")
	if got := gradientLerp(from, to, 0); got != from {
		t.Errorf("level 0 = %v, want %v", got, from)
	}
	if got := gradientLerp(from, to, 4); got != to {
		t.Errorf("level 4 = %v, want %v", got, to)
	}
	if got := gradientLerp(from, to, -5); got != from {
		t.Errorf("negative level = %v, want %v", got, from)
	}
	if got := gradientLerp(from, to, 99); got != to {
		t.Errorf("overflow level = %v, want %v", got, to)
	}
}

func TestGradientLerpInterpolatesMidpoint(t *testing.T) {
	from := lipgloss.Color("#000000")
	to := lipgloss.Color("#ffffff")
	// Integer math: level=2 of 4 → 0 + (255 * 2)/4 = 127 → #7f7f7f.
	got := string(gradientLerp(from, to, 2))
	if got != "#7f7f7f" {
		t.Errorf("level 2 interpolation = %q, want %q", got, "#7f7f7f")
	}
}

func TestAnimationIntervalMatchesOpencode(t *testing.T) {
	// Locked at 80ms to match opencode's active spinner interval. Bumping
	// this is a visual decision that should require explicit test changes.
	if AnimIntervalMs != 80 {
		t.Errorf("AnimIntervalMs = %d, want 80", AnimIntervalMs)
	}
}

func TestAnimationFrameCountIsSixteen(t *testing.T) {
	// 16 frames at 80ms = 1.28s per loop. Tuned so the eye resolves each
	// frame without the bar feeling either jittery or sluggish.
	if AnimFrames != 16 {
		t.Errorf("AnimFrames = %d, want 16", AnimFrames)
	}
}
