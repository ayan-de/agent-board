export const sidebarStyles = {
  container: `
    body {
      background: var(--ab-bg-secondary);
      color: var(--ab-fg-primary);
      padding: 8px;
      font-family: system-ui;
      margin: 0;
    }
    .project-list {
      display: flex;
      flex-direction: column;
      gap: 4px;
    }
    .project-item {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 8px 12px;
      border-radius: 4px;
      cursor: pointer;
      transition: background 0.15s;
    }
    .project-item:hover {
      background: var(--ab-bg-elevated);
    }
    .project-item.selected {
      background: var(--ab-selection-bg);
    }
    .project-icon {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 16px;
      height: 16px;
    }
    .project-icon svg {
      width: 14px;
      height: 14px;
    }
    .project-name {
      font-size: 13px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
    .empty {
      color: var(--ab-fg-muted);
      font-size: 12px;
      text-align: center;
      padding: 16px;
    }
    .loading {
      color: var(--ab-fg-muted);
      font-size: 12px;
      text-align: center;
      padding: 16px;
    }
    .error {
      color: var(--ab-accent-red);
      font-size: 12px;
      text-align: center;
      padding: 16px;
    }
  `,
};