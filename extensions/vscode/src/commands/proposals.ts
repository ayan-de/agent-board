import * as vscode from 'vscode';
import { ApiClient } from '../api/client';
import { AppState } from '../state/appState';

export function registerProposals(context: vscode.ExtensionContext, apiClient: ApiClient, appState: AppState) {
    context.subscriptions.push(
        vscode.commands.registerCommand('agentboard.createProposal', async () => {
            const selected = appState.getSelectedTicket();
            if (!selected) {
                vscode.window.showWarningMessage('Select a ticket first');
                return;
            }
            vscode.window.showInformationMessage(`Creating proposal for ${selected.id}...`);
            try {
                const proposal = await apiClient.createProposal({ ticketID: selected.id });
                vscode.window.showInformationMessage(`Proposal ${proposal.id} created`);
            } catch (err) {
                vscode.window.showErrorMessage(`Failed: ${err}`);
            }
        }),

        vscode.commands.registerCommand('agentboard.approveProposal', async () => {
            const selected = appState.getSelectedTicket();
            if (!selected) return;
            try {
                const proposals = await apiClient.getProposal(`prop-${selected.id}`).catch(() => null);
                if (!proposals) {
                    vscode.window.showWarningMessage('No proposal found for this ticket');
                    return;
                }
                await apiClient.approveProposal(`prop-${selected.id}`);
                vscode.window.showInformationMessage(`Proposal approved`);
            } catch (err) {
                vscode.window.showErrorMessage(`Failed: ${err}`);
            }
        })
    );
}