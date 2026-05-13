import * as vscode from 'vscode';

export function activate(context: vscode.ExtensionContext) {
    vscode.window.showInformationMessage('AgentBoard extension activated');
}

export function deactivate() {}