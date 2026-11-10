// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

/** Bytes as a person reads them: 91 GB, 812 MB. */
export function bytes(n: number): string {
	const units = ['B', 'KB', 'MB', 'GB', 'TB'];
	let i = 0;
	while (n >= 1000 && i < units.length - 1) {
		n /= 1000;
		i++;
	}
	return `${n < 10 && i > 0 ? n.toFixed(1) : Math.round(n)} ${units[i]}`;
}

/** A duration as a person says it roughly: 9 h, 25 min. */
export function roughly(seconds: number): string {
	return seconds >= 3600 ? `${Math.round(seconds / 3600)} h` : `${Math.max(1, Math.round(seconds / 60))} min`;
}

/** A day as the window writes it everywhere: 3 Nov. */
export function day(d: Date): string {
	return `${d.getDate()} ${['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'][d.getMonth()]}`;
}

const clockFormat = new Intl.DateTimeFormat('en-GB', { hour: '2-digit', minute: '2-digit' });

/** A time of day as the window writes it: 09:41. */
export function clock(d: Date): string {
	return clockFormat.format(d);
}

// A photo's date is the clock its camera showed, kept as UTC: read in UTC,
// it is that clock, wherever the window runs.
const takenDayFormat = new Intl.DateTimeFormat('en-GB', { timeZone: 'UTC', day: 'numeric', month: 'long', year: 'numeric' });
const takenTimeFormat = new Intl.DateTimeFormat('en-GB', { timeZone: 'UTC', hour: '2-digit', minute: '2-digit' });

/** The day a photo was taken: 14 July 2019. */
export function takenDay(d: Date): string {
	return takenDayFormat.format(d);
}

/** The time a photo was taken: 18:32. */
export function takenTime(d: Date): string {
	return takenTimeFormat.format(d);
}
