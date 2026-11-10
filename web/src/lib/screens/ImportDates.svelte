<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- I3b: the photos a reset clock dated wrong, found by the day their camera
     says, ticked, and given the day they were taken. Nothing is written
     here: the dates go with the copy, in a .xmp beside each photo. -->
<script lang="ts">
	import { Check, ChevronRight, Film } from '@lucide/svelte';
	import { importThumbUrl, type ImportFile, type Redate } from '#lib/api.js';
	import { text } from '#lib/strings.js';
	import PrimaryButton from '#lib/components/PrimaryButton.svelte';
	import SecondaryButton from '#lib/components/SecondaryButton.svelte';

	let {
		files,
		redates,
		onuse,
		onback
	}: {
		files: ImportFile[];
		redates: Redate[];
		/** The picked photos were taken at when, a clock written as UTC. */
		onuse: (names: string[], when: string) => void;
		onback: () => void;
	} = $props();

	// The camera's clock is kept as UTC: read in UTC it is the clock itself.
	const utc = (iso: string) => new Date(iso);
	const dayKey = (f: ImportFile) => (f.taken ? f.taken.slice(0, 10) : '');
	const dayFormat = new Intl.DateTimeFormat('en-GB', { timeZone: 'UTC', day: 'numeric', month: 'long', year: 'numeric' });
	const timeFormat = new Intl.DateTimeFormat('en-GB', { timeZone: 'UTC', hour: '2-digit', minute: '2-digit' });

	// The days the camera says, oldest first; the files with no date last.
	const days = $derived.by(() => {
		const by = new Map<string, ImportFile[]>();
		for (const f of files) {
			const list = by.get(dayKey(f));
			if (list) list.push(f);
			else by.set(dayKey(f), [f]);
		}
		return [...by.entries()].sort(([a], [b]) => (a === '' ? 1 : b === '' ? -1 : a.localeCompare(b)));
	});
	// The date given to each file already, by its path.
	const given = $derived(new Map(redates.flatMap((r) => r.names.map((n) => [n, r.when] as const))));

	let picked = $state(new Set<string>());
	let opened = $state(new Set<string>());
	const flip = (set: Set<string>, keys: string[], on: boolean) => {
		const next = new Set(set);
		for (const k of keys) {
			if (on) next.add(k);
			else next.delete(k);
		}
		return next;
	};

	const chosen = $derived(files.filter((f) => picked.has(f.path)));
	const clocks = $derived(chosen.filter((f) => f.taken).map((f) => utc(f.taken!).getTime()).sort((a, b) => a - b));
	const first = $derived(clocks.length ? new Date(clocks[0]) : null);
	const last = $derived(clocks.length ? new Date(clocks[clocks.length - 1]) : null);

	// The date and time given, as the date-time field writes it: the first
	// picked photo's own until changed.
	let when = $state('');
	$effect(() => {
		// Nothing picked: the next photos picked bring their own time.
		if (chosen.length === 0) when = '';
		else if (!when && first) when = first.toISOString().slice(0, 16);
	});
	const whenDate = $derived(when ? new Date(when + ':00Z') : null);
	const goesTo = $derived(
		whenDate ? `${whenDate.getUTCFullYear()}/${String(whenDate.getUTCMonth() + 1).padStart(2, '0')}` : ''
	);
	const said = (d: Date) => `${dayFormat.format(d)}, ${timeFormat.format(d)}`;

	function use() {
		if (!whenDate || chosen.length === 0) return;
		onuse(
			chosen.map((f) => f.path),
			whenDate.toISOString().replace('.000Z', 'Z')
		);
		picked = new Set();
		when = '';
	}
</script>

