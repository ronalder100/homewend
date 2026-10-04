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
		status,
		banner
	}: {
		sidebar: SidebarData;
		place: Place;
		onplace?: (p: Place) => void;
		ontoggle?: (account: string) => void;
		children: Snippet;
		status?: Snippet;
		banner?: Snippet;
	} = $props();

	// Below 1100px the sidebar folds to its rail and opens over the content;
	// above, ☰ folds and unfolds it in place. Over the page it closes with ☰,
	// its shortcut, a click beside it, or Takeouts and Settings, which leave
	// the photos; a year or an album keeps it open, to look at the next.
	let width = $state(1280);
	const narrow = $derived(width < 1100);
	// Half a laptop screen: no room even for the rail; ☰ still opens the sidebar.
	const tiny = $derived(width < 720);
	let folded = $state(false);
	let open = $state(false);
	const rail = $derived(narrow ? !open : folded);

	function menu() {
		if (narrow) open = !open;
		else folded = !folded;
	}
	function placed(p: Place) {
		if (narrow && (p.kind === 'takeouts' || p.kind === 'settings')) open = false;
		onplace?.(p);
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

<div class="window" class:narrow class:tiny>
	<Titlebar open={narrow && open} onmenu={menu} />
	{@render banner?.()}
	<div class="body">
		<div class="side" class:over={narrow && open}>
			<Sidebar data={sidebar} {place} {rail} onplace={placed} {ontoggle} onexpand={expand} />
		</div>
		{#if narrow && open}
			<!-- Open over the page, the sidebar closes with a click anywhere
			     beside it, the titlebar included; choosing a row in it leaves it open. -->
			<button class="scrim" aria-label="close" onclick={() => (open = false)}></button>
		{/if}
		<div class="main">
			<!-- A page replaces the last at once, as in a native app: a fade
			     between them reads as a flash. -->
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
	.tiny .side {
		width: 0;
	}
	.tiny .side:not(.over) {
		visibility: hidden;
	}
	.side {
		height: 100%;
		z-index: 2;
	}
	.side :global(nav) {
		transition: width 180ms ease;
	}
	.side.over {
		animation: slide 180ms ease;
		position: absolute;
		left: 0;
		top: 0;
		bottom: 0;
		box-shadow: 0 12px 40px rgba(10, 10, 30, 0.16);
	}
	@keyframes slide {
		from {
			transform: translateX(-12px);
			opacity: 0.6;
		}
	}
	.scrim {
		position: fixed;
		inset: 0;
		z-index: 1;
		cursor: default;
		/* Over the titlebar too: a drag region would swallow the click. */
		-webkit-app-region: no-drag;
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
