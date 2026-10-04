// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// What the page asks of the window: the theme to draw its own buttons in, the
// system's folder picker, the file manager, and the app's updates.
const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('shell', {
	setTheme: (theme) => ipcRenderer.send('theme', theme),
	chooseFolder: (start) => ipcRenderer.invoke('choose-folder', start),
	showInFolder: (path) => ipcRenderer.send('show-in-folder', path),
	openFolder: (path) => ipcRenderer.send('open-folder', path),
	onToggleSidebar: (fn) => ipcRenderer.on('toggle-sidebar', () => fn()),
	notify: (title, body) => ipcRenderer.send('notify', title, body),
	onUpdate: (fn) => {
		ipcRenderer.on('update-available', (_event, version) => fn(version));
		ipcRenderer.send('update-listen');
	},
	announceUpdate: (version, title, body) => ipcRenderer.send('announce-update', version, title, body),
	installUpdate: () => ipcRenderer.send('install-update')
});
