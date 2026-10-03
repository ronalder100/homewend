// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// The engine, as the page reaches it: one function per call of internal/web.

export type Theme = 'system' | 'light' | 'dark';

export interface Settings {
	theme?: Theme;
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
