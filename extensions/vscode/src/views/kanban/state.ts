import { Ticket } from '../../api/types';

export interface KanbanState {
    columns: { name: string; status: string; tickets: Ticket[] }[];
    selectedColumn: number;
    selectedTicket: number;
}