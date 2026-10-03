// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// What the sidebar shows, for any number of Google accounts.

export interface Account {
	id: string;
	name: string;
	email: string;
	/** Whether its photos are in what the window shows. */
	shown: boolean;
}

/** "complete": every file the takeouts list is here; "arriving": parts still coming. */
export type YearState = 'complete' | 'arriving' | 'none';

export interface Year {
	/** A year, or a bucket the engine names ("Still sorting", "No date"). */
	label: string;
	count: number;
	state: YearState;
}

export interface Album {
	account: string;
	name: string;
	count: number;
}

export interface Sidebar {
	accounts: Account[];
	total: number;
	years: Year[];
	albums: Album[];
	takeouts: number;
}

/** Where the window is: one row of the sidebar is this. */
export type Place =
	| { kind: 'all' }
	| { kind: 'year'; label: string }
	| { kind: 'album'; account: string; name: string }
	| { kind: 'takeouts' }
	| { kind: 'settings' };

export function samePlace(a: Place, b: Place): boolean {
	return JSON.stringify(a) === JSON.stringify(b);
}

export function initials(name: string): string {
	return name
		.split(/\s+/)
		.filter(Boolean)
		.slice(0, 2)
		.map((w) => w[0].toUpperCase())
		.join('');
}

/** How many of a list show before "N more". */
export const FOLD = { accounts: 4, albums: 3 } as const;
