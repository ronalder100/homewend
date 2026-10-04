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
	finished?: boolean;
	lastLooked?: string;
	unassigned: number;
	error?: string;
	problem?: 'signed-out' | 'no-space' | 'expired';
	need?: number;
	free?: number;
}

const q = (o: Record<string, string | number | undefined>) =>
	new URLSearchParams(
		Object.entries(o).filter(([, v]) => v !== undefined && v !== '') as [string, string][]
	).toString();

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
	local?: { parts: number; of: number; declared?: number; present?: number };
}

export const getTakeouts = (account: string) => call<Takeout[]>('GET', `/api/takeouts?${q({ account })}`);

export const logout = (account: string) => call<object>('POST', `/api/logout?${q({ account })}`);
export const getVersion = () => call<{ version: string }>('GET', '/api/about');

export const pathOf = (hash: string) => call<{ path: string }>('GET', `/api/path/${hash}`);

export const folderOf = (o: { account?: string; year?: string; album?: string; noDate?: boolean }) =>
	call<{ path: string }>(
		'GET',
		`/api/folder?${q({ account: o.account, year: o.year, album: o.album, nodate: o.noDate ? '1' : undefined })}`
	);

export const getLatest = () => call<GridPhoto[]>('GET', '/api/latest');
export const getAbout = (hash: string) => call<{ albums: string[]; accounts: string[] }>('GET', `/api/about/${hash}`);

export const readContents = (account: string, exportId: string) =>
	call<{ years: string[] }>('POST', `/api/takeouts/contents?${q({ account, export: exportId })}`);
