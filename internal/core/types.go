package core

type AgentSession struct {
	SessionID string
	TicketID  string
	Agent     string
	StartedAt int64
	Status    string
	PaneID    string
	WindowID  string
}

type AgentPane struct {
	SessionID string
	TicketID  string
	Agent     string
	PaneID    string
	WindowID  string
	Status    string
	Outcome   string
	Summary   string
}

type CreateProposalInput struct {
	TicketID string
}

type ApplyRunOutcomeInput struct {
	TicketID      string
	Outcome       string
	ResumeCommand string
}

type FinishRunInput struct {
	TicketID      string
	SessionID     string
	Outcome       string
	Summary       string
	ResumeCommand string
}

type RunCompletion struct {
	TicketID      string
	SessionID     string
	Outcome       string
	Summary       string
	ResumeCommand string
}
