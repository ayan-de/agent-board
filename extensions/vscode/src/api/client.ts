import {
    Ticket, Proposal, AgentSession, Session, RunCompletion,
    CreateProposalInput, FinishRunInput, StartAdHocRunInput,
    MoveStatusInput, CreateTicketInput,
} from './types';
import * as endpoints from './endpoints';

export interface ListTicketsOptions {
    status?: string;
}

export class ApiClient {
    private baseUrl: string;

    constructor(baseUrl: string) {
        this.baseUrl = baseUrl;
    }

    private url(path: string): string {
        return this.baseUrl + path;
    }

    // Tickets
    async listTickets(options: ListTicketsOptions = {}): Promise<Ticket[]> {
        const url = options.status
            ? `${this.url(endpoints.Tickets.list())}?status=${options.status}`
            : this.url(endpoints.Tickets.list());
        const res = await fetch(url);
        if (!res.ok) throw new Error(await res.text());
        return res.json() as Promise<Ticket[]>;
    }

    async getTicket(id: string): Promise<Ticket> {
        const res = await fetch(this.url(endpoints.Tickets.get(id)));
        if (!res.ok) throw new Error(await res.text());
        return res.json() as Promise<Ticket>;
    }

    async createTicket(input: CreateTicketInput): Promise<Ticket> {
        const res = await fetch(this.url(endpoints.Tickets.create()), {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(input),
        });
        if (!res.ok) throw new Error(await res.text());
        return res.json() as Promise<Ticket>;
    }

    async updateTicket(id: string, ticket: Partial<Ticket>): Promise<Ticket> {
        const res = await fetch(this.url(endpoints.Tickets.update(id)), {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(ticket),
        });
        if (!res.ok) throw new Error(await res.text());
        return res.json() as Promise<Ticket>;
    }

    async deleteTicket(id: string): Promise<void> {
        const res = await fetch(this.url(endpoints.Tickets.delete(id)), { method: 'DELETE' });
        if (!res.ok) throw new Error(await res.text());
    }

    async moveStatus(id: string, status: string): Promise<void> {
        const res = await fetch(this.url(endpoints.Tickets.moveStatus(id)), {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ status } as MoveStatusInput),
        });
        if (!res.ok) throw new Error(await res.text());
    }

    // Proposals
    async createProposal(input: CreateProposalInput): Promise<Proposal> {
        const res = await fetch(this.url(endpoints.Proposals.create()), {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(input),
        });
        if (!res.ok) throw new Error(await res.text());
        return res.json() as Promise<Proposal>;
    }

    async getProposal(id: string): Promise<Proposal> {
        const res = await fetch(this.url(endpoints.Proposals.get(id)));
        if (!res.ok) throw new Error(await res.text());
        return res.json() as Promise<Proposal>;
    }

    async approveProposal(id: string): Promise<void> {
        const res = await fetch(this.url(endpoints.Proposals.approve(id)), { method: 'POST' });
        if (!res.ok) throw new Error(await res.text());
    }

    // Runs
    async startApprovedRun(proposalId: string): Promise<Session> {
        const res = await fetch(this.url(endpoints.Runs.startApproved(proposalId)), { method: 'POST' });
        if (!res.ok) throw new Error(await res.text());
        return res.json() as Promise<Session>;
    }

    async startAdHocRun(input: StartAdHocRunInput): Promise<Session> {
        const res = await fetch(this.url(endpoints.Runs.startAdHoc()), {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(input),
        });
        if (!res.ok) throw new Error(await res.text());
        return res.json() as Promise<Session>;
    }

    async finishRun(input: FinishRunInput): Promise<void> {
        const res = await fetch(this.url(endpoints.Runs.finish(input.sessionID)), {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(input),
        });
        if (!res.ok) throw new Error(await res.text());
    }

    // Sessions
    async getActiveSessions(): Promise<AgentSession[]> {
        const res = await fetch(this.url(endpoints.Sessions.list()));
        if (!res.ok) throw new Error(await res.text());
        return res.json() as Promise<AgentSession[]>;
    }

    async getPaneContent(sessionId: string, lines = 100): Promise<string> {
        const res = await fetch(`${this.url(endpoints.Sessions.pane(sessionId))}?lines=${lines}`);
        if (!res.ok) throw new Error(await res.text());
        const data = await res.json() as { content: string };
        return data.content;
    }

    async switchToPane(sessionId: string): Promise<void> {
        const res = await fetch(this.url(endpoints.Sessions.switch(sessionId)), { method: 'POST' });
        if (!res.ok) throw new Error(await res.text());
    }

    async getLogs(sessionId: string): Promise<string[]> {
        const res = await fetch(this.url(endpoints.Sessions.logs(sessionId)));
        if (!res.ok) throw new Error(await res.text());
        const data = await res.json() as { logs: string[] };
        return data.logs;
    }
}