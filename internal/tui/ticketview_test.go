package tui

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/keybinding"
	"github.com/ayan-de/agent-board/internal/store"
	"github.com/ayan-de/agent-board/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func newTestTicketView(t *testing.T) (TicketViewModel, *store.Store) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := store.Open(dbPath, []string{"backlog", "in_progress", "review", "done"}, "AGT-")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	km := keybinding.DefaultKeyMap()
	resolver := keybinding.NewResolver(km)

	defaultTheme := &theme.Theme{
		Primary: lipgloss.Color("69"), Text: lipgloss.Color("15"),
		TextMuted: lipgloss.Color("240"), Background: lipgloss.Color("#000"),
		BackgroundPanel: lipgloss.Color("236"), Border: lipgloss.Color("240"),
		Success: lipgloss.Color("42"), Accent: lipgloss.Color("213"),
	}

	testAgents := []config.DetectedAgent{
		{Name: "claude-code", LogoClr: "#D97757"},
		{Name: "opencode", LogoClr: "#808080"},
		{Name: "codex", LogoClr: "#10A37F"},
		{Name: "cursor", LogoClr: "#F0DB4F"},
	}
	m := NewTicketViewModel(s, resolver, defaultTheme, testAgents)
	return m, s
}

func TestNewTicketViewModel(t *testing.T) {
	m, _ := newTestTicketView(t)

	if m.store == nil {
		t.Error("store is nil")
	}
	if m.resolver == nil {
		t.Error("resolver is nil")
	}
	if m.mode != ticketViewMode {
		t.Errorf("mode = %v, want ticketViewMode", m.mode)
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
}

func TestTicketViewModelInit(t *testing.T) {
	m, _ := newTestTicketView(t)
	cmd := m.Init()
	if cmd != nil {
		t.Errorf("Init() = %v, want nil", cmd)
	}
}

func TestTicketViewModelSetTicket(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:       "Test Ticket",
		Description: "A description",
		Status:      "backlog",
		Priority:    "high",
	})

	m = m.SetTicket(&ticket)
	if m.ticket == nil {
		t.Fatal("ticket is nil after SetTicket")
	}
	if m.ticket.ID != ticket.ID {
		t.Errorf("ticket.ID = %q, want %q", m.ticket.ID, ticket.ID)
	}
}

func TestTicketViewModelWindowSize(t *testing.T) {
	m, _ := newTestTicketView(t)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if m.width != 100 {
		t.Errorf("width = %d, want 100", m.width)
	}
	if m.height != 30 {
		t.Errorf("height = %d, want 30", m.height)
	}
}

func TestTicketViewModelFieldNavigation(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:       "Nav Test",
		Description: "Desc",
		Status:      "backlog",
	})
	m = m.SetTicket(&ticket)

	fieldCount := len(m.fields)
	if fieldCount == 0 {
		t.Fatal("no fields defined")
	}

	nextKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	for i := 0; i < fieldCount+2; i++ {
		m, _ = m.Update(nextKey)
	}
	if m.cursor != fieldCount-1 {
		t.Errorf("cursor = %d after overflow, want %d", m.cursor, fieldCount-1)
	}

	prevKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	for i := 0; i < fieldCount+2; i++ {
		m, _ = m.Update(prevKey)
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d after underflow, want 0", m.cursor)
	}
}

func TestTicketViewModelViewRendersFields(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24
	m.viewport.Width = m.width - 6
	m.viewport.Height = m.height - 7

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:       "Render Me",
		Description: "Some details here",
		Status:      "in_progress",
		Priority:    "high",
		Agent:       "claude-code",
		Branch:      "feat/auth",
	})
	m = m.SetTicket(&ticket)

	view := m.View()
	if view == "" {
		t.Fatal("view is empty")
	}

	checks := []string{
		ticket.ID,
		"Render Me",
		"Some details here",
		"in_progress",
		"high",
		"claude-code",
		"feat/auth",
	}
	for _, want := range checks {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q", want)
		}
	}
}

