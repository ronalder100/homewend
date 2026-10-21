<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- A2c, A2d: the years the account has on Google, how many photos each; one to bring home. -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { app, sidebarOf, startFailure, startJob } from '#lib/app.svelte.js';
	import { getLibrary, getTakeouts, type Takeout } from '#lib/api.js';
	import { Years } from '#lib/years.svelte.js';
	import { chooseUrl } from '#lib/places.js';
	import { text } from '#lib/strings.js';
	import PrimaryButton from '#lib/components/PrimaryButton.svelte';
	import SecondaryButton from '#lib/components/SecondaryButton.svelte';
	import { ArrowLeft, Folder, Loader, CircleCheck, CalendarX, CloudDownload } from '@lucide/svelte';

	// The account the address names; one that is gone, the first.
	const accounts = $derived(app.overview ? sidebarOf(app.overview).accounts : []);
	const account = $derived.by(() => {
		const a = page.url.searchParams.get('account') ?? '';
		return accounts.some((x) => x.id === a) ? a : (accounts[0]?.id ?? '');
	});
	const email = $derived(app.overview?.accounts.find((a) => a.id === account)?.email ?? '');
	const years = new Years(() => account);
	let year = $state(Number(page.url.searchParams.get('year')) || 0);
	let failed = $state('');

	// Without a count, every year from this one back: Google says which it
	// has only when asked.
	const thisYear = new Date().getFullYear();
	const list = $derived(
		years.counted ?? Array.from({ length: thisYear - 1990 + 1 }, (_, n) => ({ year: thisYear - n, items: -1, videos: 0 }))
	);
	// The year picked, or the one named, is one the account has; else none.
	$effect(() => {
		if (years.counted && !years.counted.some((c) => c.year === year)) year = 0;
	});
	const running = $derived(app.jobs[account]?.running ?? false);

	// The years already asked of Google, each as its last takeout stands: so
	// that one is not asked twice without knowing it.
	type Asked = { state: keyof typeof text.askedYear; day: string };
	let takeouts = $state<Takeout[]>([]);
	$effect(() => {
		const a = account;
		takeouts = [];
		if (a) getTakeouts(a).then((l) => a === account && (takeouts = l), () => {});
	});
	const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
	const asked = $derived.by(() => {
		const by = new Map<number, Asked>();
		// Newest first, as Google lists them: the first of a year is its last.
		for (const t of takeouts) {
			if (!t.known || !t.year || by.has(t.year)) continue;
			const l = t.local;
			const state =
				l && l.of > 0 && l.parts >= l.of
					? 'downloaded'
					: t.status === 'preparing'
						? 'preparing'
						: t.status === 'expired'
							? 'expired'
							: l && l.parts > 0
								? 'partly'
								: 'ready';
			const d = new Date(t.Created);
			by.set(t.year, { state, day: `${d.getDate()} ${months[d.getMonth()]}` });
		}
		return by;
	});

	// The year picked is in view, the one an address named included.
	function reveal(node: HTMLElement, on: boolean) {
		if (on) node.scrollIntoView({ block: 'nearest' });
		return { update: (now: boolean) => now && node.scrollIntoView({ block: 'nearest' }) };
	}

	async function start() {
		failed = '';
		if (!year) return;
		const { dir } = await getLibrary();
		if (!dir) return (failed = text.noLibraryYet);
		try {
			await startJob({ account, year });
			goto('/bringing');
		} catch (e) {
			failed = startFailure(e);
		}
	}
</script>

