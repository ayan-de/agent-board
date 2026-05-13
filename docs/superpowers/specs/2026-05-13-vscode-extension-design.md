# VSCode Extension — AgentBoard Frontend

## Context

The Go backend (`agentboard`) has been separated into a clean architecture:
`internal/core/` defines interfaces, `internal/api/` exposes REST+WebSocket.
The TUI (`internal/tui/`) consumes `core.Orchestrator` interface, and now a
VSCode extension will do the same — different frontend, same logic.

The goal: user installs the VSCode extension, activates it, and gets a full
AgentBoard Kanban experience without any manual setup.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────┐
│                  VSCode Extension Host                   │
│   (TypeScript — activation, lifecycle, process mgmt)    │
├─────────────────────────────────────────────────────────┤
│  Go Backend (agentboard --api)                          │
│  ├── REST API  (:8080)  ← HTTP calls                   │
│  └── WebSocket  (:8080/ws) ← real-time completions      │
├─────────────────────────────────────────────────────────┤
│  Extension connects to Go backend running as child      │
│  process (downloaded from GitHub releases on first use) │
└─────────────────────────────────────────────────────────┘
```

### Extension Structure

```
extensions/vscode/
├── src/
│   ├── extension.ts          # Entry point, activation, process lifecycle
│   ├── commands/              # VSCode commands (registerCommand)
│   │   ├── tickets.ts         # Create, delete, move ticket commands
│   │   ├── agents.ts          # Start agent, approve proposal, etc.
│   │   └── board.ts           # Refresh, settings commands
│   ├── views/
│   │   ├── kanban/
│   │   │   ├── kanbanProvider.ts    # WebviewProvider for Kanban board
│   │   │   ├── kanban.ts            # Kanban state + render logic (posts messages)
│   │   │   ├── ticketCard.ts       # Ticket card renderer
│   │   │   └── column.ts           # Column renderer
│   │   ├── ticketDetail/
│   │   │   └── ticketDetailProvider.ts
│   │   └── dashboard/
│   │       └── dashboardProvider.ts  # Agent session output display
│   ├── api/
│   │   ├── client.ts          # HTTP client (fetch wrapper)
│   │   ├── types.ts          # TypeScript types matching Go core types
│   │   └── wsClient.ts       # WebSocket client (real-time completions)
│   ├── process/
│   │   └── backendManager.ts  # Download, spawn, manage agentboard process
│   └── state/
│       └── appState.ts         # Shared state (selected ticket, active sessions)
├── package.json
├── tsconfig.json
└── vsc-extension-quickstart.md
```

### Backend Process Lifecycle

1. **First activation**: Extension checks if `agentboard` binary exists in extension storage
2. **Not found**: Downloads from GitHub releases (detects OS + arch), shows "Installing..." notification
3. **Found**: Spawns `agentboard --api --addr :PORT` (PORT = 8080 default, auto-detect if taken)
4. **Health check**: Polls `http://localhost:PORT/health` until OK
5. **On deactivation**: Terminates the child process (unless user preference says "keep running")

### Communication: Webview ↔ Extension Host

VSCode Webviews communicate with the extension host via `postMessage` / `acquireVscodeApi`.

```
Webview (Kanban)                    Extension Host
      │                                   │
      │──── postMessage (action) ────────>│
      │                                   │ calls API client
      │                                   │──── HTTP request ────> Go backend
      │                                   │
      │<--- sendResponse (result) ────────│
      │
      │<--- WebSocket push (completion) ─│ (via webviewOptions, state update)
```

All state lives in the extension host (not webview). Webview is purely render-only.

### VSCode Theme Color System

The extension reads VSCode theme colors at startup and generates CSS custom properties injected into all webviews. This ensures AgentBoard respects user's chosen color theme (Dark+, Light+, One Dark Pro, etc.).

**CSS custom properties set on `document.body` from VSCode theme tokens:**

```typescript
// In src/util/vscodeTheme.ts
interface ThemeColors {
  // Backgrounds
  '--ab-bg-primary': string;    // activityBar.background
  '--ab-bg-secondary': string;   // sideBar.background
  '--ab-bg-tertiary': string;    // editor.background
  '--ab-bg-elevated': string;   // dropdown.background or editorWidget.background

  // Foregrounds / text
  '--ab-fg-primary': string;     // editor.foreground
  '--ab-fg-secondary': string;   // descriptionForeground
  '--ab-fg-muted': string;      // editorWidget.foreground (dimmer)

  // Borders
  '--ab-border': string;         // editorWidget.border or focusBorder
  '--ab-border-subtle': string;  // panel.border (softer)

  // Accent / interactive
  '--ab-accent-primary': string; // activityBar.activeIcon.foreground or focusBorder
  '--ab-accent-blue': string;    // list.activeSelectionBackground
  '--ab-accent-green': string;  // gitDecoration.modifiedResourceForeground (status ok)
  '--ab-accent-yellow': string; // list.warningForeground (pending)
  '--ab-accent-red': string;    // list.errorForeground (failed)

  // Kanban-specific
  '--ab-col-backlog': string;    // (subtle tint, computed from accent)
  '--ab-col-progress': string;   // (subtle tint, computed from accent)
  '--ab-col-review': string;     // (subtle tint, computed from accent)
  '--ab-col-done': string;       // (subtle tint, computed from accent)

  // Selection
  '--ab-selection-bg': string;  // list.activeSelectionBackground
  '--ab-selection-fg': string;  // list.activeSelectionForeground
}
```

