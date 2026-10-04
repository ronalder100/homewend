// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// The desktop app is a window on the page "homewend ui" serves. It has no
// interface and no logic of its own: it starts the engine, shows its address,
// and stops it when the window goes.
const { app, BrowserWindow, Menu, Notification, dialog, ipcMain, nativeTheme, screen, shell } = require('electron');
const fs = require('node:fs');
const path = require('node:path');
const { spawn } = require('node:child_process');
const readline = require('node:readline');
const { autoUpdater } = require('electron-updater');
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
let win;

// Where the window was, as a native app reopens where it was left.
const placeFile = () => path.join(app.getPath('userData'), 'window.json');
function lastPlace() {
	try {
		return JSON.parse(fs.readFileSync(placeFile(), 'utf8'));
	} catch {
		return null;
	}
}
const inside = (p, a) => p.x >= a.x && p.y >= a.y && p.x < a.x + a.width && p.y < a.y + a.height;
function keepPlace() {
	if (!win.isMinimized()) fs.writeFileSync(placeFile(), JSON.stringify(win.getNormalBounds()));
}

// The menu of a native app, without a browser's: no reload, no zoom, no
// developer tools once packaged. On the Mac the sidebar opens from the View
// menu, ⌃⌘S, as in Apple's own apps, since its place in the titlebar belongs
// to the traffic lights. Elsewhere the menu is never drawn: it only carries
// the shortcuts to close and quit.
function menu() {
	const dev = app.isPackaged ? [] : [{ type: 'separator' }, { role: 'reload' }, { role: 'toggleDevTools' }];
	if (process.platform !== 'darwin') return Menu.buildFromTemplate([{ role: 'fileMenu', submenu: [{ role: 'close' }, { role: 'quit' }, ...dev] }]);
	return Menu.buildFromTemplate([
		{ role: 'appMenu' },
		{ role: 'fileMenu' },
		{ role: 'editMenu' },
		{
			label: 'View',
			submenu: [
				{ label: 'Show Sidebar', accelerator: 'Ctrl+Cmd+S', click: () => win?.webContents.send('toggle-sidebar') },
				{ type: 'separator' },
				{ role: 'togglefullscreen' },
				...dev
			]
		},
		{ role: 'windowMenu' }
	]);
}

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
	let last = lastPlace();
	// A screen unplugged since: the size is kept, the system places the window.
	if (last && !screen.getAllDisplays().some((d) => inside(last, d.workArea))) last = { width: last.width, height: last.height };
	win = new BrowserWindow({
		width: Math.max(MIN.width, Math.min(DESIGN.width, area.width)),
		height: Math.max(MIN.height, Math.min(DESIGN.height, area.height)),
		...last,
		// Shown once the page is drawn: never an empty window first.
		show: false,
		autoHideMenuBar: true,
		// A tiling window manager gives the window the size of its tile and
		// cuts off what does not fit a minimum: there the page fits itself.
		minWidth: windowButtons ? MIN.width : undefined,
		minHeight: windowButtons ? MIN.height : undefined,
		backgroundColor: theme('bg'),
		// The site's icon; the Mac and Windows take it from the app bundle.
		icon: path.join(__dirname, 'build', 'icon.png'),
		// The page draws the titlebar; the system keeps its own buttons.
		titleBarStyle: 'hidden',
		titleBarOverlay: process.platform === 'darwin' ? true : windowButtons && overlay(),
		webPreferences: { preload: path.join(__dirname, 'preload.js'), spellcheck: false }
	});
	win.once('ready-to-show', () => win.show());
	win.on('close', keepPlace);
	// A pinch enlarges the page in a browser; an app's interface keeps its size.
	win.webContents.setVisualZoomLevelLimits(1, 1);
	// The person's choice in Settings, or the system's when they chose none.
	ipcMain.on('theme', (_event, theme) => {
		if (['system', 'light', 'dark'].includes(theme)) nativeTheme.themeSource = theme;
	});
	// D2: the system's own notification, for what happens while nobody looks.
	ipcMain.on('notify', (_event, title, body) => {
		if (Notification.isSupported() && !win.isFocused()) {
			const n = new Notification({ title: String(title), body: String(body) });
			n.on('click', () => win.show());
			n.show();
		}
	});
	ipcMain.on('open-folder', (_event, dir) => {
		if (typeof dir === 'string' && path.isAbsolute(dir)) shell.openPath(dir);
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
	Menu.setApplicationMenu(menu());
	updates();
	win.loadURL(url);
}

// Updates come from the project's GitHub releases (build.publish), are
// downloaded in the background and installed on the next start, or at once
// from Settings. HOMEWEND_UPDATES points at another place that serves the
// same files, to try an update without publishing one.
const UPDATE_EVERY = 6 * 60 * 60 * 1000;
function updates() {
	if (!app.isPackaged) return;
	if (process.env.HOMEWEND_UPDATES) autoUpdater.setFeedURL({ provider: 'generic', url: process.env.HOMEWEND_UPDATES });
	autoUpdater.on('update-downloaded', (info) => win?.webContents.send('update-ready', info.version));
	// No release yet, or no network: the app carries on as it is.
	autoUpdater.on('error', (err) => console.error('update:', err.message));
	ipcMain.on('install-update', () => autoUpdater.quitAndInstall());
	const check = () => autoUpdater.checkForUpdates().catch(() => {});
	check();
	setInterval(check, UPDATE_EVERY);
}

// One app, one window: opening it again brings the open window forward.
if (!app.requestSingleInstanceLock()) app.exit();
app.on('second-instance', () => {
	if (!win) return;
	if (win.isMinimized()) win.restore();
	win.focus();
});

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
