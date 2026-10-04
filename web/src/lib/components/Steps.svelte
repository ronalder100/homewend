<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- The first run's three steps, under the titlebar (A2–A5). -->
<script lang="ts">
	import { Check } from '@lucide/svelte';

	let { steps, at }: { steps: string[]; at: number } = $props();
</script>

<ol>
	{#each steps as label, i (i)}
		<li class:done={i < at} class:now={i === at}>
			<span class="dot">{#if i < at}<Check size={14} strokeWidth={3} />{:else}{i + 1}{/if}</span>
			{label}
		</li>
	{/each}
</ol>

<style>
	ol {
		list-style: none;
		margin: 0;
		padding: 0 40px;
		display: grid;
		grid-auto-flow: column;
		grid-auto-columns: 1fr;
		border-bottom: 1px solid var(--line);
		flex-shrink: 0;
	}
	li {
		height: 64px;
		display: flex;
		align-items: center;
		gap: 12px;
		color: var(--faint);
		border-bottom: 2px solid transparent;
		margin-bottom: -1px;
	}
	li.now {
		color: var(--fg);
		font-weight: 600;
		border-bottom-color: var(--accent);
	}
	li.done {
		color: var(--muted);
	}
	.dot {
		width: 28px;
		height: 28px;
		border-radius: 50%;
		display: grid;
		place-items: center;
		font-size: var(--text-callout);
		font-weight: 600;
		border: 1.5px solid var(--border);
	}
	.now .dot {
		border: 0;
		background: var(--accent);
		color: var(--on-accent);
	}
	.done .dot {
		border: 0;
		background: var(--ok);
		color: var(--on-accent);
	}
</style>
