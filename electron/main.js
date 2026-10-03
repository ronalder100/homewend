// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// The desktop app is a window on the page "homewend ui" serves. It has no
// interface and no logic of its own: it starts the engine, shows its address,
// and stops it when the window goes.
const { app, BrowserWindow, Menu, MenuItem, dialog, ipcMain, nativeTheme, screen, shell } = require('electron');
const path = require('node:path');
const { spawn } = require('node:child_process');
const readline = require('node:readline');
const tokens = require('./tokens.json').variables;

// Where the engine is: inside the app once packaged (electron-builder's
// extraResources), else set by whoever runs it from source, else the PATH.
const engine = app.isPackaged
	? path.join(process.resourcesPath, process.platform === 'win32' ? 'homewend.exe' : 'homewend')
	: process.env.HOMEWEND_BIN || 'homewend';

// The design's window, shrunk to fit a smaller screen; never below the
// smallest window the design draws.
const DESIGN = { width: 1280, height: 800 };
const MIN = { width: 960, height: 640 };
const TITLEBAR_HEIGHT = 44;

// Scrollbars float over the content and show only while it scrolls, on every
// system, as macOS does by default.
app.commandLine.appendSwitch('enable-features', 'OverlayScrollbar');

// Tiling window managers (sway, i3, Hyprland…) place and close windows from
// the keyboard and do nothing with minimise or maximise buttons. They set no
// XDG_CURRENT_DESKTOP, or their own name; desktops with window buttons (GNOME,
// KDE, Xfce…) set theirs.
const TILING = /^$|sway|i3|hyprland|river|niri|bspwm|qtile/i;
const windowButtons =
	process.platform !== 'linux' || !TILING.test(process.env.XDG_CURRENT_DESKTOP || '');

let child;

function theme(name) {
	const mode = nativeTheme.shouldUseDarkColors ? 'dark' : 'light';
	return tokens[name].value.find((v) => v.theme.mode === mode).value;
}

function overlay() {
	// One pixel short, so the titlebar's bottom line runs under the buttons.
	return { color: theme('bg'), symbolColor: theme('fg'), height: TITLEBAR_HEIGHT - 1 };
}

function startEngine() {
	return new Promise((resolve, reject) => {
		child = spawn(engine, ['ui'], { stdio: ['pipe', 'pipe', 'inherit'] });
		child.once('error', reject);
		child.once('exit', (code) => reject(new Error(`homewend ui exited with ${code}`)));
		// The first line is the address; nothing else is read from it.
		readline.createInterface({ input: child.stdout }).once('line', resolve);
	});
}

async function open() {
	const url = await startEngine();
	const area = screen.getPrimaryDisplay().workAreaSize;
	const win = new BrowserWindow({
		width: Math.max(MIN.width, Math.min(DESIGN.width, area.width)),
		height: Math.max(MIN.height, Math.min(DESIGN.height, area.height)),
		minWidth: MIN.width,
		minHeight: MIN.height,
		backgroundColor: theme('bg'),
		// The page draws the titlebar; the system keeps its own buttons.
		titleBarStyle: 'hidden',
		titleBarOverlay: process.platform === 'darwin' ? true : windowButtons && overlay(),
		webPreferences: { preload: path.join(__dirname, 'preload.js') }
	});
	// The person's choice in Settings, or the system's when they chose none.
	ipcMain.on('theme', (_event, theme) => {
		if (['system', 'light', 'dark'].includes(theme)) nativeTheme.themeSource = theme;
	});
	ipcMain.on('show-in-folder', (_event, file) => {
		if (typeof file === 'string' && path.isAbsolute(file)) shell.showItemInFolder(file);
	});
	ipcMain.handle('choose-folder', async (_event, start) => {
		const r = await dialog.showOpenDialog(win, {
			defaultPath: start || undefined,
			properties: ['openDirectory', 'createDirectory']
		});
		return r.canceled ? '' : r.filePaths[0];
	});
	nativeTheme.on('updated', () => {
		win.setBackgroundColor(theme('bg'));
		if (process.platform !== 'darwin' && windowButtons) win.setTitleBarOverlay(overlay());
	});
	// Links leave the window for the system browser: Google's pages are never
	// shown inside the app.
	win.webContents.setWindowOpenHandler(({ url: target }) => {
		shell.openExternal(target);
		return { action: 'deny' };
	});
	// On the Mac the sidebar's place in the titlebar belongs to the traffic
	// lights: it opens from the View menu, ⌃⌘S, as in Apple's own apps.
	if (process.platform === 'darwin') {
		const menu = Menu.getApplicationMenu();
		const view = menu?.items.find((m) => m.role === 'viewmenu' || m.label === 'View');
		view?.submenu?.insert(
			0,
			new MenuItem({
				label: 'Show Sidebar',
				accelerator: 'Ctrl+Cmd+S',
				click: () => win.webContents.send('toggle-sidebar')
			})
		);
		Menu.setApplicationMenu(menu);
	}
	win.loadURL(url);
}

app.whenReady().then(open).catch((err) => {
	console.error(err.message);
	app.quit();
});

app.on('window-all-closed', () => app.quit());

// Closing its input stops the engine even if this process is killed first.
app.on('quit', () => {
	if (child) {
		child.stdin.end();
		child.kill();
	}
});
