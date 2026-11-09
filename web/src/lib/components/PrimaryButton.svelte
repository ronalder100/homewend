<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<script lang="ts">
	import type { Snippet } from 'svelte';

	let { onclick, disabled = false, children }: { onclick?: () => void; disabled?: boolean; children: Snippet } = $props();
</script>

<button {onclick} {disabled}>{@render children()}</button>

<style>
	/* The secondary button's size: the two stand side by side, and one
	   takes the other's place (Pause, Resume). */
	button {
		height: 44px;
		padding: 0 20px;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		border-radius: 12px;
		background: var(--accent);
		color: var(--on-accent);
		box-shadow: inset 0 0 0 1px #00000026;
		font-size: var(--text-body);
		font-weight: 600;
		white-space: nowrap;
	}
	button:hover:not(:disabled) {
		filter: brightness(1.06);
	}
	/* Greyed when it cannot be done now: the screen says why. */
	button:disabled {
		opacity: 0.4;
		cursor: default;
	}
</style>
