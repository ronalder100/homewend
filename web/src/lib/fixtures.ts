// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// Made-up data for the components page only: the screens read the engine.
import type { Sidebar } from './library.js';

export const twoAccounts: Sidebar = {
	accounts: [
		{ id: 'a', name: 'Alex Morgan', email: 'alex.morgan@example.com', shown: true },
		{ id: 'b', name: 'Sam Rivera', email: 'sam.rivera@example.com', shown: true }
	],
	total: 38584,
	years: [
		{ label: '2025', count: 2031, state: 'complete' },
		{ label: '2024', count: 1610, state: 'arriving' },
		{ label: '2023', count: 2950, state: 'complete' },
		{ label: '2022', count: 812, state: 'arriving' },
		{ label: 'Still sorting', count: 1240, state: 'arriving' },
		{ label: 'No date', count: 37, state: 'none' }
	],
	albums: [
		{ account: 'a', name: 'Greece 2019', count: 412 },
		{ account: 'a', name: 'Wedding', count: 1088 },
		{ account: 'a', name: 'Paris 2021', count: 230 },
		{ account: 'a', name: 'Lisbon', count: 96 },
		{ account: 'b', name: 'Kids', count: 6530 }
	],
	takeouts: 5
};

export const tenAccounts: Sidebar = {
	...twoAccounts,
	accounts: [
		...twoAccounts.accounts,
		...['Jordan Lee', 'Casey Kim', 'Riley Chen', 'Taylor Brooks', 'Morgan Diaz', 'Avery Patel', 'Quinn Ross', 'Drew Evans'].map(
			(name, i) => ({ id: `x${i}`, name, email: `${name.toLowerCase().replace(' ', '.')}@example.com`, shown: false })
		)
	]
};
