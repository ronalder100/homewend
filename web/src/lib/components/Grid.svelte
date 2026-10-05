<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- The photos, by month, newest first. Only the rows on screen exist: the
     grid has to scroll through hundreds of thousands of tiles. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Play } from '@lucide/svelte';
	import { pathOf, type GridPhoto } from '#lib/api.js';
	import ContextMenu, { folderItems } from './ContextMenu.svelte';
	import { text } from '#lib/strings.js';

	let {
		photos,
		header,
		onopen,
		withYear = true
	}: {
		photos: GridPhoto[];
		header?: Snippet;
		onopen?: (index: number) => void;
		/** On a year's page the months need no year. */
		withYear?: boolean;
	} = $props();

	// The design's tile: at least 104px, as many columns as fit, gap 8.
	const MIN_TILE = 104;
	const GAP = 8;
	const MONTH = 40; // a month's heading, with the space above it
	const OVERSCAN = 600; // pixels drawn beyond the screen, both ways

	let width = $state(0);
	let height = $state(0);
	let scrollTop = $state(0);
	let headerHeight = $state(0);

	// A photo's menu, at the end of its tile: its folder, its path.
	let menu = $state<{ path: string; anchor: Element } | null>(null);
	async function context(e: MouseEvent, hash: string) {
		e.preventDefault();
		const anchor = e.currentTarget as HTMLElement;
		const { path } = await pathOf(hash);
		menu = { path, anchor };
	}

	const cols = $derived(Math.max(1, Math.floor((width + GAP) / (MIN_TILE + GAP))));
	const tile = $derived(width > 0 ? (width - GAP * (cols - 1)) / cols : MIN_TILE);

	type Row =
		| { kind: 'month'; label: string; count: number; top: number }
		| { kind: 'photos'; items: GridPhoto[]; top: number };

	const monthName = $derived(
		new Intl.DateTimeFormat('en', withYear ? { month: 'long', year: 'numeric', timeZone: 'UTC' } : { month: 'long', timeZone: 'UTC' })
	);

	function monthOf(p: GridPhoto): string {
		return p.taken ? monthName.format(new Date(p.taken)) : text.noDate;
	}

	const layout = $derived.by(() => {
		const rows: Row[] = [];
		let top = 0;
		let i = 0;
		while (i < photos.length) {
			const label = monthOf(photos[i]);
			let j = i;
			while (j < photos.length && monthOf(photos[j]) === label) j++;
			rows.push({ kind: 'month', label, count: j - i, top });
			top += MONTH;
			for (let k = i; k < j; k += cols) {
				rows.push({ kind: 'photos', items: photos.slice(k, Math.min(k + cols, j)), top });
				top += tile + GAP;
			}
			i = j;
		}
		return { rows, total: top };
	});

	// The rows on screen: found by halving, not by walking the list.
	const visible = $derived.by(() => {
		const rows = layout.rows;
		const from = scrollTop - headerHeight - OVERSCAN;
		const to = scrollTop - headerHeight + height + OVERSCAN;
		let lo = 0;
		let hi = rows.length;
		while (lo < hi) {
			const mid = (lo + hi) >> 1;
			if (rows[mid].top + MONTH + tile < from) lo = mid + 1;
			else hi = mid;
		}
		const out: Row[] = [];
		for (let r = lo; r < rows.length && rows[r].top < to; r++) out.push(rows[r]);
		return out;
	});
</script>

<div
	class="scroller"
	bind:clientHeight={height}
	onscroll={(e) => (scrollTop = (e.currentTarget as HTMLElement).scrollTop)}
>
	<div class="head" bind:clientHeight={headerHeight}>{@render header?.()}</div>
	<div class="grid" bind:clientWidth={width} style:height="{layout.total}px">
		{#each visible as row (row.top)}
			{#if row.kind === 'month'}
				<h3 style:transform="translateY({row.top}px)">
					{row.label} <span>{row.count.toLocaleString('en')}</span>
				</h3>
			{:else}
				{#each row.items as p, c (p.hash)}
					<button
						class="tile"
						style:width="{tile}px"
						style:height="{tile}px"
						style:transform="translate({c * (tile + GAP)}px, {row.top}px)"
						onclick={() => onopen?.(photos.indexOf(p))}
						oncontextmenu={(e) => context(e, p.hash)}
					>
						<img
							src="/api/thumb/{p.hash}"
							alt={p.name}
							loading="lazy"
							decoding="async"
							onload={(e) => (e.currentTarget as HTMLElement).classList.add('in')}
						/>
						{#if p.video}<span class="video"><Play size={14} fill="currentColor" /></span>{/if}
					</button>
				{/each}
			{/if}
		{/each}
	</div>
</div>
{#if menu}<ContextMenu items={folderItems(menu.path, true)} anchor={menu.anchor} onclose={() => (menu = null)} />{/if}

<style>
	.scroller {
		height: 100%;
		overflow-y: auto;
		padding: 0 40px 40px;
	}
	.grid {
		position: relative;
	}
	h3,
	.tile {
		position: absolute;
		top: 0;
		left: 0;
	}
	h3 {
		height: 40px;
		padding-top: 16px;
		font-size: var(--text-body);
		font-weight: 600;
		color: var(--fg);
		white-space: nowrap;
	}
	h3 span {
		margin-left: 8px;
		font-size: var(--text-callout);
		font-weight: 400;
		color: var(--faint);
	}
	.tile {
		border-radius: 6px;
		overflow: hidden;
		background: var(--line);
	}
	img {
		width: 100%;
		height: 100%;
		object-fit: cover;
		display: block;
		opacity: 0;
		transition: opacity 160ms ease;
	}
	img:global(.in) {
		opacity: 1;
	}
	.video {
		position: absolute;
		right: 6px;
		bottom: 6px;
		color: #fff;
		filter: drop-shadow(0 1px 2px rgba(0, 0, 0, 0.5));
	}
	:global(.narrow) .scroller {
		padding: 0 24px 24px;
	}
</style>
