import * as vscode from 'vscode';
import { ApiClient } from '../api/client';
import { AppState } from '../state/appState';

export function registerRuns(context: vscode.ExtensionContext, apiClient: ApiClient, appState: AppState) {
    context.subscriptions.push(
        vscode.commands.registerCommand('agentboard.startRun', async () => {
            const selected = appState.getSelectedTicket();
            if (!selected) return;
            try {
                vscode.window.showInformationMessage(`Starting run for ${selected.id}...`);
                const session = await apiClient.startApprovedRun(`prop-${selected.id}`);
                const sessions = await apiClient.getActiveSessions();
                appState.setActiveSessions(sessions);
                vscode.window.showInformationMessage(`Run started: ${session.id}`);
            } catch (err) {
                vscode.window.showErrorMessage(`Start failed: ${err}`);
            }
        }),

        vscode.commands.registerCommand('agentboard.startAdHocRun', async () => {
            const agent = await vscode.window.showQuickPick(['opencode', 'claude-code', 'codex', 'gemini'], {
                placeHolder: 'Select agent',
            });
            if (!agent) return;

            const prompt = await vscode.window.showInputBox({
                prompt: 'What should the agent do?',
                placeHolder: 'Describe the task...',
            });
            if (!prompt) return;

            try {
                const session = await apiClient.startAdHocRun({ agent, prompt });
                vscode.window.showInformationMessage(`Ad-hoc run started: ${session.id}`);
            } catch (err) {
                vscode.window.showErrorMessage(`Failed: ${err}`);
            }
        }),

        vscode.commands.registerCommand('agentboard.stopRun', async () => {
            if (!appState.selectedSessionId) return;
            try {
                await apiClient.switchToPane(appState.selectedSessionId);
            } catch (err) {
                vscode.window.showErrorMessage(`Stop failed: ${err}`);
            }
        })
    );
}