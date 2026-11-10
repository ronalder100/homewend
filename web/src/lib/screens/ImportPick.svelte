<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- I5, I5b: the photos inside a group that stays out, a project most often,
     picked one by one. The likely photos first; the rest, by kind, when
     asked. -->
<script lang="ts">
	import { untrack } from 'svelte';
	import { Check, ChevronRight, Film } from '@lucide/svelte';
	import { importThumbUrl, type ImportFile, type ImportGroup } from '#lib/api.js';
	import { text } from '#lib/strings.js';
	import PrimaryButton from '#lib/components/PrimaryButton.svelte';

	let {
		group,
		files,
		picked: before,
		onpick,
		onback
	}: {
		group: ImportGroup;
		files: ImportFile[];
		/** The paths picked before, of this group. */
		picked: string[];
		onpick: (paths: string[]) => void;
		onback: () => void;
	} = $props();

	// Picked from where the person left it; changed here only.
	let picked = $state(untrack(() => new Set(before)));
	let all = $state(false);
	const mine = $derived(files.filter((f) => f.group === group.folder));
	// A camera's date and a real size: a photo. No date but a real size:
	// maybe one. A few hundred pixels: an icon or a logo.
	const kinds = $derived([
		{ name: text.pickPhotos, files: mine.filter((f) => (f.taken || f.video) && !f.small) },
		{ name: text.pickMaybe, files: mine.filter((f) => !f.taken && !f.video && !f.small) },
		{ name: text.pickIcons, files: mine.filter((f) => f.small) }
	]);
	const shown = $derived(all ? kinds.filter((k) => k.files.length > 0) : kinds.slice(0, 1));
	function flip(path: string) {
		const next = new Set(picked);
		if (!next.delete(path)) next.add(path);
		picked = next;
	}
</script>

<div class="pick">
	<button class="back" onclick={onback}><ChevronRight size={14} />{text.backToPlan}</button>
	<span class="pill">{group.folder}</span>
	<h1>{text.whichAreYours}</h1>
	<p class="lead">{text.pickLead(group.folder, group.kind)}</p>
	{#each shown as k (k.name)}
		<section>
			<header><h2>{k.name}</h2><span>{k.files.length.toLocaleString('en')}</span></header>
			{#if k.files.length === 0}
				<p class="none">{text.pickNone}</p>
			{/if}
			<div class="tiles" class:icons={k.name === text.pickIcons}>
				{#each k.files as f (f.path)}
					{@const on = picked.has(f.path)}
					<button class="tile" class:on title={f.path} onclick={() => flip(f.path)}>
						{#if f.video}<span class="video"><Film size={20} /></span>{:else}<img src={importThumbUrl(f.path)} alt={f.path} loading="lazy" />{/if}
						{#if on}<span class="tick"><Check size={14} /></span>{/if}
					</button>
				{/each}
			</div>
		</section>
	{/each}
	<button class="link" onclick={() => (all = !all)}>{all ? text.pickOnlyPhotos : text.pickShowAll(mine.length)}</button>
	<div class="bar">
		<span><b>{text.nPicked(picked.size)}</b> {text.ofImagesIn(mine.length, group.folder)}</span>
		<PrimaryButton onclick={() => onpick([...picked])}>{text.bringInThese(picked.size)}</PrimaryButton>
	</div>
</div>

<style>
	.pick {
		width: 100%;
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	.back {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		align-self: flex-start;
		color: var(--accent);
		font-weight: 600;
	}
	.back :global(svg) {
		transform: rotate(180deg);
	}
	.pill {
		align-self: flex-start;
		height: 28px;
		padding: 0 12px;
		display: inline-flex;
		align-items: center;
		border-radius: 999px;
		background: var(--accent-soft);
		color: var(--accent);
		font-family: var(--font-mono);
		font-size: var(--text-callout);
	}
	h1 {
		font-size: var(--text-title-1);
		font-weight: 700;
		color: var(--fg);
	}
	.lead {
		max-width: 560px;
		color: var(--muted);
	}
	section header {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		padding-bottom: 8px;
		border-bottom: 1px solid var(--line);
	}
	h2 {
		font-size: var(--text-title-3);
		font-weight: 600;
		color: var(--fg);
	}
	section header span,
	.none {
		color: var(--muted);
	}
	.tiles {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
		gap: 4px;
		padding: 8px 0;
	}
	.tiles.icons {
		grid-template-columns: repeat(auto-fill, minmax(64px, 1fr));
	}
	.tile {
		position: relative;
		aspect-ratio: 1;
		border-radius: 6px;
		overflow: hidden;
		background: var(--soft);
	}
	.tile img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.icons .tile img {
		object-fit: contain;
		padding: 8px;
	}
	.tile.on {
		outline: 3px solid var(--accent);
		outline-offset: -3px;
	}
	.video {
		width: 100%;
		height: 100%;
		display: grid;
		place-items: center;
		color: var(--muted);
	}
	.tick {
		position: absolute;
		top: 6px;
		right: 6px;
		width: 22px;
		height: 22px;
		display: grid;
		place-items: center;
		border-radius: 50%;
		background: var(--accent);
		color: var(--on-accent);
	}
	.link {
		align-self: flex-start;
		color: var(--accent);
		font-weight: 600;
	}
	.bar {
		position: sticky;
		bottom: 0;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
		padding: 12px 0;
		background: var(--bg);
		border-top: 1px solid var(--line);
		color: var(--muted);
	}
	.bar b {
		color: var(--fg);
	}
</style>
