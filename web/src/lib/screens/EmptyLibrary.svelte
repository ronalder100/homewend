<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- Before the first photo arrives: Apple Photos' empty state. -->
<script lang="ts">
	import { Images } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { text } from '#lib/strings.js';
	import { shownJob } from '#lib/app.svelte.js';
	import PrimaryButton from '#lib/components/PrimaryButton.svelte';

	const job = $derived(shownJob());
</script>

<div class="empty">
	<Images size={40} strokeWidth={1.5} />
	<h2>{text.noPhotos}</h2>
	<p>{text.noPhotosDesc}</p>
	{#if job}
		<button class="link" onclick={() => goto('/bringing')}>{text.seeDownload}</button>
	{:else}
		<span class="go"><PrimaryButton onclick={() => goto('/choose')}>{text.bringHome}</PrimaryButton></span>
	{/if}
</div>

<style>
	.empty {
		height: 100%;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 8px;
		padding: 40px;
		text-align: center;
		color: var(--faint);
	}
	h2 {
		font-size: var(--text-title-2);
		font-weight: 600;
		color: var(--muted);
	}
	p {
		color: var(--muted);
	}
	.link {
		color: var(--accent);
		font-weight: 600;
	}
	.go {
		margin-top: 16px;
	}
</style>
