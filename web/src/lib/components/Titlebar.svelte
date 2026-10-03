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
	let importing = $state(false);
</script>

<header>
	{#if menu && !mac}
		<button class="menu" aria-label={text.menu} class:open onclick={onmenu}
			><i></i><i></i><i></i></button
		>
	{/if}
	<Brand />
	<span class="spacer"></span>
	<span class="anchor">
		<SecondaryButton small onclick={() => (importing = !importing)}
			><FolderInput size={14} />{text.importFolder}</SecondaryButton
		>
		{#if importing}
			<!-- C3: importing a folder is phase 2; the button says what is coming. -->
			<div class="popover" role="dialog" aria-label={text.importTitle}>
				<div class="row">
					<span class="tile"><FolderInput size={18} /></span><span class="chip">{text.upcoming}</span>
				</div>
				<b>{text.importTitle}</b>
				<p>{text.importDesc}</p>
			</div>
		{/if}
	</span>
</header>
<svelte:window
	onkeydown={(e) => e.key === 'Escape' && (importing = false)}
	onclick={(e) => {
		// A click anywhere but the popover and its button closes it.
		if (importing && !(e.target as Element).closest('.anchor')) importing = false;
	}}
/>

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
	.anchor {
		position: relative;
	}
	.popover {
		position: absolute;
		right: 0;
		top: 40px;
		z-index: 20;
		width: 340px;
		padding: 24px;
		display: flex;
		flex-direction: column;
		gap: 12px;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 12px;
		box-shadow: 0 12px 40px rgba(10, 10, 30, 0.16);
		-webkit-app-region: no-drag;
	}
	.popover .row {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}
	.popover b {
		font-size: var(--text-title-2);
		font-weight: 700;
		color: var(--fg);
	}
	.tile {
		width: 36px;
		height: 36px;
		display: grid;
		place-items: center;
		border-radius: 8px;
		background: var(--accent-soft);
		color: var(--accent);
	}
	.chip {
		padding: 2px 8px;
		border-radius: 999px;
		background: var(--chip);
		color: #0a0a0a;
		font-size: var(--text-subheadline);
		font-weight: 600;
	}
	.popover p {
		font-size: var(--text-callout);
		line-height: 1.5;
		color: var(--muted);
	}
</style>
