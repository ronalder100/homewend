// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// An account's photos on Google, counted by year: read, asked for once if
// there is none, and looked at every two seconds while the count runs.
import { countYears, getYears, type YearCounts } from './api.js';

export class Years {
	counts = $state<YearCounts | null>(null);
	/** When the count began here, for the seconds it has run. */
	since = $state(0);
	tick = $state(Date.now());

	/** The years, newest first, once counted. */
	get counted() {
		return this.counts?.years?.length ? this.counts.years : null;
	}
	/** Not known yet: no year is offered rather than ones the account may not have. */
	get finding() {
		return !this.counts || this.counts.counting;
	}
	get seconds() {
		return Math.max(0, Math.floor((this.tick - this.since) / 1000));
	}
	itemsOf(y: number) {
		return this.counted?.find((c) => c.year === y)?.items;
	}

	/** Called while a component is created: it follows account(). */
	constructor(account: () => string) {
		$effect(() => {
			const a = account();
			if (!a) return;
			this.counts = null;
			let asked = false;
			let stop = false;
			const look = async (): Promise<void> => {
				try {
					const c = await getYears(a);
					if (stop) return;
					if (!c.years && !c.counting && !c.error && !asked) {
						asked = true;
						await countYears(a).catch(() => {});
						this.since = Date.now();
						return look();
					}
					if (c.counting && !this.since) this.since = Date.now();
					this.counts = c;
				} catch (e) {
					// No count: every year is offered.
					if (!stop) this.counts = { counting: false, error: String(e) };
				}
			};
			look();
			const t = setInterval(() => {
				this.tick = Date.now();
				if (this.counts?.counting) look();
			}, 1000);
			return () => {
				stop = true;
				clearInterval(t);
			};
		});
	}
}
