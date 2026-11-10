<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- I1–I8: a card or a folder, what comes in, the copy, a card taken out, the end.
     Each title answers what a person wants to know there; the numbers that
     matter are the biggest thing on the screen. -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { ChevronRight, CircleCheck, Folder, HardDrive, Loader, TriangleAlert } from '@lucide/svelte';
	import { app, refresh } from '#lib/app.svelte.js';
	import {
		ApiError,
		getCards,
		getImport,
		getLibrary,
		planImport,
		startImport,
		stopImport,
		type Card,
		againImport,
		type Choice,
		type Filed,
		type ImportAsk,
		type ImportGroup,
		type ImportPlan,
		type Redate,
		type ImportState
	} from '#lib/api.js';
	import { bytes, roughly, takenDay } from '#lib/format.js';
	import { forget, pace } from '#lib/pace.js';
	import { text } from '#lib/strings.js';
	import PrimaryButton from '#lib/components/PrimaryButton.svelte';
	import SecondaryButton from '#lib/components/SecondaryButton.svelte';
	import ImportDates from './ImportDates.svelte';
	import ImportPick from './ImportPick.svelte';

	let im = $state<ImportState | null>(null);
	const plan = $derived(im?.plan);
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

	// How the folder is named on every screen, as the engine names it.
	const name = $derived(im?.top ?? '');
	// Said by the engine when the run began: a card taken out is no longer
	// in the list, and is still a card.
	const isCard = $derived(!!im?.card);

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
			const s = await getImport();
			// The cards are looked at again before a card taken out is shown:
			// a list from before it came out would carry the copy on at once.
			if (s.problem === 'taken-out') await lookCards();
			im = s;
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
				// What was given before the window closed goes on with it.
				redates = im.redates ?? [];
				choice = im.choice ?? null;
				bringBack = im.bring_back ?? [];
			}
		});
		lookCards();
		getLibrary().then((l) => (free = l.free), () => {});
		const t = setInterval(() => {
			if (im?.running) load();
			// The cards come and go: one put in while the window is open, the
			// one taken out put back.
			lookCards();
		}, 1000);
		return () => clearInterval(t);
	});

	// What came in is in the sidebar's counts and years at once.
	$effect(() => {
		if (view === 'done' || view === 'stopped') refresh();
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
	// The engine keeps what was asked, and runs it again.
	const carryOn = () => start(againImport);

	async function start(call: () => Promise<ImportState>) {
		choosing = false;
		// A run measures its own pace, not the last one's.
		forget('import');
		try {
			im = await call();
		} catch (e) {
			// A second click while the first one runs: that run is the one shown.
			if (e instanceof ApiError && e.status === 409) return load();
			const said = e instanceof ApiError && e.status === 400 ? text.importBadProfile : (e as Error).message;
			im = { running: false, n: 0, of: 0, done: 0, total: 0, moved: 0, bytes: 0, error: said };
		}
	}
	function look(dir: string) {
		if (!profile) return;
		// The room left is read again for each card: an import before took some.
		getLibrary().then((l) => (free = l.free), () => {});
		redates = [];
		link = false;
		choice = null;
		bringBack = [];
		start(() => planImport({ dir, profile }));
	}

	// Dates given to photos a reset clock dated wrong (I3b): kept here
	// until the copy, and looked at again with them, which reads nothing
	// twice.
	let redates = $state<Redate[]>([]);
	let dating = $state(false);
	// Photos deleted from the library come in again only when ticked, by
	// name, or all.
	let bringBack = $state<string[]>([]);
	function back(name: string) {
		bringBack = bringBack.includes(name) ? bringBack.filter((n) => n !== name) : [...bringBack, name];
		lookAgain();
	}
	function backAll(on: boolean) {
		bringBack = on ? ['all'] : [];
		lookAgain();
	}

	// What the person has said so far, for the next look or the copy.
	const asking = (): ImportAsk => ({ dir: im!.dir!, profile: im!.profile || profile, redates, choice, bring_back: bringBack });
	const lookAgain = () => start(() => planImport(asking()));

	// Which of the folder's groups come in (I4), and the photos picked from
	// one that stays out (I5): null, each as its kind says. A change looks
	// again, which reads nothing twice.
	let choice = $state<Choice | null>(null);
	let picking = $state<string | null>(null);
	function rechoose(next: Choice) {
		choice = next;
		lookAgain();
	}
	const without = (list: string[], s: string) => list.filter((x) => x !== s);
	function tick(g: ImportGroup) {
		const c = choice ?? { out: [], in: [], pick: [] };
		rechoose(
			g.in
				? { ...c, in: without(c.in, g.folder), out: [...without(c.out, g.folder), g.folder] }
				: { ...c, out: without(c.out, g.folder), in: [...without(c.in, g.folder), g.folder] }
		);
	}
	// Each file's group, by its path: a folder may hold a hundred thousand.
	const groupOf = $derived(new Map((plan?.files ?? []).map((f) => [f.path, f.group])));
	const pickedIn = (g: string) => (choice?.pick ?? []).filter((p) => groupOf.get(p) === g);
	// What would come in: the files of the groups in, and those picked.
	const coming = $derived.by(() => {
		const whole = new Set((plan?.groups ?? []).filter((g) => g.in).map((g) => g.folder));
		const picked = new Set(choice?.pick ?? []);
		return (plan?.files ?? []).filter((f) => whole.has(f.group) || picked.has(f.path));
	});
	function pick(g: string, paths: string[]) {
		const c = choice ?? { out: [], in: [], pick: [] };
		const others = c.pick.filter((p) => groupOf.get(p) !== g);
		picking = null;
		rechoose({ ...c, pick: [...others, ...paths] });
	}
	const range = (g: ImportGroup) =>
		g.first && g.last ? text.takenRange(takenDay(new Date(g.first)), takenDay(new Date(g.last))) : '';
	function give(names: string[], when: string) {
		// A photo given a date again keeps only the last.
		const others = redates
			.map((r) => ({ ...r, names: r.names.filter((n) => !names.includes(n)) }))
			.filter((r) => r.names.length > 0);
		redates = [...others, { names, when }];
		dating = false;
		lookAgain();
	}
	async function chooseFolder() {
		const picked = await window.shell?.chooseFolder();
		if (picked) look(picked);
	}

	// Hard links rather than copies: asked each time, copy unless said.
	let link = $state(false);
	// The years opened to show their months.
	let opened = $state(new Set<string>());
	function toggle(year: string) {
		const next = new Set(opened);
		if (!next.delete(year)) next.add(year);
		opened = next;
	}
	// What is in of a year, from its months filed.
	const filedIn = (filed: Filed, year: string) => Object.values(filed[year] ?? {}).reduce((n, k) => n + k, 0);
	// "verified / to copy", for a year or a month of the copy.
	const verified = (n: number, c: { photos: number; videos: number }) =>
		`${n.toLocaleString('en')} / ${(c.photos + c.videos).toLocaleString('en')}`;
	const counts = (c: { photos: number; videos: number }) =>
		`${c.photos ? c.photos.toLocaleString('en') : '—'}${c.videos ? ` · ${c.videos.toLocaleString('en')}` : ''}`;
	const fits = $derived(!plan || !free || plan.bytes <= free);
	// The copy's speed and time left, as a download's are measured.
	const run = $derived(im?.running && im.kind === 'import' ? pace('import', im.moved, im.bytes) : null);
	const fileShare = $derived(im && im.total > 0 ? im.done / im.total : 0);
	const share = $derived(im && im.of > 0 ? (im.n - 1 + fileShare) / im.of : 0);
</script>

<!-- The years, newest first; a year opened shows its months below it (I8b). -->
<!-- During the copy (I6), each with what is in and checked, of what to copy. -->
{#snippet byYear(years: ImportPlan['years'], filed?: Filed)}
	{#if years.length > 0}
		<section class="years">
			<header><h2>{text.byYearImport}</h2><span>{filed ? text.verifiedLegend : text.byYearImportLegend}</span></header>
			{#each years as y (y.year)}
				{@const open = opened.has(y.year)}
				<button class="year" class:open disabled={!y.months?.length} onclick={() => toggle(y.year)}>
					<span class="name">{#if y.months?.length}<ChevronRight size={16} />{/if}{y.year || text.noDate}</span>
					<span class="count">{filed ? verified(filedIn(filed, y.year), y) : counts(y)}</span>
				</button>
				{#if open}
					{#each y.months ?? [] as m (m.month)}
						<div class="year month">
							<span class="name">{text.monthNames[m.month - 1]}</span>
							<span class="count"
								>{filed ? verified(filed[y.year]?.[m.month] ?? 0, m) : counts(m)}</span
							>
						</div>
					{/each}
				{/if}
			{/each}
		</section>
	{/if}
{/snippet}

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
	{:else if view === 'plan' && plan && im && picking !== null && plan.groups.some((g) => g.folder === picking)}
		<ImportPick
			group={plan.groups.find((g) => g.folder === picking)!}
			files={plan.files}
			picked={pickedIn(picking)}
			onpick={(paths) => pick(picking!, paths)}
			onback={() => (picking = null)}
		/>
	{:else if view === 'plan' && plan && im && dating}
		<ImportDates files={coming} {redates} onuse={give} onback={() => (dating = false)} />
	{:else if view === 'plan' && plan && im}
		<!-- I3: what comes in, by year; what stays out. -->
		<span class="pill">{name}</span>
		<h1>{text.toCopy(plan.photos, plan.videos)}</h1>
		<!-- How it ends, as the engine says it, for this screen and the command line alike. -->
		{#if plan.outcome !== 'new'}<p class="lead">{text.planLead[plan.outcome](plan)}</p>{:else if plan.in_library > 0}<p class="lead">{text.toCopyLead}</p>{/if}
		<div class="numbers">
			<div><b>{bytes(plan.bytes)}</b><span>{text.importToCopy}</span></div>
			<div><b>{plan.in_library.toLocaleString('en')}</b><span>{text.importInLibrary}</span></div>
			{#if plan.other_files > 0}<div><b>{plan.other_files.toLocaleString('en')}</b><span>{text.importLeftOut}</span></div>{/if}
			{#if plan.unreadable > 0}<div class="warn"><b>{plan.unreadable.toLocaleString('en')}</b><span>{text.importUnreadable}</span></div>{/if}
			{#if free}<div class:warn={!fits}><b>{bytes(free)}</b><span>{text.importFree}</span></div>{/if}
		</div>
		{#if im.canLink && plan.photos + plan.videos > 0}
			<!-- On the library's disk: a second name for each file instead of a copy. -->
			<fieldset class="how">
				<legend>{text.importHow}</legend>
				<label class:on={!link}
					><input type="radio" bind:group={link} value={false} /><b>{text.importHowCopy}</b><span>{text.importHowCopyDesc(bytes(plan.bytes))}</span></label
				>
				<label class:on={link}
					><input type="radio" bind:group={link} value={true} /><b>{text.importHowLink}</b><span>{text.importHowLinkDesc}</span></label
				>
			</fieldset>
		{/if}
		{#if !fits && !link}<p class="problem"><TriangleAlert size={16} />{text.importNoRoom}</p>{/if}
		<div class="actions">
			<PrimaryButton
				disabled={(!fits && !link) || plan.outcome !== 'new' && plan.outcome !== 'redated'}
				onclick={() => start(() => startImport({ ...asking(), link: link && !!im!.canLink }))}>{link ? text.linkIn : text.copyAndVerify}</PrimaryButton
			>
			<SecondaryButton onclick={() => (choosing = true)}>{text.back}</SecondaryButton>
		</div>
		<p class="whose">{text.importGoesTo} <b>{im.profile || profile}</b></p>
		<!-- A camera whose clock was reset: its photos given their day before the copy. -->
		<p class="note">
			{#if redates.length > 0}{text.datesGiven(redates.reduce((n, r) => n + r.names.length, 0))}
				{#if plan.not_redated > 0}<span class="warnText">{text.notRedated(plan.not_redated)}</span>{/if}
			{/if}
			<button class="link" onclick={() => (dating = true)}>{text.anyDatesWrong}</button>
		</p>
		{#if plan.deleted > 0}
			<!-- The card still holds what was deleted from the library: it stays out
			     unless ticked back, one by its name or all. -->
			{@const all = bringBack.includes('all')}
			{@const named = [...bringBack.filter((n) => n !== 'all'), ...(plan.deleted_names ?? [])]}
			<section class="years deleted">
				<header>
					<h2>{text.deletedHead(plan.deleted)}</h2>
					<label class="back"><input type="checkbox" checked={all} onchange={() => backAll(!all)} />{text.bringAllBack}</label>
				</header>
				{#each named as n (n)}
					<label class="year back">
						<span class="name mono"
							><input type="checkbox" checked={all || bringBack.includes(n)} disabled={all} onchange={() => back(n)} />{n}</span
						>
					</label>
				{/each}
				{#if plan.deleted - plan.brought_back > (plan.deleted_names?.length ?? 0)}
					<div class="year"><span class="name">{text.othersMore(plan.deleted - plan.brought_back - (plan.deleted_names?.length ?? 0))}</span></div>
				{/if}
			</section>
		{/if}
		{#if plan.groups.length > 0}
			<!-- I4: one row per group, with why; a project is one row, not its folders. -->
			<section class="years groups">
				<header><h2>{text.whatComesIn}</h2><span>{text.whatComesInLegend}</span></header>
				{#each plan.groups as g (g.folder)}
					{@const picks = pickedIn(g.folder).length}
					<div class="group" class:out={!g.in}>
						<input type="checkbox" checked={g.in} onchange={() => tick(g)} aria-label={g.folder} />
						<span class="what">
							<b class="mono">{g.folder === '.' ? name : g.folder}</b>
							<small>
								{g.kind === 'photos'
									? [range(g) || text.whyPhotos, g.in_library ? text.nInLibrary(g.in_library) : ''].filter(Boolean).join(' · ')
									: g.kind === 'project'
										? text.whyProject(g.mark ?? '')
										: g.kind === 'previews'
											? text.whyPreviews(g.mark ?? '')
											: text.whyUnsure}
							</small>
							{#if !g.in}<button class="link" onclick={() => (picking = g.folder)}>{text.choosePhotosInside(picks)}</button>{/if}
						</span>
						<span class="tag {g.kind}">{text.groupKind[g.kind]}</span>
						<span class="count">{!g.in && picks ? `${picks} / ` : ''}{(g.photos + g.videos).toLocaleString('en')}</span>
					</div>
				{/each}
			</section>
		{/if}
		{@render byYear(plan.years)}
		{#if plan.others.length > 0}
			<section class="years">
				<header><h2>{text.leftOut}</h2><span>{text.leftOutLead}</span></header>
				{#each plan.others as o (o.name)}
					<div class="year"><span class="name mono">{o.name}</span><span class="count">{o.files.toLocaleString('en')} · {bytes(o.bytes)}</span></div>
				{/each}
				{#if plan.others_more}<div class="year"><span class="name">{text.othersMore(plan.others_more)}</span></div>{/if}
			</section>
		{/if}
	{:else if view === 'copying' && im}
		<!-- I6: each file copied and read back, one at a time. -->
		<span class="pill live"><span class="spin"><Loader size={14} /></span>{name}</span>
		<h1>{text.importCopying}</h1>
		<p class="lead">{text.importCopyingLead}</p>
		<div class="track"><b style:width="{share * 100}%"></b></div>
		<div class="numbers">
			<div><b>{text.nOf(im.n, im.of)}</b><span>{im.stage === 'read' ? text.datesRead : text.importCopied}</span></div>
			{#if im.total > 0}<div><b>{bytes(im.done)} / {bytes(im.total)}</b><span>{im.name}</span></div>{/if}
			{#if run}
				<div><b>{text.perSecond(bytes(run.rate))}</b><span>{text.speed}</span></div>
				<div><b>{text.aboutTime(roughly(run.left))}</b><span>{text.leftAtSpeed}</span></div>
			{/if}
		</div>
		<div class="actions"><SecondaryButton onclick={() => stopImport().then((s) => (im = s))}>{text.importStop}</SecondaryButton></div>
		{#if plan}{@render byYear(plan.years, im.filed ?? {})}{/if}
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
		<!-- How it ended, as the engine says it. -->
		{@const end =
			view === 'stopped'
				? { title: text.importStopped, lead: text.importStoppedLead }
				: done && done.outcome !== 'new'
					? { title: text.importNothingNew, lead: (text.doneLead[done.outcome as keyof typeof text.doneLead] ?? text.planLead[done.outcome])(done) }
					: { title: text.importDone(done?.placed ?? 0), lead: text.importDoneLead }}
		<h1>{end.title}</h1>
		<p class="lead">{end.lead}</p>
		{#if done}
			<div class="numbers">
				<div><b>{done.placed.toLocaleString('en')}</b><span>{text.importPlaced}</span></div>
				<div><b>{done.in_library.toLocaleString('en')}</b><span>{text.importInLibrary}</span></div>
				{#if done.undated > 0}<div><b>{done.undated.toLocaleString('en')}</b><span>{text.importUndated}</span></div>{/if}
				{#if done.redated > 0}<div><b>{done.redated.toLocaleString('en')}</b><span>{text.importRedated}</span></div>{/if}
				{#if done.brought_back > 0}<div><b>{done.brought_back.toLocaleString('en')}</b><span>{text.importDeletedBack}</span></div>{/if}
				{#if done.deleted > done.brought_back}<div><b>{(done.deleted - done.brought_back).toLocaleString('en')}</b><span>{text.importDeletedOut}</span></div>{/if}
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
		<!-- All of it came in: what came in is what the look said would. -->
		{#if view === 'done' && plan}{@render byYear(plan.years)}{/if}
	{:else if im}
		<span class="pill err">{name}</span>
		<h1>{text.importFailed}</h1>
		<p class="lead">
			{im.problem === 'no-space'
				? text.importNoRoom
				: im.problem === 'no-link'
					? text.importNoLink
				: im.problem === 'busy'
					? text.importBusy
					: im.problem === 'library-gone'
						? text.libraryGone
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
	}
	/* The page scrolls whole, its scrollbar at the window's edge; what is
	   read stays a column a line can be followed across. Picking dates
	   takes more: the photos and the panel beside them. */
	.page > :global(*) {
		max-width: 760px;
	}
	.page > :global(.dates) {
		max-width: 1200px;
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
	.how {
		width: 100%;
		display: flex;
		flex-direction: column;
		gap: 8px;
		border: 0;
		padding: 0;
	}
	.how legend {
		margin-bottom: 8px;
		font-size: var(--text-callout);
		font-weight: 600;
		color: var(--muted);
	}
	.how label {
		display: grid;
		grid-template-columns: auto 1fr;
		column-gap: 12px;
		padding: 12px 16px;
		border: 1px solid var(--border);
		border-radius: 12px;
		cursor: pointer;
	}
	.how label.on {
		border-color: var(--accent);
	}
	.how input {
		grid-row: span 2;
		accent-color: var(--accent);
	}
	.how b {
		color: var(--fg);
	}
	.how span {
		color: var(--muted);
		font-size: var(--text-callout);
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
	.link {
		color: var(--accent);
		font-weight: 600;
	}
	.back {
		display: flex;
		align-items: center;
		gap: 8px;
		cursor: pointer;
	}
	.back input {
		accent-color: var(--accent);
	}
	.warnText {
		color: var(--warn);
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
	button.year {
		width: 100%;
		text-align: left;
	}
	button.year:not(:disabled):hover {
		background: var(--soft);
	}
	.year .name {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		color: var(--fg);
		font-weight: 600;
	}
	.year.open .name :global(svg) {
		transform: rotate(90deg);
	}
	.year.month {
		padding-left: 24px;
	}
	.year.month .name {
		font-weight: 400;
		color: var(--muted);
	}
	.group {
		display: flex;
		align-items: flex-start;
		gap: 12px;
		padding: 12px 0;
		border-bottom: 1px solid var(--line);
	}
	.group input {
		margin-top: 4px;
		accent-color: var(--accent);
	}
	.group .what {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 2px;
	}
	.group .what b {
		color: var(--fg);
		overflow-wrap: anywhere;
	}
	.group .what small {
		color: var(--muted);
		font-size: var(--text-callout);
	}
	.group.out .what b {
		color: var(--muted);
	}
	.tag {
		padding: 2px 8px;
		border-radius: 999px;
		font-size: var(--text-footnote);
		font-weight: 600;
		background: var(--ok-soft);
		color: var(--ok);
		white-space: nowrap;
	}
	.tag.unsure {
		background: var(--warn-soft);
		color: var(--warn);
	}
	.tag.project,
	.tag.previews {
		background: var(--soft);
		color: var(--muted);
	}
	.mono {
		font-family: var(--font-mono);
	}
	.year .count {
		color: var(--muted);
		font-variant-numeric: tabular-nums;
	}
</style>
