import { Ticket } from '../../api/types';
import { renderTicketCard } from './ticketCard';

export function renderColumn(
    name: string,
    status: string,
    tickets: Ticket[],
    isFocused: boolean,
    selectedIndex: number
): string {
    const ticketCards = tickets
        .map((t, i) => renderTicketCard(t, i === selectedIndex))
        .join('');

    return `
    <div class="kanban-column ${isFocused ? 'focused' : ''}" data-status="${status}">
      <div class="column-header">
        <span>${name}</span>
        <span class="count">${tickets.length}</span>
      </div>
      <div class="column-tickets">
        ${ticketCards || '<div class="empty-col">No tickets</div>'}
      </div>
    </div>`;
}