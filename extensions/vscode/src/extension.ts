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

async function ensureBackend(): Promise<{ apiClient: ApiClient; appState: AppState }> {
    if (apiClient && appState) {
        return { apiClient, appState };
    }
    const baseUrl = await backendManager!.ensureRunning();
    apiClient = new ApiClient(baseUrl);
    appState = new AppState();
    const tickets = await apiClient.listTickets();
    appState.setTickets(tickets);
    return { apiClient, appState };
}

export function activate(context: vscode.ExtensionContext) {
    const outputChannel = vscode.window.createOutputChannel('AgentBoard');
    backendManager = new BackendManager(outputChannel);

    // Start backend in background
    backendManager.ensureRunning().then((baseUrl) => {
        apiClient = new ApiClient(baseUrl);
        appState = new AppState();
        return apiClient.listTickets();
    }).then((tickets) => {
        appState!.setTickets(tickets);
    }).catch((err) => {
        outputChannel.appendLine(`Backend start error: ${err}`);
    });

    // Open KanbanPanel (editor tab) via command
    context.subscriptions.push(
        vscode.commands.registerCommand('agentboard.open', async () => {
            try {
                const { apiClient: ac, appState: as } = await ensureBackend();
                KanbanPanel.open(ac, as);
            } catch (err) {
                vscode.window.showErrorMessage(`AgentBoard error: ${err}`);
            }
        })
    );

    // Open KanbanPanel when sidebar icon is clicked
    context.subscriptions.push(
        vscode.commands.registerCommand('agentboard.openSidebar', async () => {
            try {
                const { apiClient: ac, appState: as } = await ensureBackend();
                KanbanPanel.open(ac, as);
            } catch (err) {
                vscode.window.showErrorMessage(`AgentBoard error: ${err}`);
            }
        })
    );

    // WebviewViewProvider: sidebar view shows project list
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
                    // TODO: Switch backend to use selected project's database
                    ensureBackend().then(({ apiClient: ac, appState: as }) => {
                        KanbanPanel.open(ac, as);
                    }).catch((err) => {
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
