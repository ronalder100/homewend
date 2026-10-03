// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// Where the window is, as an address: the back button and a reload keep it.
import type { Place } from './library.js';

export function urlOf(p: Place): string {
	switch (p.kind) {
		case 'all':
			return '/library';
		case 'year':
			return `/library?year=${encodeURIComponent(p.label)}`;
		case 'album':
			return `/library?account=${encodeURIComponent(p.account)}&album=${encodeURIComponent(p.name)}`;
		case 'takeouts':
			return '/takeouts';
		case 'settings':
			return '/settings';
	}
}

export function placeOf(url: { pathname: string; searchParams: { has(k: string): boolean; get(k: string): string | null } }): Place {
	const q = url.searchParams;
	if (url.pathname.startsWith('/settings')) return { kind: 'settings' };
	if (url.pathname.startsWith('/takeouts')) return { kind: 'takeouts' };
	if (q.has('album')) return { kind: 'album', account: q.get('account') ?? '', name: q.get('album') ?? '' };
	if (q.has('year')) return { kind: 'year', label: q.get('year') ?? '' };
	return { kind: 'all' };
}
