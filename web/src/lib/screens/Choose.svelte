<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- A2: everything, one year, or a takeout already made. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { app } from '#lib/app.svelte.js';
	import { getLibrary, putLibrary, startGet } from '#lib/api.js';
	import { text } from '#lib/strings.js';
	import PrimaryButton from '#lib/components/PrimaryButton.svelte';
	import SecondaryButton from '#lib/components/SecondaryButton.svelte';
	import Avatar from '#lib/components/Avatar.svelte';
	import { sidebarOf } from '#lib/app.svelte.js';

	type Kind = 'year' | 'everything' | 'asked';
	let kind = $state<Kind>('everything');
	// The last four years, newest first: what most people want to try.
	const thisYear = new Date().getFullYear();
	const years = [0, 1, 2, 3].map((n) => thisYear - n);
	// Any older year from a list: Google says which it has only when asked.
	const older = Array.from({ length: thisYear - 4 - 1990 + 1 }, (_, n) => thisYear - 4 - n);
	let year = $state(thisYear - 1);
	let dir = $state('');
	let failed = $state('');

	const accounts = $derived(app.overview ? sidebarOf(app.overview).accounts : []);
	let account = $state('');
	$effect(() => {
		if (!account && accounts.length > 0) account = accounts[0].id;
	});

	onMount(async () => (dir = (await getLibrary()).dir));

	async function change() {
		const picked = await window.shell?.chooseFolder(dir);
		if (picked) dir = (await putLibrary(picked)).dir;
	}

	async function start() {
		failed = '';
		if (kind === 'asked') return goto('/takeouts');
		if (!dir) return (failed = text.noLibraryYet);
		try {
			await startGet({ account, year: kind === 'year' ? year : 0 });
			goto('/bringing');
		} catch (e) {
			failed = String(e);
		}
	}
</script>

<div class="page">
	<h1>{text.chooseTitle}</h1>
	<p class="lead">{text.chooseLead}</p>

	{#if accounts.length > 1}
		<div class="accounts" role="radiogroup">
			{#each accounts as a, i (a.id)}
				<button role="radio" aria-checked={account === a.id} class:on={account === a.id} onclick={() => (account = a.id)}>
					<Avatar account={a} index={i} />
					<b>{a.email}</b>
				</button>
			{/each}
		</div>
	{/if}

	<div class="cards" role="radiogroup">
		<div role="radio" tabindex="0" aria-checked={kind === 'year'} class="card" class:on={kind === 'year'} onclick={() => (kind = 'year')} onkeydown={(e) => e.key === 'Enter' && (kind = 'year')}>
			<span class="radio"></span>
			<h2>{text.oneYear}</h2>
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
						onkeydown={(e) => e.key === 'Enter' && ((kind = 'year'), (year = y))}>{y}</span
					>
				{/each}
				<select
					class="chip"
					class:on={kind === 'year' && year < years[years.length - 1]}
					aria-label={text.olderYear}
					onclick={(e) => e.stopPropagation()}
					onchange={(e) => {
						kind = 'year';
						year = Number((e.currentTarget as HTMLSelectElement).value);
					}}
				>
					<option value="" selected={year >= years[years.length - 1]} disabled>{text.olderYear}</option>
					{#each older as y (y)}<option value={y} selected={year === y}>{y}</option>{/each}
				</select>
			</span>
		</div>
		<div role="radio" tabindex="0" aria-checked={kind === 'everything'} class="card" class:on={kind === 'everything'} onclick={() => (kind = 'everything')} onkeydown={(e) => e.key === 'Enter' && (kind = 'everything')}>
			<span class="radio"></span>
			<h2>{text.everything}</h2>
			<p>{text.everythingDesc}</p>
		</div>
		<div role="radio" tabindex="0" aria-checked={kind === 'asked'} class="card" class:on={kind === 'asked'} onclick={() => (kind = 'asked')} onkeydown={(e) => e.key === 'Enter' && (kind = 'asked')}>
			<span class="radio"></span>
			<h2>{text.oneAsked}</h2>
			<p>{text.oneAskedDesc}</p>
			<span class="link">{text.seeTakeouts}</span>
		</div>
	</div>

	<div class="dest">
		<span class="label">{text.photosGoTo}</span>
		<span class="path" title={dir}>{dir || text.noLibraryYet}</span>
		<SecondaryButton small onclick={change}>{dir ? text.change : text.chooseFolder}</SecondaryButton>
	</div>

	<div class="go">
		<PrimaryButton onclick={start}>
			{kind === 'year' ? text.startYear(year) : kind === 'asked' ? text.seeTakeouts : text.startEverything}
		</PrimaryButton>
		{#if failed}<span class="err">{failed}</span>{/if}
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
		background: var(--accent-soft);
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
		cursor: pointer;
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
		background: var(--accent-soft);
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
		cursor: pointer;
	}
	.chip.on {
		border-color: var(--accent);
		color: var(--accent);
		font-weight: 600;
	}
	.link {
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
	.label {
		color: var(--muted);
		white-space: nowrap;
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
