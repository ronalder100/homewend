// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// What the window knows: the settings and the library's overview, read from
// the engine and kept in step with it.
import {
	getJob,
	getOverview,
	getSettings,
	putSettings,
	type JobState,
	type Overview,
	type Settings
} from './api.js';
import type { Sidebar } from './library.js';
import { text } from './strings.js';

export const app = $state<{
	settings: Settings;
	overview: Overview | null;
	jobs: Record<string, JobState>;
	/** Seconds left at the speed of the last minute, per account. */
	left: Record<string, number>;
	error: string;
}>({
	settings: {},
	overview: null,
	jobs: {},
	left: {},
	error: ''
});

const samples: Record<string, { t: number; done: number }[]> = {};

function measure(account: string, j: JobState) {
	const now = Date.now();
	const s = (samples[account] = [
		...(samples[account] ?? []).filter((x) => now - x.t < 60000 && x.done <= j.done),
		{ t: now, done: j.done }
	]);
	const rate = (j.done - s[0].done) / ((now - s[0].t) / 1000);
	if (j.running && j.total > 0 && rate > 0) app.left[account] = (j.total - j.done) / rate;
	else delete app.left[account];
}

/** Asks every second where each account's download stands. */
export function watchJobs(): () => void {
	let wasRunning = false;
	const tick = async () => {
		for (const a of app.overview?.accounts ?? []) {
			try {
				app.jobs[a.id] = await getJob(a.id);
				measure(a.id, app.jobs[a.id]);
			} catch {
				// The engine went away: the window says so elsewhere.
			}
		}
		// Photos arrive while a job runs: the counts follow them.
		const running = Object.values(app.jobs).some((j) => j.running);
		if (running || wasRunning) app.overview = await getOverview();
		wasRunning = running;
	};
	tick();
	const t = setInterval(tick, 1000);
	return () => clearInterval(t);
}

/** The account whose download the window shows: the one running, or the first. */
export function shownJob(): { account: string; job: JobState } | null {
	const entries = Object.entries(app.jobs);
	const running = entries.find(([, j]) => j.running);
	const any = running ?? entries.find(([, j]) => j.stage || j.error || j.finished);
	return any ? { account: any[0], job: any[1] } : null;
}

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
		// An account is its address: Homewend asks Google for nothing more.
		accounts: o.accounts.map((a) => ({ id: a.id, email: a.email, name: a.email, shown: a.shown, photos: a.photos })),
		total: o.total,
		years: o.years.map((y) => ({
			label: y.year || text.noDate,
			count: y.count,
			state: yearState(y.year)
		})),
		albums: o.albums,
		takeouts: o.takeouts
	};
}

/** A year is complete when the download that brought it says every photo
 *  arrived, arriving while it still counts; otherwise nothing is said. */
function yearState(year: string): 'complete' | 'arriving' | 'none' {
	for (const j of Object.values(app.jobs)) {
		const y = j.years.find((x) => x.year === year);
		if (!y) continue;
		if (y.of > 0 && y.arrived >= y.of) return 'complete';
		return j.running ? 'arriving' : 'none';
	}
	return 'none';
}
