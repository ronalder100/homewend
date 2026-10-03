<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- B7: one photo, over the grid. ← and → move, Esc closes. -->
<script lang="ts">
	import { X, ChevronLeft, ChevronRight, FolderOpen } from '@lucide/svelte';
	import { pathOf, type GridPhoto } from '#lib/api.js';
	import { text } from '#lib/strings.js';

	let {
		photos,
		index = $bindable()
	}: { photos: GridPhoto[]; index: number | null } = $props();

	const p = $derived(index === null ? null : photos[index]);
	// What Chromium shows by itself; anything else shows its thumbnail.
	const ext = $derived(p ? p.name.toLowerCase().split('.').pop() ?? '' : '');
	const image = $derived(['jpg', 'jpeg', 'png', 'gif', 'webp'].includes(ext));
	const video = $derived(['mp4', 'webm', 'm4v'].includes(ext));
	const taken = new Intl.DateTimeFormat('en', {
		day: 'numeric',
		month: 'long',
		year: 'numeric',
		hour: '2-digit',
		minute: '2-digit',
		timeZone: 'UTC'
	});

	const move = (d: number) => {
		if (index !== null) index = Math.max(0, Math.min(photos.length - 1, index + d));
	};
	async function reveal() {
		if (p) window.shell?.showInFolder((await pathOf(p.hash)).path);
	}
	function key(e: KeyboardEvent) {
		if (index === null) return;
		if (e.key === 'Escape') index = null;
		if (e.key === 'ArrowLeft') move(-1);
		if (e.key === 'ArrowRight') move(1);
	}
</script>

<svelte:window onkeydown={key} />

{#if p}
	<div class="viewer">
		<header>
			<button class="round" aria-label={text.close} onclick={() => (index = null)}><X size={18} /></button>
			<div class="meta">
				<b>{p.taken ? taken.format(new Date(p.taken)) : text.noDate}</b>
				<span>{p.name}</span>
			</div>
			{#if window.shell}
				<button class="pill" onclick={reveal}><FolderOpen size={14} />{text.openInFolder}</button>
			{/if}
		</header>
		<div class="stage">
			{#key p.hash}
				{#if video}
					<!-- svelte-ignore a11y_media_has_caption -->
					<video src="/api/original/{p.hash}" controls autoplay></video>
				{:else}
					<img src="/api/{image ? 'original' : 'thumb'}/{p.hash}" alt={p.name} />
				{/if}
			{/key}
		</div>
		<button class="round nav left" aria-label={text.previous} onclick={() => move(-1)}><ChevronLeft size={20} /></button>
		<button class="round nav right" aria-label={text.next} onclick={() => move(1)}><ChevronRight size={20} /></button>
	</div>
{/if}

<style>
	.viewer {
		position: fixed;
		inset: 44px 0 0 0;
		z-index: 10;
		display: flex;
		flex-direction: column;
		background: #0b0c10f2;
		color: #fff;
	}
	header {
		display: flex;
		align-items: center;
		gap: 16px;
		padding: 12px 16px;
	}
	.meta {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
	}
	.meta b {
		font-size: var(--text-body);
	}
	.meta span {
		font-family: var(--font-mono);
		font-size: var(--text-subheadline);
		color: #ffffff99;
	}
	.round {
		width: 36px;
		height: 36px;
		border-radius: 50%;
		display: grid;
		place-items: center;
		background: #ffffff14;
		color: #fff;
	}
	.pill {
		height: 32px;
		padding: 0 12px;
		display: flex;
		align-items: center;
		gap: 6px;
		border-radius: 999px;
		background: #ffffff14;
		color: #fff;
		font-size: var(--text-callout);
		font-weight: 600;
	}
	/* The photo fits the space left, whole: never taller than the window. */
	.stage {
		flex: 1;
		min-height: 0;
		position: relative;
		margin: 0 72px 24px;
	}
	img,
	video {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		object-fit: contain;
	}
	.nav {
		position: absolute;
		top: 50%;
	}
	.left {
		left: 16px;
	}
	.right {
		right: 16px;
	}
</style>
