<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<script lang="ts">
	import { page } from '$app/state';
	import { app } from '#lib/app.svelte.js';
	import { getPhotos, type GridPhoto } from '#lib/api.js';
	import { placeOf } from '#lib/places.js';
	import { text } from '#lib/strings.js';
	import Grid from '#lib/components/Grid.svelte';
	import EmptyLibrary from '#lib/screens/EmptyLibrary.svelte';
	import Viewer from '#lib/components/Viewer.svelte';

	let open = $state<number | null>(null);

	const place = $derived(placeOf(page.url));
	const title = $derived(
		place.kind === 'year' ? place.label : place.kind === 'album' ? place.name : text.allPhotos
	);

	let photos = $state<GridPhoto[]>([]);
	let loaded = $state(false);

	// The whole list, a chunk at a time: the grid needs every photo's month to
	// lay itself out, and the first chunk shows while the rest arrives.
	const CHUNK = 5000;
	$effect(() => {
		const p = place;
		let stopped = false;
		photos = [];
		loaded = false;
		(async () => {
			const q =
				p.kind === 'year'
					? p.label === text.noDate
						? { noDate: true }
						: { year: p.label }
					: p.kind === 'album'
						? { album: p.name }
						: {};
			for (let offset = 0; !stopped; offset += CHUNK) {
				const chunk = await getPhotos({ ...q, offset, limit: CHUNK });
				if (stopped) return;
				photos = offset === 0 ? chunk : photos.concat(chunk);
				if (chunk.length < CHUNK) break;
			}
			loaded = true;
		})();
		return () => (stopped = true);
	});
</script>

{#if app.overview && app.overview.total === 0}
	<EmptyLibrary />
{:else}
	<Grid {photos} onopen={(i) => (open = i)}>
		{#snippet header()}
			<div class="head">
				<h1>{title}</h1>
				<p>{text.photosAndVideos(photos.length)}{loaded ? '' : '…'}</p>
			</div>
		{/snippet}
	</Grid>
	<Viewer {photos} bind:index={open} />
{/if}

<style>
	.head {
		padding: 32px 0 8px;
	}
	h1 {
		font-size: var(--text-display);
		font-weight: 700;
		color: var(--fg);
	}
	p {
		margin-top: 8px;
		font-size: var(--text-body);
		color: var(--muted);
	}
	:global(.narrow) h1 {
		font-size: var(--text-large-title);
	}
</style>
