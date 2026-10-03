// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// The one thing the page asks of the window: which theme to draw its own
// buttons and background in.
const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('shell', {
	setTheme: (theme) => ipcRenderer.send('theme', theme)
});
