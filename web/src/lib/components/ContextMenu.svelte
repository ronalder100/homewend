<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- B1's menu: one bubble whose arrow touches the end of the row it is for,
     its first item on the row's centre line. Ours, the same on every system. -->
<script lang="ts">
	import { FolderOpen, Copy } from '@lucide/svelte';
	import { text } from '#lib/strings.js';

	let {
		x,
		y,
		start = x,
		path,
		file = false,
		onclose
	}: {
		x: number;
		y: number;
		/** Where the row begins, for the bubble that opens to its left. */
		start?: number;
		path: string;
		/** A photo: its folder opens with the photo picked in it. */
		file?: boolean;
		onclose: () => void;
	} = $props();

	// Where the window ends, the bubble opens to the left of its row instead.
	const WIDTH = 208;
	let innerWidth = $state(Infinity);
	const flip = $derived(x + 7 + WIDTH > innerWidth);

	function open() {
		if (file) window.shell?.showInFolder(path);
		else window.shell?.openFolder(path);
		onclose();
	}
	async function copy() {
		// The folder's path, as the item says, also for a photo.
		await navigator.clipboard.writeText(file ? path.replace(/[\\/][^\\/]*$/, '') : path);
		onclose();
	}
</script>

<svelte:window
	bind:innerWidth
	onkeydown={(e) => e.key === 'Escape' && onclose()}
	onmousedown={(e) => !(e.target as Element).closest('.menu') && onclose()}
/>

<!-- 8px padding, 32px items: the first item's centre is 24px below the top. -->
<div class="menu" class:flip style:left="{flip ? start - 7 - WIDTH : x + 7}px" style:top="{y - 24}px" role="menu">
	<button role="menuitem" onclick={open}><FolderOpen size={16} />{text.openInFolder}</button>
	<button role="menuitem" onclick={copy}><Copy size={16} />{text.copyFolderPath}</button>
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
