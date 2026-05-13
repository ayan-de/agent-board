import * as vscode from 'vscode';
import { ApiClient } from '../api/client';
import { AppState } from '../state/appState';
import { registerTickets } from './tickets';
import { registerProposals } from './proposals';
import { registerRuns } from './runs';
import { registerBoard } from './board';

export function registerCommands(context: vscode.ExtensionContext, apiClient: ApiClient, appState: AppState) {
    registerTickets(context, apiClient, appState);
    registerProposals(context, apiClient, appState);
    registerRuns(context, apiClient, appState);
    registerBoard(context, appState);
}