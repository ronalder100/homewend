<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<script lang="ts">
	import { Menu, FolderInput } from '@lucide/svelte';
	import Brand from './Brand.svelte';
	import SecondaryButton from './SecondaryButton.svelte';
	import { text } from '#lib/strings.js';

	let { menu = true, onmenu }: { menu?: boolean; onmenu?: () => void } = $props();

	// On the Mac the corner belongs to the traffic lights: the sidebar opens
	// from the View menu and ⌃⌘S there, as in Apple's own apps.
	const mac = typeof navigator !== 'undefined' && /Mac/.test(navigator.platform);
</script>

<header>
	{#if menu && !mac}
		<button class="menu" aria-label={text.menu} onclick={onmenu}><Menu size={20} /></button>
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
	.menu {
		width: 24px;
		height: 24px;
		display: grid;
		place-items: center;
		color: var(--fg);
	}
	.spacer {
		flex: 1;
	}
</style>
