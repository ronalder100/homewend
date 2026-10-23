// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// The desktop app is a window on the page "homewend ui" serves. It has no
// interface and no logic of its own: it starts the engine, shows its address,
// and stops it when the app quits.
const { app, BrowserWindow, Menu, Notification, dialog, ipcMain, nativeImage, nativeTheme, powerSaveBlocker, screen, shell } = require('electron');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawn } = require('node:child_process');
const readline = require('node:readline');
const util = require('node:util');
const { autoUpdater } = require('electron-updater');
const tokens = require('./tokens.json').variables;

// What the app and its engine say goes to a file as well: opened from the
// menu or the Finder, nobody sees the terminal. One file, in Electron's
// folder for the app's logs, set aside as .old once it passes 5 MB.
const logFile = path.join(app.getPath('logs'), 'homewend.log');
fs.mkdirSync(path.dirname(logFile), { recursive: true });
try {
	if (fs.statSync(logFile).size > 5e6) fs.renameSync(logFile, logFile + '.old');
} catch {}
const logStream = fs.createWriteStream(logFile, { flags: 'a' });
for (const name of ['log', 'error']) {
	const say = console[name];
	console[name] = (...args) => {
		say(...args);
		logStream.write(`${new Date().toISOString()} ${util.format(...args)}\n`);
	};
}
console.log(`Homewend ${app.getVersion()} started`);

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
const ICON = path.join(__dirname, 'build', 'icon.png');

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
// Set once the app is quitting: the window closes for good, and the engine
// stopping is not a failure.
let quitting = false;

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

// Opened from the Finder, an app gets the system's PATH alone, without
// Homebrew's, where ffmpeg is for the thumbnails of videos.
const env =
	process.platform === 'darwin'
		? { ...process.env, PATH: [process.env.PATH, '/opt/homebrew/bin', '/usr/local/bin'].filter(Boolean).join(':') }
		: process.env;

// What stops the app is said in a box: opened from the Finder there is no
// terminal, and the app would only vanish.
function fail(err) {
	console.error(err.message);
	dialog.showErrorBox('Homewend stopped', err.message);
	app.quit();
}

function startEngine() {
	return new Promise((resolve, reject) => {
		// windowsHide: the engine is a console program, and started from a
		// window it would get a console window of its own on Windows (Node's
		// child_process.spawn).
		child = spawn(engine, ['ui'], { stdio: ['pipe', 'pipe', 'pipe'], env, windowsHide: true });
		// Its last words, still passed on, for the box if it stops.
		let said = '';
		child.stderr.on('data', (data) => {
			process.stderr.write(data);
			logStream.write(data);
			said = (said + data).slice(-1000);
		});
		let up = false;
		child.once('error', reject);
		child.once('exit', (code) => {
			const err = new Error([`homewend ui exited with ${code}`, said.trim()].filter(Boolean).join('\n\n'));
			if (!up) reject(err);
			// Under a page already open: nothing it shows works any more.
			else if (!quitting) fail(err);
		});
		// The first line is the address; nothing else is read from it.
		readline.createInterface({ input: child.stdout }).once('line', (line) => {
			up = true;
			resolve(line);
		});
	});
}