**Usage in CSS:**

```css
.kanban-column {
  background: var(--ab-bg-secondary);
  border: 1px solid var(--ab-border);
}

.ticket-card {
  background: var(--ab-bg-elevated);
  color: var(--ab-fg-primary);
  border-left: 3px solid var(--ab-accent-blue);
}

.ticket-card.selected {
  background: var(--ab-selection-bg);
  color: var(--ab-selection-fg);
  border-left-color: var(--ab-accent-primary);
}

.status-badge.done {
  background: var(--ab-accent-green);
  color: var(--ab-bg-primary);
}
```

**How it works:**
1. On extension activation, `vscodeTheme.ts` reads `vscode.workspace.getConfiguration('workbench.colorTheme')` and maps theme tokens via `vscode.ColorThemeKind` detection
2. Reads actual color values via `getColor('activityBar.activeIcon.foreground')` etc. from the active theme
3. Generates `ThemeColors` object and passes it to each webview on creation via `asWebviewUri` + webview options
4. Webview sets CSS custom properties on `document.body` before rendering anything

This approach means the Kanban board never hardcodes a color — it always uses the current theme's values, just like a native VSCode view.

## UI Sections (mapped from TUI)

| TUI Component | VSCode Equivalent | Notes |
|---|---|---|
| Kanban board | Webview panel (HTML/CSS grid) | 4 columns, ticket cards |
| Ticket detail | Side panel (Webview) | Title, description, status, agent, proposal |
| Agent dashboard | Webview panel | Shows agent pane output (tmux pane content) |
| Command palette | `QuickPick` | `: ` keybinding |
| Notifications | `vscode.window.showNotification` | |
| Help overlay | `MarkdownString` in hover | |
| Keybindings | `package.json` keybindings | Maps TUI keys to VSCode commands |

### Scalable Extension Structure

```
extensions/vscode/
├── src/
│   ├── extension.ts                # Entry point, activation, process lifecycle
│   │
│   ├── commands/                   # All VSCode commands (scalable)
│   │   ├── index.ts               # Re-exports all commands
│   │   ├── tickets.ts             # Create, delete, move, update ticket
│   │   ├── proposals.ts           # Create proposal, approve, view
│   │   ├── runs.ts                # Start run, finish run, ad-hoc
│   │   ├── sessions.ts            # Switch pane, send input, view logs
│   │   └── board.ts               # Refresh, settings, open kanban, toggle dashboard
│   │
│   ├── views/                      # All Webview providers (scalable)
│   │   ├── kanban/
│   │   │   ├── KanbanProvider.ts  # WebviewProvider — owns webview lifecycle
│   │   │   ├── state.ts           # Kanban state (column focus, ticket selection)
│   │   │   ├── renderer.ts        # Renders HTML/CSS, receives postMessage
│   │   │   ├── column.ts          # Column rendering helper
│   │   │   └── ticketCard.ts      # Ticket card rendering helper
│   │   ├── ticketDetail/
│   │   │   ├── TicketDetailProvider.ts
│   │   │   └── renderer.ts        # Ticket detail HTML renderer
│   │   ├── dashboard/
│   │   │   ├── DashboardProvider.ts
│   │   │   └── renderer.ts        # Dashboard HTML renderer
│   │   └── shared/
│   │       ├── styles.ts          # CSS generation (tokens, base styles)
│   │       └── icons.ts           # SVG icon helpers
│   │
│   ├── api/                       # Backend communication (stable interface)
│   │   ├── client.ts             # Fetch wrapper, base URL, error handling
│   │   ├── types.ts              # TypeScript types matching Go core + store types
│   │   ├── endpoints.ts          # Endpoint path constants (single source of truth)
│   │   └── wsClient.ts          # WebSocket client (reconnect, heartbeat)
│   │
│   ├── process/                   # Backend process lifecycle
│   │   ├── backendManager.ts     # Download, spawn, port-find, health-check, kill
│   │   └── backendProcess.ts     # ChildProcess wrapper with stdout/stderr capture
│   │
│   ├── state/                     # Shared extension state
│   │   ├── appState.ts          # Global state: selectedTicket, activeSessions, columns
│   │   └── stateSync.ts         # Syncs state changes → all webviews
│   │
│   └── util/                      # Shared utilities
│       ├── logger.ts             # Extension logger (channels to OutputChannel)
│       └── vscodeTheme.ts        # Reads VSCode theme colors → CSS token map
│
├── package.json                   # Commands, keybindings, views, extension dependencies
├── tsconfig.json
└── vsc-extension-quickstart.md
```

