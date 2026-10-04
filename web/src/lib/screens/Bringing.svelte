<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- A3, A4, A5: Google preparing, the parts arriving, or paused. -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { Folder, CircleCheck, ArrowRight, Loader } from '@lucide/svelte';
	import { app, jobsKnown, shownJob } from '#lib/app.svelte.js';
	import { getLatest, getTakeouts, startGet, stopJob, type GridPhoto, type Takeout } from '#lib/api.js';
	import { onMount } from 'svelte';
	import { bytes, roughly } from '#lib/format.js';
	import { fraction, headline } from '#lib/job.js';
	import { text } from '#lib/strings.js';
	import PrimaryButton from '#lib/components/PrimaryButton.svelte';
	import SecondaryButton from '#lib/components/SecondaryButton.svelte';
	import Avatar from '#lib/components/Avatar.svelte';

	const shown = $derived(shownJob());
	// No download to show, once that is known: the page to start one.
	$effect(() => {
		if (jobsKnown() && !shown) goto('/choose', { replaceState: true });
	});
	const job = $derived(shown?.job);
	const waiting = $derived(job?.stage === 'waiting');
	const paused = $derived(job && !job.running && !job.finished && !job.error && job.stage);
	const title = $derived(
		job?.finished ? text.finishedTitle : paused ? text.pausedTitle : waiting ? text.preparingTitle : text.bringingTitle
	);
	const lead = $derived(paused ? text.pausedLead : waiting ? text.preparingLead : text.bringingLead);
	const email = $derived(app.overview?.accounts.find((a) => a.id === shown?.account)?.email ?? '');

	// The takeout as Google lists it: when it was asked, until when it is kept.
	let takeout = $state<Takeout | null>(null);
	$effect(() => {
		const id = job?.export;
		const account = shown?.account;
		if (!id || !account || takeout?.id && id.startsWith(takeout.id)) return;
		getTakeouts(account).then((l) => (takeout = l.find((t) => id.startsWith(t.id)) ?? null));
	});

	// The photos that arrived last, refreshed while they arrive.
	let latest = $state<GridPhoto[]>([]);
	onMount(() => {
		const load = () => getLatest().then((l) => (latest = l), () => {});
		load();
		const t = setInterval(load, 5000);
		return () => clearInterval(t);
	});

	const clock = new Intl.DateTimeFormat('en-GB', { hour: '2-digit', minute: '2-digit' });
	const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
	const dayOf = (d: Date) => `${d.getDate()} ${months[d.getMonth()]}`;
	let now = $state(Date.now());
	onMount(() => {
		const t = setInterval(() => (now = Date.now()), 30000);
		return () => clearInterval(t);
	});
	const sameDay = (d: Date) => new Date(now).toDateString() === d.toDateString();

	const left = $derived(
		shown && app.left[shown.account] ? text.aboutTime(roughly(app.left[shown.account])) : ''
	);
</script>

