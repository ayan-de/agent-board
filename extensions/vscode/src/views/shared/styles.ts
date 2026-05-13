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
  color: var(--ab-fg-primary, #cccccc);
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

.empty-col {
  padding: 16px;
  text-align: center;
  color: var(--ab-fg-muted, #6e6e6e);
  font-size: 12px;
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
  flex-wrap: wrap;
}

.ticket-card .agent-badge {
  background: var(--ab-bg-primary, #333333);
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 10px;
}

.ticket-card .status-badge {
  display: inline-block;
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 10px;
  text-transform: uppercase;
}

.ticket-card .status-badge.backlog { background: var(--ab-col-backlog, #3c3c3c); }
.ticket-card .status-badge.in-progress { background: var(--ab-col-progress, #2d4a6d); }
.ticket-card .status-badge.review { background: var(--ab-col-review, #4a3d6d); }
.ticket-card .status-badge.done { background: var(--ab-col-done, #2d4a2d); color: #fff; }
`;
}