import * as vscode from 'vscode';
import * as path from 'path';
import * as fs from 'fs';
import { BackendProcess } from './backendProcess';

function getPlatformBinary(): string {
    const platform = process.platform;
    const arch = process.arch;
    if (platform === 'darwin' && arch === 'arm64') return 'agentboard-darwin-arm64';
    if (platform === 'darwin' && arch === 'x64') return 'agentboard-darwin-amd64';
    if (platform === 'linux') return 'agentboard';
    if (platform === 'win32') return 'agentboard.exe';
    throw new Error(`Unsupported platform: ${platform}-${arch}`);
}

function findAvailablePort(start: number): Promise<number> {
    return new Promise((resolve) => {
        const net = require('net');
        const server = net.createServer();
        server.listen(start, () => {
            server.close(() => resolve(start));
        });
        server.on('error', () => resolve(start + 1));
    });
}

export class BackendManager {
    private backend: BackendProcess | null = null;
    private port: number = 8080;
    private binaryPath: string = '';
    private statusBarItem: vscode.StatusBarItem;
    private outputChannel: vscode.OutputChannel;

    constructor(outputChannel: vscode.OutputChannel) {
        this.outputChannel = outputChannel;
        this.statusBarItem = vscode.window.createStatusBarItem();
    }

    async ensureRunning(): Promise<string> {
        if (this.backend?.isRunning()) {
            return `http://localhost:${this.port}`;
        }

        vscode.window.showInformationMessage('AgentBoard: downloading backend...');

        const binaryName = getPlatformBinary();
        const extensionPath = vscode.extensions.getExtension('ayan-de.agentboard')!.extensionUri.fsPath;
        this.binaryPath = path.join(extensionPath, 'bin', binaryName);

        if (!fs.existsSync(this.binaryPath)) {
            await this.downloadBinary(binaryName, extensionPath);
        }

        this.port = await findAvailablePort(8080);

        this.backend = new BackendProcess(this.port);
        await this.backend.start(this.binaryPath);
        await this.waitForHealth();

        this.statusBarItem.text = '$(rocket) AgentBoard';
        this.statusBarItem.tooltip = `AgentBoard running on port ${this.port}`;
        this.statusBarItem.show();

        vscode.window.showInformationMessage(`AgentBoard ready on port ${this.port}`);
        return `http://localhost:${this.port}`;
    }

    private async downloadBinary(binaryName: string, extensionPath: string): Promise<void> {
        const binDir = path.join(extensionPath, 'bin');
        fs.mkdirSync(binDir, { recursive: true });

        const targetPath = path.join(binDir, binaryName);
        const url = `https://github.com/ayan-de/agent-board/releases/latest/download/${binaryName}`;

        try {
            const response = await fetch(url);
            if (!response.ok) {
                throw new Error(`Download failed: ${response.statusText}`);
            }
            const buffer = await response.arrayBuffer();
            fs.writeFileSync(targetPath, Buffer.from(buffer));
            fs.chmodSync(targetPath, 0o755);
        } catch (err) {
            vscode.window.showErrorMessage(`Failed to download AgentBoard: ${err}`);
            throw err;
        }
    }

    private async waitForHealth(timeout = 15000): Promise<void> {
        const url = `http://localhost:${this.port}/health`;
        const deadline = Date.now() + timeout;

        while (Date.now() < deadline) {
            try {
                const res = await fetch(url);
                if (res.ok) return;
            } catch {
                // still starting
            }
            await new Promise(r => setTimeout(r, 500));
        }
        throw new Error('Backend did not become healthy in time');
    }

    async stop(): Promise<void> {
        if (this.backend) {
            await this.backend.stop();
            this.backend = null;
            this.statusBarItem.hide();
        }
    }

    getBaseUrl(): string {
        return `http://localhost:${this.port}`;
    }
}