**Why this scales:**
- `commands/` and `views/` are flat — add a new file, register it in `index.ts`. No deep nesting.
- `views/shared/styles.ts` centralizes CSS generation so any new view uses the same tokens.
- `api/endpoints.ts` is the single source of truth for URL paths — changing the API means updating one file.
- `state/appState.ts` is the single source of truth for shared state — webviews never hold their own state.

## API Endpoints Used by Extension

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/tickets` | List all tickets |
| POST | `/tickets` | Create ticket |
| GET | `/tickets/{id}` | Get ticket detail |
| PUT | `/tickets/{id}` | Update ticket |
| DELETE | `/tickets/{id}` | Delete ticket |
| POST | `/tickets/{id}/status` | Move ticket to column |
| POST | `/proposals` | Create proposal (decompose ticket) |
| GET | `/proposals/{id}` | View proposal |
| POST | `/proposals/{id}/approve` | Approve proposal |
| POST | `/runs/{id}/start` | Start approved run |
| POST | `/runs/adhoc` | Start ad-hoc agent |
| POST | `/runs/{id}/finish` | Finish run (called by agent or manually) |
| GET | `/sessions` | List active sessions |
| GET | `/sessions/{id}/pane` | Get pane output |
| POST | `/sessions/{id}/switch` | Switch to pane |
| GET | `/ws?session=global` | WebSocket (run completion events) |

## Type Mapping (Go → TypeScript)

```typescript
// Matches core/types.go:AgentSession
interface AgentSession {
  sessionID: string;
  ticketID: string;
  agent: string;
  startedAt: number;
  status: string;  // running | completed | failed | cancelled
  paneID: string;
  windowID: string;
}

// Matches core/types.go:CreateProposalInput
interface CreateProposalInput {
  ticketID: string;
}

// Matches store.Ticket
interface Ticket {
  id: string;
  title: string;
  description: string;
  status: string;  // backlog | in_progress | review | done
  agent: string | null;
  branch: string;
  createdAt: string;
  updatedAt: string;
  dependsOn: string;
}

// Matches store.Proposal
interface Proposal {
  id: string;
  ticketID: string;
  agent: string;
  prompt: string;
  status: string;  // pending | approved | running | completed
  createdAt: string;
}
```

## Keybindings (VSCode → TUI keys)

| TUI Key | VSCode Command | Action |
|---------|---------------|--------|
| `h/l` or `←/→` | `agentboard.columnLeft/Right` | Move kanban column focus |
| `j/k` or `↑/↓` | `agentboard.ticketUp/Down` | Move ticket selection |
| `Enter` | `agentboard.openTicket` | Open ticket detail |
| `a` | `agentboard.addTicket` | Add new ticket |
| `d` | `agentboard.deleteTicket` | Delete ticket (confirm dialog) |
| `s` | `agentboard.startAgent` | Start agent on selected ticket |
| `p` | `agentboard.approveProposal` | Approve pending proposal |
| `r` | `agentboard.refreshBoard` | Refresh board state |
| `i` | `agentboard.toggleDashboard` | Toggle agent dashboard |
| `:` | `agentboard.openCommandPalette` | Open command palette |
| `?` | `agentboard.showHelp` | Show help |
| `1-4` | `agentboard.goToColumn` | Jump to column |
| `Esc` | (built-in) | Close view / cancel |

## Implementation Order

1. **Extension scaffold** — `yo code` skeleton, package.json, tsconfig
2. **Backend manager** — download binary, spawn process, health check, shutdown
3. **API client** — TypeScript types + fetch-based HTTP client + WebSocket client
4. **Kanban view** — webview with HTML/CSS rendering, postMessage to extension host
5. **Ticket CRUD commands** — create, update, delete, move status via API
6. **Proposal/Run flow** — create proposal, approve, start run, poll completion
7. **Agent dashboard** — webview showing pane content via `/sessions/{id}/pane`
8. **Command palette** — QuickPick mapped to all commands
9. **Keybindings** — register all shortcuts in package.json

## Verification

- Activate extension → binary downloaded → server starts → "AgentBoard ready" notification
- Kanban board renders with tickets from API
- Create ticket → appears in board
- Move ticket to In Progress → status updates via API
- Create proposal → proposal appears → approve → run starts
- Agent dashboard shows live pane output
- Deactivate extension → server process terminates