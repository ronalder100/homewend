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
		path,
		onclose
	}: { x: number; y: number; path: string; onclose: () => void } = $props();

	function open() {
		window.shell?.openFolder(path);
		onclose();
	}
	async function copy() {
		await navigator.clipboard.writeText(path);
		onclose();
	}
</script>

<svelte:window
	onkeydown={(e) => e.key === 'Escape' && onclose()}
	onmousedown={(e) => !(e.target as Element).closest('.menu') && onclose()}
/>

<!-- 8px padding, 32px items: the first item's centre is 24px below the top. -->
<div class="menu" style:left="{x + 7}px" style:top="{y - 24}px" role="menu">
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
