<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- I1–I8: a card or a folder, what comes in, the copy, a card taken out, the end.
     Each title answers what a person wants to know there; the numbers that
     matter are the biggest thing on the screen. -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { CircleCheck, Folder, HardDrive, Loader, TriangleAlert } from '@lucide/svelte';
	import { app } from '#lib/app.svelte.js';
	import {
		ApiError,
		getCards,
		getImport,
		getLibrary,
		planImport,
		startImport,
		stopImport,
		type Card,
		type ImportState
	} from '#lib/api.js';
	import { bytes } from '#lib/format.js';
	import { text } from '#lib/strings.js';
	import PrimaryButton from '#lib/components/PrimaryButton.svelte';
	import SecondaryButton from '#lib/components/SecondaryButton.svelte';

	let im = $state<ImportState | null>(null);
	let cards = $state<Card[]>([]);
	let free = $state(0);
	// An import that ended before this window opened is not shown again: the
	// screen starts at the choice. One under way, or a card out, is.
	let choosing = $state(true);

	// Whose photos they are: the profiles of the accounts, or a name typed
	// when there is no account yet.
	const profiles = $derived([...new Set((app.overview?.accounts ?? []).map((a) => a.profile))].sort());
	let profile = $state('');
	$effect(() => {
		if (!profile && profiles.length > 0) profile = profiles[0];
	});

	const name = $derived(nameOf(im?.dir ?? ''));
	function nameOf(dir: string) {
		return cards.find((c) => c.path === dir)?.name || dir.split(/[\\/]/).filter(Boolean).pop() || dir;
	}
	const isCard = $derived(cards.some((c) => c.path === im?.dir));

	type View = 'choose' | 'reading' | 'plan' | 'copying' | 'taken-out' | 'done' | 'stopped' | 'failed';
	const view = $derived<View>(
		!im || (choosing && !im.running && im.problem !== 'taken-out')
			? 'choose'
			: im.running
				? im.kind === 'plan'
					? 'reading'
					: 'copying'
				: im.problem === 'taken-out'
					? 'taken-out'
					: im.kind === 'import' && im.imported && !im.error
						? im.stopped
							? 'stopped'
							: 'done'
						: im.kind === 'plan' && im.plan && !im.error
							? 'plan'
							: im.stopped
								? 'choose'
								: 'failed'
	);

	async function load() {
		try {
			im = await getImport();
		} catch {
			// The engine answers within a second; the next ask will do.
		}
	}
	async function lookCards() {
		cards = await getCards().catch(() => cards);
	}
	onMount(() => {
		load().then(() => {
			// A copy under way, or waiting for its card, is where the window opens.
			if (im?.running || im?.problem === 'taken-out') {
				choosing = false;
				if (im.profile) profile = im.profile;
			}
		});
		lookCards();
		getLibrary().then((l) => (free = l.free), () => {});
		const t = setInterval(() => {
			if (im?.running) load();
			// The cards come and go, during a copy too: the one taken out must
			// be seen gone before it can be seen back.
			lookCards();
		}, 1000);
		return () => clearInterval(t);
	});

	// The card put back in: the look or the copy carries on by itself. What
	// is in already is found by content and not copied again.
	let restarting = false;
	$effect(() => {
		if (!restarting && view === 'taken-out' && im?.dir && im.profile && cards.some((c) => c.path === im?.dir)) {
			restarting = true;
			carryOn().finally(() => (restarting = false));
		}
	});
	const carryOn = () =>
		start(() => (im!.kind === 'plan' ? planImport(im!.dir!, im!.profile!) : startImport(im!.dir!, im!.profile!)));

	async function start(call: () => Promise<ImportState>) {
		choosing = false;
		try {
			im = await call();
		} catch (e) {
			// A second click while the first one runs: that run is the one shown.
			if (e instanceof ApiError && e.status === 409) return load();
			const said = e instanceof ApiError && e.status === 400 ? text.importBadProfile : (e as Error).message;
			im = { running: false, n: 0, of: 0, done: 0, total: 0, error: said };
		}
	}
	function look(dir: string) {
		if (!profile) return;
		// The room left is read again for each card: an import before took some.
		getLibrary().then((l) => (free = l.free), () => {});
		start(() => planImport(dir, profile));
	}
	async function chooseFolder() {
		const picked = await window.shell?.chooseFolder();
		if (picked) look(picked);
	}

	const plan = $derived(im?.plan);
	const otherFiles = $derived((plan?.others ?? []).reduce((n, o) => n + o.files, 0));
	const fits = $derived(!plan || !free || plan.bytes <= free);
	const fileShare = $derived(im && im.total > 0 ? im.done / im.total : 0);
	const share = $derived(im && im.of > 0 ? (im.n - 1 + fileShare) / im.of : 0);
