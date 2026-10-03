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
