<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<script lang="ts">
	import type { Snippet } from 'svelte';

	let { onclick, children }: { onclick?: () => void; children: Snippet } = $props();
</script>

<button {onclick}>{@render children()}</button>

<style>
	button {
		height: 52px;
		padding: 0 24px;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		border-radius: 12px;
		background: var(--accent);
		color: var(--on-accent);
		box-shadow: inset 0 0 0 1px #00000026;
		font-size: var(--text-title-3);
		font-weight: 600;
		white-space: nowrap;
	}
	button:hover {
		filter: brightness(1.06);
	}
</style>
