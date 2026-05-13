import * as vscode from 'vscode';
import { AppState } from '../state/appState';

export function registerBoard(context: vscode.ExtensionContext, appState: AppState) {
    context.subscriptions.push(
        vscode.commands.registerCommand('agentboard.toggleDashboard', () => {
            appState.toggleDashboard();
        }),

        vscode.commands.registerCommand('agentboard.openCommandPalette', async () => {
            const items = [
                { label: 'Refresh Board', command: 'agentboard.refreshBoard' },
                { label: 'Add Ticket', command: 'agentboard.addTicket' },
                { label: 'Delete Ticket', command: 'agentboard.deleteTicket' },
                { label: 'Create Proposal', command: 'agentboard.createProposal' },
                { label: 'Approve Proposal', command: 'agentboard.approveProposal' },
                { label: 'Start Run', command: 'agentboard.startRun' },
                { label: 'Start Ad-Hoc Run', command: 'agentboard.startAdHocRun' },
                { label: 'Toggle Dashboard', command: 'agentboard.toggleDashboard' },
                { label: 'Show Help', command: 'agentboard.showHelp' },
            ];

            const selected = await vscode.window.showQuickPick(
                items.map(i => ({ label: i.label })),
                { placeHolder: 'AgentBoard commands...' }
            );

            if (selected) {
                const cmd = items.find(i => i.label === selected.label)?.command;
                if (cmd) vscode.commands.executeCommand(cmd);
            }
        }),

        vscode.commands.registerCommand('agentboard.showHelp', () => {
            vscode.window.showInformationMessage(
                'AgentBoard: Kanban Board for AI Coding Agents\n\n' +
                'Navigate: Arrow keys / h,j,k,l\n' +
                'Add: a | Delete: d | Refresh: r\n' +
                'Start agent: s | Approve: p\n' +
                'Dashboard: i | Command Palette: :'
            );
        }),

        vscode.commands.registerCommand('agentboard.columnLeft', () => {
            appState.moveSelection('left');
        }),

        vscode.commands.registerCommand('agentboard.columnRight', () => {
            appState.moveSelection('right');
        }),

        vscode.commands.registerCommand('agentboard.ticketUp', () => {
            appState.moveSelection('up');
        }),

        vscode.commands.registerCommand('agentboard.ticketDown', () => {
            appState.moveSelection('down');
        }),

        vscode.commands.registerCommand('agentboard.goToColumn', async (_, idx: number) => {
            appState.selectColumn(idx - 1);
        })
    );
}