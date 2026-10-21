<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- A2: everything, one year, or a takeout already made. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { app, startFailure, startJob } from '#lib/app.svelte.js';
	import { countYears, getLibrary, getYears, putLibrary, type YearCounts } from '#lib/api.js';
	import { text } from '#lib/strings.js';
	import PrimaryButton from '#lib/components/PrimaryButton.svelte';
	import Avatar from '#lib/components/Avatar.svelte';
	import { Plus, Folder, ChevronDown, ArrowRight, Loader } from '@lucide/svelte';
	import { bytes } from '#lib/format.js';
	import { sidebarOf } from '#lib/app.svelte.js';

	type Kind = 'year' | 'everything' | 'asked';
	let kind = $state<Kind>('everything');
	// The last four years, newest first: what most people want to try.
	const thisYear = new Date().getFullYear();
	// The years the account has on Google, once counted; until then, or if
	// the count failed, every year is offered.
	let counts = $state<YearCounts | null>(null);
	const counted = $derived(counts?.years?.length ? counts.years : null);
	const itemsOf = (y: number) => counted?.find((c) => c.year === y)?.items;
	// Oldest first, as the design reads them: the three newest years.
	const years = $derived(
		counted ? counted.slice(0, 3).map((c) => c.year).reverse() : [3, 2, 1].map((n) => thisYear - n)
	);
	// Any other year from a list.
	const older = $derived(
		counted
			? counted.slice(3).map((c) => c.year)
			: [thisYear, ...Array.from({ length: thisYear - 4 - 1990 + 1 }, (_, n) => thisYear - 4 - n)]
	);
	let year = $state(thisYear - 1);
	let dir = $state('');
	let free = $state(0);
	let failed = $state('');

	const accounts = $derived(app.overview ? sidebarOf(app.overview).accounts : []);
	let account = $state('');
	// The account and year the address names: an account just added, a
	// takeout to ask for again. Without one, or once it is gone, the first.
	$effect(() => {
		const q = page.url.searchParams;
		const [a, y] = [q.get('account'), Number(q.get('year'))];
		if (a) account = a;
		if (y) [kind, year] = ['year', y];
	});
	$effect(() => {
		if (accounts.length > 0 && !accounts.some((a) => a.id === account)) account = accounts[0].id;
	});

	const running = $derived(app.jobs[account]?.running ?? false);

	// The account's count: read, asked for once if there is none, and looked
	// at every two seconds while it runs (about a minute).
	let countSince = $state(0);
	let tick = $state(Date.now());
	$effect(() => {
		const a = account;
		if (!a) return;
		counts = null;
		let asked = false;
		let stop = false;
		const look = async () => {
			try {
				const c = await getYears(a);
				if (stop) return;
				if (!c.years && !c.counting && !c.error && !asked) {
					asked = true;
					await countYears(a).catch(() => {});
					countSince = Date.now();
					return look();
				}
				if (c.counting && !countSince) countSince = Date.now();
				counts = c;
			} catch {
				// No count: every year is offered.
			}
		};
		look();
		const t = setInterval(() => {
			tick = Date.now();
			if (counts?.counting) look();
		}, 2000);
		return () => {
			stop = true;
			clearInterval(t);
		};
	});
	// A year the account does not have is not offered.
	$effect(() => {
		if (counted && !counted.some((c) => c.year === year)) year = counted[0].year;
	});

	onMount(async () => ({ dir, free } = await getLibrary()));

	async function change() {
		const picked = await window.shell?.chooseFolder(dir);
		if (picked) {
			await putLibrary(picked);
			({ dir, free } = await getLibrary());
		}
	}

	async function start() {
		failed = '';
		// A takeout from the list needs the folder too, and the list has no
		// place to choose it.
		if (!dir) return (failed = text.noLibraryYet);
		if (kind === 'asked') return goto('/takeouts');
		try {
			await startJob({ account, year: kind === 'year' ? year : 0 });
			goto('/bringing');
		} catch (e) {
			failed = startFailure(e);
		}
	}
</script>

