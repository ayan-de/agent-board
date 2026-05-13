import { Ticket } from '../../api/types';

export function renderTicketCard(ticket: Ticket, isSelected: boolean): string {
    const statusClass = ticket.status.replace('_', '-');
    const agentBadge = ticket.agent
        ? `<span class="agent-badge">${escapeHtml(ticket.agent)}</span>`
        : '';

    return `
    <div class="ticket-card ${isSelected ? 'selected' : ''}"
         data-id="${ticket.id}"
         data-status="${ticket.status}">
      <div class="ticket-id">${escapeHtml(ticket.id)}</div>
      <div class="ticket-title">${escapeHtml(ticket.title)}</div>
      <div class="ticket-meta">
        ${agentBadge}
        <span class="status-badge ${statusClass}">${ticket.status.replace('_', ' ')}</span>
      </div>
    </div>`;
}

function escapeHtml(s: string): string {
    return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}