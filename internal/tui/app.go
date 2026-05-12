package tui

import (
	"strings"

	"github.com/ayan-de/agent-board/internal/board"
	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/keybinding"
	"github.com/ayan-de/agent-board/internal/store"
	"github.com/ayan-de/agent-board/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type App struct {
	board     *board.BoardService
	renderer  *Renderer
	state     board.BoardViewState
	resolver  *keybinding.Resolver
	palette   CommandPalette
	modal     ConfirmModal
	textInput TextInputModal
	notification NotificationStack

	width  int
	height int
	quit   bool
}

type AppDeps struct {
	Orchestrator board.Orchestrator
}

func NewApp(cfg *config.Config, s *store.Store, reg *theme.Registry, deps AppDeps) (*App, error) {
	boardSvc := board.NewBoardService(s, deps.Orchestrator, cfg, reg)

	renderer := NewRenderer(0, 0)

	km := keybinding.DefaultKeyMap()
	if len(cfg.TUI.Keybindings) > 0 {
		keybinding.ApplyConfig(&km, cfg.TUI.Keybindings)
	}
	resolver := keybinding.NewResolver(km)

	a := &App{
		board:     boardSvc,
		renderer:  renderer,
		state:     boardSvc.GetState(),
		resolver:  resolver,
		palette:   NewCommandPalette(nil, nil),
		modal:     ConfirmModal{},
		textInput: TextInputModal{},
		notification: NotificationStack{},
	}

	a.palette.SetTheme(reg.Active())
	a.modal.SetTheme(reg.Active())
	a.textInput.SetTheme(reg.Active())
	a.notification.SetTheme(reg.Active())

	return a, nil
}

func (a *App) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.renderer.SetSize(msg.Width, msg.Height)
		a.modal.SetSize(msg.Width, msg.Height)
		a.textInput.SetSize(msg.Width, msg.Height)
		a.notification.SetSize(msg.Width, msg.Height)
		return a, nil

	case tea.KeyMsg:
		if a.modal.Active() {
			var cmd tea.Cmd
			a.modal, cmd = a.modal.Update(msg)
			return a, cmd
		}
		if a.textInput.Active() {
			var cmd tea.Cmd
			a.textInput, cmd = a.textInput.Update(msg)
			return a, cmd
		}
		if a.palette.Active() {
			a.palette, _ = a.palette.Update(msg)
			return a, nil
		}

		intent := a.resolveIntent(msg)
		if intent != nil {
			a.state = a.board.ProcessIntent(intent)
		}
		return a, nil

	case boardIntentMsg:
		intent := extractIntent(msg)
		if intent != nil {
			a.state = a.board.ProcessIntent(intent)
		}
		return a, nil

	case notificationDismissMsg:
		a.notification = a.notification.HandleDismiss(msg)
		a.state.Notification = nil
		return a, nil
	}

	return a, nil
}

func (a *App) resolveIntent(msg tea.KeyMsg) board.Intent {
	key := msg.String()
	action, _ := a.resolver.Resolve(key)

	switch action {
	case keybinding.ActionQuit:
		a.modal.Open("Quit AgentBoard", "Are you sure you want to quit?", func() tea.Cmd { return tea.Quit }, nil)
		return nil
	case keybinding.ActionForceQuit:
		return nil
	case keybinding.ActionOpenTicket:
		ticket := a.selectedTicket()
		if ticket != nil {
			return board.IntentSelectTicket{TicketID: ticket.ID}
		}
		return nil
	case keybinding.ActionAddTicket:
		return board.IntentCreateTicket{ColumnIndex: a.state.Kanban.ColIndex}
	case keybinding.ActionDeleteTicket:
		ticket := a.selectedTicket()
		if ticket != nil {
			return board.IntentDeleteTicket{TicketID: ticket.ID}
		}
		return nil
	case keybinding.ActionPrevColumn:
		if a.state.Kanban.ColIndex > 0 {
			return board.IntentOpenView{View: board.ViewBoard}
		}
		return nil
	case keybinding.ActionNextColumn:
		if a.state.Kanban.ColIndex < len(a.state.Kanban.ColumnDefs)-1 {
			return board.IntentOpenView{View: board.ViewBoard}
		}
		return nil
	case keybinding.ActionPrevTicket:
		return board.IntentOpenView{View: board.ViewBoard}
	case keybinding.ActionNextTicket:
		return board.IntentOpenView{View: board.ViewBoard}
	case keybinding.ActionJumpColumn1:
		return board.IntentOpenView{View: board.ViewBoard}
	case keybinding.ActionJumpColumn2:
		return board.IntentOpenView{View: board.ViewBoard}
	case keybinding.ActionJumpColumn3:
		return board.IntentOpenView{View: board.ViewBoard}
	case keybinding.ActionJumpColumn4:
		return board.IntentOpenView{View: board.ViewBoard}
	case keybinding.ActionShowHelp:
		if a.state.ActiveView == board.ViewHelp {
			return board.IntentOpenView{View: board.ViewBoard}
		}
		return board.IntentOpenView{View: board.ViewHelp}
	case keybinding.ActionShowDashboard:
		if a.state.ActiveView == board.ViewDashboard {
			return board.IntentOpenView{View: board.ViewBoard}
		}
		return board.IntentRefreshDashboard{}
	case keybinding.ActionOpenPalette:
		return board.IntentShowPalette{}
	case keybinding.ActionRefresh:
		return board.IntentRefreshDashboard{}
	default:
		return nil
	}
}

