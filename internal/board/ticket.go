package board

import (
	"context"
	"fmt"
	"strings"

	"github.com/ayan-de/agent-board/internal/config"
	"github.com/ayan-de/agent-board/internal/store"
)

func TicketSelectField(b *BoardService, fieldName string) BoardViewState {
	b.state.Ticket.Mode = ModeTicketEdit
	b.state.Ticket.EditBuffer = getFieldValue(b.state.Ticket.Ticket, fieldName)
	return *b.state
}

func getFieldValue(t *store.Ticket, field string) string {
	switch field {
	case "title":
		return t.Title
	case "description":
		return t.Description
	case "branch":
		return t.Branch
	case "tags":
		return strings.Join(t.Tags, ", ")
	default:
		return ""
	}
}

func TicketCommitField(b *BoardService, field, value string) BoardViewState {
	if b.state.Ticket == nil || b.state.Ticket.Ticket == nil {
		return *b.state
	}
	switch field {
	case "title":
		b.state.Ticket.Ticket.Title = value
	case "description":
		b.state.Ticket.Ticket.Description = value
	case "branch":
		b.state.Ticket.Ticket.Branch = value
	case "tags":
		b.state.Ticket.Ticket.Tags = parseTags(value)
	}
	_, _ = b.store.UpdateTicket(context.Background(), *b.state.Ticket.Ticket)
	b.state.Ticket.Mode = ModeTicketView
	b.state.Ticket.EditBuffer = ""
	return *b.state
}

func parseTags(input string) []string {
	if input == "" {
		return nil
	}
	tags := strings.Split(input, ",")
	result := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t != "" {
			result = append(result, t)
		}
	}
	return result
}

func TicketCycleStatus(b *BoardService) BoardViewState {
	if b.state.Ticket == nil || b.state.Ticket.Ticket == nil {
		return *b.state
	}
	statuses := [4]string{"backlog", "in_progress", "review", "done"}
	currentIdx := -1
	for i, s := range statuses {
		if s == b.state.Ticket.Ticket.Status {
			currentIdx = i
			break
		}
	}
	if currentIdx == -1 {
		currentIdx = 0
	}
	nextIdx := (currentIdx + 1) % len(statuses)
	newStatus := statuses[nextIdx]
	return KanbanMoveTicket(b, b.state.Ticket.Ticket.ID, newStatus)
}

func TicketAssignAgent(b *BoardService, agentName string) BoardViewState {
	if b.state.Ticket == nil || b.state.Ticket.Ticket == nil {
		return *b.state
	}
	b.state.Ticket.Ticket.Agent = agentName
	_, _ = b.store.UpdateTicket(context.Background(), *b.state.Ticket.Ticket)
	b.state.Ticket.Mode = ModeTicketView
	b.state.Notification = &NotificationState{
		Title:   "Agent assignment updated",
		Message: fmt.Sprintf("%s assigned to %s", agentName, b.state.Ticket.Ticket.ID),
		Variant: NotificationSuccess,
	}
	return *b.state
}

func TicketSetPriority(b *BoardService, priority string) BoardViewState {
	if b.state.Ticket == nil || b.state.Ticket.Ticket == nil {
		return *b.state
	}
	b.state.Ticket.Ticket.Priority = priority
	_, _ = b.store.UpdateTicket(context.Background(), *b.state.Ticket.Ticket)
	b.state.Ticket.Mode = ModeTicketView
	return *b.state
}

