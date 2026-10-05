<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- C1: the exports on Google Takeout, each account's. -->
<script module lang="ts">
	import type { Takeout as Seen } from '#lib/api.js';
	// Each account's list as last read, kept while the window is open.
	const seen = new Map<string, Seen[]>();
</script>

<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount, untrack } from 'svelte';
	import { Archive, Ellipsis, FolderOpen, RotateCw, Timer } from '@lucide/svelte';
	import ContextMenu, { type MenuItem } from '#lib/components/ContextMenu.svelte';
	import { app, resigning, sidebarOf, signInAgain } from '#lib/app.svelte.js';
	import { ApiError, folderOf, getTakeouts, readContents, startGet, type Takeout } from '#lib/api.js';
	import { bytes } from '#lib/format.js';
	import { text } from '#lib/strings.js';
	import Avatar from '#lib/components/Avatar.svelte';
	import Skeleton from '#lib/components/Skeleton.svelte';

	const accounts = $derived(app.overview ? sidebarOf(app.overview).accounts : []);
	let account = $state('');
	// The download this account runs, to say which takeout it is.
	const job = $derived(app.jobs[account]);
	const email = $derived(accounts.find((a) => a.id === account)?.email ?? '');
	$effect(() => {
		if (!account && accounts.length > 0) account = accounts[0].id;
	});

	let list = $state<Takeout[] | null>(null);
	let failed = $state('');
	let signedOut = $state(false);
	let tries = $state(0);
	// One sign-in at a time, shared with the download's banner: none can start
	// while one is open, and the page reads the list again once this
	// account's is done, wherever it was started.
	const busy = $derived(resigning.account !== '');
	const signingIn = $derived(resigning.account === account);
	let was = '';
	$effect(() => {
		const now = resigning.account;
		if (was === untrack(() => account) && now === '') untrack(() => tries++);
		was = now;
	});
	let reading = $state('');
	$effect(() => {
		const a = account;
		void tries;
		if (!a) return;
		// The list seen last shows at once; Google's answer replaces it.
		list = seen.get(a) ?? null;
		failed = '';
		signedOut = false;
		getTakeouts(a).then(
			async (l) => {
				seen.set(a, l);
				if (account !== a) return;
				list = l;
				// A ready takeout made on Google: read its list of files, one at a
				// time, to say what it holds.
				for (const t of l) {
					if (t.known || t.years?.length || t.status !== 'ready' || account !== a) continue;
					reading = t.id;
					try {
						t.years = (await readContents(a, t.id)).years;
						list = [...l];
					} catch {
						break; // no download from this account yet: nothing to read with
					} finally {
						reading = '';
					}
				}
			},
			(e) => {
				failed = String(e);
				signedOut = e instanceof ApiError && e.status === 401;
			}
		);
	});

	// "1 Oct, 09:14", "Until 8 Oct": the design's dates.
	const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
	const day = { format: (d: Date) => `${d.getDate()} ${months[d.getMonth()]}` };
	const clock = new Intl.DateTimeFormat('en-GB', { hour: '2-digit', minute: '2-digit' });
	const when = { format: (d: Date) => `${day.format(d)}, ${clock.format(d)}` };

	function holds(t: Takeout) {
		if (t.known) return t.year ? text.holdsYear(t.year) : text.holdsAll;
		const y = t.years ?? [];
		if (y.length === 1) return y[0];
		if (y.length > 1) return `${y[0]}–${y[y.length - 1]}`;
		return reading === t.id ? text.reading : text.holdsUnknown;
	}

	// One row's state: its status, the line under it, its colour, and what can
	// be done. The colour is the status's, on the dot and on the button.
	type Act = 'download' | 'resume' | 'retry' | 'timer' | 'more' | '';
	type Row = { status: string; detail: string; tone: string; act: Act };
	function rowOf(t: Takeout): Row {
		const until = t.Expires ? (t.status === 'expired' ? text.since : text.until)(day.format(new Date(t.Expires))) : '';
		const l = t.local;
		const all = l && l.of > 0 && l.parts === l.of;
		if (job?.running && job.export?.startsWith(t.id))
			return { status: text.statusDownloading, detail: text.partsOf(job.parts, job.of), tone: 'accent', act: '' };
		if (t.status === 'preparing') return { status: text.statusPreparing, detail: '', tone: 'warn', act: 'timer' };
		if (all && (l.present ?? 0) < (l.declared ?? 0)) {
			const missing = ((l.declared ?? 0) - (l.present ?? 0)).toLocaleString('en');
			return { status: text.statusMissing(missing), detail: until, tone: 'err', act: t.status === 'expired' ? 'more' : 'retry' };
		}
		if (all) return { status: text.statusAllHere, detail: until, tone: 'ok', act: 'more' };
		if (t.status === 'expired') return { status: text.statusExpired, detail: until, tone: 'faint', act: 'more' };
		if (l) return { status: text.statusPaused, detail: text.partsOf(l.parts, l.of), tone: 'accent', act: 'resume' };
		return { status: text.statusReady, detail: until, tone: 'ok', act: 'download' };
	}

	async function download(t: Takeout, again = false) {
		await startGet({ account, export: t.id, again });
		goto('/bringing');
	}
	// An expired takeout is asked of Google once more, as it was asked.
	async function askAgain(t: Takeout) {
		await startGet({ account, year: t.known && t.year ? t.year : undefined, fresh: true });
		goto('/bringing');
	}
	async function openFolder() {
		window.shell?.openFolder((await folderOf({ account })).path);
	}
	// The ⋯ of a row: the same menu as a folder's, at the end of the button.
	// Pressed again, the ⋯ closes it.
	let menu = $state<{ id: string; anchor: Element; x: number; y: number; start: number; items: MenuItem[] } | null>(null);
	function more(e: MouseEvent, t: Takeout) {
		if (menu?.id === t.id) {
			menu = null;
			return;
		}
		const anchor = e.currentTarget as HTMLElement;
		const b = anchor.getBoundingClientRect();
		const items: MenuItem[] =
			t.status === 'expired'
				? [{ icon: RotateCw, label: text.askAgain, run: () => askAgain(t) }]
				: [{ icon: RotateCw, label: text.downloadAgain, run: () => download(t, true) }];
		if (t.local) items.push({ icon: FolderOpen, label: text.openInFolder, run: openFolder });
		menu = { id: t.id, anchor, x: b.right, y: b.top + b.height / 2, start: b.left, items };
	}

	// How long Google has been preparing, counted while the page is open.
	let now = $state(Date.now());
	onMount(() => {
		const t = setInterval(() => (now = Date.now()), 1000);
		return () => clearInterval(t);
	});
	function since(iso: string) {
		const s = Math.max(0, Math.floor((now - new Date(iso).getTime()) / 1000));
		const p = (n: number) => String(n).padStart(2, '0');
		return `${Math.floor(s / 3600)}:${p(Math.floor(s / 60) % 60)}:${p(s % 60)}`;
	}
