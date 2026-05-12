package board

import (
	"context"
	"fmt"
	"time"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/store"
)



func KanbanSelectTicket(b *BoardService, ticketID string) BoardViewState {
	ticket, err := b.store.GetTicket(context.Background(), ticketID)
	if err != nil {
		return *b.state
	}
	proposal, _ := b.store.GetActiveProposalForTicket(context.Background(), ticketID)
	b.state.Ticket = &TicketViewState{
		Ticket:   &ticket,
		Proposal: &proposal,
	}
	b.state.ActiveView = ViewTicket
	return *b.state
}

func KanbanCreateTicket(b *BoardService, colIndex int) BoardViewState {
	if colIndex >= len(b.state.Kanban.Columns) {
		return *b.state
	}
	col := b.state.Kanban.Columns[colIndex].Def
	ticket, err := b.store.CreateTicket(context.Background(), store.Ticket{
		Title:  "New Ticket",
		Status: col.Status,
	})
	if err != nil {
		return *b.state
	}
	b.loadKanbanState()
	b.state.Notification = &NotificationState{
		Title:   "Ticket created",
		Message: fmt.Sprintf("%s: %s", ticket.ID, ticket.Title),
		Variant: NotificationSuccess,
	}
	return *b.state
}

func KanbanDeleteTicket(b *BoardService, ticketID string) BoardViewState {
	_ = b.store.DeleteTicket(context.Background(), ticketID)
	b.loadKanbanState()
	return *b.state
}

func KanbanMoveTicket(b *BoardService, ticketID, newStatus string) BoardViewState {
	_ = b.store.MoveStatus(context.Background(), ticketID, newStatus)
	b.loadKanbanState()
	if b.state.Ticket != nil && b.state.Ticket.Ticket != nil && b.state.Ticket.Ticket.ID == ticketID {
		b.state.Ticket.Ticket.Status = newStatus
	}
	if newStatus == "in_progress" {
		if b.state.Ticket != nil {
			b.state.Ticket.Loading = true
		}
		return ProposalCreate(b, ticketID)
	}
	return *b.state
}

func KanbanPrevColumn(b *BoardService) BoardViewState {
	if b.state.Kanban.ColIndex > 0 {
		b.state.Kanban.ColIndex--
	}
	return *b.state
}

func KanbanNextColumn(b *BoardService) BoardViewState {
	if b.state.Kanban.ColIndex < len(b.state.Kanban.ColumnDefs)-1 {
		b.state.Kanban.ColIndex++
	}
	return *b.state
}

func KanbanPrevTicket(b *BoardService) BoardViewState {
	colIdx := b.state.Kanban.ColIndex
	if colIdx >= 0 && colIdx < len(b.state.Kanban.Cursors) {
		if b.state.Kanban.Cursors[colIdx] > 0 {
			b.state.Kanban.Cursors[colIdx]--
			if b.state.Kanban.Cursors[colIdx] < b.state.Kanban.ScrollOff[colIdx] {
				b.state.Kanban.ScrollOff[colIdx] = b.state.Kanban.Cursors[colIdx]
			}
		}
	}
	return *b.state
}

func KanbanNextTicket(b *BoardService) BoardViewState {
	colIdx := b.state.Kanban.ColIndex
	if colIdx >= 0 && colIdx < len(b.state.Kanban.Columns) && colIdx < len(b.state.Kanban.Cursors) {
		numTickets := len(b.state.Kanban.Columns[colIdx].Tickets)
		if b.state.Kanban.Cursors[colIdx] < numTickets-1 {
			b.state.Kanban.Cursors[colIdx]++
		}
	}
	return *b.state
}

func KanbanJumpColumn(b *BoardService, index int) BoardViewState {
	if index >= 0 && index < len(b.state.Kanban.ColumnDefs) {
		b.state.Kanban.ColIndex = index
	}
	return *b.state
}



func KanbanHandleTabChange(b *BoardService, tab KanbanTab) BoardViewState {
	b.state.Kanban.Tab = tab
	return *b.state
}

func KanbanNavigateMonth(b *BoardService, direction int) BoardViewState {
	if direction == 1 {
		b.state.Kanban.MonthOffset++
	} else if direction == -1 && b.state.Kanban.MonthOffset > 0 {
		b.state.Kanban.MonthOffset--
	}
	from, to := MonthWindow(time.Now(), b.state.Kanban.MonthOffset)
	fromPtr := &from
	toPtr := &to
	tickets, _ := b.store.ListTickets(context.Background(), store.TicketFilters{From: fromPtr, To: toPtr})
	b.state.Kanban.Columns = groupByStatusDynamic(tickets, b.state.Kanban.ColumnDefs)
	return *b.state
}

func groupByStatusDynamic(tickets []store.Ticket, columnDefs []config.Column) []KanbanColumn {
	cols := make([]KanbanColumn, len(columnDefs))
	statusMap := make(map[string]int)
	for i, col := range columnDefs {
		statusMap[col.Status] = i
	}
	for _, t := range tickets {
		if idx, ok := statusMap[t.Status]; ok {
			cols[idx].Tickets = append(cols[idx].Tickets, t)
		}
	}
	for i := range columnDefs {
		cols[i].Def = columnDefs[i]
	}
	return cols
}

func MonthWindow(initDate time.Time, offset int) (from, to time.Time) {
	from = initDate.AddDate(0, offset, 0)
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	to = from.AddDate(0, 1, -1)
	to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, to.Location())
	return from, to
}
