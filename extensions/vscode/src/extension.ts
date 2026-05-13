import * as vscode from 'vscode';
import * as path from 'path';
import * as fs from 'fs';
import { BackendManager } from './process/backendManager';
import { ApiClient } from './api/client';
import { AppState } from './state/appState';
import { KanbanPanel } from './views/kanban/KanbanPanel';
import { renderSidebarHtml, SidebarProject } from './views/sidebar/sidebarTemplate';

let backendManager: BackendManager;
let apiClient: ApiClient | undefined;
let appState: AppState | undefined;

export function activate(context: vscode.ExtensionContext) {
    const outputChannel = vscode.window.createOutputChannel('AgentBoard');
    backendManager = new BackendManager(outputChannel);

    const initBackend = async () => {
        const baseUrl = await backendManager!.ensureRunning();
        apiClient = new ApiClient(baseUrl);
        appState = new AppState();
        const tickets = await apiClient.listTickets();
        appState.setTickets(tickets);
    };

    backendManager.ensureRunning().then(() => {
        return initBackend();
    }).catch((err) => {
        outputChannel.appendLine(`Backend start error: ${err}`);
    });

    context.subscriptions.push(
        vscode.commands.registerCommand('agentboard.open', async () => {
            if (!apiClient || !appState) {
                await initBackend();
            }
            if (apiClient && appState) {
                KanbanPanel.open(apiClient, appState);
            }
        })
    );

    context.subscriptions.push(
        vscode.commands.registerCommand('agentboard.refreshBoard', async () => {
            if (!apiClient || !appState) return;
            const tickets = await apiClient.listTickets();
            appState.setTickets(tickets);
        })
    );

    context.subscriptions.push(
        vscode.commands.registerCommand('agentboard.addTicket', async () => {
            if (!apiClient || !appState) return;
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
        })
    );

    context.subscriptions.push(
        vscode.commands.registerCommand('agentboard.deleteTicket', async () => {
            if (!apiClient || !appState) return;
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
        })
    );

    context.subscriptions.push(
        vscode.commands.registerCommand('agentboard.columnLeft', () => {
            appState?.moveSelection('left');
        })
    );

    context.subscriptions.push(
        vscode.commands.registerCommand('agentboard.columnRight', () => {
            appState?.moveSelection('right');
        })
    );

    context.subscriptions.push(
        vscode.commands.registerCommand('agentboard.ticketUp', () => {
            appState?.moveSelection('up');
        })
    );

    context.subscriptions.push(
        vscode.commands.registerCommand('agentboard.ticketDown', () => {
            appState?.moveSelection('down');
        })
    );

    const provider: vscode.WebviewViewProvider = {
        resolveWebviewView(view) {
            view.webview.options = { enableScripts: true };

            const projectsDir = path.join(process.env.HOME || '', '.agentboard', 'projects');

            let projects: SidebarProject[] = [];
            try {
                projects = fs.readdirSync(projectsDir)
                    .filter(f => fs.statSync(path.join(projectsDir, f)).isDirectory())
                    .map(name => {
                        const projPath = path.join(projectsDir, name);
                        const dbPath = path.join(projPath, 'board.db');
                        return { name, path: projPath, hasDb: fs.existsSync(dbPath) };
                    });
            } catch {
                projects = [];
            }

            view.webview.html = renderSidebarHtml(projects);

            view.webview.onDidReceiveMessage((msg) => {
                if (msg.type === 'selectProject') {
                    initBackend().catch((err) => {
                        vscode.window.showErrorMessage(`Error opening project: ${err}`);
                    });
                }
            });
        }
    };

    context.subscriptions.push(
        vscode.window.registerWebviewViewProvider('agentboard.kanban', provider)
    );

    context.subscriptions.push(
        vscode.commands.registerCommand('agentboard.helloWorld', () => {
            vscode.window.showInformationMessage('AgentBoard extension activated');
        })
    );
}

export function deactivate() {
    backendManager?.stop();
}