</script>

<div class="page">
	<header>
		<div>
			<h1>{text.takeoutsTitle}</h1>
			{#if accounts.length === 1}<p class="who">{accounts[0].email}</p>{/if}
		</div>
	</header>

	{#if accounts.length > 1}
		<div class="tabs">
			{#each accounts as a, i (a.id)}
				<button class:on={account === a.id} onclick={() => (account = a.id)}>
					<Avatar account={a} index={i} />{a.name}
				</button>
			{/each}
		</div>
	{/if}

	{#if failed && list === null}
		<!-- What went wrong is the engine's to log; the person is told what
		     they can do. -->
		<div class="failed {signedOut ? 'warn' : 'accent'}" title={failed}>
			{#if signedOut}
				<p>{signingIn ? text.signingIn : text.bannerSignedOut(email)}</p>
				<button class="pill" disabled={busy} onclick={() => signInAgain(account)}>{text.signInAgain}</button>
			{:else}
				<p>{text.takeoutsFailed}</p>
				<button class="pill" onclick={() => tries++}>{text.retry}</button>
			{/if}
		</div>
	{:else if list === null}
		<Skeleton label={text.readingTakeouts} />
	{:else if list.length === 0}
		<p class="muted">{text.noTakeouts}</p>
	{:else}
		<table>
			<thead>
				<tr><th>{text.takeoutCol}</th><th>{text.askedCol}</th><th>{text.sizeCol}</th><th>{text.statusCol}</th><th></th></tr>
			</thead>
			<tbody>
				{#each list as t (t.id)}
					{@const r = rowOf(t)}
					<tr>
						<td>
							<span class="name"><Archive size={16} /><span><b>{holds(t)}</b><small>{t.id}</small></span></span>
						</td>
						<td>{when.format(new Date(t.Created))}</td>
						<td>{t.Bytes ? bytes(t.Bytes) : '—'}{t.Parts?.length ? ` · ${text.parts(t.Parts.length)}` : ''}</td>
						<td>
							<span class="status {r.tone}"><i></i><span><b>{r.status}</b>{#if r.detail}<small>{r.detail}</small>{/if}</span></span>
						</td>
						<td class="act {r.tone}">
							{#if r.act === 'download'}<button class="pill" onclick={() => download(t)}>{text.download}</button>
							{:else if r.act === 'resume'}<button class="pill" onclick={() => download(t)}>{text.resume}</button>
							{:else if r.act === 'retry'}<button class="pill" onclick={() => download(t, true)}>{text.retry}</button>
							{:else if r.act === 'timer'}<span class="slot timer"><Timer size={14} />{since(t.Created)}</span>
							{:else if r.act === 'more'}
								<span class="slot"
									><button
										class="more"
										aria-label={text.moreActions}
										onclick={(e) => more(e, t)}><Ellipsis size={16} /></button></span
								>
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
</div>

{#if menu}<ContextMenu x={menu.x} y={menu.y} start={menu.start} items={menu.items} anchor={menu.anchor} onclose={() => (menu = null)} />{/if}

<style>
	.page {
		padding: 32px 40px;
		display: flex;
		flex-direction: column;
		gap: 24px;
	}
	h1 {
		font-size: var(--text-display);
		font-weight: 700;
		color: var(--fg);
	}
	.who {
		margin-top: 8px;
		color: var(--muted);
	}
	.tabs {
		display: flex;
		gap: 8px;
	}
	.tabs button {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 4px 12px 4px 4px;
		border: 1px solid var(--border);
		border-radius: 999px;
	}
	.tabs button.on {
		background: var(--line);
		border-color: transparent;
		font-weight: 600;
		color: var(--fg);
	}
	/* Fixed columns: every row's status and action fall on the same lines. */
	table {
		width: 100%;
		border-collapse: collapse;
		table-layout: fixed;
	}
	th {
		text-align: left;
		padding: 0 16px 8px 0;
		font-size: var(--text-subheadline);
		font-weight: 600;
		letter-spacing: 0.8px;
		color: var(--faint);
	}
	th:nth-child(1) {
		width: 26%;
	}
	th:nth-child(2),
	th:nth-child(3) {
		width: 17%;
	}
	th:nth-child(5) {
		width: 96px;
	}
	td {
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		padding: 12px 16px 12px 0;
		border-top: 1px solid var(--line);
		color: var(--text);
	}
	.name,
	.status {
		display: flex;
		align-items: center;
		gap: 12px;
		color: var(--muted);
	}
	.status {
		gap: 8px;
	}
	.name b {
		display: block;
		color: var(--fg);
		font-weight: 600;
	}
	.status b {
		display: block;
		color: var(--text);
		font-weight: 400;
	}
	small {
		display: block;
		font-size: var(--text-subheadline);
		color: var(--faint);
	}
	.name small {
		font-family: var(--font-mono);
	}
	.status i {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: var(--c);
		flex-shrink: 0;
	}
	/* A status's colour, on its dot and on its button. */
	.ok {
		--c: var(--ok);
		--c-soft: var(--ok-soft);
	}
	.accent {
		--c: var(--accent);
		--c-soft: var(--accent-soft);
	}
	.warn {
		--c: var(--warn);
		--c-soft: var(--warn-soft);
	}
	.err {
		--c: var(--err);
		--c-soft: var(--err-soft);
	}
	.faint {
		--c: var(--faint);
	}
	.act {
		padding-right: 0;
		overflow: visible;
	}
	/* Every action, button, timer or ⋯, sits in the same 96px slot, centred. */
	.pill,
	.slot {
		width: 96px;
		height: 28px;
		display: inline-flex;
		align-items: center;
		justify-content: center;
	}
	.pill {
		border-radius: 999px;
		background: var(--c-soft);
		color: var(--c);
		font-size: var(--text-callout);
		font-weight: 700;
	}
	.pill:hover {
		filter: brightness(1.1);
	}
	.timer {
		gap: 6px;
		color: var(--c);
		font-size: var(--text-callout);
		font-weight: 600;
		font-variant-numeric: tabular-nums;
	}
	.more {
		width: 28px;
		height: 28px;
		display: grid;
		place-items: center;
		border-radius: 50%;
		color: var(--muted);
	}
	.more:hover {
		background: var(--field);
	}
	/* Half a screen: the table scrolls sideways rather than break its lines. */
	.page:has(table) {
		overflow-x: auto;
	}
	:global(.narrow) .page {
		padding: 24px;
	}
	.muted {
		color: var(--muted);
	}
	.failed .pill {
		width: auto;
		padding: 0 14px;
	}
	.failed {
		display: flex;
		align-items: center;
		gap: 16px;
		color: var(--muted);
	}
</style>
