<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<script lang="ts">
	import { FolderInput } from '@lucide/svelte';
	import Brand from './Brand.svelte';
	import SecondaryButton from './SecondaryButton.svelte';
	import { text } from '#lib/strings.js';

	let {
		menu = true,
		open = false,
		onmenu
	}: { menu?: boolean; /** the sidebar is open */ open?: boolean; onmenu?: () => void } = $props();

	// On the Mac the corner belongs to the traffic lights: the sidebar opens
	// from the View menu and ⌃⌘S there, as in Apple's own apps.
	const mac = typeof navigator !== 'undefined' && /Mac/.test(navigator.platform);
</script>

<header>
	{#if menu && !mac}
		<button class="menu" aria-label={text.menu} class:open onclick={onmenu}
			><i></i><i></i><i></i></button
		>
	{/if}
	<Brand />
	<span class="spacer"></span>
	<SecondaryButton small><FolderInput size={14} />{text.importFolder}</SecondaryButton>
</header>

<style>
	header {
		height: 44px;
		flex-shrink: 0;
		display: flex;
		align-items: center;
		gap: 8px;
		/* In the desktop app the window's own controls sit in this bar: the
		   traffic lights on the left on a Mac, the buttons on the right
		   elsewhere. The titlebar-area variables say where they are; in a
		   browser they are unset and the plain padding applies. */
		padding-left: max(16px, calc(env(titlebar-area-x, 0px) + 8px));
		padding-right: max(
			16px,
			calc(100vw - env(titlebar-area-x, 0px) - env(titlebar-area-width, 100vw) + 8px)
		);
		background: var(--bg);
		border-bottom: 1px solid var(--line);
		-webkit-app-region: drag;
	}
	header :global(button) {
		-webkit-app-region: no-drag;
	}
	/* ☰ turns into ✕ while the sidebar is open: three bars, the middle one
	   fades and the outer two cross. */
	.menu {
		width: 24px;
		height: 24px;
		position: relative;
	}
	.menu i {
		position: absolute;
		left: 4px;
		width: 16px;
		height: 2px;
		border-radius: 1px;
		background: var(--fg);
		transition:
			transform 200ms ease,
			opacity 200ms ease;
	}
	.menu i:nth-child(1) {
		top: 6px;
	}
	.menu i:nth-child(2) {
		top: 11px;
	}
	.menu i:nth-child(3) {
		top: 16px;
	}
	.menu.open i:nth-child(1) {
		transform: translateY(5px) rotate(45deg);
	}
	.menu.open i:nth-child(2) {
		opacity: 0;
	}
	.menu.open i:nth-child(3) {
		transform: translateY(-5px) rotate(-45deg);
	}
	@media (prefers-reduced-motion: reduce) {
		.menu i {
			transition: none;
		}
	}
	.spacer {
		flex: 1;
	}
</style>
