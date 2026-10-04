<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- D1's strip under the titlebar: an icon, what happened, and at most one
     action; its tone says how much it matters. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import { slide } from 'svelte/transition';

	let {
		tone = 'accent',
		icon,
		title,
		detail,
		action,
		onaction
	}: {
		tone?: 'accent' | 'ok' | 'warn' | 'err';
		icon: Snippet;
		title: string;
		detail: string;
		action?: string;
		onaction?: () => void;
	} = $props();
</script>

<div class="banner {tone}" transition:slide={{ duration: 180 }}>
	<span class="icon">{@render icon()}</span>
	<div class="text"><b>{title}</b><span>{detail}</span></div>
	{#if action}<button class="act" onclick={onaction}>{action}</button>{/if}
</div>

<style>
	.banner {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px 24px;
		border-bottom: 1px solid var(--line);
		background: var(--accent-soft);
		font-size: var(--text-body);
	}
	.err {
		background: var(--err-soft);
	}
	.warn {
		background: var(--warn-soft);
	}
	.ok {
		background: var(--ok-soft);
	}
	.icon {
		display: grid;
		color: var(--accent);
	}
	.err .icon {
		color: var(--err);
	}
	.warn .icon {
		color: var(--warn);
	}
	.ok .icon {
		color: var(--ok);
	}
	.text {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}
	b {
		color: var(--fg);
		font-weight: 600;
	}
	.text span {
		font-size: var(--text-callout);
		color: var(--muted);
	}
	.act {
		height: 32px;
		padding: 0 12px;
		border-radius: 8px;
		background: var(--accent);
		color: var(--on-accent);
		font-size: var(--text-callout);
		font-weight: 600;
		white-space: nowrap;
	}
</style>
