<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- The window around every screen after sign-in: titlebar, sidebar, content. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import Titlebar from './Titlebar.svelte';
	import Sidebar from './Sidebar.svelte';
	import type { Place, Sidebar as SidebarData } from '#lib/library.js';

	let {
		sidebar,
		place,
		onplace,
		ontoggle,
		children,
		status
	}: {
		sidebar: SidebarData;
		place: Place;
		onplace?: (p: Place) => void;
		ontoggle?: (account: string) => void;
		children: Snippet;
		status?: Snippet;
	} = $props();

	// Below 1100px the sidebar folds to its rail and opens over the content;
	// above, ☰ folds and unfolds it in place. Only ☰ (or its shortcut) closes
	// it: choosing a year or an album leaves it open.
	let width = $state(1280);
	const narrow = $derived(width < 1100);
	let folded = $state(false);
	let open = $state(false);
	const rail = $derived(narrow ? !open : folded);

	function menu() {
		if (narrow) open = !open;
		else folded = !folded;
	}
	function expand() {
		if (narrow) open = true;
		else folded = false;
	}
	$effect(() => window.shell?.onToggleSidebar(menu));

	function key(e: KeyboardEvent) {
		const mac = /Mac/.test(navigator.platform);
		if ((mac && e.ctrlKey && e.metaKey && e.key === 's') || (!mac && e.ctrlKey && e.key === 'b')) {
			e.preventDefault();
			menu();
		}
	}
</script>

<svelte:window bind:innerWidth={width} onkeydown={key} />

<div class="window" class:narrow>
	<Titlebar open={narrow && open} onmenu={menu} />
	<div class="body">
		<div class="side" class:over={narrow && open}>
			<Sidebar data={sidebar} {place} {rail} {onplace} {ontoggle} onexpand={expand} />
		</div>
		<div class="main">
			<div class="content">{@render children()}</div>
			{#if status}<footer>{@render status()}</footer>{/if}
		</div>
	</div>
</div>

<style>
	.window {
		height: 100%;
		display: flex;
		flex-direction: column;
	}
	.body {
		flex: 1;
		min-height: 0;
		display: flex;
		position: relative;
	}
	.narrow .side {
		width: 56px;
		flex-shrink: 0;
	}
	.side {
		height: 100%;
		z-index: 2;
	}
	.side.over {
		position: absolute;
		left: 0;
		top: 0;
		bottom: 0;
		box-shadow: 0 12px 40px rgba(10, 10, 30, 0.16);
	}
	.main {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
	}
	.content {
		flex: 1;
		min-height: 0;
		overflow-y: auto;
	}
	/* A screen that scrolls by itself, like the grid, fills it exactly. */
	.content > :global(:only-child) {
		min-height: 100%;
	}
	footer {
		height: 56px;
		flex-shrink: 0;
		display: flex;
		align-items: center;
		gap: 16px;
		padding: 0 24px;
		border-top: 1px solid var(--line);
	}
</style>
