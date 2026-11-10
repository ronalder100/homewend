// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// The engine, as the page reaches it: one function per call of internal/web.

export type Theme = 'system' | 'light' | 'dark';

export interface Settings {
	theme?: Theme;
	hidden?: string[];
	/** The profile of each account, by id, once it has been asked. */
	profiles?: Record<string, string>;
}

export interface AccountInfo {
	/** "google/<email>" */
	id: string;
	service: string;
	email: string;
	/** Whose photos they are: the folder of the library they go in. */
	profile: string;
}

export interface Overview {
	accounts: (AccountInfo & { shown: boolean; photos: number })[];
	total: number;
	years: { year: string; count: number }[];
	albums: { account: string; name: string; count: number }[];
	takeouts: number;
}

/** A call the engine refused; status 401 means Google signed the account out. */
export class ApiError extends Error {
	constructor(
		readonly status: number,
		message: string
	) {
		super(message);
	}
}

async function call<T>(method: string, path: string, body?: unknown): Promise<T> {
	const res = await fetch(path, {
		method,
		headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
		body: body === undefined ? undefined : JSON.stringify(body)
	});
	if (!res.ok) throw new ApiError(res.status, `${method} ${path}: ${res.status} ${await res.text()}`);
	return res.json();
}

export const getSettings = () => call<Settings>('GET', '/api/settings');
export const putSettings = (s: Settings) => call<Settings>('PUT', '/api/settings', s);
export const putProfile = (account: string, profile: string) =>
	call<{ profile: string }>('PUT', `/api/profile?account=${encodeURIComponent(account)}`, { profile });
export const getOverview = () => call<Overview>('GET', '/api/overview');
/** Opens the system browser on Google's sign-in and waits for it. */
export const login = (account?: string) =>
	call<AccountInfo>('POST', '/api/login' + (account ? `?account=${encodeURIComponent(account)}` : ''));

export interface GridPhoto {
	hash: string;
	name: string;
	taken?: string;
	video?: boolean;
}

export function getPhotos(q: { year?: string; album?: string; noDate?: boolean; offset: number; limit: number }) {
	const p = new URLSearchParams({ offset: String(q.offset), limit: String(q.limit) });
	if (q.year) p.set('year', q.year);
	if (q.album) p.set('album', q.album);
	if (q.noDate) p.set('nodate', '1');
	return call<GridPhoto[]>('GET', `/api/photos?${p}`);
}

export interface JobState {
	running: boolean;
	year: number;
	stage?: string;
	export?: string;
	parts: number;
	of: number;
	done: number;
	total: number;
	years: { year: string; arrived: number; of: number }[];
	retry?: string;
	/** When the stage began, as the engine saw it. */
	stageSince?: string;
	finished?: boolean;
	lastLooked?: string;
	unassigned: number;
	error?: string;
	problem?: 'signed-out' | 'no-space' | 'expired' | 'not-offered' | 'password-closed';
	/** With not-offered: the years Google has photos of. */
	offered?: number[];
	need?: number;
	free?: number;
}

const q = (o: Record<string, string | number | undefined>) =>
	new URLSearchParams(
		Object.entries(o).filter(([, v]) => v !== undefined && v !== '') as [string, string][]
	).toString();

/** How many photos and videos an account has on Google, by year, newest first. */
export interface YearCount {
	year: number;
	items: number;
	videos: number;
}
export interface YearCounts {
	counted?: string;
	years?: YearCount[];
	/** Old: counted again in the background, this one shown meanwhile. */
	stale?: boolean;
	counting: boolean;
	error?: string;
}
export const getYears = (account: string) => call<YearCounts>('GET', `/api/years?${q({ account })}`);
export const countYears = (account: string) => call<Record<string, never>>('POST', `/api/years?${q({ account })}`);
export const getLibrary = () => call<{ dir: string; free: number }>('GET', '/api/library');
export const putLibrary = (dir: string) => call<{ dir: string }>('PUT', '/api/library', { dir });
export const startGet = (o: { account: string; year?: number; export?: string; fresh?: boolean; again?: boolean }) =>
	call<JobState>(
		'POST',
		`/api/get?${q({ account: o.account, year: o.year, export: o.export, new: o.fresh ? '1' : undefined, again: o.again ? '1' : undefined })}`
	);
export const getJob = (account: string) => call<JobState>('GET', `/api/job?${q({ account })}`);
export const stopJob = (account: string) => call<JobState>('POST', `/api/job/stop?${q({ account })}`);

export interface Takeout {
	id: string;
	status: 'preparing' | 'ready' | 'expired';
	year: number;
	known: boolean;
	Created: string;
	Bytes: number;
	Parts: unknown[] | null;
	Expires: string;
	years?: string[];
	/** What of it the library holds; absent when it never came here. */
	local?: { profile: string; parts: number; of: number; declared?: number; present?: number };
}

export const getTakeouts = (account: string) => call<Takeout[]>('GET', `/api/takeouts?${q({ account })}`);

export const logout = (account: string) => call<object>('POST', `/api/logout?${q({ account })}`);
export const getVersion = () => call<{ version: string }>('GET', '/api/about');

export const pathOf = (hash: string) => call<{ path: string }>('GET', `/api/path/${hash}`);

export const folderOf = (o: { account?: string; profile?: string; year?: string; album?: string; noDate?: boolean }) =>
	call<{ path: string }>(
		'GET',
		`/api/folder?${q({ account: o.account, profile: o.profile, year: o.year, album: o.album, nodate: o.noDate ? '1' : undefined })}`
	);