func TestTicketViewModelViewShowsCursor(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24
	m.viewport.Width = m.width - 6
	m.viewport.Height = m.height - 7

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Cursor Test",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	view := m.View()
	if !strings.Contains(view, "▸") {
		t.Error("view missing cursor marker '▸'")
	}
}

func TestTicketViewModelCycleStatus(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Status Cycle",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	if cmd == nil {
		t.Fatal("expected command from cycleStatus")
	}
	msg := cmd()
	sc, ok := msg.(statusChangedMsg)
	if !ok {
		t.Fatalf("expected statusChangedMsg, got %T", msg)
	}
	if sc.newStatus != "in_progress" {
		t.Errorf("sc.newStatus = %q, want %q", sc.newStatus, "in_progress")
	}
}

func TestTicketViewModelCycleStatusWraps(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Wrap Test",
		Status: "done",
	})
	m = m.SetTicket(&ticket)

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if cmd == nil {
		t.Fatal("expected command from cycleStatus")
	}
	msg := cmd()
	sc, ok := msg.(statusChangedMsg)
	if !ok {
		t.Fatalf("expected statusChangedMsg, got %T", msg)
	}
	if sc.newStatus != "backlog" {
		t.Errorf("sc.newStatus = %q after wrap, want %q", sc.newStatus, "backlog")
	}
}

func TestTicketViewModelCycleStatusReturnsCmd(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Persist Test",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if cmd == nil {
		t.Fatal("expected command")
	}
	msg := cmd()
	if _, ok := msg.(statusChangedMsg); !ok {
		t.Errorf("got %T, want statusChangedMsg", msg)
	}
}


func TestTicketViewModelEditTitle(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24
	m.viewport.Width = m.width - 6
	m.viewport.Height = m.height - 7

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Old Title",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

	if m.mode != ticketEditMode {
		t.Errorf("mode = %v, want ticketEditMode", m.mode)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})

	if m.editBuffer != "Old TitleX" {
		t.Errorf("editBuffer = %q, want %q", m.editBuffer, "Old TitleX")
	}
	if m.mode != ticketEditMode {
		t.Errorf("mode = %v, want ticketEditMode", m.mode)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.mode != ticketViewMode {
		t.Errorf("mode = %v after enter, want ticketViewMode", m.mode)
	}
	if m.ticket.Title != "Old TitleX" {
		t.Errorf("title = %q, want %q", m.ticket.Title, "Old TitleX")
	}

	loaded, _ := s.GetTicket(ctx, ticket.ID)
	if loaded.Title != "Old TitleX" {
		t.Errorf("persisted title = %q, want %q", loaded.Title, "Old TitleX")
	}
}

func TestTicketViewModelEditCancel(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Keep This",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Z'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if m.mode != ticketViewMode {
		t.Errorf("mode = %v after cancel, want ticketViewMode", m.mode)
	}
	if m.ticket.Title != "Keep This" {
		t.Errorf("title = %q after cancel, want %q", m.ticket.Title, "Keep This")
	}
}

func TestTicketViewModelEditBackspace(t *testing.T) {
	m, _ := newTestTicketView(t)
	ctx := context.Background()

	ticket, _ := m.store.CreateTicket(ctx, store.Ticket{
		Title:  "Hello",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})

	if m.editBuffer != "Hell" {
		t.Errorf("editBuffer = %q after backspace, want %q", m.editBuffer, "Hell")
	}
}

func TestTicketViewModelEditDescriptionField(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:       "Desc Edit",
		Description: "Old desc",
		Status:      "backlog",
	})
	m = m.SetTicket(&ticket)

	descIdx := -1
	for i, f := range m.fields {
		if f.label == "Description" {
			descIdx = i
			break
		}
	}
	if descIdx < 0 {
		t.Fatal("Description field not found")
	}

	m.cursor = descIdx
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

	if m.mode != ticketEditMode {
		t.Errorf("mode = %v, want ticketEditMode", m.mode)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'!'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.ticket.Description != "Old desc!" {
		t.Errorf("description = %q, want %q", m.ticket.Description, "Old desc!")
	}
}

