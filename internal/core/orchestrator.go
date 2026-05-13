package core

import (
	"context"

	"github.com/ayan-de/agent-board/internal/store"
)

type Orchestrator interface {
	CreateProposal(ctx context.Context, input CreateProposalInput) (store.Proposal, error)
	ApproveProposal(ctx context.Context, proposalID string) error
	StartApprovedRun(ctx context.Context, proposalID string) (store.Session, error)
	StartAdHocRun(ctx context.Context, agent, prompt string) (store.Session, error)
	FinishRun(ctx context.Context, input FinishRunInput) error
	GetLogs(sessionID string) []string
	SendInput(sessionID, input string) error
	GetActiveSessions() []*AgentSession
	GetPaneContent(sessionID string, lines int) (string, error)
	SwitchToPane(sessionID string) error
	CompletionChan() <-chan RunCompletion
}
