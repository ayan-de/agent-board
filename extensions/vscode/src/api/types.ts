// Matches store.Ticket
export interface Ticket {
    id: string;
    title: string;
    description: string;
    status: 'backlog' | 'in_progress' | 'review' | 'done';
    agent: string | null;
    branch: string;
    createdAt: string;
    updatedAt: string;
    dependsOn: string;
}

// Matches store.Proposal
export interface Proposal {
    id: string;
    ticketID: string;
    agent: string;
    prompt: string;
    status: 'pending' | 'approved' | 'running' | 'completed';
    createdAt: string;
}

// Matches core/types.go:AgentSession
export interface AgentSession {
    sessionID: string;
    ticketID: string;
    agent: string;
    startedAt: number;
    status: 'running' | 'completed' | 'failed' | 'cancelled';
    paneID: string;
    windowID: string;
}

// Matches store.Session
export interface Session {
    id: string;
    ticketID: string;
    agent: string;
    startedAt: string;
    endedAt: string | null;
    status: 'running' | 'completed' | 'failed' | 'cancelled';
    contextKey: string;
}

// Matches core/types.go:RunCompletion
export interface RunCompletion {
    ticketID: string;
    sessionID: string;
    outcome: string;
    summary: string;
    resumeCommand: string;
}

// Request bodies
export interface CreateProposalInput {
    ticketID: string;
}

export interface FinishRunInput {
    ticketID: string;
    sessionID: string;
    outcome: string;
    summary: string;
    resumeCommand: string;
}

export interface StartAdHocRunInput {
    agent: string;
    prompt: string;
}

export interface MoveStatusInput {
    status: string;
}

export interface CreateTicketInput {
    title: string;
    description?: string;
    status?: string;
}