func TestTicketViewModelViewNoTicket(t *testing.T) {
	m, _ := newTestTicketView(t)
	m.width = 80
	m.height = 24

	view := m.View()
	if !strings.Contains(view, "No ticket") {
		t.Errorf("view without ticket should say 'No ticket', got: %q", view)
	}
}

func TestTicketViewModelEscReturnsToViewMode(t *testing.T) {
	m, _ := newTestTicketView(t)
	ctx := context.Background()

	ticket, _ := m.store.CreateTicket(ctx, store.Ticket{
		Title:  "Esc Test",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if m.mode != ticketEditMode {
		t.Fatalf("mode = %v, want ticketEditMode", m.mode)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if m.mode != ticketViewMode {
		t.Errorf("mode = %v after esc in edit, want ticketViewMode", m.mode)
	}
}

func TestTicketViewModelTagsDisplayed(t *testing.T) {
	m, _ := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24
	m.viewport.Width = m.width - 6
	m.viewport.Height = m.height - 7

	ticket, _ := m.store.CreateTicket(ctx, store.Ticket{
		Title:  "Tagged",
		Status: "backlog",
		Tags:   []string{"auth", "api"},
	})
	m = m.SetTicket(&ticket)

	view := m.View()
	if !strings.Contains(view, "auth") || !strings.Contains(view, "api") {
		t.Errorf("view missing tags, got: %s", view)
	}
}

func TestTicketViewModelDependsOnDisplayed(t *testing.T) {
	m, _ := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24
	m.viewport.Width = m.width - 6
	m.viewport.Height = m.height - 7

	ticket, _ := m.store.CreateTicket(ctx, store.Ticket{
		Title:     "Dep Test",
		Status:    "backlog",
		DependsOn: []string{"AGT-01", "AGT-02"},
	})
	m = m.SetTicket(&ticket)

	view := m.View()
	if !strings.Contains(view, "AGT-01") || !strings.Contains(view, "AGT-02") {
		t.Errorf("view missing depends_on, got: %s", view)
	}
}

func TestTicketViewModelTimestampsDisplayed(t *testing.T) {
	m, _ := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24
	m.viewport.Width = m.width - 6
	m.viewport.Height = m.height - 7

	ticket, _ := m.store.CreateTicket(ctx, store.Ticket{
		Title:  "Time Test",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	view := m.View()
	if !strings.Contains(view, "Created") || !strings.Contains(view, "Updated") {
		t.Error("view missing timestamp labels")
	}
}

func TestTicketViewModelEditNonEditableField(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "No Edit ID",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	for i, f := range m.fields {
		if f.label == "ID" && !f.editable {
			m.cursor = i
			m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
			if m.mode == ticketEditMode {
				t.Error("e key should not enter edit mode for non-editable field")
			}
			return
		}
	}
}

func TestTicketViewModelAgentSelectMode(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Agent Test",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if m.mode != ticketAgentSelectMode {
		t.Fatalf("mode = %v, want ticketAgentSelectMode", m.mode)
	}
	if m.agentCursor != 0 {
		t.Errorf("agentCursor = %d, want 0", m.agentCursor)
	}
}

func TestTicketViewModelAgentSelectNavigate(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Agent Nav",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	downKey := tea.KeyMsg{Type: tea.KeyDown}
	m, _ = m.Update(downKey)
	if m.agentCursor != 1 {
		t.Errorf("agentCursor = %d after down, want 1", m.agentCursor)
	}

	upKey := tea.KeyMsg{Type: tea.KeyUp}
	m, _ = m.Update(upKey)
	if m.agentCursor != 0 {
		t.Errorf("agentCursor = %d after up, want 0", m.agentCursor)
	}

	for i := 0; i < 10; i++ {
		m, _ = m.Update(upKey)
	}
	if m.agentCursor != 0 {
		t.Errorf("agentCursor = %d after underflow, want 0", m.agentCursor)
	}
}

func TestTicketViewModelAgentSelectAssign(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Assign Me",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.ticket.Agent != "claude-code" {
		t.Errorf("agent = %q, want %q", m.ticket.Agent, "claude-code")
	}
	if m.mode != ticketViewMode {
		t.Errorf("mode = %v after select, want ticketViewMode", m.mode)
	}

	loaded, _ := s.GetTicket(ctx, ticket.ID)
	if loaded.Agent != "claude-code" {
		t.Errorf("persisted agent = %q, want %q", loaded.Agent, "claude-code")
	}
}

func TestTicketViewModelAgentSelectNone(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Unassign Me",
		Status: "backlog",
		Agent:  "opencode",
	})
	m = m.SetTicket(&ticket)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.ticket.Agent != "" {
		t.Errorf("agent = %q after None, want empty", m.ticket.Agent)
	}

	loaded, _ := s.GetTicket(ctx, ticket.ID)
	if loaded.Agent != "" {
		t.Errorf("persisted agent = %q, want empty", loaded.Agent)
	}
}

func TestTicketViewModelAgentSelectDropdownRenders(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24
	m.viewport.Width = m.width - 6
	m.viewport.Height = m.height - 7

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Dropdown Render",
		Status: "backlog",
	})
	m = m.SetTicket(&ticket)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	if m.mode != ticketAgentSelectMode {
		t.Error("mode should be ticketAgentSelectMode after 'a'")
	}

	view := m.View()
	if !strings.Contains(view, "Select Agent") {
		t.Error("view should show Select Agent header")
	}
}

