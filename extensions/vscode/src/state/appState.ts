import { Ticket, AgentSession } from '../api/types';

export interface KanbanColumn {
    name: string;
    status: string;
    tickets: Ticket[];
}

export class AppState {
    columns: KanbanColumn[] = [
        { name: 'Backlog', status: 'backlog', tickets: [] },
        { name: 'In Progress', status: 'in_progress', tickets: [] },
        { name: 'Review', status: 'review', tickets: [] },
        { name: 'Done', status: 'done', tickets: [] },
    ];

    selectedColumnIndex: number = 0;
    selectedTicketIndex: number = 0;
    selectedTicketId: string | null = null;

    activeSessions: AgentSession[] = [];
    dashboardOpen: boolean = false;
    selectedSessionId: string | null = null;

    onStateChange?: (state: AppState) => void;

    private emit() {
        this.onStateChange?.(this);
    }

    setTickets(tickets: Ticket[]) {
        for (const col of this.columns) {
            col.tickets = tickets.filter(t => t.status === col.status);
        }
        this.emit();
    }

    selectColumn(index: number) {
        this.selectedColumnIndex = index;
        this.selectedTicketIndex = 0;
        const col = this.columns[index];
        if (col.tickets.length > 0) {
            this.selectedTicketId = col.tickets[0].id;
        } else {
            this.selectedTicketId = null;
        }
        this.emit();
    }

    moveSelection(direction: 'up' | 'down' | 'left' | 'right') {
        if (direction === 'left') {
            if (this.selectedColumnIndex > 0) this.selectColumn(this.selectedColumnIndex - 1);
            return;
        }
        if (direction === 'right') {
            if (this.selectedColumnIndex < this.columns.length - 1) this.selectColumn(this.selectedColumnIndex + 1);
            return;
        }
        const col = this.columns[this.selectedColumnIndex];
        if (direction === 'up') {
            if (this.selectedTicketIndex > 0) {
                this.selectedTicketIndex--;
                this.selectedTicketId = col.tickets[this.selectedTicketIndex]?.id ?? null;
                this.emit();
            }
        }
        if (direction === 'down') {
            if (this.selectedTicketIndex < col.tickets.length - 1) {
                this.selectedTicketIndex++;
                this.selectedTicketId = col.tickets[this.selectedTicketIndex]?.id ?? null;
                this.emit();
            }
        }
    }

    setActiveSessions(sessions: AgentSession[]) {
        this.activeSessions = sessions;
        this.emit();
    }

    toggleDashboard() {
        this.dashboardOpen = !this.dashboardOpen;
        this.emit();
    }

    getSelectedTicket(): Ticket | null {
        const col = this.columns[this.selectedColumnIndex];
        return col.tickets.find(t => t.id === this.selectedTicketId) ?? null;
    }
}