<div class="page">
	<h1>{text.chooseTitle}</h1>
	<p class="lead">{text.chooseLead}</p>

	<div class="accounts" role="radiogroup">
		{#each accounts as a, i (a.id)}
			<button role="radio" aria-checked={account === a.id} class:on={account === a.id} onclick={() => (account = a.id)}>
				<Avatar account={a} index={i} />
				<b>{a.email}</b>
			</button>
		{/each}
		<button class="add" onclick={() => goto('/add')}><Plus size={14} />{text.addAccount}</button>
	</div>

	<div class="cards" role="radiogroup">
		<div role="radio" tabindex="0" aria-checked={kind === 'year'} class="card" class:on={kind === 'year'} onclick={() => (kind = 'year')} onkeydown={(e) => e.key === 'Enter' && (kind = 'year')}>
			<span class="title"><span class="radio"></span><h2>{text.oneYear}</h2></span>
			<p>{text.oneYearDesc}</p>
			<span class="chips">
				{#each years as y (y)}
					<span
						role="button"
						tabindex="0"
						class="chip"
						class:on={kind === 'year' && year === y}
						onclick={(e) => {
							e.stopPropagation();
							kind = 'year';
							year = y;
						}}
						onkeydown={(e) => e.key === 'Enter' && ((kind = 'year'), (year = y))}
						>{y}{#if itemsOf(y) !== undefined}<small>{(itemsOf(y) ?? 0).toLocaleString('en')}</small>{/if}</span
					>
				{/each}
				<select
					class="chip"
					class:on={kind === 'year' && !years.includes(year)}
					aria-label={text.olderYear}
					onclick={(e) => e.stopPropagation()}
					onchange={(e) => {
						kind = 'year';
						year = Number((e.currentTarget as HTMLSelectElement).value);
					}}
				>
					<option value="" selected={years.includes(year)} disabled>{text.olderYear}</option>
					{#each older as y (y)}<option value={y} selected={year === y}
							>{itemsOf(y) !== undefined ? text.yearWithCount(y, itemsOf(y) ?? 0) : y}</option
						>{/each}
				</select>
			</span>
			{#if counts?.counting}
				<span class="counting"
					><span class="spin"><Loader size={14} /></span>{text.countingPhotos} · {text.pleaseWait}
					{text.elapsed(Math.max(0, Math.floor((tick - countSince) / 1000)))}</span
				>
			{:else if counts?.error}
				<span class="counting">{text.countFailed}</span>
			{/if}
		</div>
		<div role="radio" tabindex="0" aria-checked={kind === 'everything'} class="card" class:on={kind === 'everything'} onclick={() => (kind = 'everything')} onkeydown={(e) => e.key === 'Enter' && (kind = 'everything')}>
			<span class="title"><span class="radio"></span><h2>{text.everything}</h2></span>
			<p>{text.everythingDesc}</p>
			{#if counted}<span class="total">{text.itemsCount(counted.reduce((n, c) => n + c.items, 0))}</span>{/if}
		</div>
		<div role="radio" tabindex="0" aria-checked={kind === 'asked'} class="card" class:on={kind === 'asked'} onclick={() => (kind = 'asked')} onkeydown={(e) => e.key === 'Enter' && (kind = 'asked')}>
			<span class="title"><span class="radio"></span><h2>{text.oneAsked}</h2></span>
			<p>{text.oneAskedDesc}</p>
			<span class="link">{text.seeTakeouts}<ArrowRight size={14} /></span>
		</div>
	</div>

	<div class="dest">
		<span class="folder"><Folder size={20} fill="currentColor" /></span>
		<span class="label">{text.photosGoTo}</span>
		<span class="path" title={dir}>{dir || text.noLibraryYet}</span>
		{#if dir && free}<span class="free">{text.free(bytes(free))}</span>{/if}
		<button class="change" onclick={change}>{dir ? text.change : text.chooseFolder}</button>
	</div>

	<div class="go">
		<!-- One download at a time for each account: the one running is shown
		     instead of a start the engine would refuse. -->
		{#if running}
			<PrimaryButton onclick={() => goto('/bringing')}>{text.seeDownload}</PrimaryButton>
			<span class="hint">{text.alreadyBringing}</span>
		{:else}
			<PrimaryButton onclick={start}>
				{kind === 'year' ? text.startYear(year) : kind === 'asked' ? text.seeTakeouts : text.startEverything}
			</PrimaryButton>
			{#if failed}<span class="err">{failed}</span>{:else if kind !== 'asked'}<span class="hint">{text.notifyWhenReady}</span>{/if}
		{/if}
	</div>
</div>

<style>
	.page {
		max-width: 1000px;
		margin: 0 auto;
		padding: 40px;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 24px;
		text-align: center;
	}
	h1 {
		font-size: var(--text-display);
		font-weight: 700;
		color: var(--fg);
	}
	.lead {
		margin-top: -12px;
		font-size: var(--text-title-2);
		color: var(--muted);
	}
	.accounts {
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: 8px;
	}
	.accounts button {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 6px 12px 6px 6px;
		border: 1px solid var(--border);
		border-radius: 999px;
		text-align: left;
	}
	.accounts button.on {
		border-color: var(--accent);
	}
	.accounts .add {
		border: 0;
		display: flex;
		align-items: center;
		gap: 6px;
		color: var(--accent);
		font-size: var(--text-callout);
		font-weight: 600;
	}
	.accounts b {
		display: block;
		font-size: var(--text-body);
		color: var(--fg);
	}
	.cards {
		width: 100%;
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 16px;
		text-align: left;
	}
	.card {
		cursor: default;
		position: relative;
		display: flex;
		flex-direction: column;
		gap: 8px;
		padding: 20px;
		border: 1px solid var(--border);
		border-radius: 16px;
		align-items: flex-start;
	}
	.card.on {
		border: 2px solid var(--accent);
		padding: 19px;
	}
	.title {
		display: flex;
		align-items: center;
		gap: 12px;
	}
	.radio {
		width: 16px;
		height: 16px;
		border-radius: 50%;
		border: 1.5px solid var(--faint);
	}
	.card.on .radio {
		border: 5px solid var(--accent);
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
	.chips {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
		margin-top: 4px;
	}
	.chip {
		height: 28px;
		padding: 0 10px;
		display: inline-flex;
		align-items: center;
		border: 1px solid var(--border);
		border-radius: 8px;
		font-size: var(--text-callout);
		color: var(--text);
	}
	select.chip {
		font: inherit;
		font-size: var(--text-callout);
		background: transparent;
		cursor: default;
	}
	.chip small {
		margin-left: 6px;
		font-size: var(--text-footnote);
		color: var(--faint);
	}
	.counting {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		font-size: var(--text-footnote);
		color: var(--muted);
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
	.total {
		font-size: var(--text-callout);
		font-weight: 600;
		color: var(--fg);
	}
	.chip.on {
		border-color: var(--accent);
		color: var(--accent);
		font-weight: 600;
	}
	.link {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		color: var(--accent);
		font-weight: 600;
		font-size: var(--text-callout);
	}
	.dest {
		width: 100%;
		display: flex;
		align-items: center;
		gap: 16px;
		padding: 12px 16px;
		border: 1px solid var(--border);
		border-radius: 12px;
		background: var(--soft);
		text-align: left;
	}
	.folder {
		display: grid;
		color: var(--folder);
	}
	.label {
		color: var(--muted);
		white-space: nowrap;
	}
	.free {
		color: var(--ok);
		font-size: var(--text-callout);
		white-space: nowrap;
	}
	.change {
		color: var(--accent);
		font-weight: 600;
		font-size: var(--text-callout);
		white-space: nowrap;
	}
	.hint {
		color: var(--faint);
		font-size: var(--text-callout);
	}
	.path {
		flex: 1;
		min-width: 0;
		font-family: var(--font-mono);
		font-size: var(--text-callout);
		color: var(--fg);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		user-select: text;
	}
	.go {
		display: flex;
		align-items: center;
		gap: 16px;
	}
	.err {
		color: var(--err);
	}
</style>
