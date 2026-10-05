// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// A context menu whose items wait for the engine: the right-click is kept
// until the answer comes, and dropped if it no longer stands by then.
export class Opener<T> {
	/** The menu shown, with the element that opened it. */
	menu = $state<(T & { anchor: HTMLElement }) | null>(null);
	#opening: HTMLElement | null = null;

	/** Opens the menu for the element right-clicked, once load answers: unless
	 *  another right-click came since, a click or a key dismissed it, or the
	 *  element left the page (a row scrolled away). */
	async open(e: MouseEvent, load: () => Promise<T>) {
		e.preventDefault();
		const anchor = (this.#opening = e.currentTarget as HTMLElement);
		const dismiss = () => {
			if (this.#opening === anchor) this.#opening = null;
		};
		addEventListener('pointerdown', dismiss, { capture: true, once: true });
		addEventListener('keydown', dismiss, { capture: true, once: true });
		try {
			const value = await load();
			if (this.#opening === anchor && anchor.isConnected) this.menu = { ...value, anchor };
		} finally {
			removeEventListener('pointerdown', dismiss, { capture: true });
			removeEventListener('keydown', dismiss, { capture: true });
		}
	}

	close = () => {
		this.menu = null;
		this.#opening = null;
	};
}
