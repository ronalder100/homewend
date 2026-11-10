// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// How fast bytes go, over the last minute, and so how long is left: one
// measure for a download and for an import.
const samples: Record<string, { t: number; done: number }[]> = {};

/** Bytes a second for key, and seconds left to total; none until known. */
export function pace(key: string, done: number, total: number): { rate: number; left: number } | null {
	const now = Date.now();
	const s = (samples[key] = [
		...(samples[key] ?? []).filter((x) => now - x.t < 60000 && x.done <= done),
		{ t: now, done }
	]);
	const rate = (done - s[0].done) / ((now - s[0].t) / 1000);
	return total > 0 && rate > 0 ? { rate, left: (total - done) / rate } : null;
}

/** Forgets key's pace: an account gone. */
export function forget(key: string) {
	delete samples[key];
}