<div class="page">
	<h1>{text.whichYear}</h1>
	<p class="lead">
		{years.counted && email ? text.whichYearLead(email, years.counted.length) : text.whichYearLeadUnknown}
	</p>

	<div class="list" class:finding={years.finding}>
		{#if years.finding}
			<div class="wait">
				<span class="spin"><Loader size={28} /></span>
				<b>{text.findingYears}</b>
				<time>{text.pleaseWait} {text.elapsed(years.seconds)}</time>
				<p>{text.findingYearsWhy}</p>
			</div>
		{:else}
			<header><span>{text.yearsHead}</span><span>{years.counted ? text.yearsLegend : ''}</span></header>
			<div class="rows" role="radiogroup">
				{#each list as y (y.year)}
					<button role="radio" aria-checked={year === y.year} class:on={year === y.year} onclick={() => (year = y.year)} use:reveal={year === y.year}>
						<span class="radio"></span>
						<span class="folder"><Folder size={18} fill="currentColor" /></span>
						<span class="name">{y.year}</span>
						{#if asked.get(y.year)}
							{@const a = asked.get(y.year)!}
							<span class="asked {a.state}">
								{#if a.state === 'downloaded'}<CircleCheck size={12} />{:else if a.state === 'expired'}<CalendarX size={12} />{:else if a.state === 'preparing'}<Loader size={12} />{:else}<CloudDownload size={12} />{/if}
								{text.askedYear[a.state]} · {text.askedOn(a.day)}
							</span>
						{/if}
						<span class="videos">{y.videos > 0 ? text.videosCount(y.videos) : ''}</span>
						<span class="count">{y.items >= 0 ? y.items.toLocaleString('en') : ''}</span>
					</button>
				{/each}
			</div>
		{/if}
	</div>
	{#if years.counts?.error}<p class="note">{text.countFailed}</p>{/if}

	<div class="go">
		<SecondaryButton onclick={() => goto(chooseUrl(account))}><ArrowLeft size={16} />{text.back}</SecondaryButton>
		{#if running}
			<PrimaryButton onclick={() => goto('/bringing')}>{text.seeDownload}</PrimaryButton>
			<span class="hint">{text.alreadyBringing}</span>
		{:else}
			<PrimaryButton onclick={start}>{year ? text.startYear(year) : text.startTakeout}</PrimaryButton>
			{#if failed}<span class="err">{failed}</span>
			{:else if years.finding}<span class="hint">{text.chooseOnceHere}</span>
			{:else if year && years.itemsOf(year) !== undefined}<span class="hint"
					>{text.itemsCount(years.itemsOf(year) ?? 0)}. {text.notifyWhenReady}</span
				>
			{:else}<span class="hint">{text.notifyWhenReady}</span>{/if}
		{/if}
	</div>
</div>

<style>
	.page {
		height: 100%;
		max-width: 1000px;
		margin: 0 auto;
		padding: 40px;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 20px;
		text-align: center;
	}
	h1 {
		font-size: var(--text-display);
		font-weight: 700;
		color: var(--fg);
	}
	.lead {
		margin-top: -8px;
		max-width: 640px;
		font-size: var(--text-title-2);
		color: var(--muted);
	}
	.list {
		width: 640px;
		max-width: 100%;
		flex: 1;
		min-height: 240px;
		display: flex;
		flex-direction: column;
		padding: 16px 24px;
		border: 1px solid var(--border);
		border-radius: 16px;
		background: var(--surface);
		text-align: left;
		overflow: hidden;
	}
	.list.finding {
		justify-content: center;
		align-items: center;
	}
	.wait {
		max-width: 420px;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 10px;
		text-align: center;
	}
	.wait b {
		font-size: var(--text-title-3);
		color: var(--fg);
	}
	.wait time {
		font-family: var(--font-mono);
		color: var(--muted);
	}
	.wait p {
		color: var(--muted);
		line-height: 1.4;
	}
	.wait :global(svg) {
		color: var(--accent);
	}
	.spin {
		display: inline-flex;
		animation: spin 1s linear infinite;
	}
	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
	header {
		display: flex;
		justify-content: space-between;
		padding: 0 12px 10px;
		border-bottom: 1px solid var(--line);
		font-size: var(--text-callout);
		color: var(--faint);
	}
	.rows {
		flex: 1;
		overflow-y: auto;
	}
	.rows button {
		width: 100%;
		height: 44px;
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 0 12px;
		border-bottom: 1px solid var(--line);
		text-align: left;
	}
	.rows button.on {
		border-radius: 10px;
		border-bottom-color: transparent;
		background: var(--accent-soft);
	}
	.radio {
		flex: none;
		width: 16px;
		height: 16px;
		border-radius: 50%;
		border: 1.5px solid var(--faint);
	}
	.on .radio {
		border: 5px solid var(--accent);
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
	.on .name,
	.on .count {
		font-weight: 700;
		color: var(--accent);
	}
	.asked {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 3px 8px;
		border-radius: 999px;
		font-size: var(--text-footnote);
		font-weight: 600;
		white-space: nowrap;
		color: var(--accent);
		background: var(--accent-soft);
	}
	.asked.downloaded {
		color: var(--ok);
		background: var(--ok-soft);
	}
	.asked.expired {
		color: var(--muted);
		background: var(--soft);
	}
	.videos {
		width: 100px;
		text-align: right;
		font-size: var(--text-callout);
		color: var(--faint);
	}
	.count {
		width: 64px;
		text-align: right;
		color: var(--text);
		font-variant-numeric: tabular-nums;
	}
	.note {
		color: var(--muted);
		font-size: var(--text-callout);
	}
	/* Under the list's left edge, where the hint beside it cannot move it. */
	.go {
		width: 640px;
		max-width: 100%;
		display: flex;
		align-items: center;
		gap: 16px;
	}
	.hint {
		color: var(--faint);
		font-size: var(--text-callout);
	}
	.err {
		color: var(--err);
	}
</style>
