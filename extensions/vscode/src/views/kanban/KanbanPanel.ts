import * as vscode from 'vscode';
import { ApiClient } from '../../api/client';
import { AppState } from '../../state/appState';
import { getThemeColors } from '../../util/vscodeTheme';
import { renderKanban } from './renderer';
import { KanbanState } from './state';

export class KanbanPanel {
    public static readonly viewType = 'agentboard.kanbanPanel';
    public static currentPanel: KanbanPanel | undefined;

    private readonly panel: vscode.WebviewPanel;
    private apiClient: ApiClient;
    private appState: AppState;

    constructor(
        apiClient: ApiClient,
        appState: AppState,
        column: vscode.ViewColumn = vscode.ViewColumn.Active
    ) {
        this.apiClient = apiClient;
        this.appState = appState;

        this.panel = vscode.window.createWebviewPanel(
            KanbanPanel.viewType,
            'AgentBoard',
            { viewColumn: column, preserveFocus: true },
            { enableScripts: true, localResourceRoots: [] }
        );

        KanbanPanel.currentPanel = this;

        this.panel.webview.html = this.buildHtml();

        this.panel.webview.onDidReceiveMessage((msg) => {
            this.handleMessage(msg);
        });

        this.appState.onStateChange = () => {
            this.refreshView();
        };

        this.refreshView();

        this.panel.onDidDispose(() => {
            KanbanPanel.currentPanel = undefined;
        });
    }

    private buildHtml(): string {
        const colors = getThemeColors();
        const state = this.buildState();
        return renderKanban(state, colors);
    }

    private handleMessage(msg: { type: string; [key: string]: unknown }) {
        switch (msg.type) {
            case 'navigate':
                this.appState.moveSelection(msg.direction as 'up' | 'down' | 'left' | 'right');
                break;
            case 'selectTicket':
                this.selectTicket(msg.ticketId as string);
                break;
            case 'command':
                vscode.commands.executeCommand(msg.command as string);
                break;
        }
    }

    private refreshView() {
        this.panel.webview.html = this.buildHtml();
    }

    private selectTicket(ticketId: string) {
        for (let ci = 0; ci < this.appState.columns.length; ci++) {
            const col = this.appState.columns[ci];
            const ti = col.tickets.findIndex(t => t.id === ticketId);
            if (ti !== -1) {
                this.appState.selectedColumnIndex = ci;
                this.appState.selectedTicketIndex = ti;
                this.appState.selectedTicketId = ticketId;
                this.refreshView();
                return;
            }
        }
    }

    private buildState(): KanbanState {
        return {
            columns: this.appState.columns,
            selectedColumn: this.appState.selectedColumnIndex,
            selectedTicket: this.appState.selectedTicketIndex,
        };
    }

    public static open(apiClient: ApiClient, appState: AppState) {
        if (KanbanPanel.currentPanel) {
            KanbanPanel.currentPanel.panel.reveal(vscode.ViewColumn.Active, true);
            return;
        }
        new KanbanPanel(apiClient, appState);
    }
}
