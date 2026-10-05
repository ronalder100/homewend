<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- B1's menu: one bubble whose arrow touches the end of the row it is for,
     its first item on the row's centre line. Ours, the same on every system. -->
<script module lang="ts">
	import { FolderOpen, Copy } from '@lucide/svelte';
	import type { Component } from 'svelte';
	import { text } from '#lib/strings.js';

	export type MenuItem = { icon: Component; label: string; run: () => void | Promise<void> };

	/** A folder's two items, or a photo's: open it, copy its path. */
	export function folderItems(path: string, file = false): MenuItem[] {
		return [
			{
				icon: FolderOpen,
				label: text.openInFolder,
				// A photo's folder opens with the photo picked in it.
				run: () => (file ? window.shell?.showInFolder(path) : window.shell?.openFolder(path))
			},
			{
				icon: Copy,
				label: text.copyFolderPath,
				// The folder's path, as the item says, also for a photo.
				run: () => navigator.clipboard.writeText(file ? path.replace(/[\\/][^\\/]*$/, '') : path)
			}
		];
	}
</script>

<script lang="ts">
	let {
		x,
		y,
		start = x,
		items,
		anchor,
		onclose
	}: {
		x: number;
		y: number;
		/** Where the row begins, for the bubble that opens to its left. */
		start?: number;
		items: MenuItem[];
		/** The button that opened it, which closes it itself. */
		anchor?: Element;
		onclose: () => void;
	} = $props();

	// Where the window ends, the bubble opens to the left of its row instead.
	const WIDTH = 208;
	let innerWidth = $state(Infinity);
	const flip = $derived(x + 7 + WIDTH > innerWidth);

	let bubble = $state<HTMLElement>();

	async function pick(item: MenuItem) {
		onclose();
		await item.run();
	}
</script>

<svelte:window
	bind:innerWidth
	onkeydown={(e) => e.key === 'Escape' && onclose()}
	onscrollcapture={onclose}
	onmousedown={(e) => !bubble?.contains(e.target as Node) && !anchor?.contains(e.target as Node) && onclose()}
/>

<!-- Placed once, where its row was: whatever scrolls closes it, rather than
     leave it pointing at another row. -->
<!-- 8px padding, 32px items: the first item's centre is 24px below the top. -->
<div bind:this={bubble} class="menu" class:flip style:left="{flip ? start - 7 - WIDTH : x + 7}px" style:top="{y - 24}px" role="menu">
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
