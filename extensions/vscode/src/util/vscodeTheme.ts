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

export function getThemeColors(): ThemeColors {
    const fgPrimary = getColor('colorForeground', '#cccccc');
    const bgEditor = getColor('colorEditor.background', '#1e1e1e');
    const bgSideBar = getColor('colorSideBar.background', '#252526');
    const bgActivityBar = getColor('colorActivityBar.background', '#333333');
    const borderColor = getColor('colorEditorWidget.border', '#3c3c3c');
    const panelBorder = getColor('colorPanel.border', '#3c3c3c');
    const activeIcon = getColor('colorActiveIcon.foreground', '#ffffff');
    const selectionBg = getColor('colorSelection.background', '#264f78');
    const selectionFg = getColor('colorSelection.foreground', '#ffffff');
    const errorFg = getColor('colorError.foreground', '#f48771');
    const warningFg = getColor('colorWarning.foreground', '#dcdcaa');
    const modifiedFg = getColor('colorModified.foreground', '#6d984a');

    return {
        '--ab-bg-primary': bgActivityBar,
        '--ab-bg-secondary': bgSideBar,
        '--ab-bg-tertiary': bgEditor,
        '--ab-bg-elevated': getColor('colorDropdown.background', '#3c3c3c'),
        '--ab-fg-primary': fgPrimary,
        '--ab-fg-secondary': getColor('colorDescription.foreground', '#858585'),
        '--ab-fg-muted': getColor('colorEditorWidget.foreground', '#6e6e6e'),
        '--ab-border': borderColor,
        '--ab-border-subtle': panelBorder,
        '--ab-accent-primary': activeIcon,
        '--ab-accent-blue': selectionBg,
        '--ab-accent-green': modifiedFg,
        '--ab-accent-yellow': warningFg,
        '--ab-accent-red': errorFg,
        '--ab-col-backlog': '#3c3c3c',
        '--ab-col-progress': '#2d4a6d',
        '--ab-col-review': '#4a3d6d',
        '--ab-col-done': '#2d4a2d',
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