func TicketToggleDependsOn(b *BoardService, dependsOnID string) BoardViewState {
	if b.state.Ticket == nil || b.state.Ticket.Ticket == nil {
		return *b.state
	}
	if dependsOnID == b.state.Ticket.Ticket.ID {
		return *b.state
	}
	found := false
	for i, d := range b.state.Ticket.Ticket.DependsOn {
		if d == dependsOnID {
			b.state.Ticket.Ticket.DependsOn = append(b.state.Ticket.Ticket.DependsOn[:i], b.state.Ticket.Ticket.DependsOn[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		b.state.Ticket.Ticket.DependsOn = append(b.state.Ticket.Ticket.DependsOn, dependsOnID)
	}
	return *b.state
}

func TicketSaveDependsOn(b *BoardService) BoardViewState {
	if b.state.Ticket == nil || b.state.Ticket.Ticket == nil {
		return *b.state
	}
	_, _ = b.store.UpdateTicket(context.Background(), *b.state.Ticket.Ticket)
	b.state.Ticket.Mode = ModeTicketView
	return *b.state
}

func TicketOpenAgentSelect(b *BoardService) BoardViewState {
	if b.state.Ticket == nil {
		return *b.state
	}
	b.state.Ticket.Mode = ModeTicketAgentSelect
	b.state.Ticket.Agents = config.DetectAgents()
	b.state.Ticket.Cursor = 0
	return *b.state
}

func TicketSelectAgentAtCursor(b *BoardService) BoardViewState {
	if b.state.Ticket == nil {
		return *b.state
	}
	agents := b.state.Ticket.Agents
	cursor := b.state.Ticket.Cursor
	if cursor >= 0 && cursor < len(agents) {
		return TicketAssignAgent(b, agents[cursor].Name)
	}
	return *b.state
}

func TicketOpenPrioritySelect(b *BoardService) BoardViewState {
	b.state.Ticket.Mode = ModeTicketPrioritySelect
	return *b.state
}

func TicketOpenDependsOnSelect(b *BoardService) BoardViewState {
	b.state.Ticket.Mode = ModeTicketDependsOnSelect
	tickets, _ := b.store.ListTickets(context.Background(), store.TicketFilters{})
	if len(tickets) > 5 {
		tickets = tickets[:5]
	}
	b.state.Ticket.DependsOnTickets = tickets
	return *b.state
}

func TicketCancelEdit(b *BoardService) BoardViewState {
	if b.state.Ticket == nil {
		return *b.state
	}
	if b.state.Ticket.Mode == ModeTicketAgentSelect ||
		b.state.Ticket.Mode == ModeTicketPrioritySelect ||
		b.state.Ticket.Mode == ModeTicketDependsOnSelect {
		b.state.Ticket.Mode = ModeTicketView
	} else if b.state.Ticket.Mode == ModeTicketEdit {
		b.state.Ticket.Mode = ModeTicketView
		b.state.Ticket.EditBuffer = ""
	}
	return *b.state
}

func TicketViewProposal(b *BoardService) BoardViewState {
	if b.state.Ticket == nil || b.state.Ticket.Proposal == nil {
		return *b.state
	}
	b.SetNotification("Proposal", b.state.Ticket.Proposal.Prompt, NotificationInfo)
	return *b.state
}

func TicketMoveCursor(b *BoardService, direction int) BoardViewState {
	if b.state.Ticket == nil {
		return *b.state
	}
	maxCursor := 6
	switch b.state.Ticket.Mode {
	case ModeTicketAgentSelect:
		maxCursor = len(b.state.Ticket.Agents) - 1
	case ModeTicketDependsOnSelect:
		maxCursor = len(b.state.Ticket.DependsOnTickets) - 1
	}
	if maxCursor < 0 {
		maxCursor = 0
	}
	if b.state.Ticket.Cursor > maxCursor {
		b.state.Ticket.Cursor = maxCursor
	}
	if direction > 0 && b.state.Ticket.Cursor < maxCursor {
		b.state.Ticket.Cursor++
	} else if direction < 0 && b.state.Ticket.Cursor > 0 {
		b.state.Ticket.Cursor--
	}
	return *b.state
}

func TicketReturnToBoard(b *BoardService) BoardViewState {
	b.state.ActiveView = ViewBoard
	b.state.Ticket = nil
	return *b.state
}
