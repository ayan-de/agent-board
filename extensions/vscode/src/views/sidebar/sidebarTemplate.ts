import { sidebarStyles } from './sidebarStyles';

export interface SidebarProject {
  name: string;
  path: string;
  hasDb: boolean;
}

export function renderSidebarHtml(projects: SidebarProject[], selectedPath?: string): string {
  const projectItems = projects.map(p => {
    const dbIcon = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 22h14a2 2 0 0 0 2-2V7.5L14.5 2H6a2 2 0 0 0-2 2v4"/><polyline points="14 2 14 8 20 8"/><path d="M3 15h6"/><path d="M6 12v6"/></svg>`;
    const folderIcon = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 20h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.93a2 2 0 0 1-1.66-.9l-.82-1.2A2 2 0 0 0 7.93 3H6a2 2 0 0 0-2 2v13c0 1.1.9 2 2 2z"/></svg>`;
    const icon = p.hasDb ? dbIcon : folderIcon;
    const selected = p.path === selectedPath ? ' selected' : '';
    return `<div class="project-item${selected}" data-path="${p.path}">
      <span class="project-icon">${icon}</span>
      <span class="project-name">${p.name}</span>
    </div>`;
  }).join('\n');

  const content = projectItems || `<p class="empty">No projects found in ~/.agentboard/projects</p>`;

  return `<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <style>${sidebarStyles.container}</style>
</head>
<body>
  <div class="project-list">
    ${content}
  </div>
  <script>
    const vscode = acquireVsCodeApi();
    document.querySelectorAll('.project-item').forEach(el => {
      el.addEventListener('click', () => {
        document.querySelectorAll('.project-item').forEach(e => e.classList.remove('selected'));
        el.classList.add('selected');
        vscode.postMessage({ type: 'selectProject', path: el.dataset.path });
      });
    });
  </script>
</body>
</html>`;
}

export function renderLoadingHtml(): string {
  return `<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <style>${sidebarStyles.container}</style>
</head>
<body>
  <p class="loading">Loading projects...</p>
</body>
</html>`;
}

export function renderErrorHtml(message: string): string {
  return `<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <style>${sidebarStyles.container}</style>
</head>
<body>
  <p class="error">Error: ${message}</p>
</body>
</html>`;
}