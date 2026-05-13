import { KanbanState } from './state';
import { renderColumn } from './column';
import { generateBaseCSS } from '../shared/styles';
import { ThemeColors } from '../../util/vscodeTheme';

export function renderKanban(state: KanbanState, colors: ThemeColors): string {
    const columns = state.columns.map((col, i) =>
        renderColumn(col.name, col.status, col.tickets, i === state.selectedColumn, state.selectedTicket)
    ).join('\n');

    const colorEntries = Object.entries(colors)
        .map(([k, v]) => `document.body.style.setProperty('${k}', '${v}');`)
        .join('\n    ');

    return `<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <style>${generateBaseCSS(colors)}</style>
</head>
<body>
  <script>
  (function() {
      ${colorEntries}
  })();
  </script>
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

    document.addEventListener('keydown', (e) => {
        const keyMap = {
            'ArrowLeft': 'left', 'ArrowRight': 'right',
            'ArrowUp': 'up', 'ArrowDown': 'down',
        };
        const action = keyMap[e.key];
        if (action) {
            e.preventDefault();
            vscode.postMessage({ type: 'navigate', direction: action });
        }
    });

    window.addEventListener('message', (event) => {
        const msg = event.data;
        if (msg.type === 'render' && msg.html) {
            document.body.innerHTML = '';
            document.write(msg.html);
            document.close();
        }
    });
  </script>
</body>
</html>`;
}