<div class="page">
	{#if !shown || !job}
		<!-- Known within a second: a download, or the page to start one. -->
	{:else}
		<div class="left">
			<span class="pill" class:err={job.error} class:live={job.running}><i></i>{headline(job)}</span>
			<h1>{title}</h1>
			<p class="lead">{job.error || lead}</p>
			{#if email}<span class="who"><Avatar account={{ id: shown.account, name: email, email, shown: true }} />{email}</span>{/if}
			{#if waiting}
				<div class="numbers">
					{#if takeout}
						{@const asked = new Date(takeout.Created)}
						<div><b>{sameDay(asked) ? clock.format(asked) : dayOf(asked)}</b><span>{sameDay(asked) ? text.askedToday : text.asked}</span></div>
					{/if}
					{#if job.lastLooked}
						<div><b>{text.minAgo(Math.floor((now - new Date(job.lastLooked).getTime()) / 60000))}</b><span>{text.lastLooked}</span></div>
					{/if}
					<div><b>{text.sevenDays}</b><span>{text.keepsOnceReady}</span></div>
				</div>
			{:else if job.total > 0}
				<div class="track"><b style:width="{fraction(job) * 100}%"></b></div>
				<div class="numbers">
					<div><b>{text.gbOf(bytes(job.done), bytes(job.total))}</b><span>{text.downloaded}</span></div>
					<div><b>{(app.overview?.total ?? 0).toLocaleString('en')}</b><span>{text.inFolder}</span></div>
					{#if left && !paused}<div><b>{left}</b><span>{text.leftAtSpeed}</span></div>{/if}
					{#if paused && takeout?.Expires}<div><b>{dayOf(new Date(takeout.Expires))}</b><span>{text.keepsUntil}</span></div>{/if}
				</div>
			{/if}
			<div class="actions">
				{#if job.running}
					<SecondaryButton onclick={() => stopJob(shown.account)}>{text.pause}</SecondaryButton>
				{:else if !job.finished}
					<PrimaryButton onclick={() => startGet({ account: shown.account, year: job.year })}>{text.resume}</PrimaryButton>
				{/if}
			</div>
			{#if !waiting && latest.length > 0}
				<div class="strip">
					<div class="striphead">
						<span>{paused ? text.lastArrived : text.firstArrived}</span>
						<button class="link" onclick={() => goto('/library')}>{text.seeLibrary}<ArrowRight size={14} /></button>
					</div>
					<div class="tiles">
						{#each latest as p (p.hash)}<img src="/api/thumb/{p.hash}" alt={p.name} />{/each}
					</div>
				</div>
			{/if}
		</div>
		{#if waiting}
			<section class="card">
				<h2>{text.canClose}</h2>
				<p>{text.canCloseDesc}</p>
				<ol>
					<li class="now"><Loader size={16} />{text.stepPrepares}</li>
					<li><Folder size={16} />{text.stepDownloads}</li>
					<li><CircleCheck size={16} />{text.stepLands}</li>
					<li><CircleCheck size={16} />{text.stepNotify}</li>
				</ol>
				<a class="link" href="https://takeout.google.com/manage" target="_blank" rel="noreferrer">{text.seeOnTakeout}<ArrowRight size={14} /></a>
			</section>
		{/if}
		{#if !waiting && (job.years.length > 0 || job.unassigned > 0)}
			<section class="years">
				<header><h2>{text.byYear}</h2><span>{text.byYearLegend}</span></header>
				{#each job.years as y (y.year)}
					{@const done = y.of > 0 && y.arrived >= y.of}
					<div class="year">
						<span class="folder"><Folder size={18} /></span>
						<span class="name">{y.year || text.noDate}</span>
						<span class="count" class:done>{y.arrived.toLocaleString('en')} / {y.of.toLocaleString('en')}</span>
						<span class="state">
							{#if done}<CircleCheck size={18} />{:else}<span class="bar"><b style:width="{y.of ? (y.arrived / y.of) * 100 : 0}%"></b></span>{/if}
						</span>
					</div>
				{/each}
				{#if job.unassigned > 0}
					<div class="year sorting">
						<span class="folder"><Folder size={18} /></span>
						<span class="name">{text.stillSorting}<small>{text.stillSortingDesc}</small></span>
						<span class="count">{job.unassigned.toLocaleString('en')}</span>
						<span class="state arriving"><Loader size={18} /></span>
					</div>
				{/if}
			</section>
		{/if}
	{/if}
</div>

<style>
	.page {
		height: 100%;
		display: flex;
		gap: 40px;
		padding: 64px 40px 40px;
		align-items: flex-start;
	}
	.left {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 16px;
		align-items: flex-start;
	}
	.pill {
		height: 28px;
		padding: 0 12px;
		display: inline-flex;
		align-items: center;
		gap: 8px;
		border-radius: 999px;
		background: var(--accent-soft);
		color: var(--accent);
		font-size: var(--text-callout);
		font-weight: 600;
	}
	.pill i {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: currentColor;
	}
	/* While the engine works the dot breathes: the screen is alive. */
	.pill.live i {
		animation: pulse 1.6s ease-in-out infinite;
	}
	@keyframes pulse {
		50% {
			opacity: 0.25;
		}
	}
	.pill.err {
		background: var(--err-soft);
		color: var(--err);
	}
	h1 {
		font-size: var(--text-hero);
		font-weight: 700;
		letter-spacing: -2px;
		line-height: 1.05;
		color: var(--fg);
		white-space: pre-line;
	}
	.lead {
		font-size: var(--text-title-2);
		line-height: 1.5;
		color: var(--muted);
		max-width: 560px;
	}
	.who {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		padding: 4px 12px 4px 4px;
		border: 1px solid var(--border);
		border-radius: 999px;
		color: var(--fg);
		font-weight: 600;
	}
	.track {
		width: 100%;
		height: 8px;
		border-radius: 999px;
		background: var(--line);
		overflow: hidden;
	}
	.track b,
	.bar b {
		display: block;
		height: 100%;
		border-radius: 999px;
		background: var(--accent);
	}
	.numbers {
		display: flex;
		gap: 40px;
	}
	.numbers b {
		display: block;
		font-size: var(--text-title-1);
		font-weight: 700;
		color: var(--fg);
	}
	.numbers span {
		font-size: var(--text-callout);
		color: var(--muted);
	}
	.actions {
		display: flex;
		align-items: center;
		gap: 24px;
	}
	.link {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		color: var(--accent);
		font-weight: 600;
		text-decoration: none;
		font-size: var(--text-callout);
	}
	.strip {
		width: 100%;
		display: flex;
		flex-direction: column;
		gap: 12px;
		margin-top: 8px;
	}
	.striphead {
		display: flex;
		justify-content: space-between;
		font-size: var(--text-callout);
		color: var(--muted);
	}
	.tiles {
		display: grid;
		grid-template-columns: repeat(6, 1fr);
		gap: 8px;
	}
	.tiles img {
		width: 100%;
		aspect-ratio: 1;
		object-fit: cover;
		border-radius: 6px;
	}
	.card {
		width: 380px;
		flex-shrink: 0;
		display: flex;
		flex-direction: column;
		gap: 12px;
		padding: 24px;
		border: 1px solid var(--border);
		border-radius: 16px;
		background: var(--soft);
	}
	.card h2 {
		font-size: var(--text-title-2);
		font-weight: 700;
		color: var(--fg);
	}
	.card p {
		font-size: var(--text-callout);
		line-height: 1.5;
		color: var(--muted);
	}
	.card ol {
		list-style: none;
		padding: 0;
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.card li {
		display: flex;
		align-items: center;
		gap: 10px;
		color: var(--muted);
		font-size: var(--text-body);
	}
	.card li.now {
		color: var(--fg);
		font-weight: 600;
	}
	.card li.now :global(svg) {
		color: var(--accent);
	}
	.sorting .name {
		display: flex;
		flex-direction: column;
	}
	.sorting small {
		font-size: var(--text-callout);
		color: var(--faint);
		font-weight: 400;
	}
	.state.arriving {
		color: var(--accent);
	}
	.years {
		width: 380px;
		flex-shrink: 0;
		max-height: 100%;
		overflow-y: auto;
		padding: 24px;
		border: 1px solid var(--border);
		border-radius: 16px;
	}
	.years header {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		padding-bottom: 12px;
	}
	.years h2 {
		font-size: var(--text-title-2);
		font-weight: 700;
		color: var(--fg);
	}
	.years header span {
		font-size: var(--text-callout);
		color: var(--faint);
	}
	.year {
		height: 44px;
		display: flex;
		align-items: center;
		gap: 12px;
		border-bottom: 1px solid var(--line);
	}
	.folder {
		display: grid;
		color: var(--folder);
	}
	.name {
		flex: 1;
		font-weight: 500;
		color: var(--fg);
	}
	.count {
		color: var(--text);
	}
	.count.done {
		color: var(--ok);
		font-weight: 600;
	}
	.state {
		width: 40px;
		display: flex;
		justify-content: flex-end;
		color: var(--ok);
	}
	.bar {
		width: 40px;
		height: 4px;
		border-radius: 999px;
		background: var(--line);
		overflow: hidden;
	}
</style>
