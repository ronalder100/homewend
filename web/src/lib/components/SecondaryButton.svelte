<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		small = false,
		onclick,
		children
	}: { small?: boolean; onclick?: () => void; children: Snippet } = $props();
</script>

<button class:small {onclick}>{@render children()}</button>

<style>
	button {
		height: 40px;
		padding: 0 16px;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		border: 1px solid var(--border);
		border-radius: 12px;
		color: var(--accent);
		font-size: var(--text-body);
		font-weight: 600;
		white-space: nowrap;
	}
	button:hover {
		background: var(--soft);
	}
	/* The titlebar's size of the same button. */
	.small {
		height: 32px;
		padding: 0 12px;
		border-radius: 8px;
		font-size: var(--text-callout);
	}
</style>
