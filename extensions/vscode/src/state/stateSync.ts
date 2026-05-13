import { WebviewView } from 'vscode';
import { AppState } from './appState';

export class StateSync {
    private views: Set<WebviewView> = new Set();

    constructor(private state: AppState) {
        state.onStateChange = () => this.broadcast();
    }

    register(view: WebviewView) {
        this.views.add(view);
    }

    unregister(view: WebviewView) {
        this.views.delete(view);
    }

    private broadcast() {
        const data = this.serialize();
        for (const view of this.views) {
            view.webview.postMessage({ type: 'stateUpdate', data });
        }
    }

    private serialize() {
        return {
            columns: this.state.columns,
            selectedColumnIndex: this.state.selectedColumnIndex,
            selectedTicketIndex: this.state.selectedTicketIndex,
            selectedTicketId: this.state.selectedTicketId,
            activeSessions: this.state.activeSessions,
            dashboardOpen: this.state.dashboardOpen,
        };
    }
}