// A notification nobody holds is collected, and its click with it.
const notifications = new Set();
function notify(title, body, click) {
	const n = new Notification({ title: String(title), body: String(body) });
	notifications.add(n);
	const forget = () => notifications.delete(n);
	n.on('click', () => {
		forget();
		click();
	});
	n.on('close', forget);
	n.show();
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
		icon: ICON,
		// The page draws the titlebar; the system keeps its own buttons.
		titleBarStyle: 'hidden',
		titleBarOverlay: process.platform === 'darwin' ? true : windowButtons && overlay(),
		webPreferences: { preload: path.join(__dirname, 'preload.js'), spellcheck: false }
	});
	win.once('ready-to-show', () => win.show());
	win.on('close', (event) => {
		keepPlace();
		// On the Mac closing the window leaves the app running, as Mac apps
		// do and as the page tells: the download goes on, and the Dock brings
		// the window back.
		if (process.platform === 'darwin' && !quitting) {
			event.preventDefault();
			win.hide();
		}
	});
	// The page offers to close it while the download goes on.
	ipcMain.on('close-window', () => win.close());
	// A pinch enlarges the page in a browser; an app's interface keeps its size.
	win.webContents.setVisualZoomLevelLimits(1, 1);
	// The person's choice in Settings, or the system's when they chose none.
	ipcMain.on('theme', (_event, theme) => {
		if (['system', 'light', 'dark'].includes(theme)) nativeTheme.themeSource = theme;
	});
	// D2: the system's own notification, for what happens while nobody looks.
	ipcMain.on('notify', (_event, title, body) => {
		if (Notification.isSupported() && !win.isFocused()) notify(title, body, () => win.show());
	});
	// Awake while a download runs: asleep, the computer downloads nothing. The
	// screen may still go dark.
	let awake = -1;
	ipcMain.on('keep-awake', (_event, on) => {
		if (on && awake < 0) awake = powerSaveBlocker.start('prevent-app-suspension');
		else if (!on && awake >= 0) {
			powerSaveBlocker.stop(awake);
			awake = -1;
		}
	});
	ipcMain.on('open-folder', (_event, dir) => {
		if (typeof dir === 'string' && path.isAbsolute(dir)) shell.openPath(dir);
	});
	ipcMain.on('show-in-folder', (_event, file) => {
		if (typeof file === 'string' && path.isAbsolute(file)) shell.showItemInFolder(file);
	});
	// A photo dragged out of the grid is its file, as the file manager drags
	// one: dropped in a folder, a mail or an editor, the original goes.
	ipcMain.on('drag-file', (event, file) => {
		if (typeof file !== 'string' || !path.isAbsolute(file)) return;
		// The app's icon under the pointer: decoding the photo itself, which
		// may be a 50 MB original, would hold the window up.
		event.sender.startDrag({ file, icon: nativeImage.createFromPath(ICON).resize({ width: 64 }) });
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
	// A photo or a file dropped on the window would open in its place, and
	// the page would be gone with no way back: the window shows the engine's
	// page and nothing else.
	win.webContents.on('will-navigate', (event) => {
		if (new URL(event.url).origin !== new URL(url).origin) event.preventDefault();
	});
	Menu.setApplicationMenu(menu());
	updates();
	win.loadURL(url);
}

// Updates come from the project's GitHub releases (build.publish). A newer
// one is only announced, with a notification in the page's words; nothing is
// downloaded until the person asks, from the notification or Settings, and
// then it is installed and the app restarts. HOMEWEND_UPDATES points at
// another place that serves the same files, to try an update without
// publishing one.
//
// Not on the Mac: Squirrel.Mac installs only an app signed with an Apple
// identity, and ours is signed ad hoc. There asking opens the release's page,
// to download it as the first time.
const UPDATE_EVERY = 6 * 60 * 60 * 1000;
function updates() {
	// A build between releases (0.1.2-dev.5…) is not offered the last
	// release, which is older; only HOMEWEND_UPDATES, to try one, overrides it.
	if (!app.isPackaged || (app.getVersion().includes('-') && !process.env.HOMEWEND_UPDATES)) return;
	if (process.env.HOMEWEND_UPDATES) autoUpdater.setFeedURL({ provider: 'generic', url: process.env.HOMEWEND_UPDATES });
	autoUpdater.autoDownload = false;
	autoUpdater.autoInstallOnAppQuit = false;
	let announced = '';
	// The newer version found, told to the page now and whenever it asks: it
	// may be found before the page listens.
	let available = '';
	// Its release's tag, on GitHub.
	let tag = '';
	autoUpdater.on('update-available', (info) => {
		available = info.version;
		tag = info.tag || `v${info.version}`;
		win?.webContents.send('update-available', available);
	});
	ipcMain.on('update-listen', (event) => available && event.sender.send('update-available', available));
	autoUpdater.on('update-downloaded', () => autoUpdater.quitAndInstall());
	// No release yet, or no network: the app carries on as it is.
	autoUpdater.on('error', (err) => console.error('update:', err.message));
	// Asked from the notification and the strip alike: fetched once.
	let installing = false;
	const install = () => {
		if (process.platform === 'darwin') {
			shell.openExternal(releasePage(tag));
			// Nothing installs here: the strip offers its button again.
			win?.webContents.send('update-failed');
			return;
		}
		if (installing) return;
		installing = true;
		autoUpdater.downloadUpdate().catch((err) => {
			installing = false;
			console.error('update:', err.message);
			// The page stops saying it is installing, and can be asked again.
			win?.webContents.send('update-failed');
		});
	};
	ipcMain.on('install-update', install);
	ipcMain.on('announce-update', (_event, version, title, body) => {
		if (version === announced || !Notification.isSupported()) return;
		announced = version;
		notify(title, body, install);
	});
	const check = () => autoUpdater.checkForUpdates().catch(() => {});
	// Once the page can say it.
	win.webContents.once('did-finish-load', check);
	setInterval(check, UPDATE_EVERY);
}

// The page of a release: on GitHub, where build.publish points, which
// electron-builder writes into app-update.yml beside the app (the packaged
// package.json loses it); or HOMEWEND_UPDATES, the place trying one.
function releasePage(tag) {
	if (process.env.HOMEWEND_UPDATES) return process.env.HOMEWEND_UPDATES;
	const config = fs.readFileSync(path.join(process.resourcesPath, 'app-update.yml'), 'utf8');
	const field = (name) => config.match(new RegExp(`^${name}: *(\\S+)`, 'm'))?.[1];
	return `https://github.com/${field('owner')}/${field('repo')}/releases/tag/${tag}`;
}

// One app, one window: opening it again brings the open window forward.
if (!app.requestSingleInstanceLock()) app.exit();
app.on('second-instance', () => {
	if (!win) return;
	if (win.isMinimized()) win.restore();
	win.show();
	win.focus();
});

// The Dock icon clicked, the window closed on the Mac comes back.
app.on('activate', () => win?.show());
app.on('before-quit', () => {
	quitting = true;
});

// An AppImage is one file with no installer: on Linux the app puts itself in
// the desktop's menu, as an installed app is, and keeps the entry pointing at
// wherever the file is now. Written only when it would change.
function desktopEntry() {
	const appimage = process.env.APPIMAGE;
	if (process.platform !== 'linux' || !appimage) return;
	const data = process.env.XDG_DATA_HOME || path.join(os.homedir(), '.local', 'share');
	// Named as the window reports itself (its WM class and Wayland app id,
	// the package's name), so the desktop ties the window to the entry.
	const name = 'homewend-desktop';
	const icon = path.join(data, 'icons', 'hicolor', '512x512', 'apps', `${name}.png`);
	const entry = path.join(data, 'applications', `${name}.desktop`);
	// As the desktop-entry spec asks: inside quotes " ` $ \ take a backslash;
	// then, the value being a string, every backslash is doubled and a % is %%.
	const quoted = ('"' + appimage.replace(/["`$\\]/g, '\\$&') + '"').replace(/\\/g, '\\\\').replace(/%/g, '%%');
	const text = [
		'[Desktop Entry]',
		'Type=Application',
		'Name=Homewend',
		'Comment=Google Photos, on your own disk.',
		`Exec=${quoted} --no-sandbox %U`,
		`Icon=${name}`,
		'Terminal=false',
		`StartupWMClass=${name}`,
		'Categories=Graphics;',
		''
	].join('\n');
	try {
		if (!fs.existsSync(icon)) {
			fs.mkdirSync(path.dirname(icon), { recursive: true });
			fs.writeFileSync(icon, nativeImage.createFromPath(ICON).resize({ width: 512 }).toPNG());
		}
		let was = '';
		try {
			was = fs.readFileSync(entry, 'utf8');
		} catch {
			// None yet.
		}
		if (was !== text) {
			fs.mkdirSync(path.dirname(entry), { recursive: true });
			fs.writeFileSync(entry, text);
		}
	} catch (err) {
		// The menu entry is a convenience: the app runs without it.
		console.error('desktop entry:', err.message);
	}
}

app.whenReady().then(() => {
	desktopEntry();
	return open();
}).catch(fail);

app.on('window-all-closed', () => app.quit());

// Closing its input stops the engine even if this process is killed first.
app.on('quit', () => {
	if (child) {
		child.stdin.end();
		child.kill();
	}
});
