package tui

import "github.com/ayan-de/agent-board/internal/store"

type searchQueryMsg struct {
	query string
}

type searchResultsMsg struct {
	tickets []store.Ticket
}

type monthNavigateMsg struct {
	direction int
}

type tabChangeMsg struct {
	tab KanbanTab
}

type deleteTicketConfirmMsg struct {
	ticketID string
}

type deleteTicketRequestMsg struct {
	ticketID string
}

type showDeleteModalMsg struct {
	ticketID string
}

type notificationMsg struct {
	title   string
	message string
	variant NotificationVariant
}

type proposalApprovedMsg struct {
	proposalID string
}

type runStartedMsg struct {
	proposalID string
}

type viewProposalFullMsg struct {
	proposalID string
}

type adhocRunStartedMsg struct {
	agent  string
	prompt string
}

type statusChangedMsg struct {
	ticketID  string
	newStatus string
}