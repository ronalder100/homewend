<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- B7: one photo, over the grid. ← and → move, Esc or Space closes. -->
<script lang="ts">
	import { X, ChevronLeft, ChevronRight, FolderOpen } from '@lucide/svelte';
	import { fade } from 'svelte/transition';
	import { getAbout, pathOf, type GridPhoto } from '#lib/api.js';
	import { text } from '#lib/strings.js';
	import { app } from '#lib/app.svelte.js';

	let {
		photos,
		index = $bindable()
	}: { photos: GridPhoto[]; index: number | null } = $props();

	const p = $derived(index === null ? null : photos[index]);
	// What Chromium shows by itself; anything else shows its thumbnail.
	const ext = $derived(p ? p.name.toLowerCase().split('.').pop() ?? '' : '');
	const image = $derived(['jpg', 'jpeg', 'png', 'gif', 'webp'].includes(ext));
	const video = $derived(['mp4', 'webm', 'm4v'].includes(ext));
	// "Taken 14 July 2019, 18:32", as the design writes it.
	const day = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'long', year: 'numeric', timeZone: 'UTC' });
	const time = new Intl.DateTimeFormat('en-GB', { hour: '2-digit', minute: '2-digit', timeZone: 'UTC' });
	const taken = (iso: string) => text.taken(`${day.format(new Date(iso))}, ${time.format(new Date(iso))}`);

	// "IMG_4471.HEIC · Greece 2019 · James Dean": the file, its albums, whose.
	let about = $state('');
	$effect(() => {
		const hash = p?.hash;
		about = '';
		if (hash)
			getAbout(hash).then((a) => {
				if (p?.hash !== hash) return;
				// Sources are account ids; the window names an account by its address.
				const email = (id: string) => app.overview?.accounts.find((x) => x.id === id)?.email ?? id;
				about = [...a.albums, ...a.accounts.map(email)].map((x) => ` · ${x}`).join('');
			});
	});

	const move = (d: number) => {
		if (index !== null) index = Math.max(0, Math.min(photos.length - 1, index + d));
	};
	async function reveal() {
		if (p) window.shell?.showInFolder((await pathOf(p.hash)).path);
	}
	function key(e: KeyboardEvent) {
		if (index === null) return;
		// Space closes it, as it closes Quick Look.
		if (e.key === 'Escape' || e.key === ' ') {
			e.preventDefault();
			index = null;
		}
		if (e.key === 'ArrowLeft') move(-1);
		if (e.key === 'ArrowRight') move(1);
	}
</script>

<svelte:window onkeydown={key} />

{#if p}
	<div class="viewer" transition:fade={{ duration: 150 }}>
		<header>
			<button class="round" aria-label={text.close} onclick={() => (index = null)}><X size={18} /></button>
			<div class="meta">
				<b>{p.taken ? taken(p.taken) : text.noDate}</b>
				<span>{p.name}{about}</span>
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
		/* The photo takes the whole window, titlebar included. */
		inset: 0;
		z-index: 30;
		display: flex;
		flex-direction: column;
		background: #0b0c10;
		color: #fff;
	}
	header {
		display: flex;
		align-items: center;
		gap: 16px;
		padding: 12px 16px;
		/* Clear of the window buttons the system draws over the top right. */
		padding-right: calc(100vw - env(titlebar-area-x, 0px) - env(titlebar-area-width, 100vw) + 16px);
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