func TestTicketViewModelAgentSelectCancel(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()
	m.width = 80
	m.height = 24

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Cancel Agent",
		Status: "backlog",
		Agent:  "opencode",
	})
	m = m.SetTicket(&ticket)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if m.ticket.Agent != "opencode" {
		t.Errorf("agent = %q after cancel, want %q", m.ticket.Agent, "opencode")
	}
	if m.mode != ticketViewMode {
		t.Errorf("mode = %v after cancel, want ticketViewMode", m.mode)
	}
}

func TestTicketViewModelRunStartsApprovedProposal(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Run Approved",
		Status: "in_progress",
		Agent:  "opencode",
	})
	proposal, _ := s.CreateProposal(ctx, store.Proposal{
		TicketID: ticket.ID,
		Agent:    "opencode",
		Status:   "approved",
		Prompt:   "do the work",
	})

	m = m.SetTicket(&ticket)
	m = m.SetProposal(&proposal)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Fatal("expected run command for approved proposal")
	}

	msg := cmd()
	runMsg, ok := msg.(runStartedMsg)
	if !ok {
		t.Fatalf("expected runStartedMsg, got %T", msg)
	}
	if runMsg.proposalID != proposal.ID {
		t.Fatalf("proposalID = %q, want %q", runMsg.proposalID, proposal.ID)
	}
}

func TestTicketViewModelRunDoesNothingWhenProposalPending(t *testing.T) {
	m, s := newTestTicketView(t)
	ctx := context.Background()

	ticket, _ := s.CreateTicket(ctx, store.Ticket{
		Title:  "Run Pending",
		Status: "in_progress",
		Agent:  "opencode",
	})
	proposal, _ := s.CreateProposal(ctx, store.Proposal{
		TicketID: ticket.ID,
		Agent:    "opencode",
		Status:   "pending",
		Prompt:   "wait for approval",
	})

	m = m.SetTicket(&ticket)
	m = m.SetProposal(&proposal)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd != nil {
		t.Fatalf("expected no run command for pending proposal, got %T", cmd())
	}
}
