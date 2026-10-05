// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getSettings, putSettings, type Theme } from './api.js';
import { app } from './app.svelte.js';

// What the desktop shell offers the page (electron/preload.js); absent in a
// browser.
declare global {
	interface Window {
		shell?: {
			setTheme(theme: Theme): void;
			chooseFolder(start?: string): Promise<string>;
			showInFolder(path: string): void;
			dragFile(path: string): void;
			openFolder(path: string): void;
			onToggleSidebar(fn: () => void): void;
			notify(title: string, body: string): void;
			onUpdate(fn: (version: string) => void): void;
			announceUpdate(version: string, title: string, body: string): void;
			installUpdate(): void;
		};
	}
}

export const theme = $state<{ value: Theme }>({ value: 'system' });

// apply shows the theme: data-theme for the page's colours, and the shell's
// own setting so the window's buttons and background follow.
function apply(t: Theme) {
	theme.value = t;
	if (t === 'system') delete document.documentElement.dataset.theme;
	else document.documentElement.dataset.theme = t;
	window.shell?.setTheme(t);
}

export async function loadTheme() {
	const s = await getSettings();
	apply(s.theme ?? 'system');
}

export async function setTheme(t: Theme) {
	apply(t);
	app.settings = await putSettings({ ...app.settings, theme: t });
}
