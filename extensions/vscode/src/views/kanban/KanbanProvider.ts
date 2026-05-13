import * as vscode from 'vscode';
import { ApiClient } from '../../api/client';
import { AppState } from '../../state/appState';
import { StateSync } from '../../state/stateSync';
import { getThemeColors } from '../../util/vscodeTheme';
import { renderKanban } from './renderer';
import { KanbanState } from './state';

export class KanbanProvider implements vscode.WebviewViewProvider {
    public static readonly viewType = 'agentboard.kanban';

    private webviewView: vscode.WebviewView | undefined;

    constructor(
        private apiClient: ApiClient,
        private appState: AppState,
        private stateSync: StateSync
    ) {}

    resolveWebviewView(webviewView: vscode.WebviewView) {
        this.webviewView = webviewView;
        this.stateSync.register(webviewView);

        webviewView.webview.options = {
            enableScripts: true,
            localResourceRoots: [],
        };

        const colors = getThemeColors();
        const state = this.buildState();
        webviewView.webview.html = renderKanban(state, colors);

        webviewView.webview.onDidReceiveMessage((msg) => {
            this.handleMessage(msg);
        });

        this.appState.onStateChange = () => {
            this.refreshView();
        };
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
        const colors = getThemeColors();
        const state = this.buildState();
        this.webviewView!.webview.html = renderKanban(state, colors);
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
}