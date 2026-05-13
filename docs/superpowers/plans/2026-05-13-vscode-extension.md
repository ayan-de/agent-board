# VSCode Extension — AgentBoard Frontend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a VSCode extension that lets users manage AgentBoard tickets, spawn AI agents, and monitor runs — entirely from within VSCode. Extension downloads and manages the Go backend binary automatically.

**Architecture:** Extension acts as an HTTP client + process manager for the Go API server (`agentboard --api`). Webviews render the Kanban board using VSCode theme colors. State lives in the extension host; webviews are purely presentational.

**Tech Stack:** TypeScript, VSCode Extension API (webviews, commands, QuickPick), Node.js process management, native `fetch` / WebSocket, HTML/CSS for webview rendering.

---

## File Structure

```
extensions/vscode/
├── src/
│   ├── extension.ts
│   ├── commands/
│   │   ├── index.ts
│   │   ├── tickets.ts
│   │   ├── proposals.ts
│   │   ├── runs.ts
│   │   ├── sessions.ts
│   │   └── board.ts
│   ├── views/
│   │   ├── kanban/
│   │   │   ├── KanbanProvider.ts
│   │   │   ├── state.ts
│   │   │   ├── renderer.ts
│   │   │   ├── column.ts
│   │   │   └── ticketCard.ts
│   │   ├── ticketDetail/
│   │   │   ├── TicketDetailProvider.ts
│   │   │   └── renderer.ts
│   │   ├── dashboard/
│   │   │   ├── DashboardProvider.ts
│   │   │   └── renderer.ts
│   │   └── shared/
│   │       ├── styles.ts
│   │       └── icons.ts
│   ├── api/
│   │   ├── client.ts
│   │   ├── types.ts
│   │   ├── endpoints.ts
│   │   └── wsClient.ts
│   ├── process/
│   │   ├── backendManager.ts
│   │   └── backendProcess.ts
│   ├── state/
│   │   ├── appState.ts
│   │   └── stateSync.ts
│   └── util/
│       ├── logger.ts
│       └── vscodeTheme.ts
├── package.json
├── tsconfig.json
└── vsc-extension-quickstart.md
```

**Dependency order:** Go API needs `/health` endpoint first (Task 0), then extension scaffold (Task 1), then backend manager (Task 2), then API client (Task 3), then views (Tasks 4-6), then commands (Tasks 7-8), then wiring + keybindings (Task 9).

---

## Task 0: Add `/health` endpoint to Go API

**Files:**
- Modify: `internal/api/server.go:1-20`
- Modify: `internal/api/handlers.go:1-10`

The extension will poll `GET /health` on startup to know when the backend is ready. This endpoint currently doesn't exist.

- [ ] **Step 1: Add health handler to handlers.go**

Add this handler after the existing handlers:

```go
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
```

- [ ] **Step 2: Register /health route in server.go**

Find the section where routes are registered (look for `r.Route("/api", ...)` or similar pattern). Add:

```go
r.Get("/health", h.Health)
```

The health endpoint should be outside the `/api` prefix so it's reachable without authentication headers.

- [ ] **Step 3: Verify the endpoint works**

Run: `go build -o agentboard ./cmd/agentboard && ./agentboard --api --addr :8081 &`
Then: `curl http://localhost:8081/health`
Expected: `ok`

Kill the background process after testing.

- [ ] **Step 4: Commit**

```bash
git add internal/api/handlers.go internal/api/server.go
git commit -m "feat(api): add /health endpoint for extension startup check"
```

---

## Task 1: Scaffold VSCode Extension

**Files:**
- Create: `extensions/vscode/package.json`
- Create: `extensions/vscode/tsconfig.json`
- Create: `extensions/vscode/src/extension.ts`
- Create: `extensions/vscode/src/test/runTest.ts`
- Create: `extensions/vscode/src/test/suite/extension.test.ts`
- Create: `extensions/vscode/src/test/suite/index.ts`
- Create: `extensions/vscode/.vscode/extensions.json`
- Create: `extensions/vscode/.vscode/launch.json`
- Create: `extensions/vscode/.vscodeignore`
- Create: `extensions/vscode/README.md`

Use the official `yo code` generator as the base. After generating, prune the test files you don't need and add the proper extension name/publisher.

- [ ] **Step 1: Generate extension scaffold**

Run in the `extensions/` directory:
```bash
npm install -g yo generator-code
mkdir -p vscode && cd vscode
yo code --template=ext-ts --name=agentboard --displayName="AgentBoard" --publisher=ayan-de --description="Kanban board for AI coding agents" --outputDir=.
```

