<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- B1's menu: one bubble whose arrow touches the end of the row it is for,
     its first item on the row's centre line. Ours, the same on every system. -->
<script module lang="ts">
	import { FolderOpen, Copy } from '@lucide/svelte';
	import type { Component } from 'svelte';
	import { text } from '#lib/strings.js';

	export type MenuItem = { icon: Component; label: string; run: () => void | Promise<void> };

	/** A folder's two items, or a photo's: open it, copy its path. The path
	 *  may be asked of the engine only when an item is chosen. */
	export function folderItems(path: string | (() => Promise<string>), file = false): MenuItem[] {
		const where = async () => (typeof path === 'string' ? path : path());
		return [
			{
				icon: FolderOpen,
				label: text.openInFolder,
				// A photo's folder opens with the photo picked in it.
				run: async () => {
					const p = await where();
					if (file) window.shell?.showInFolder(p);
					else window.shell?.openFolder(p);
				}
			},
			{
				icon: Copy,
				label: text.copyFolderPath,
				// The folder's path, as the item says, also for a photo.
				run: async () => {
					const p = await where();
					await navigator.clipboard.writeText(file ? p.replace(/[\\/][^\\/]*$/, '') : p);
				}
			}
		];
	}
</script>

<script lang="ts">
	let {
		items,
		anchor,
		onclose
	}: {
		items: MenuItem[];
		/** What opened it, a row, a tile or a button: the bubble's arrow touches
		 *  its end, the keyboard goes back to it when the menu closes, and a
		 *  button that pops it up (aria-haspopup) closes it when pressed again. */
		anchor: Element;
		onclose: () => void;
	} = $props();

	// Placed where its opener is as it opens, and again if another opener
	// takes its place. Where the window ends, the bubble opens to the left of
	// the opener instead.
	const WIDTH = 208;
	const at = $derived(anchor.getBoundingClientRect());
	const x = $derived(at.right);
	const y = $derived(at.top + at.height / 2);
	let innerWidth = $state(Infinity);
	const flip = $derived(x + 7 + WIDTH > innerWidth);

	let bubble = $state<HTMLElement>();

	// As a native menu: it takes the keyboard when it opens, the arrows move
	// through it, and Escape or Tab closes it and gives the keyboard back to
	// what opened it. Focus that goes nowhere, as a click on its edge, is not
	// leaving it: clicks are told by mousedown.
	$effect(() => {
		bubble?.querySelector('button')?.focus();
	});
	// A button that pops the menu up (aria-haspopup) is not outside: pressed
	// again, it closes it itself. A row or a tile that opened it is outside,
	// as any other click.
	const outside = (target: EventTarget | null) =>
		!bubble?.contains(target as Node) &&
		!(anchor.hasAttribute('aria-haspopup') && anchor.contains(target as Node));
	// Its opener gone from the page, a row read again or scrolled away, the
	// menu goes too, rather than point at nothing.
	$effect(() => {
		const gone = new MutationObserver(() => !anchor.isConnected && onclose());
		gone.observe(document.body, { childList: true, subtree: true });
		return () => gone.disconnect();
	});

	// The arrows walk the items, round from the last to the first.
	function step(by: number) {
		const buttons = [...(bubble?.querySelectorAll('button') ?? [])];
		const at = buttons.indexOf(document.activeElement as HTMLButtonElement);
		// With none of them focused, down starts at the first, up at the last.
		const next = at < 0 ? (by > 0 ? 0 : buttons.length - 1) : (at + by + buttons.length) % buttons.length;
		buttons[next]?.focus();
	}
	// Closed by the keyboard or by a choice, the keyboard goes back to what
	// opened it. Read before closing: closed, its props are gone with it.
	function back() {
		const opener = anchor as HTMLElement;
		onclose();
		// Where the person scrolled to stays: the opener gets the keyboard, not
		// the view.
		opener.focus({ preventScroll: true });
	}

	async function pick(item: MenuItem) {
		back();
		await item.run();
	}
</script>

<svelte:window
	bind:innerWidth
	onkeydown={(e) => {
		if (e.key === 'Escape' || (e.key === 'Tab' && bubble?.contains(document.activeElement))) {
			e.preventDefault();
			back();
		} else if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
			e.preventDefault();
			step(e.key === 'ArrowDown' ? 1 : -1);
		}
	}}
	onscrollcapture={() => (bubble?.contains(document.activeElement) ? back() : onclose())}
	onmousedown={(e) => outside(e.target) && onclose()}
/>

<!-- Placed where its row was: whatever scrolls closes it, rather than leave
     it pointing at another row, and gives the keyboard back if it had it. -->
<!-- 8px padding, 32px items: the first item's centre is 24px below the top. -->
<div
	bind:this={bubble}
	onfocusout={(e) => e.relatedTarget && outside(e.relatedTarget) && onclose()}
	class="menu" class:flip style:left="{flip ? at.left - 7 - WIDTH : x + 7}px" style:top="{y - 24}px" role="menu">
	{#each items as item (item.label)}
		<button role="menuitem" onclick={() => pick(item)}><item.icon size={16} />{item.label}</button>
	{/each}
</div>

<style>
	.menu {
		position: fixed;
		z-index: 30;
		width: 208px;
		padding: 8px;
		display: flex;
		flex-direction: column;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 8px;
		box-shadow:
			0 12px 40px #0a0a1a14,
			0 2px 6px #0a0a1a12;
	}
	/* The arrow: a square turned, its tip on the row's end. */
	.menu::before {
		content: '';
		position: absolute;
		left: -6px;
		top: 18px;
		width: 11px;
		height: 11px;
		background: var(--surface);
		border-left: 1px solid var(--border);
		border-bottom: 1px solid var(--border);
		transform: rotate(45deg);
	}
	.flip::before {
		left: auto;
		right: -6px;
		transform: rotate(225deg);
	}
	button {
		position: relative;
		height: 32px;
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 0 8px;
		border-radius: 4px;
		color: var(--fg);
		font-size: var(--text-body);
		text-align: left;
	}
	button :global(svg) {
		color: var(--muted);
	}
	button:hover {
		background: var(--accent-soft);
		color: var(--accent);
	}
	button:hover :global(svg) {
		color: var(--accent);
	}
</style>
