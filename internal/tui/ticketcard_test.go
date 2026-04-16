package tui

import (
	"strings"
	"testing"

	"github.com/ayan-de/agent-board/internal/store"
	"github.com/ayan-de/agent-board/internal/theme"

	"github.com/charmbracelet/lipgloss"
)

func testCardTheme() *theme.Theme {
	return &theme.Theme{
		Primary: lipgloss.Color("69"), Text: lipgloss.Color("15"),
		TextMuted: lipgloss.Color("240"), Background: lipgloss.Color("#000"),
		BackgroundPanel: lipgloss.Color("236"), Border: lipgloss.Color("240"),
		BorderActive: lipgloss.Color("69"), Accent: lipgloss.Color("213"),
		Success: lipgloss.Color("42"), Error: lipgloss.Color("196"),
		Warning: lipgloss.Color("214"),
	}
}

func testTicket() store.Ticket {
	return store.Ticket{
		ID:          "AGT-01",
		Title:       "Implement Auth",
		Description: "Add JWT authentication flow",
		Status:      "in_progress",
		Priority:    "high",
		Agent:       "claude-code",
		AgentActive: false,
	}
}

func TestTicketCardCompactRenders(t *testing.T) {
	ticket := testTicket()
	card := NewTicketCardModel(ticket, false, false, 30, 0, testCardTheme())
	output := card.Render()

	if !strings.Contains(output, "AGT-01") {
		t.Error("compact card missing ticket ID")
	}
	if !strings.Contains(output, "Implement Auth") {
		t.Error("compact card missing title")
	}
}

func TestTicketCardCompactHasBorder(t *testing.T) {
	ticket := testTicket()
	card := NewTicketCardModel(ticket, false, false, 30, 0, testCardTheme())
	output := card.Render()

	if !strings.Contains(output, "╭") && !strings.Contains(output, "╰") {
		t.Error("compact card missing rounded border corners")
	}
}

func TestTicketCardCompactHeight(t *testing.T) {
	ticket := testTicket()
	card := NewTicketCardModel(ticket, false, false, 30, 0, testCardTheme())
	if card.CompactHeight() != 2 {
		t.Errorf("CompactHeight = %d, want 2", card.CompactHeight())
	}
}

func TestTicketCardExpandedRenders(t *testing.T) {
	ticket := testTicket()
	card := NewTicketCardModel(ticket, true, true, 40, 0, testCardTheme())
	output := card.Render()

	if !strings.Contains(output, "AGT-01") {
		t.Error("expanded card missing ticket ID")
	}
	if !strings.Contains(output, "Implement Auth") {
		t.Error("expanded card missing title")
	}
	if !strings.Contains(output, "Add JWT authentication flow") {
		t.Error("expanded card missing description")
	}
}

func TestTicketCardExpandedHasBorder(t *testing.T) {
	ticket := testTicket()
	card := NewTicketCardModel(ticket, true, true, 40, 0, testCardTheme())
	output := card.Render()

	if !strings.Contains(output, "╭") && !strings.Contains(output, "╰") {
		t.Error("expanded card missing rounded border corners")
	}
}

func TestTicketCardAgentDotIdle(t *testing.T) {
	ticket := testTicket()
	ticket.AgentActive = false
	card := NewTicketCardModel(ticket, false, false, 30, 0, testCardTheme())
	output := card.Render()

	if !strings.Contains(output, "○") {
		t.Error("idle agent card should show muted dot '○'")
	}
}

func TestTicketCardAgentDotActive(t *testing.T) {
	ticket := testTicket()
	ticket.AgentActive = true
	card := NewTicketCardModel(ticket, false, false, 30, 0, testCardTheme())
	output := card.Render()

	if !strings.Contains(output, "●") {
		t.Error("active agent card should show colored dot '●'")
	}
}

func TestTicketCardNoAgentNoDot(t *testing.T) {
	ticket := testTicket()
	ticket.Agent = ""
	card := NewTicketCardModel(ticket, false, false, 30, 0, testCardTheme())
	output := card.Render()

	if strings.Contains(output, "○") || strings.Contains(output, "●") {
		t.Error("no-agent card should not show any dot")
	}
}

func TestTicketCardActivityBarWhenActive(t *testing.T) {
	ticket := testTicket()
	ticket.AgentActive = true
	card := NewTicketCardModel(ticket, false, false, 30, 0, testCardTheme())
	output := card.Render()

	if !strings.Contains(output, "▓") {
		t.Error("active card should show activity bar with filled blocks")
	}
}

func TestTicketCardNoActivityBarWhenIdle(t *testing.T) {
	ticket := testTicket()
	ticket.AgentActive = false
	card := NewTicketCardModel(ticket, false, false, 30, 0, testCardTheme())
	output := card.Render()

	if strings.Contains(output, "▓") {
		t.Error("idle card should not show activity bar")
	}
}

func TestTicketCardLongTitleTruncationCompact(t *testing.T) {
	ticket := testTicket()
	ticket.Title = strings.Repeat("X", 200)
	card := NewTicketCardModel(ticket, false, false, 20, 0, testCardTheme())
	output := card.Render()

	if strings.Contains(output, ticket.Title) {
		t.Error("long title should be truncated, not shown in full")
	}
}

func TestTicketCardExpandedWrapsDescription(t *testing.T) {
	ticket := testTicket()
	ticket.Description = strings.Repeat("D", 200)
	card := NewTicketCardModel(ticket, true, true, 20, 0, testCardTheme())
	output := card.Render()

	if !strings.Contains(output, "D") {
		t.Error("expanded card should contain description text")
	}
}

func TestTicketCardEmptyDescription(t *testing.T) {
	ticket := testTicket()
	ticket.Description = ""
	card := NewTicketCardModel(ticket, false, false, 30, 0, testCardTheme())
	output := card.Render()

	if output == "" {
		t.Fatal("card with empty description should still render")
	}
}

func TestDefaultTicketCardStyles(t *testing.T) {
	styles := DefaultTicketCardStyles()
	if styles.SelectedBorder.GetBorderStyle() == lipgloss.NewStyle().GetBorderStyle() {
		t.Error("SelectedBorder should have a border set")
	}
}
