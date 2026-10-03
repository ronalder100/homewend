// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// What the window knows: the settings and the library's overview, read from
// the engine and kept in step with it.
import { getOverview, getSettings, putSettings, type Overview, type Settings } from './api.js';
import type { Sidebar } from './library.js';
import { text } from './strings.js';

export const app = $state<{ settings: Settings; overview: Overview | null; error: string }>({
	settings: {},
	overview: null,
	error: ''
});

export async function refresh() {
	try {
		const [settings, overview] = await Promise.all([getSettings(), getOverview()]);
		app.settings = settings;
		app.overview = overview;
		app.error = '';
	} catch (e) {
		app.error = String(e);
	}
}

/** Shows or hides an account's photos, and remembers it. */
export async function toggleAccount(id: string) {
	const hidden = new Set(app.settings.hidden ?? []);
	if (hidden.has(id)) hidden.delete(id);
	else hidden.add(id);
	app.settings = await putSettings({ ...app.settings, hidden: [...hidden] });
	app.overview = await getOverview();
}

/** The engine's overview as the sidebar draws it. */
export function sidebarOf(o: Overview): Sidebar {
	return {
		// Google gives an address; until it gives a name, the part before
		// the @ is what a person recognises.
		accounts: o.accounts.map((a) => ({
			id: a.id,
			email: a.email,
			name: a.email.split('@')[0],
			shown: a.shown
		})),
		total: o.total,
		years: o.years.map((y) => ({
			label: y.year || text.noDate,
			count: y.count,
			state: 'none' as const
		})),
		albums: o.albums,
		takeouts: o.takeouts
	};
}