func (a *App) selectedTicket() *store.Ticket {
	colIdx := a.state.Kanban.ColIndex
	cursorIdx := 0
	if colIdx >= 0 && colIdx < len(a.state.Kanban.Cursors) {
		cursorIdx = a.state.Kanban.Cursors[colIdx]
	}
	columns := a.state.Kanban.Columns
	if colIdx >= 0 && colIdx < len(columns) {
		tickets := columns[colIdx].Tickets
		if cursorIdx >= 0 && cursorIdx < len(tickets) {
			return &tickets[cursorIdx]
		}
	}
	return nil
}

func (a *App) View() string {
	mainView := a.renderer.Render(a.state)

	if a.palette.Active() {
		mainView = a.overlayPalette(mainView)
	}

	if a.modal.Active() {
		return a.overlayModal(mainView)
	}

	if a.textInput.Active() {
		return a.textInput.View()
	}

	if a.notification.Active() {
		return a.overlayNotification(mainView)
	}

	return mainView
}

func (a *App) overlayPalette(mainView string) string {
	paletteView := a.palette.View()
	paletteLines := strings.Split(paletteView, "\n")
	paletteHeight := len(paletteLines)
	bgLines := strings.Split(mainView, "\n")

	for len(bgLines) < a.height {
		bgLines = append(bgLines, "")
	}

	startY := a.height - paletteHeight
	var finalView strings.Builder
	for i := 0; i < a.height; i++ {
		bgLine := ""
		if i < len(bgLines) {
			bgLine = bgLines[i]
		}

		if i >= startY && i < a.height {
			row := i - startY
			paletteLine := paletteLines[row]
			finalView.WriteString(overlayLine(bgLine, paletteLine, 0))
		} else {
			finalView.WriteString(bgLine)
		}
		if i < a.height-1 {
			finalView.WriteString("\n")
		}
	}
	return finalView.String()
}

func (a *App) overlayModal(mainView string) string {
	modalBox := a.modal.ViewBox()
	modalHeight := lipgloss.Height(modalBox)
	modalWidth := lipgloss.Width(modalBox)
	bgLines := strings.Split(mainView, "\n")

	for len(bgLines) < a.height {
		bgLines = append(bgLines, "")
	}

	startY := (a.height - modalHeight) / 2
	startX := (a.width - modalWidth) / 2

	var finalView strings.Builder
	modalLines := strings.Split(modalBox, "\n")

	for i := 0; i < a.height; i++ {
		bgLine := ""
		if i < len(bgLines) {
			bgLine = bgLines[i]
		}

		if i >= startY && i < startY+modalHeight {
			row := i - startY
			modalLine := modalLines[row]
			finalView.WriteString(overlayLine(bgLine, modalLine, startX))
		} else {
			finalView.WriteString(bgLine)
		}
		if i < a.height-1 {
			finalView.WriteString("\n")
		}
	}
	return finalView.String()
}

func (a *App) overlayNotification(mainView string) string {
	notificationStack := a.notification.View()
	notificationHeight := lipgloss.Height(notificationStack)
	notificationWidth := lipgloss.Width(notificationStack)
	bgLines := strings.Split(mainView, "\n")

	for len(bgLines) < a.height {
		bgLines = append(bgLines, "")
	}

	startY := a.height - notificationHeight - 1
	if startY < 1 {
		startY = 1
	}
	startX := a.width - notificationWidth - 2
	if startX < 0 {
		startX = 0
	}

	var finalView strings.Builder
	notificationLines := strings.Split(notificationStack, "\n")

	for i := 0; i < a.height; i++ {
		bgLine := ""
		if i < len(bgLines) {
			bgLine = bgLines[i]
		}

		if i >= startY && i < startY+notificationHeight {
			row := i - startY
			notificationLine := notificationLines[row]
			finalView.WriteString(overlayLine(bgLine, notificationLine, startX))
		} else {
			finalView.WriteString(bgLine)
		}
		if i < a.height-1 {
			finalView.WriteString("\n")
		}
	}
	return finalView.String()
}

func overlayLine(bg, fg string, x int) string {
	bgWidth := lipgloss.Width(bg)
	fgWidth := lipgloss.Width(fg)

	if x < 0 {
		x = 0
	}
	if x+fgWidth > bgWidth {
		x = (bgWidth - fgWidth) / 2
	}

	left := ansiTruncate(bg, x)
	right := ansiSkip(bg, x+fgWidth)

	return left + fg + right
}

func ansiTruncate(s string, limit int) string {
	var (
		visualPos int
		inEscape  bool
	)

	for i, r := range s {
		if visualPos >= limit && !inEscape {
			return s[:i]
		}

		if r == '\x1b' {
			inEscape = true
			continue
		}

		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}

		visualPos++
	}
	return s
}

func ansiSkip(s string, skip int) string {
	var (
		visualPos int
		inEscape  bool
	)

	for i, r := range s {
		if visualPos >= skip && !inEscape {
			return s[i:]
		}

		if r == '\x1b' {
			inEscape = true
			continue
		}

		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}

		visualPos++
	}
	return ""
}