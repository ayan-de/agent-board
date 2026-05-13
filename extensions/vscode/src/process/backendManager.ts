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
    private binaryPath: string | null = null;
    private statusBarItem: vscode.StatusBarItem;
    private outputChannel: vscode.OutputChannel;
    private projectDir: string = '';

    constructor(outputChannel: vscode.OutputChannel) {
        this.outputChannel = outputChannel;
        this.statusBarItem = vscode.window.createStatusBarItem();
    }

    setProjectDir(dir: string) {
        this.projectDir = dir;
    }

    async restartBackend(): Promise<string> {
        if (this.backend) {
            await this.backend.stop();
            this.backend = null;
        }
        return this.startBackend();
    }

    private async startBackend(): Promise<string> {
        vscode.window.showInformationMessage('AgentBoard: starting backend...');

        const binaryName = getPlatformBinary();
        this.binaryPath = this.findLocalBinary(binaryName);

        if (!this.binaryPath) {
            vscode.window.showErrorMessage('agentboard binary not found. Please build it: go build -o agentboard ./cmd/agentboard');
            throw new Error('agentboard binary not found');
        }

        this.port = await findAvailablePort(8080);

        this.backend = new BackendProcess(this.port, this.projectDir);
        await this.backend.start(this.binaryPath);
        await this.waitForHealth();

        this.statusBarItem.text = '$(rocket) AgentBoard';
        this.statusBarItem.tooltip = `AgentBoard running on port ${this.port}`;
        this.statusBarItem.show();

        vscode.window.showInformationMessage(`AgentBoard ready on port ${this.port}`);
        return `http://localhost:${this.port}`;
    }

    async ensureRunning(): Promise<string> {
        if (this.backend?.isRunning()) {
            return `http://localhost:${this.port}`;
        }
        return this.startBackend();
    }

    private findLocalBinary(binaryName: string): string | null {
        const ext = process.platform === 'win32' ? '.exe' : '';
        const withExt = (p: string) => p.endsWith(ext) ? p : p + ext;

        const searchPaths = [
            // Development: hardcoded project root (for testing)
            withExt('/home/ayande/Project/AGENT-BOARD/agent-board/' + binaryName),
            // Development: current working directory (extension dev host cwd)
            withExt(path.join(process.cwd(), binaryName)),
            // Standalone: ~/.local/bin
            withExt(path.join(process.env.HOME || '', '.local', 'bin', binaryName)),
            // Unix typical locations
            '/usr/local/bin/' + binaryName,
            '/usr/bin/' + binaryName,
        ];

        for (const p of searchPaths) {
            if (fs.existsSync(p)) {
                return p;
            }
        }
        return null;
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

    getPort(): number {
        return this.port;
    }
}