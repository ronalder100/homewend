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
	import { onMount } from 'svelte';
	import { Archive, Ellipsis, FolderOpen, RotateCw, Timer } from '@lucide/svelte';
	import { app, sidebarOf } from '#lib/app.svelte.js';
	import { ApiError, folderOf, getTakeouts, login, readContents, startGet, type Takeout } from '#lib/api.js';
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
	async function signInAgain() {
		await login(account);
		tries++;
	}
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
		menu = '';
		await startGet({ account, export: t.id, again });
		goto('/bringing');
	}
	// An expired takeout is asked of Google once more, as it was asked.
	async function askAgain(t: Takeout) {
		menu = '';
		await startGet({ account, year: t.known && t.year ? t.year : undefined, fresh: true });
		goto('/bringing');
	}
	async function openFolder() {
		menu = '';
		window.shell?.openFolder((await folderOf({ account })).path);
	}
	let menu = $state('');

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
				<p>{text.bannerSignedOut(email)}</p>
				<button class="pill" onclick={signInAgain}>{text.signInAgain}</button>
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
								<span class="slot more-at">
									<button class="more" aria-label={text.moreActions} onclick={() => (menu = menu === t.id ? '' : t.id)}><Ellipsis size={16} /></button>
									{#if menu === t.id}
										<div class="menu" role="menu">
											{#if t.status === 'expired'}
												<button role="menuitem" onclick={() => askAgain(t)}><RotateCw size={14} />{text.askAgain}</button>
											{:else}
												<button role="menuitem" onclick={() => download(t, true)}><RotateCw size={14} />{text.downloadAgain}</button>
											{/if}
											{#if t.local}<button role="menuitem" onclick={openFolder}><FolderOpen size={14} />{text.openInFolder}</button>{/if}
										</div>
									{/if}
								</span>
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
</div>

<svelte:window onmousedown={(e) => menu && !(e.target as Element).closest('.more-at') && (menu = '')} onkeydown={(e) => e.key === 'Escape' && (menu = '')} />

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
	.more-at {
		position: relative;
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
	.menu {
		position: absolute;
		right: 0;
		top: 32px;
		z-index: 5;
		width: 208px;
		padding: 8px;
		display: flex;
		flex-direction: column;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 8px;
		box-shadow:
			0 12px 40px #0a0a1a14,
			0 2px 6px #0a0a1a12;
	}
	.menu button {
		height: 32px;
		padding: 0 8px;
		display: flex;
		align-items: center;
		gap: 10px;
		border-radius: 4px;
		color: var(--fg);
		text-align: left;
	}
	.menu button :global(svg) {
		color: var(--muted);
	}
	.menu button:hover {
		background: var(--accent-soft);
		color: var(--accent);
	}
	.menu button:hover :global(svg) {
		color: var(--accent);
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
