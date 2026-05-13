import * as vscode from 'vscode';
import { BackendManager } from './process/backendManager';

let backendManager: BackendManager;

export function activate(context: vscode.ExtensionContext) {
    const outputChannel = vscode.window.createOutputChannel('AgentBoard');
    backendManager = new BackendManager(outputChannel);

    backendManager.ensureRunning().catch(err => {
        vscode.window.showErrorMessage(`AgentBoard failed to start: ${err.message}`);
    });

    context.subscriptions.push(
        vscode.commands.registerCommand('agentboard.open', () => {
            backendManager.ensureRunning();
        })
    );
}

export function deactivate() {
    backendManager?.stop();
}