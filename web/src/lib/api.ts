// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// The engine, as the page reaches it: one function per call of internal/web.

export type Theme = 'system' | 'light' | 'dark';

export interface Settings {
	theme?: Theme;
	hidden?: string[];
}

export interface AccountInfo {
	id: string;
	email: string;
}

export interface Overview {
	accounts: (AccountInfo & { shown: boolean })[];
	total: number;
	years: { year: string; count: number }[];
	albums: { account: string; name: string; count: number }[];
	takeouts: number;
}

async function call<T>(method: string, path: string, body?: unknown): Promise<T> {
	const res = await fetch(path, {
		method,
		headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
		body: body === undefined ? undefined : JSON.stringify(body)
	});
	if (!res.ok) throw new Error(`${method} ${path}: ${res.status} ${await res.text()}`);
	return res.json();
}

export const getSettings = () => call<Settings>('GET', '/api/settings');
export const putSettings = (s: Settings) => call<Settings>('PUT', '/api/settings', s);
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
