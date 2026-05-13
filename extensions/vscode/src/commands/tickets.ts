import * as vscode from 'vscode';
import { ApiClient } from '../api/client';
import { AppState } from '../state/appState';

export function registerTickets(context: vscode.ExtensionContext, apiClient: ApiClient, appState: AppState) {
    context.subscriptions.push(
        vscode.commands.registerCommand('agentboard.refreshBoard', async () => {
            try {
                const tickets = await apiClient.listTickets();
                appState.setTickets(tickets);
            } catch (err) {
                vscode.window.showErrorMessage(`Refresh failed: ${err}`);
            }
        }),

        vscode.commands.registerCommand('agentboard.addTicket', async () => {
            const title = await vscode.window.showInputBox({
                prompt: 'Ticket title',
                placeHolder: 'What needs to be done?',
            });
            if (!title) return;
            try {
                const ticket = await apiClient.createTicket({ title });
                const tickets = await apiClient.listTickets();
                appState.setTickets(tickets);
                vscode.window.showInformationMessage(`Created ${ticket.id}`);
            } catch (err) {
                vscode.window.showErrorMessage(`Failed to create ticket: ${err}`);
            }
        }),

        vscode.commands.registerCommand('agentboard.deleteTicket', async () => {
            const selected = appState.getSelectedTicket();
            if (!selected) {
                vscode.window.showWarningMessage('No ticket selected');
                return;
            }
            const confirm = await vscode.window.showWarningMessage(
                `Delete ticket ${selected.id}?`,
                { modal: true },
                'Delete', 'Cancel'
            );
            if (confirm !== 'Delete') return;
            try {
                await apiClient.deleteTicket(selected.id);
                const tickets = await apiClient.listTickets();
                appState.setTickets(tickets);
            } catch (err) {
                vscode.window.showErrorMessage(`Delete failed: ${err}`);
            }
        }),

        vscode.commands.registerCommand('agentboard.moveTicket', async (_, status: string) => {
            const selected = appState.getSelectedTicket();
            if (!selected) return;
            try {
                await apiClient.moveStatus(selected.id, status);
                const tickets = await apiClient.listTickets();
                appState.setTickets(tickets);
            } catch (err) {
                vscode.window.showErrorMessage(`Move failed: ${err}`);
            }
        }),

        vscode.commands.registerCommand('agentboard.openTicket', async () => {
            const selected = appState.getSelectedTicket();
            if (!selected) return;
            const ticket = await apiClient.getTicket(selected.id);
            const panel = vscode.window.createWebviewPanel(
                'agentboard.ticketDetail',
                `Ticket: ${ticket.id}`,
                vscode.ViewColumn.Beside,
                { enableScripts: true }
            );
            panel.webview.html = `<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><style>
body { font-family: var(--vscode-font-family, sans-serif); padding: 20px; color: var(--vscode-editor-foreground, #ccc); background: var(--vscode-editor-background, #1e1e1e); }
h1 { font-size: 16px; margin-bottom: 10px; }
.meta { color: var(--vscode-descriptionForeground, #858585); font-size: 12px; margin-bottom: 20px; }
.desc { margin-bottom: 20px; }
.field { margin-bottom: 10px; }
.label { font-weight: 600; font-size: 12px; color: var(--vscode-descriptionForeground, #858585); }
.value { font-size: 13px; }
</style></head>
<body>
<h1>${escapeHtml(ticket.title)}</h1>
<div class="meta">${ticket.id} &middot; ${ticket.status.replace('_', ' ')}</div>
<div class="desc">${escapeHtml(ticket.description || '(no description)')}</div>
<div class="field"><span class="label">Agent:</span> <span class="value">${ticket.agent || 'unassigned'}</span></div>
<div class="field"><span class="label">Branch:</span> <span class="value">${ticket.branch || '-'}</span></div>
</body>
</html>`;
        })
    );
}

function escapeHtml(s: string): string {
    return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}