export const getLatest = () => call<GridPhoto[]>('GET', '/api/latest');
export const getAbout = (hash: string) =>
	call<{ albums: string[]; accounts: string[]; local: string[] }>('GET', `/api/about/${hash}`);

export const readContents = (account: string, exportId: string) =>
	call<{ years: string[] }>('POST', `/api/takeouts/contents?${q({ account, export: exportId })}`);

/** A disk with a camera's photos on it: a card, a camera, a phone. */
export interface Card {
	path: string;
	name: string;
}

/** Photos whose camera had the date wrong, and the date they were taken:
 *  the first of them by the camera's clock takes it, the others keep their
 *  distance from it. */
export interface Redate {
	/** Paths below the folder. */
	names: string[];
	/** A clock, written as UTC: "2025-09-14T10:21:00Z" is 10:21 where it was taken. */
	when: string;
}

/** A photo or video of the folder, as its camera dated it. */
export interface ImportFile {
	path: string;
	/** The camera's clock, as UTC; none, no date. */
	taken?: string;
	video?: boolean;
	/** Its group's folder. */
	group: string;
	/** An image of a few hundred pixels at most: an icon, a logo. */
	small?: boolean;
}

/** A group of the folder's photos: its own folders, a project, an app's previews. */
export interface ImportGroup {
	folder: string;
	kind: 'photos' | 'unsure' | 'project' | 'previews';
	/** What made it so: "package.json", "Lightroom". */
	mark?: string;
	/** Comes in whole. */
	in: boolean;
	photos: number;
	videos: number;
	in_library: number;
	first?: string;
	last?: string;
}

/** Which groups come in: left out, brought in against their kind, photos picked one by one. */
export interface Choice {
	out: string[];
	in: string[];
	pick: string[];
}

/** What a look and its copy both count, and how it ends (the engine's library.Counts). */
export interface Counts {
	in_library: number;
	twins: number;
	/** Moved to a date given; given one but filed in another profile. */
	redated: number;
	not_redated: number;
	/** Deleted from the library: left out, and brought back; those left out by path, and how many more. */
	deleted: number;
	brought_back: number;
	deleted_names?: string[];
	deleted_more?: number;
	unreadable: number;
	outcome: Outcome;
}

export interface ImportPlan extends Counts {
	photos: number;
	videos: number;
	bytes: number;
	files: ImportFile[];
	groups: ImportGroup[];
	/** Newest first; "" is the photos with no date. */
	years: { year: string; photos: number; videos: number; months?: { month: number; photos: number; videos: number }[] }[];
	/** What stays out, by its name at the folder's top: "FWUP0013.DAT", "FFDB/". */
	others: { name: string; files: number; bytes: number }[];
	/** Those left unnamed past the engine's limit; the files of all of them. */
	others_more?: number;
	other_files: number;
}

export interface Imported extends Counts {
	placed: number;
	bytes: number;
	undated: number;
}

/** In the library and checked, by year and month; year "" month 0, the undated. */
export type Filed = Record<string, Record<number, number>>;

/** How a look or a copy ends, decided by the engine. */
export type Outcome = 'new' | 'redated' | 'left-out' | 'deleted' | 'known-noted' | 'known' | 'unread' | 'empty';

/** What the page asks of a look or a copy; kept in the state for a copy taken up again. */
export interface ImportAsk {
	dir: string;
	/** Whose photos they are. */
	profile: string;
	redates?: Redate[];
	choice?: Choice | null;
	/** Hard links, not copies. */
	link?: boolean;
	/** Photos deleted from the library to bring in again, by name, or "all". */
	bring_back?: string[];
}

/** Among the names to bring back, every photo deleted from the library. */
export const BRING_ALL_BACK = 'all';

/** Where the window's import stands: the look at a folder, then the copy. */
export interface ImportState extends Partial<ImportAsk> {
	running: boolean;
	/** Whether dir was a card when the run began. */
	card?: boolean;
	/** How the folder is named on every screen. */
	top?: string;
	kind?: 'plan' | 'import';
	/** The file being read or copied, n of of; done of total its bytes. */
	stage?: 'read' | 'copy';
	n: number;
	of: number;
	name?: string;
	done: number;
	total: number;
	filed?: Filed;
	/** The folder's bytes gone through, of all of them: the copy's pace. */
	moved: number;
	bytes: number;
	plan?: ImportPlan;
	/** The folder is on the library's disk: it can come in as hard links. */
	canLink?: boolean;
	imported?: Imported;
	stopped?: boolean;
	error?: string;
	problem?: 'no-link' | 'taken-out' | 'no-space' | 'busy' | 'library-gone' | 'unknown-name' | 'ambiguous-name';
	need?: number;
	free?: number;
}

export const getCards = () => call<Card[]>('GET', '/api/cards');
/** Runs what ran last once more, as it was asked: a card come back. */
export const againImport = () => call<ImportState>('POST', '/api/import/again');
export const planImport = (ask: ImportAsk) => call<ImportState>('POST', '/api/import/plan', ask);
export const startImport = (ask: ImportAsk) => call<ImportState>('POST', '/api/import', ask);
export const importThumbUrl = (path: string) => `/api/import/thumb?${q({ file: path })}`;
export const getImport = () => call<ImportState>('GET', '/api/import');
export const stopImport = () => call<ImportState>('POST', '/api/import/stop');
