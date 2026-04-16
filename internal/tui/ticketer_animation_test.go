package tui

import (
	"strings"
	"testing"

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

func TestActivityBarWrapsTo8Frames(t *testing.T) {
	bar8 := stripAnsi(ActivityBar(8, 20, nil))
	bar0 := stripAnsi(ActivityBar(0, 20, nil))
	if bar8 != bar0 {
		t.Errorf("frame 8 stripped = %q, want frame 0 stripped = %q", bar8, bar0)
	}
}