</script>

<div class="page">
	{#if view === 'choose'}
		<!-- I1: a card in, or a folder to choose. -->
		<h1>{cards.length > 0 ? text.importChoose : text.importNoCard}</h1>
		<p class="lead">{cards.length > 0 ? text.importChooseLead : text.importNoCardLead}</p>
		<div class="choices">
			{#each cards as c (c.path)}
				<button class="choice card" disabled={!profile} onclick={() => look(c.path)}>
					<span class="tile"><HardDrive size={22} /></span>
					<span class="what"><b>{c.name}</b><small>{c.path}</small></span>
					<span class="go">{text.lookAtCard}</span>
				</button>
			{/each}
			<button class="choice" disabled={!profile || !window.shell} onclick={chooseFolder}>
				<span class="tile"><Folder size={22} /></span>
				<span class="what"><b>{text.anotherFolder}</b><small>{text.anotherFolderDesc}</small></span>
				<span class="go">{text.chooseFolderEllipsis}</span>
			</button>
		</div>
		<label class="whose">
			{profiles.length === 0 ? text.whosePhotos : text.importGoesTo}
			{#if profiles.length > 1}
				<select bind:value={profile}>
					{#each profiles as p (p)}<option value={p}>{p}</option>{/each}
				</select>
			{:else if profiles.length === 1}
				<b>{profile}</b>
			{:else}
				<input bind:value={profile} placeholder={text.profilePlaceholder} spellcheck="false" />
			{/if}
		</label>
	{:else if view === 'reading' && im}
		<!-- I2: every file read once, to know it; nothing copied yet. -->
		<span class="pill live"><span class="spin"><Loader size={14} /></span>{text.importReading(name)}</span>
		<h1>{text.importReadingTitle}</h1>
		<p class="lead">{text.importReadingLead}</p>
		<div class="track"><b style:width="{share * 100}%"></b></div>
		{#if im.of > 0}<div class="numbers"><div><b>{text.nOf(im.n, im.of)}</b><span>{text.filesRead}</span></div></div>{/if}
		<div class="actions"><SecondaryButton onclick={() => stopImport().then((s) => ((im = s), (choosing = true)))}>{text.importStop}</SecondaryButton></div>
	{:else if view === 'plan' && plan && im}
		<!-- I3: what comes in, by year; what stays out. -->
		<span class="pill">{name}</span>
		<h1>{text.toCopy(plan.photos, plan.videos)}</h1>
		{#if plan.photos + plan.videos === 0}<p class="lead">{text.nothingNewLead}</p>{:else if plan.in_library > 0}<p class="lead">{text.toCopyLead}</p>{/if}
		<div class="numbers">
			<div><b>{bytes(plan.bytes)}</b><span>{text.importToCopy}</span></div>
			<div><b>{plan.in_library.toLocaleString('en')}</b><span>{text.importInLibrary}</span></div>
			{#if otherFiles > 0}<div><b>{otherFiles.toLocaleString('en')}</b><span>{text.importLeftOut}</span></div>{/if}
			{#if plan.unreadable > 0}<div class="warn"><b>{plan.unreadable.toLocaleString('en')}</b><span>{text.importUnreadable}</span></div>{/if}
			{#if free}<div class:warn={!fits}><b>{bytes(free)}</b><span>{text.importFree}</span></div>{/if}
		</div>
		{#if !fits}<p class="problem"><TriangleAlert size={16} />{text.importNoRoom}</p>{/if}
		<div class="actions">
			<PrimaryButton disabled={!fits || plan.photos + plan.videos === 0} onclick={() => start(() => startImport(im!.dir!, profile))}
				>{text.copyAndVerify}</PrimaryButton
			>
			<SecondaryButton onclick={() => (choosing = true)}>{text.back}</SecondaryButton>
		</div>
		<p class="whose">{text.importGoesTo} <b>{im.profile || profile}</b></p>
		{#if plan.years.length > 0}
			<section class="years">
				<header><h2>{text.byYearImport}</h2><span>{text.byYearImportLegend}</span></header>
				{#each plan.years as y (y.year)}
					<div class="year"><span class="name">{y.year || text.noDate}</span><span class="count">{y.photos.toLocaleString('en')}{y.videos ? ` · ${y.videos.toLocaleString('en')}` : ''}</span></div>
				{/each}
			</section>
		{/if}
		{#if plan.others.length > 0}
			<section class="years">
				<header><h2>{text.leftOut}</h2><span>{text.leftOutLead}</span></header>
				{#each plan.others as o (o.folder)}
					<div class="year"><span class="name" class:mono={o.folder !== '.'}>{o.folder === '.' ? text.importTop : o.folder}</span><span class="count">{o.files.toLocaleString('en')} · {bytes(o.bytes)}</span></div>
				{/each}
			</section>
		{/if}
	{:else if view === 'copying' && im}
		<!-- I6: each file copied and read back, one at a time. -->
		<span class="pill live"><span class="spin"><Loader size={14} /></span>{name}</span>
		<h1>{text.importCopying}</h1>
		<p class="lead">{text.importCopyingLead}</p>
		<div class="track"><b style:width="{share * 100}%"></b></div>
		<div class="numbers">
			<div><b>{text.nOf(im.n, im.of)}</b><span>{text.importCopied}</span></div>
			{#if im.total > 0}<div><b>{bytes(im.done)} / {bytes(im.total)}</b><span>{im.name}</span></div>{/if}
		</div>
		<div class="actions"><SecondaryButton onclick={() => stopImport().then((s) => (im = s))}>{text.importStop}</SecondaryButton></div>
	{:else if view === 'taken-out' && im}
		<!-- I7: the card came out; put back, the copy carries on by itself. -->
		<span class="pill warn">{text.importWaitingFor(name)}</span>
		<h1>{text.importTakenOut(name)}</h1>
		<p class="lead">{im.kind === 'plan' ? text.importTakenOutReadingLead : text.importTakenOutLead}</p>
		{#if im.imported}
			<div class="numbers">
				<div><b>{im.imported.placed.toLocaleString('en')}</b><span>{text.importPlaced}</span></div>
			</div>
		{/if}
		<div class="actions">
			<!-- A card is found again by itself; a disk or a share, once back, is carried on from here. -->
			{#if !isCard}<PrimaryButton onclick={carryOn}>{text.carryOn}</PrimaryButton>{/if}
			<SecondaryButton onclick={() => stopImport().then((s) => (im = s))}
				>{im.kind === 'plan' ? text.importStop : text.importEndWithout}</SecondaryButton
			>
		</div>
		{#if im.kind !== 'plan'}<p class="note">{text.importEndWithoutDesc}</p>{/if}
	{:else if (view === 'done' || view === 'stopped') && im}
		<!-- I8: what came in; the card can go. -->
		{@const done = im.imported}
		<span class="pill" class:ok={view === 'done'}>{#if view === 'done'}<CircleCheck size={14} />{/if}{name}</span>
		<h1>{view === 'stopped' ? text.importStopped : text.importDone(done?.placed ?? 0)}</h1>
		<p class="lead">{view === 'stopped' ? text.importStoppedLead : text.importDoneLead}</p>
		{#if done}
			<div class="numbers">
				<div><b>{done.placed.toLocaleString('en')}</b><span>{text.importPlaced}</span></div>
				<div><b>{done.in_library.toLocaleString('en')}</b><span>{text.importInLibrary}</span></div>
				{#if done.undated > 0}<div><b>{done.undated.toLocaleString('en')}</b><span>{text.importUndated}</span></div>{/if}
				{#if done.unreadable > 0}<div class="warn"><b>{done.unreadable.toLocaleString('en')}</b><span>{text.importUnreadable}</span></div>{/if}
			</div>
		{/if}
		{#if isCard}
			<p class="note"><b>{text.canTakeOut(name)}</b> {text.nothingChanged}</p>
		{/if}
		<div class="actions">
			<PrimaryButton onclick={() => goto('/library')}>{text.seeLibraryNow}</PrimaryButton>
			<SecondaryButton onclick={() => (choosing = true)}>{text.back}</SecondaryButton>
		</div>
	{:else if im}
		<span class="pill err">{name}</span>
		<h1>{text.importFailed}</h1>
		<p class="lead">
			{im.problem === 'no-space'
				? text.importNoRoom
				: im.problem === 'busy'
					? text.importBusy
					: im.problem === 'library-gone'
						? text.importLibraryGone
						: im.error}
		</p>
		<div class="actions"><PrimaryButton onclick={() => (choosing = true)}>{text.lookAgain}</PrimaryButton></div>
	{/if}
</div>

<style>
	.page {
		height: 100%;
		overflow-y: auto;
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 16px;
		padding: 48px 40px 40px;
		max-width: 760px;
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
	.pill.ok {
		background: var(--ok-soft);
		color: var(--ok);
	}
	.pill.warn {
		background: var(--warn-soft);
		color: var(--warn);
	}
	.pill.err {
		background: var(--err-soft);
		color: var(--err);
	}
	.spin {
		display: grid;
		animation: spin 1s linear infinite;
	}
	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
	.choices {
		width: 100%;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.choice {
		display: flex;
		align-items: center;
		gap: 16px;
		padding: 16px;
		border: 1px solid var(--border);
		border-radius: 12px;
		background: var(--surface);
		text-align: left;
	}
	.choice.card {
		border-color: var(--accent);
	}
	.choice:disabled {
		opacity: 0.5;
	}
	.choice:not(:disabled):hover {
		background: var(--soft);
	}
	.tile {
		width: 44px;
		height: 44px;
		display: grid;
		place-items: center;
		border-radius: 10px;
		background: var(--accent-soft);
		color: var(--accent);
	}
	.what {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
	}
	.what b {
		font-size: var(--text-title-3);
		color: var(--fg);
	}
	.what small {
		color: var(--muted);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.go {
		color: var(--accent);
		font-weight: 600;
	}
	.whose {
		color: var(--muted);
		display: flex;
		align-items: center;
		gap: 8px;
	}
	.whose b {
		color: var(--fg);
	}
	.whose select,
	.whose input {
		height: 32px;
		padding: 0 8px;
		border: 1px solid var(--border);
		border-radius: 8px;
		background: var(--surface);
		color: var(--fg);
	}
	.numbers {
		display: flex;
		flex-wrap: wrap;
		gap: 16px 40px;
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
	.numbers .warn b {
		color: var(--warn);
	}
	.track {
		width: 100%;
		height: 8px;
		border-radius: 4px;
		background: var(--soft);
		overflow: hidden;
	}
	.track b {
		display: block;
		height: 100%;
		background: var(--accent);
		transition: width 0.5s;
	}
	.actions {
		display: flex;
		gap: 12px;
	}
	.problem {
		display: flex;
		align-items: center;
		gap: 8px;
		color: var(--warn);
		font-weight: 600;
	}
	.note {
		color: var(--muted);
	}
	.note b {
		color: var(--fg);
	}
	.years {
		width: 100%;
		display: flex;
		flex-direction: column;
	}
	.years header {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		padding-bottom: 8px;
		border-bottom: 1px solid var(--line);
	}
	.years h2 {
		font-size: var(--text-title-3);
		font-weight: 600;
		color: var(--fg);
	}
	.years header span {
		color: var(--muted);
		font-size: var(--text-callout);
	}
	.year {
		display: flex;
		justify-content: space-between;
		padding: 10px 0;
		border-bottom: 1px solid var(--line);
	}
	.year .name {
		color: var(--fg);
		font-weight: 600;
	}
	.mono {
		font-family: var(--font-mono);
	}
	.year .count {
		color: var(--muted);
		font-variant-numeric: tabular-nums;
	}
</style>