<div class="dates">
	<button class="back" onclick={onback}><ChevronRight size={14} />{text.backToPlan}</button>
	<h1>{text.anyDatesWrong}</h1>
	<p class="lead">{text.anyDatesWrongLead}</p>
	<div class="split">
		<section class="days">
			{#each days as [day, list] (day)}
				{@const all = list.every((f) => picked.has(f.path))}
				{@const open = opened.has(day)}
				<div class="day" class:open>
					<input
						type="checkbox"
						checked={all}
						indeterminate={!all && list.some((f) => picked.has(f.path))}
						onchange={(e) => (picked = flip(picked, list.map((f) => f.path), e.currentTarget.checked))}
						aria-label={day ? dayFormat.format(utc(day)) : text.noDate}
					/>
					<button class="name" onclick={() => (opened = flip(opened, [day], !open))}>
						<ChevronRight size={16} />{day ? dayFormat.format(utc(day)) : text.noDate}
					</button>
					<span class="count">{list.length.toLocaleString('en')}</span>
				</div>
				{#if open}
					<div class="tiles">
						{#each list as f (f.path)}
							{@const on = picked.has(f.path)}
							<button class="tile" class:on title={f.path} onclick={() => (picked = flip(picked, [f.path], !on))}>
								{#if f.video}<span class="video"><Film size={20} /></span>{:else}<img src={importThumbUrl(f.path)} alt={f.path} loading="lazy" />{/if}
								{#if on}<span class="tick"><Check size={14} /></span>{/if}
								{#if given.has(f.path)}<span class="given">{dayFormat.format(utc(given.get(f.path)!))}</span>{/if}
							</button>
						{/each}
					</div>
				{/if}
			{/each}
		</section>
		{#if chosen.length > 0}
			<aside class="panel">
				<b class="picked">{text.photosPicked(chosen.length)}</b>
				{#if first && last}
					<span class="label">{text.cameraSays}</span>
					<span>{first.getTime() === last.getTime() ? said(first) : `${said(first)} – ${timeFormat.format(last)}`}</span>
				{/if}
				<label class="label" for="when">{text.takenOn}</label>
				<input id="when" type="datetime-local" bind:value={when} />
				{#if goesTo}
					<p class="where">
						{text.goInto}
						<b>{goesTo}</b>{#if first && last && Math.round((last.getTime() - first.getTime()) / 1000) > 0}{text.keepingBetween(Math.round((last.getTime() - first.getTime()) / 1000))}{/if}.
						{text.xmpBeside(chosen[0].path.split('/').pop() ?? '')}
					</p>
				{/if}
				<div class="actions">
					<PrimaryButton disabled={!whenDate} onclick={use}>{text.useThisDate}</PrimaryButton>
					<SecondaryButton onclick={() => ((picked = new Set()), (when = ''))}>{text.cancel}</SecondaryButton>
				</div>
			</aside>
		{/if}
	</div>
</div>

<style>
	.dates {
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
	h1 {
		font-size: var(--text-title-1);
		font-weight: 700;
		color: var(--fg);
	}
	.lead {
		max-width: 560px;
		color: var(--muted);
	}
	.split {
		display: flex;
		flex-wrap: wrap;
		gap: 24px;
		align-items: flex-start;
	}
	.days {
		flex: 1 1 360px;
		min-width: 0;
	}
	.day {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 0;
		border-bottom: 1px solid var(--line);
	}
	.day input {
		accent-color: var(--accent);
	}
	.day .name {
		flex: 1;
		display: inline-flex;
		align-items: center;
		gap: 4px;
		text-align: left;
		color: var(--fg);
		font-weight: 600;
	}
	.day.open .name :global(svg) {
		transform: rotate(90deg);
	}
	.count {
		color: var(--muted);
		font-variant-numeric: tabular-nums;
	}
	.tiles {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(96px, 1fr));
		gap: 4px;
		padding: 8px 0;
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
	.given {
		position: absolute;
		left: 0;
		right: 0;
		bottom: 0;
		padding: 2px 4px;
		background: #000000a0;
		color: #fff;
		font-size: var(--text-footnote);
	}
	.panel {
		flex: 0 1 320px;
		display: flex;
		flex-direction: column;
		gap: 8px;
		padding: 20px;
		border: 1px solid var(--border);
		border-radius: 12px;
		background: var(--surface);
		position: sticky;
		top: 0;
	}
	.picked {
		font-size: var(--text-title-2);
		color: var(--fg);
	}
	.label {
		margin-top: 8px;
		font-size: var(--text-callout);
		font-weight: 600;
		color: var(--muted);
	}
	.panel input {
		height: 36px;
		padding: 0 8px;
		border: 1px solid var(--border);
		border-radius: 8px;
		background: var(--bg);
		color: var(--fg);
	}
	.where {
		color: var(--muted);
		font-size: var(--text-callout);
		line-height: 1.5;
	}
	.where b {
		color: var(--fg);
	}
	.actions {
		display: flex;
		gap: 12px;
		margin-top: 8px;
	}
</style>