Answer prompts: TypeScript, yes to git, no to webview (we'll add manually).

- [ ] **Step 2: Review generated files**

Open `package.json` and check:
- `name` should be `agentboard` or `agentboard-vscode`
- `publisher` should be `ayan-de`
- `activationEvents` should include `"onCommand:agentboard.helloWorld"`
- `contributes.commands` should include the hello world command

- [ ] **Step 3: Replace extension.ts with minimal activation**

Replace the generated `extension.ts` with:

```typescript
import * as vscode from 'vscode';

export function activate(context: vscode.ExtensionContext) {
    vscode.window.showInformationMessage('AgentBoard extension activated');
}

export function deactivate() {}
```

- [ ] **Step 4: Verify extension loads in development**

Run the extension (F5 in VSCode). Check the output panel for errors and confirm the notification appears.

- [ ] **Step 5: Commit**

```bash
git add extensions/vscode/
git commit -m "feat(vscode): scaffold extension with yo code"
```

---

## Task 2: Backend Manager (download + spawn + lifecycle)

**Files:**
- Create: `extensions/vscode/src/process/backendProcess.ts`
- Create: `extensions/vscode/src/process/backendManager.ts`
- Modify: `extensions/vscode/src/extension.ts` (register backend lifecycle)
- Modify: `extensions/vscode/package.json` (add extension dependency: `@vscode/sqlite3` not needed — extension doesn't use DB)

- [ ] **Step 1: Write backendProcess.ts**

```typescript
import { ChildProcess, spawn, kill } from 'child_process';

export interface BackendConfig {
    port: number;
    storagePath: string;  // Extension storage path for binary
}

export class BackendProcess {
    private process: ChildProcess | null = null;
    private port: number;

    constructor(port: number) {
        this.port = port;
    }

    start(binaryPath: string): Promise<void> {
        return new Promise((resolve, reject) => {
            this.process = spawn(binaryPath, ['--api', '--addr', `:${this.port}`], {
                stdio: ['ignore', 'pipe', 'pipe'],
                detached: false,
            });

            this.process.stdout?.on('data', (data) => {
                console.log(`[agentboard] ${data.toString().trim()}`);
            });

            this.process.stderr?.on('data', (data) => {
                console.error(`[agentboard] ${data.toString().trim()}`);
            });

            this.process.on('error', reject);
            this.process.on('exit', (code) => {
                if (code !== 0 && code !== null) {
                    console.error(`[agentboard] exited with code ${code}`);
                }
            });

            // Give the server a moment to start
            setTimeout(resolve, 500);
        });
    }

    stop(): Promise<void> {
        return new Promise((resolve) => {
            if (!this.process) {
                resolve();
                return;
            }
            this.process.on('exit', () => resolve());
            this.process.kill('SIGTERM');
            setTimeout(() => {
                if (this.process) {
                    this.process.kill('SIGKILL');
                }
                resolve();
            }, 3000);
        });
    }

    isRunning(): boolean {
        return this.process !== null && !this.process.killed;
    }
}
```

- [ ] **Step 2: Write backendManager.ts**

```typescript
import * as vscode from 'vscode';
import * as os from 'os';
import * as path from 'path';
import * as fs from 'fs';
import { BackendProcess, BackendConfig } from './backendProcess';

const RELEASE_URL = 'https://api.github.com/repos/ayan-de/agent-board/releases/latest';

function getPlatformBinary(): string {
    const platform = process.platform;
    const arch = process.arch;
    if (platform === 'darwin' && arch === 'arm64') return 'agentboard-darwin-arm64';
    if (platform === 'darwin' && arch === 'x64') return 'agentboard-darwin-amd64';
    if (platform === 'linux') return 'agentboard';
    if (platform === 'win32') return 'agentboard.exe';
    throw new Error(`Unsupported platform: ${platform}-${arch}`);
}

function findAvailablePort(start: number): number {
    const net = require('net');
    return new Promise((resolve) => {
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

        // Detect OS and get correct binary name
        const binaryName = getPlatformBinary();
        const storagePath = vscode.extensions.getExtension('ayan-de.agentboard')!
            .extensionUri.fsPath;
        this.binaryPath = path.join(storagePath, 'bin', binaryName);

        // Download if not exists
        if (!fs.existsSync(this.binaryPath)) {
            await this.downloadBinary(binaryName, storagePath);
        }

        // Find available port
        this.port = await findAvailablePort(8080);

        // Spawn backend
        this.backend = new BackendProcess(this.port);
        await this.backend.start(this.binaryPath);

        // Wait for health check
        await this.waitForHealth();

        this.statusBarItem.text = `$(rocket) AgentBoard`;
        this.statusBarItem.tooltip = `AgentBoard running on port ${this.port}`;
        this.statusBarItem.show();

        vscode.window.showInformationMessage(`AgentBoard ready on port ${this.port}`);
        return `http://localhost:${this.port}`;
    }

    private async downloadBinary(binaryName: string, storagePath: string): Promise<void> {
        const binDir = path.join(storagePath, 'bin');
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
```

- [ ] **Step 3: Update extension.ts to use BackendManager**

```typescript
import { BackendManager } from './process/backendManager';

let backendManager: BackendManager;

export function activate(context: vscode.ExtensionContext) {
    const outputChannel = vscode.window.createOutputChannel('AgentBoard');
    backendManager = new BackendManager(outputChannel);

    // Start backend on activation
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
```

- [ ] **Step 4: Test backend starts**

Run the extension (F5). Check:
1. Output channel shows "AgentBoard: downloading backend..." or uses existing binary
2. Status bar shows "🚀 AgentBoard" after startup
3. `curl http://localhost:8080/health` returns `ok`

- [ ] **Step 5: Commit**

```bash
git add extensions/vscode/src/process/ extensions/vscode/src/extension.ts
git commit -m "feat(vscode): backend manager — download, spawn, health check"
```

---

## Task 3: API Client (HTTP + WebSocket + types)

**Files:**
- Create: `extensions/vscode/src/api/types.ts`
- Create: `extensions/vscode/src/api/endpoints.ts`
- Create: `extensions/vscode/src/api/client.ts`
- Create: `extensions/vscode/src/api/wsClient.ts`

- [ ] **Step 1: Write api/types.ts**

```typescript
// Matches store.Ticket (internal/store/tickets.go)
export interface Ticket {
    id: string;
    title: string;
    description: string;
    status: 'backlog' | 'in_progress' | 'review' | 'done';
    agent: string | null;
    branch: string;
    createdAt: string;
    updatedAt: string;
    dependsOn: string;
}

// Matches store.Proposal
export interface Proposal {
    id: string;
    ticketID: string;
    agent: string;
    prompt: string;
    status: 'pending' | 'approved' | 'running' | 'completed';
    createdAt: string;
}

// Matches core/types.go:AgentSession
export interface AgentSession {
    sessionID: string;
    ticketID: string;
    agent: string;
    startedAt: number;
    status: 'running' | 'completed' | 'failed' | 'cancelled';
    paneID: string;
    windowID: string;
}

// Matches core/types.go:Session (store.Session)
export interface Session {
    id: string;
    ticketID: string;
    agent: string;
    startedAt: string;
    endedAt: string | null;
    status: 'running' | 'completed' | 'failed' | 'cancelled';
    contextKey: string;
}

// Matches core/types.go:RunCompletion
export interface RunCompletion {
    ticketID: string;
    sessionID: string;
    outcome: string;
    summary: string;
    resumeCommand: string;
}

// Request bodies
export interface CreateProposalInput {
    ticketID: string;
}

export interface FinishRunInput {
    ticketID: string;
    sessionID: string;
    outcome: string;
    summary: string;
    resumeCommand: string;
}

export interface StartAdHocRunInput {
    agent: string;
    prompt: string;
}

export interface MoveStatusInput {
    status: string;
}

export interface CreateTicketInput {
    title: string;
    description?: string;
    status?: string;
}

export interface SendInputRequest {
    input: string;
}

// Kanban column definition
export interface ColumnDef {
    name: string;
    status: string;
}
```

- [ ] **Step 2: Write api/endpoints.ts**

```typescript
// Single source of truth for all API URL paths

export const Tickets = {
    list: () => '/tickets',
    create: () => '/tickets',
    get: (id: string) => `/tickets/${id}`,
    update: (id: string) => `/tickets/${id}`,
    delete: (id: string) => `/tickets/${id}`,
    moveStatus: (id: string) => `/tickets/${id}/status`,
} as const;

export const Proposals = {
    create: () => '/proposals',
    get: (id: string) => `/proposals/${id}`,
    approve: (id: string) => `/proposals/${id}/approve`,
} as const;

export const Runs = {
    startApproved: (proposalId: string) => `/runs/${proposalId}/start`,
    startAdHoc: () => '/runs/adhoc',
    finish: (sessionId: string) => `/runs/${sessionId}/finish`,
} as const;

export const Sessions = {
    list: () => '/sessions',
    listActive: () => '/sessions/list',
    logs: (id: string) => `/sessions/${id}/logs`,
    pane: (id: string) => `/sessions/${id}/pane`,
    switch: (id: string) => `/sessions/${id}/switch`,
} as const;

export const WebSocket = {
    global: () => '/ws?session=global',
} as const;
```

- [ ] **Step 3: Write api/client.ts**

```typescript
import {
    Ticket, Proposal, AgentSession, Session, RunCompletion,
    CreateProposalInput, FinishRunInput, StartAdHocRunInput,
    MoveStatusInput, CreateTicketInput, ColumnDef
} from './types';
import * as endpoints from './endpoints';

export interface ListTicketsOptions {
    status?: string;
}

export class ApiClient {
    private baseUrl: string;

    constructor(baseUrl: string) {
        this.baseUrl = baseUrl;
    }

    private url(path: string): string {
        return this.baseUrl + path;
    }

    // Tickets
    async listTickets(options: ListTicketsOptions = {}): Promise<Ticket[]> {
        const url = options.status
            ? `${this.url(endpoints.Tickets.list())}?status=${options.status}`
            : this.url(endpoints.Tickets.list());
        const res = await fetch(url);
        if (!res.ok) throw new Error(await res.text());
        return res.json();
    }

    async getTicket(id: string): Promise<Ticket> {
        const res = await fetch(this.url(endpoints.Tickets.get(id)));
        if (!res.ok) throw new Error(await res.text());
        return res.json();
    }

    async createTicket(input: CreateTicketInput): Promise<Ticket> {
        const res = await fetch(this.url(endpoints.Tickets.create()), {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(input),
        });
        if (!res.ok) throw new Error(await res.text());
        return res.json();
    }

    async updateTicket(id: string, ticket: Partial<Ticket>): Promise<Ticket> {
        const res = await fetch(this.url(endpoints.Tickets.update(id)), {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(ticket),
        });
        if (!res.ok) throw new Error(await res.text());
        return res.json();
    }

    async deleteTicket(id: string): Promise<void> {
        const res = await fetch(this.url(endpoints.Tickets.delete(id)), { method: 'DELETE' });
        if (!res.ok) throw new Error(await res.text());
    }

    async moveStatus(id: string, status: string): Promise<void> {
        const res = await fetch(this.url(endpoints.Tickets.moveStatus(id)), {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ status } as MoveStatusInput),
        });
        if (!res.ok) throw new Error(await res.text());
    }

    // Proposals
    async createProposal(input: CreateProposalInput): Promise<Proposal> {
        const res = await fetch(this.url(endpoints.Proposals.create()), {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(input),
        });
        if (!res.ok) throw new Error(await res.text());
        return res.json();
    }

    async getProposal(id: string): Promise<Proposal> {
        const res = await fetch(this.url(endpoints.Proposals.get(id)));
        if (!res.ok) throw new Error(await res.text());
        return res.json();
    }

    async approveProposal(id: string): Promise<void> {
        const res = await fetch(this.url(endpoints.Proposals.approve(id)), { method: 'POST' });
        if (!res.ok) throw new Error(await res.text());
    }

    // Runs
    async startApprovedRun(proposalId: string): Promise<Session> {
        const res = await fetch(this.url(endpoints.Runs.startApproved(proposalId)), { method: 'POST' });
        if (!res.ok) throw new Error(await res.text());
        return res.json();
    }

    async startAdHocRun(input: StartAdHocRunInput): Promise<Session> {
        const res = await fetch(this.url(endpoints.Runs.startAdHoc()), {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(input),
        });
        if (!res.ok) throw new Error(await res.text());
        return res.json();
    }

    async finishRun(input: FinishRunInput): Promise<void> {
        const res = await fetch(this.url(endpoints.Runs.finish(input.sessionID)), {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(input),
        });
        if (!res.ok) throw new Error(await res.text());
    }

    // Sessions
    async getActiveSessions(): Promise<AgentSession[]> {
        const res = await fetch(this.url(endpoints.Sessions.list()));
        if (!res.ok) throw new Error(await res.text());
        return res.json();
    }

    async getPaneContent(sessionId: string, lines = 100): Promise<string> {
        const res = await fetch(`${this.url(endpoints.Sessions.pane(sessionId))}?lines=${lines}`);
        if (!res.ok) throw new Error(await res.text());
        const data = await res.json() as { content: string };
        return data.content;
    }

    async switchToPane(sessionId: string): Promise<void> {
        const res = await fetch(this.url(endpoints.Sessions.switch(sessionId)), { method: 'POST' });
        if (!res.ok) throw new Error(await res.text());
    }

    async getLogs(sessionId: string): Promise<string[]> {
        const res = await fetch(this.url(endpoints.Sessions.logs(sessionId)));
        if (!res.ok) throw new Error(await res.text());
        const data = await res.json() as { logs: string[] };
        return data.logs;
    }
}
```

- [ ] **Step 4: Write api/wsClient.ts**

```typescript
import { RunCompletion } from './types';

export type WsMessageHandler = (completion: RunCompletion) => void;

export class WsClient {
    private ws: WebSocket | null = null;
    private handlers: WsMessageHandler[] = [];
    private url: string;
    private reconnectDelay = 1000;
    private maxReconnectDelay = 30000;
    private reconnectTimer: ReturnType<typeof setTimeout> | null = null;

    constructor(url: string) {
        this.url = url;
    }

    connect(): void {
        try {
            this.ws = new WebSocket(this.url);

            this.ws.onopen = () => {
                this.reconnectDelay = 1000;
            };

            this.ws.onmessage = (event) => {
                try {
                    const msg = JSON.parse(event.data) as { type: string; data: RunCompletion };
                    if (msg.type === 'run_completion') {
                        this.handlers.forEach(h => h(msg.data));
                    }
                } catch {
                    // ignore parse errors
                }
            };

            this.ws.onclose = () => {
                this.scheduleReconnect();
            };

            this.ws.onerror = () => {
                this.ws?.close();
            };
        } catch {
            this.scheduleReconnect();
        }
    }

    private scheduleReconnect(): void {
        if (this.reconnectTimer) return;
        this.reconnectTimer = setTimeout(() => {
            this.reconnectTimer = null;
            this.connect();
            this.reconnectDelay = Math.min(this.reconnectDelay * 2, this.maxReconnectDelay);
        }, this.reconnectDelay);
    }

    onCompletion(handler: WsMessageHandler): void {
        this.handlers.push(handler);
    }

    disconnect(): void {
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
        this.ws?.close();
        this.ws = null;
    }
}
```

- [ ] **Step 5: Verify types compile**

Run: `cd extensions/vscode && npx tsc --noEmit`
Expected: No errors (types match Go structs)

- [ ] **Step 6: Commit**

```bash
git add extensions/vscode/src/api/
git commit -m "feat(vscode): API client — HTTP, WebSocket, types"
```

---

## Task 4: State Management

**Files:**
- Create: `extensions/vscode/src/state/appState.ts`
- Create: `extensions/vscode/src/state/stateSync.ts`
- Modify: `extensions/vscode/src/extension.ts` (initialize state)

- [ ] **Step 1: Write state/appState.ts**

```typescript
import { Ticket, AgentSession, ColumnDef } from '../api/types';

export interface KanbanColumn {
    name: string;
    status: string;
    tickets: Ticket[];
}

export class AppState {
    columns: KanbanColumn[] = [
        { name: 'Backlog', status: 'backlog', tickets: [] },
        { name: 'In Progress', status: 'in_progress', tickets: [] },
        { name: 'Review', status: 'review', tickets: [] },
        { name: 'Done', status: 'done', tickets: [] },
    ];

    selectedColumnIndex: number = 0;
    selectedTicketIndex: number = 0;
    selectedTicketId: string | null = null;

    activeSessions: AgentSession[] = [];
    dashboardOpen: boolean = false;
    selectedSessionId: string | null = null;

    onStateChange?: (state: AppState) => void;

    private emit() {
        this.onStateChange?.(this);
    }

    setTickets(tickets: Ticket[]) {
        for (const col of this.columns) {
            col.tickets = tickets.filter(t => t.status === col.status);
        }
        this.emit();
    }

    selectColumn(index: number) {
        this.selectedColumnIndex = index;
        this.selectedTicketIndex = 0;
        const col = this.columns[index];
        if (col.tickets.length > 0) {
            this.selectedTicketId = col.tickets[0].id;
        } else {
            this.selectedTicketId = null;
        }
        this.emit();
    }

    moveSelection(direction: 'up' | 'down' | 'left' | 'right') {
        if (direction === 'left') {
            if (this.selectedColumnIndex > 0) this.selectColumn(this.selectedColumnIndex - 1);
            return;
        }
        if (direction === 'right') {
            if (this.selectedColumnIndex < this.columns.length - 1) this.selectColumn(this.selectedColumnIndex + 1);
            return;
        }
        const col = this.columns[this.selectedColumnIndex];
        if (direction === 'up') {
            if (this.selectedTicketIndex > 0) {
                this.selectedTicketIndex--;
                this.selectedTicketId = col.tickets[this.selectedTicketIndex]?.id ?? null;
                this.emit();
            }
        }
        if (direction === 'down') {
            if (this.selectedTicketIndex < col.tickets.length - 1) {
                this.selectedTicketIndex++;
                this.selectedTicketId = col.tickets[this.selectedTicketIndex]?.id ?? null;
                this.emit();
            }
        }
    }

    setActiveSessions(sessions: AgentSession[]) {
        this.activeSessions = sessions;
        this.emit();
    }

    toggleDashboard() {
        this.dashboardOpen = !this.dashboardOpen;
        this.emit();
    }

    getSelectedTicket(): Ticket | null {
        const col = this.columns[this.selectedColumnIndex];
        return col.tickets.find(t => t.id === this.selectedTicketId) ?? null;
    }
}
```

- [ ] **Step 2: Write state/stateSync.ts**

```typescript
import { WebviewView } from 'vscode';
import { AppState } from './appState';

export class StateSync {
    private views: Set<WebviewView> = new Set();

    constructor(private state: AppState) {
        state.onStateChange = (s) => this.broadcast(s);
    }

    register(view: WebviewView) {
        this.views.add(view);
    }

    unregister(view: WebviewView) {
        this.views.delete(view);
    }

    private broadcast(state: AppState) {
        const payload = JSON.stringify({ type: 'stateUpdate', data: this.serialize(state) });
        for (const view of this.views) {
            view.webview.postMessage({ type: 'stateUpdate', data: this.serialize(state) });
        }
    }

    private serialize(state: AppState) {
        return {
            columns: state.columns,
            selectedColumnIndex: state.selectedColumnIndex,
            selectedTicketIndex: state.selectedTicketIndex,
            selectedTicketId: state.selectedTicketId,
            activeSessions: state.activeSessions,
            dashboardOpen: state.dashboardOpen,
        };
    }
}
```

- [ ] **Step 3: Update extension.ts to initialize state**

Add to `activate()`:

```typescript
import { AppState } from './state/appState';
import { StateSync } from './state/stateSync';

const appState = new AppState();
const stateSync = new StateSync(appState);
```

- [ ] **Step 4: Commit**

```bash
git add extensions/vscode/src/state/ extensions/vscode/src/extension.ts
git commit -m "feat(vscode): app state + state sync"
```

---

## Task 5: VSCode Theme Color System

**Files:**
- Create: `extensions/vscode/src/util/vscodeTheme.ts`
- Create: `extensions/vscode/src/views/shared/styles.ts`

- [ ] **Step 1: Write util/vscodeTheme.ts**

```typescript
import * as vscode from 'vscode';

export interface ThemeColors {
    '--ab-bg-primary': string;
    '--ab-bg-secondary': string;
    '--ab-bg-tertiary': string;
    '--ab-bg-elevated': string;
    '--ab-fg-primary': string;
    '--ab-fg-secondary': string;
    '--ab-fg-muted': string;
    '--ab-border': string;
    '--ab-border-subtle': string;
    '--ab-accent-primary': string;
    '--ab-accent-blue': string;
    '--ab-accent-green': string;
    '--ab-accent-yellow': string;
    '--ab-accent-red': string;
    '--ab-col-backlog': string;
    '--ab-col-progress': string;
    '--ab-col-review': string;
    '--ab-col-done': string;
    '--ab-selection-bg': string;
    '--ab-selection-fg': string;
}

function getColor(key: string, fallback: string): string {
    const val = vscode.workspace.getConfiguration('workbench').get<string>(key);
    return val ?? fallback;
}

function getColorValue(token: string): string {
    try {
        const color = new vscode.ThemeColor(token);
        const resolved = vscode.TreeViewItemResolveTextColor !== undefined
            ? vscode.workspace.getConfiguration('editor').get<string>(token.replace('.', ' '))
            : undefined;
        // Use vscode's getColor method from ThemeColor API
        return `var(--vscode-${token.replace('.', '-')})`;
    } catch {
        return '#808080';
    }
}

export function getThemeColors(): ThemeColors {
    const fgPrimary = getColor('colorForeground', '#cccccc');
    const bgEditor = getColor('colorEditorBackground', '#1e1e1e');
    const bgSideBar = getColor('colorSideBarBackground', '#252526');
    const bgActivityBar = getColor('colorActivityBarBackground', '#333333');
    const borderColor = getColor('colorEditorWidget.border', '#3c3c3c');
    const panelBorder = getColor('colorPanelBorder', '#3c3c3c');
    const activeIcon = getColor('colorActiveIconForeground', '#ffffff');
    const selectionBg = getColor('colorSelectionBackground', '#264f78');
    const selectionFg = getColor('colorSelectionForeground', '#ffffff');
    const errorFg = getColor('colorErrorForeground', '#f48771');
    const warningFg = getColor('colorWarningForeground', '#dcdcaa');
    const modifiedFg = getColor('colorModifiedForeground', '#6d984a');

    const tintBacklog = '#3c3c3c';
    const tintProgress = '#2d4a6d';
    const tintReview = '#4a3d6d';
    const tintDone = '#2d4a2d';

    return {
        '--ab-bg-primary': bgActivityBar,
        '--ab-bg-secondary': bgSideBar,
        '--ab-bg-tertiary': bgEditor,
        '--ab-bg-elevated': getColor('colorDropdown.background', '#3c3c3c'),
        '--ab-fg-primary': fgPrimary,
        '--ab-fg-secondary': getColor('colorDescriptionForeground', '#858585'),
        '--ab-fg-muted': getColor('colorEditorWidget.foreground', '#6e6e6e'),
        '--ab-border': borderColor,
        '--ab-border-subtle': panelBorder,
        '--ab-accent-primary': activeIcon,
        '--ab-accent-blue': selectionBg,
        '--ab-accent-green': modifiedFg,
        '--ab-accent-yellow': warningFg,
        '--ab-accent-red': errorFg,
        '--ab-col-backlog': tintBacklog,
        '--ab-col-progress': tintProgress,
        '--ab-col-review': tintReview,
        '--ab-col-done': tintDone,
        '--ab-selection-bg': selectionBg,
        '--ab-selection-fg': selectionFg,
    };
}

export function injectThemeColorsScript(colors: ThemeColors): string {
    const entries = Object.entries(colors)
        .map(([k, v]) => `document.body.style.setProperty('${k}', '${v}');`)
        .join('\n    ');
    return `<script>
    (function() {
        ${entries}
    })();
    </script>`;
}
```

- [ ] **Step 2: Write views/shared/styles.ts**

```typescript
import { ThemeColors } from '../../util/vscodeTheme';

export function generateBaseCSS(colors: ThemeColors): string {
    const colorEntries = Object.entries(colors)
        .map(([k, v]) => `  ${k}: ${v};`)
        .join('\n');

    return `
:root {
${colorEntries}
}

* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

body {
  font-family: var(--vscode-font-family, 'Segoe WPC', 'Segoe UI', sans-serif);
  font-size: var(--vscode-font-size, 13px);
  color: var(----ab-fg-primary, #cccccc);
  background: var(--ab-bg-tertiary, #1e1e1e);
  overflow: hidden;
  height: 100vh;
}

.ab-root {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
}

.ab-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--ab-bg-secondary, #252526);
  border-bottom: 1px solid var(--ab-border, #3c3c3c);
}

.ab-toolbar button {
  background: var(--ab-bg-elevated, #3c3c3c);
  color: var(--ab-fg-primary, #cccccc);
  border: 1px solid var(--ab-border, #3c3c3c);
  padding: 4px 12px;
  border-radius: 3px;
  cursor: pointer;
  font-size: 12px;
}

.ab-toolbar button:hover {
  background: var(--ab-selection-bg, #264f78);
}

.kanban-board {
  display: flex;
  flex: 1;
  overflow-x: auto;
  overflow-y: hidden;
  gap: 12px;
  padding: 12px;
}

.kanban-column {
  display: flex;
  flex-direction: column;
  flex: 0 0 280px;
  max-height: 100%;
  background: var(--ab-bg-secondary, #252526);
  border: 1px solid var(--ab-border, #3c3c3c);
  border-radius: 6px;
}

.kanban-column.focused {
  border-color: var(--ab-accent-primary, #ffffff);
}

.column-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 14px;
  font-size: 12px;
  font-weight: 600;
  border-bottom: 1px solid var(--ab-border-subtle, #3c3c3c);
}

.column-header .count {
  background: var(--ab-bg-elevated, #3c3c3c);
  border-radius: 10px;
  padding: 2px 8px;
  font-size: 11px;
}

.column-tickets {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.ticket-card {
  background: var(--ab-bg-elevated, #3c3c3c);
  color: var(--ab-fg-primary, #cccccc);
  border-radius: 6px;
  padding: 10px 12px;
  cursor: pointer;
  border-left: 3px solid var(--ab-accent-blue, #264f78);
  transition: background 0.1s;
}

.ticket-card:hover {
  background: var(--ab-selection-bg, #264f78);
}

.ticket-card.selected {
  background: var(--ab-selection-bg, #264f78);
  color: var(--ab-selection-fg, #ffffff);
  border-left-color: var(--ab-accent-primary, #ffffff);
}

.ticket-card .ticket-title {
  font-size: 13px;
  font-weight: 500;
  margin-bottom: 4px;
}

.ticket-card .ticket-id {
  font-size: 11px;
  color: var(--ab-fg-muted, #6e6e6e);
  margin-bottom: 4px;
}

.ticket-card .ticket-meta {
  display: flex;
  gap: 8px;
  font-size: 11px;
  color: var(--ab-fg-secondary, #858585);
}

.ticket-card .status-badge {
  display: inline-block;
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 10px;
  text-transform: uppercase;
}

.ticket-card .status-badge.backlog { background: var(--ab-col-backlog); }
.ticket-card .status-badge.in_progress { background: var(--ab-col-progress); }
.ticket-card .status-badge.review { background: var(--ab-col-review); }
.ticket-card .status-badge.done { background: var(--ab-col-done); color: #fff; }
`;
}
```

- [ ] **Step 3: Commit**

```bash
git add extensions/vscode/src/util/ extensions/vscode/src/views/shared/
git commit -m "feat(vscode): VSCode theme color system + base CSS tokens"
```

---

## Task 6: Kanban View

**Files:**
- Create: `extensions/vscode/src/views/kanban/kanbanProvider.ts`
- Create: `extensions/vscode/src/views/kanban/state.ts`
- Create: `extensions/vscode/src/views/kanban/renderer.ts`
- Create: `extensions/vscode/src/views/kanban/column.ts`
- Create: `extensions/vscode/src/views/kanban/ticketCard.ts`

- [ ] **Step 1: Write kanban/state.ts**

```typescript
import { ColumnDef, Ticket } from '../../api/types';

export interface KanbanState {
    columns: { name: string; status: string; tickets: Ticket[] }[];
    selectedColumn: number;
    selectedTicket: number;
}
```

- [ ] **Step 2: Write kanban/column.ts**

```typescript
import { Ticket } from '../../api/types';

export function renderColumn(
    name: string,
    status: string,
    tickets: Ticket[],
    isFocused: boolean,
    selectedIndex: number
): string {
    const ticketCards = tickets
        .map((t, i) => renderTicketCard(t, i === selectedIndex))
        .join('');

    return `
    <div class="kanban-column ${isFocused ? 'focused' : ''}" data-status="${status}">
      <div class="column-header">
        <span>${name}</span>
        <span class="count">${tickets.length}</span>
      </div>
      <div class="column-tickets">
        ${ticketCards || '<div class="empty-col">No tickets</div>'}
      </div>
    </div>`;
}

function renderTicketCard(ticket: Ticket, isSelected: boolean): string {
    return `
    <div class="ticket-card ${isSelected ? 'selected' : ''}" data-id="${ticket.id}">
      <div class="ticket-id">${ticket.id}</div>
      <div class="ticket-title">${escapeHtml(ticket.title)}</div>
      <div class="ticket-meta">
        ${ticket.agent ? `<span class="agent-badge">${escapeHtml(ticket.agent)}</span>` : ''}
        <span class="status-badge ${ticket.status}">${ticket.status.replace('_', ' ')}</span>
      </div>
    </div>`;
}

function escapeHtml(s: string): string {
    return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
```

- [ ] **Step 3: Write kanban/ticketCard.ts**

```typescript
// Ticket card renderer — kept separate for maintainability
import { Ticket } from '../../api/types';

export function renderTicketCard(ticket: Ticket, isSelected: boolean): string {
    const statusClass = ticket.status.replace('_', '-');
    const agentBadge = ticket.agent
        ? `<span class="agent-badge">${escapeHtml(ticket.agent)}</span>`
        : '';

    return `
    <div class="ticket-card ${isSelected ? 'selected' : ''}"
         data-id="${ticket.id}"
         data-status="${ticket.status}">
      <div class="ticket-id">${escapeHtml(ticket.id)}</div>
      <div class="ticket-title">${escapeHtml(ticket.title)}</div>
      <div class="ticket-meta">
        ${agentBadge}
        <span class="status-badge ${statusClass}">${ticket.status.replace('_', ' ')}</span>
      </div>
    </div>`;
}

function escapeHtml(s: string): string {
    return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
```

- [ ] **Step 4: Write kanban/renderer.ts**

```typescript
import { KanbanState } from './state';
import { renderColumn } from './column';
import { generateBaseCSS } from '../shared/styles';
import { ThemeColors } from '../../util/vscodeTheme';
import { injectThemeColorsScript } from '../../util/vscodeTheme';

export function renderKanban(state: KanbanState, colors: ThemeColors): string {
    const columns = state.columns.map((col, i) =>
        renderColumn(col.name, col.status, col.tickets, i === state.selectedColumn, state.selectedTicket)
    ).join('\n');

    return `<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <style>${generateBaseCSS(colors)}</style>
</head>
<body>
  <div class="ab-root">
    <div class="ab-toolbar">
      <button id="btn-add">+ Add Ticket</button>
      <button id="btn-refresh">↻ Refresh</button>
      <button id="btn-dashboard">Dashboard</button>
    </div>
    <div class="kanban-board" id="kanban-board">
      ${columns}
    </div>
  </div>
  <script>
    const vscode = acquireVsCodeApi();

    document.getElementById('btn-add')?.addEventListener('click', () => {
        vscode.postMessage({ type: 'command', command: 'agentboard.addTicket' });
    });

    document.getElementById('btn-refresh')?.addEventListener('click', () => {
        vscode.postMessage({ type: 'command', command: 'agentboard.refreshBoard' });
    });

    document.getElementById('btn-dashboard')?.addEventListener('click', () => {
        vscode.postMessage({ type: 'command', command: 'agentboard.toggleDashboard' });
    });

    document.querySelectorAll('.ticket-card').forEach(card => {
        card.addEventListener('click', () => {
            vscode.postMessage({ type: 'selectTicket', ticketId: card.dataset.id });
        });
    });

    // Handle keyboard navigation from extension
    window.addEventListener('message', (event) => {
        const msg = event.data;
        if (msg.type === 'stateUpdate') {
            // Re-render with new state
            const board = document.getElementById('kanban-board');
            if (board && msg.data.columns) {
                // State update — can do incremental update
            }
        }
    });

    // Expose keyboard handler to extension
    document.addEventListener('keydown', (e) => {
        const keyMap = {
            'ArrowLeft': 'left', 'ArrowRight': 'right',
            'ArrowUp': 'up', 'ArrowDown': 'down',
            'Enter': 'select', 'a': 'add', 'd': 'delete',
        };
        const action = keyMap[e.key];
        if (action) {
            e.preventDefault();
            vscode.postMessage({ type: 'navigate', direction: action });
        }
    });
  </script>
</body>
</html>`;
}
```

- [ ] **Step 5: Write kanbanProvider.ts**

```typescript
import * as vscode from 'vscode';
import { ApiClient } from '../../api/client';
import { AppState } from '../../state/appState';
import { StateSync } from '../../state/stateSync';
import { getThemeColors, injectThemeColorsScript } from '../../util/vscodeTheme';
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

        // Sync initial state
        this.appState.onStateChange = (state) => {
            const newState = this.buildState();
            webviewView.webview.postMessage({ type: 'stateUpdate', data: newState });
        };
    }

    private handleMessage(msg: { type: string; [key: string]: unknown }) {
        switch (msg.type) {
            case 'navigate':
                this.appState.moveSelection(msg.direction as 'up' | 'down' | 'left' | 'right');
                this.refreshView();
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
        this.webviewView?.webview.postMessage({
            type: 'render',
            html: renderKanban(state, colors)
        });
    }

    private selectTicket(ticketId: string) {
        // Find ticket in state and update selection
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
```

- [ ] **Step 6: Register the provider in extension.ts**

Add to `activate()`:

```typescript
import { KanbanProvider } from './views/kanban/kanbanProvider';

const apiClient = new ApiClient('http://localhost:8080');
const kanbanProvider = new KanbanProvider(apiClient, appState, stateSync);

context.subscriptions.push(
    vscode.window.registerWebviewViewProvider(KanbanProvider.viewType, kanbanProvider)
);
```

- [ ] **Step 7: Register the view in package.json**

Add to `contributes.views.webview`:

```json
{
    "type": "webview",
    "id": "agentboard.kanban",
    "name": "AgentBoard",
    "icon": "media/icon.png"
}
```

- [ ] **Step 8: Commit**

```bash
git add extensions/vscode/src/views/kanban/
git commit -m "feat(vscode): kanban webview — board, columns, ticket cards"
```

---

## Task 7: Commands (Ticket CRUD + Proposals + Runs)

**Files:**
- Create: `extensions/vscode/src/commands/index.ts`
- Create: `extensions/vscode/src/commands/tickets.ts`
- Create: `extensions/vscode/src/commands/proposals.ts`
- Create: `extensions/vscode/src/commands/runs.ts`

- [ ] **Step 1: Write commands/index.ts**

```typescript
export function registerCommands(context: vscode.ExtensionContext, apiClient: ApiClient, appState: AppState) {
    const tickets = require('./tickets').register(context, apiClient, appState);
    const proposals = require('./proposals').register(context, apiClient, appState);
    const runs = require('./runs').register(context, apiClient, appState);
}
```

- [ ] **Step 2: Write commands/tickets.ts**

```typescript
import * as vscode from 'vscode';

export function register(context: vscode.ExtensionContext, apiClient: ApiClient, appState: AppState) {
    context.subscriptions.push(
        vscode.commands.registerCommand('agentboard.refreshBoard', async () => {
            const tickets = await apiClient.listTickets();
            appState.setTickets(tickets);
            vscode.window.showInformationMessage('Board refreshed');
        }),

        vscode.commands.registerCommand('agentboard.addTicket', async () => {
            const title = await vscode.window.showInputBox({
                prompt: 'Ticket title',
                placeHolder: 'What needs to be done?',
            });
            if (!title) return;

            const ticket = await apiClient.createTicket({ title });
            const tickets = await apiClient.listTickets();
            appState.setTickets(tickets);
            vscode.window.showInformationMessage(`Created ${ticket.id}`);
        }),

        vscode.commands.registerCommand('agentboard.deleteTicket', async () => {
            const selected = appState.getSelectedTicket();
            if (!selected) {
                vscode.window.showWarningMessage('No ticket selected');
                return;
            }
            const confirm = await vscode.window.showWarningMessage(
                `Delete ticket ${selected.id}?`,
                { modal: true },
                'Delete', 'Cancel'
            );
            if (confirm !== 'Delete') return;

            await apiClient.deleteTicket(selected.id);
            const tickets = await apiClient.listTickets();
            appState.setTickets(tickets);
        }),

        vscode.commands.registerCommand('agentboard.moveTicket', async (_, status: string) => {
            const selected = appState.getSelectedTicket();
            if (!selected) return;
            await apiClient.moveStatus(selected.id, status);
            const tickets = await apiClient.listTickets();
            appState.setTickets(tickets);
        }),

        vscode.commands.registerCommand('agentboard.openTicket', async () => {
            const selected = appState.getSelectedTicket();
            if (!selected) return;
            vscode.commands.executeCommand('agentboard.viewTicketDetail', selected.id);
        }),

        vscode.commands.registerCommand('agentboard.viewTicketDetail', async (_, ticketId: string) => {
            const ticket = await apiClient.getTicket(ticketId);
            const panel = vscode.window.createWebviewPanel(
                'agentboard.ticketDetail',
                `Ticket: ${ticket.id}`,
                vscode.ViewColumn.Beside,
                { enableScripts: true }
            );
            panel.webview.html = `<html><body>
                <h1>${ticket.title}</h1>
                <p>${ticket.description || '(no description)'}</p>
                <p><strong>Status:</strong> ${ticket.status}</p>
                <p><strong>Agent:</strong> ${ticket.agent || 'unassigned'}</p>
                <p><strong>Branch:</strong> ${ticket.branch || '-'}</p>
            </body></html>`;
        })
    );
}
```

- [ ] **Step 3: Write commands/proposals.ts**

```typescript
import * as vscode from 'vscode';

export function register(context: vscode.ExtensionContext, apiClient: ApiClient, appState: AppState) {
    context.subscriptions.push(
        vscode.commands.registerCommand('agentboard.createProposal', async () => {
            const selected = appState.getSelectedTicket();
            if (!selected) {
                vscode.window.showWarningMessage('Select a ticket first');
                return;
            }
            vscode.window.showInformationMessage(`Creating proposal for ${selected.id}...`);
            try {
                const proposal = await apiClient.createProposal({ ticketID: selected.id });
                vscode.window.showInformationMessage(`Proposal ${proposal.id} created`);
            } catch (err) {
                vscode.window.showErrorMessage(`Failed: ${err}`);
            }
        }),

        vscode.commands.registerCommand('agentboard.approveProposal', async () => {
            const selected = appState.getSelectedTicket();
            if (!selected) return;
            try {
                const proposal = await apiClient.getProposal(`prop-${selected.id}`);
                await apiClient.approveProposal(proposal.id);
                vscode.window.showInformationMessage(`Proposal ${proposal.id} approved`);
            } catch (err) {
                vscode.window.showErrorMessage(`Failed: ${err}`);
            }
        })
    );
}
```

- [ ] **Step 4: Write commands/runs.ts**

```typescript
import * as vscode from 'vscode';

export function register(context: vscode.ExtensionContext, apiClient: ApiClient, appState: AppState) {
    context.subscriptions.push(
        vscode.commands.registerCommand('agentboard.startRun', async () => {
            const selected = appState.getSelectedTicket();
            if (!selected) return;

            // Get approved proposal
            try {
                const proposal = await apiClient.getProposal(`prop-${selected.id}`);
                vscode.window.showInformationMessage(`Starting run for ${selected.id}...`);
                const session = await apiClient.startApprovedRun(proposal.id);
                const sessions = await apiClient.getActiveSessions();
                appState.setActiveSessions(sessions);
                vscode.window.showInformationMessage(`Run started: ${session.id}`);
            } catch (err) {
                vscode.window.showErrorMessage(`Start failed: ${err}`);
            }
        }),

        vscode.commands.registerCommand('agentboard.startAdHocRun', async () => {
            const agent = await vscode.window.showQuickPick(['claude-code', 'opencode', 'cursor'], {
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
            await apiClient.switchToPane(appState.selectedSessionId);
        })
    );
}
```

- [ ] **Step 5: Register all commands in extension.ts**

Replace the hello world command registration with:

```typescript
import { registerCommands } from './commands/index';

registerCommands(context, apiClient, appState);
```

- [ ] **Step 6: Commit**

```bash
git add extensions/vscode/src/commands/
git commit -m "feat(vscode): commands — ticket CRUD, proposals, runs"
```

---

## Task 8: Command Palette + Keybindings

**Files:**
- Create: `extensions/vscode/src/commands/board.ts`
- Modify: `extensions/vscode/package.json`
- Modify: `extensions/vscode/src/extension.ts`

- [ ] **Step 1: Write commands/board.ts**

```typescript
import * as vscode from 'vscode';

export function register(context: vscode.ExtensionContext, appState: AppState) {
    context.subscriptions.push(
        vscode.commands.registerCommand('agentboard.toggleDashboard', () => {
            appState.toggleDashboard();
        }),

        vscode.commands.registerCommand('agentboard.openCommandPalette', async () => {
            const items = [
                { label: 'Refresh Board', command: 'agentboard.refreshBoard' },
                { label: 'Add Ticket', command: 'agentboard.addTicket' },
                { label: 'Delete Ticket', command: 'agentboard.deleteTicket' },
                { label: 'Create Proposal', command: 'agentboard.createProposal' },
                { label: 'Approve Proposal', command: 'agentboard.approveProposal' },
                { label: 'Start Run', command: 'agentboard.startRun' },
                { label: 'Start Ad-Hoc Run', command: 'agentboard.startAdHocRun' },
                { label: 'Toggle Dashboard', command: 'agentboard.toggleDashboard' },
            ];

            const selected = await vscode.window.showQuickPick(
                items.map(i => ({ label: i.label })),
                { placeHolder: 'AgentBoard commands...' }
            );

            if (selected) {
                const cmd = items.find(i => i.label === selected.label)?.command;
                if (cmd) vscode.commands.executeCommand(cmd);
            }
        }),

        vscode.commands.registerCommand('agentboard.showHelp', () => {
            vscode.window.showInformationMessage(
                'AgentBoard: Kanban Board for AI Coding Agents\n\n' +
                'Navigate: Arrow keys / h,j,k,l\n' +
                'Add: a | Delete: d | Refresh: r\n' +
                'Start agent: s | Approve: p\n' +
                'Dashboard: i | Command Palette: :'
            );
        }),

        vscode.commands.registerCommand('agentboard.columnLeft', () => {
            appState.moveSelection('left');
        }),

        vscode.commands.registerCommand('agentboard.columnRight', () => {
            appState.moveSelection('right');
        }),

        vscode.commands.registerCommand('agentboard.ticketUp', () => {
            appState.moveSelection('up');
        }),

        vscode.commands.registerCommand('agentboard.ticketDown', () => {
            appState.moveSelection('down');
        }),

        vscode.commands.registerCommand('agentboard.goToColumn', async (_, idx: number) => {
            appState.selectColumn(idx - 1);
        })
    );
}
```

- [ ] **Step 2: Register board commands in commands/index.ts**

Add to `registerCommands`:

```typescript
const board = require('./board').register(context, appState);
```

- [ ] **Step 3: Update package.json with keybindings**

Add to `contributes.keybindings`:

```json
[
    { "command": "agentboard.columnLeft", "key": "h", "when": "agentboard.kanban.focused" },
    { "command": "agentboard.columnRight", "key": "l", "when": "agentboard.kanban.focused" },
    { "command": "agentboard.ticketUp", "key": "k", "when": "agentboard.kanban.focused" },
    { "command": "agentboard.ticketDown", "key": "j", "when": "agentboard.kanban.focused" },
    { "command": "agentboard.addTicket", "key": "a", "when": "agentboard.kanban.focused" },
    { "command": "agentboard.deleteTicket", "key": "d", "when": "agentboard.kanban.focused" },
    { "command": "agentboard.refreshBoard", "key": "r", "when": "agentboard.kanban.focused" },
    { "command": "agentboard.startRun", "key": "s", "when": "agentboard.kanban.focused" },
    { "command": "agentboard.approveProposal", "key": "p", "when": "agentboard.kanban.focused" },
    { "command": "agentboard.toggleDashboard", "key": "i", "when": "agentboard.kanban.focused" },
    { "command": "agentboard.openCommandPalette", "key": ":", "when": "agentboard.kanban.focused" },
    { "command": "agentboard.showHelp", "key": "?", "when": "agentboard.kanban.focused" },
    { "command": "agentboard.goToColumn", "key": "1", "when": "agentboard.kanban.focused" },
    { "command": "agentboard.goToColumn", "key": "2", "when": "agentboard.kanban.focused" },
    { "command": "agentboard.goToColumn", "key": "3", "when": "agentboard.kanban.focused" },
    { "command": "agentboard.goToColumn", "key": "4", "when": "agentboard.kanban.focused" }
]
```

- [ ] **Step 4: Add activation events to package.json**

Ensure `activationEvents` includes:

```json
"onCommand:agentboard.addTicket",
"onCommand:agentboard.refreshBoard",
"onCommand:agentboard.createProposal",
"onCommand:agentboard.approveProposal",
"onCommand:agentboard.startRun",
"onCommand:agentboard.openCommandPalette",
"onView:agentboard.kanban"
```

- [ ] **Step 5: Commit**

```bash
git add extensions/vscode/src/commands/board.ts extensions/vscode/package.json
git commit -m "feat(vscode): command palette + keybindings"
```

---

## Task 9: Wire Everything — Final Integration in extension.ts

**Files:**
- Modify: `extensions/vscode/src/extension.ts`

- [ ] **Step 1: Write final extension.ts**

```typescript
import * as vscode from 'vscode';
import { BackendManager } from './process/backendManager';
import { ApiClient } from './api/api/client';
import { AppState } from './state/appState';
import { StateSync } from './state/stateSync';
import { KanbanProvider } from './views/kanban/kanbanProvider';
import { registerCommands } from './commands/index';
import { WsClient } from './api/wsClient';
import * as endpoints from './api/endpoints';

let backendManager: BackendManager;
let apiClient: ApiClient;
let appState: AppState;
let stateSync: StateSync;
let wsClient: WsClient;

export function activate(context: vscode.ExtensionContext) {
    const outputChannel = vscode.window.createOutputChannel('AgentBoard');
    backendManager = new BackendManager(outputChannel);

    // Start backend — get base URL when ready
    backendManager.ensureRunning().then((baseUrl) => {
        apiClient = new ApiClient(baseUrl);

        appState = new AppState();
        stateSync = new StateSync(appState);

        // Register commands
        registerCommands(context, apiClient, appState);

        // Register Kanban view
        const kanbanProvider = new KanbanProvider(apiClient, appState, stateSync);
        context.subscriptions.push(
            vscode.window.registerWebviewViewProvider(KanbanProvider.viewType, kanbanProvider)
        );

        // Load initial tickets
        apiClient.listTickets().then((tickets) => {
            appState.setTickets(tickets);
        }).catch((err) => {
            outputChannel.appendLine(`Failed to load tickets: ${err}`);
        });

        // Connect WebSocket for real-time completions
        wsClient = new WsClient(baseUrl + endpoints.WebSocket.global());
        wsClient.connect();
        wsClient.onCompletion((completion) => {
            // Update board when a run completes
            apiClient.listTickets().then((tickets) => {
                appState.setTickets(tickets);
            });
            vscode.window.showInformationMessage(
                `Run completed: ${completion.ticketID} — ${completion.summary}`
            );
        });

    }).catch((err) => {
        vscode.window.showErrorMessage(`AgentBoard failed to start: ${err.message}`);
        outputChannel.appendLine(`Failed to start: ${err}`);
    });

    // Register the open command for manual activation
    context.subscriptions.push(
        vscode.commands.registerCommand('agentboard.open', () => {
            backendManager.ensureRunning();
        })
    );
}

export function deactivate() {
    wsClient?.disconnect();
    backendManager?.stop();
}
```

- [ ] **Step 2: Verify TypeScript compiles**

Run: `cd extensions/vscode && npx tsc --noEmit`
Expected: No errors (fix any import or type mismatches before proceeding)

- [ ] **Step 3: Full integration test**

1. Run extension in development (F5)
2. Check output: "AgentBoard: downloading backend..." or startup
3. Kanban webview appears in the Activity Bar
4. Try `agentboard.refreshBoard` command → tickets appear
5. Try `agentboard.addTicket` → ticket created
6. Check `curl http://localhost:8080/tickets` shows new ticket

- [ ] **Step 4: Commit**

```bash
git add extensions/vscode/src/extension.ts
git commit -m "feat(vscode): wire all components in extension activation"
```

---

## Verification

After completing all tasks:

1. **Extension activates**: Binary downloaded → server starts → "AgentBoard ready" notification
2. **Kanban board renders**: Webview shows 4 columns with ticket data from API
3. **Create ticket**: `a` key → input box → ticket appears in Backlog column
4. **Move ticket**: Right-click → "Move to In Progress" → API called → UI updates
5. **Create proposal**: Command palette → "Create Proposal" → proposal created via API
6. **Approve + start run**: Proposal approved → run started → dashboard shows active session
7. **Real-time update**: WebSocket receives completion → board auto-refreshes
8. **Deactivate**: Extension deactivates → server process terminated

Each task is independently testable. Commit after each task to keep the history clean.