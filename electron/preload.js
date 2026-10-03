// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// What the page asks of the window: the theme to draw its own buttons in, the
// system's folder picker, and the file manager.
const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('shell', {
	setTheme: (theme) => ipcRenderer.send('theme', theme),
	chooseFolder: (start) => ipcRenderer.invoke('choose-folder', start),
	showInFolder: (path) => ipcRenderer.send('show-in-folder', path)
});
