<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- C1: the exports on Google Takeout, each account's. -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { Archive, Download } from '@lucide/svelte';
	import { app, sidebarOf } from '#lib/app.svelte.js';
	import { getTakeouts, startGet, type Takeout } from '#lib/api.js';
	import { bytes } from '#lib/format.js';
	import { text } from '#lib/strings.js';
	import Avatar from '#lib/components/Avatar.svelte';
	import SecondaryButton from '#lib/components/SecondaryButton.svelte';

	const accounts = $derived(app.overview ? sidebarOf(app.overview).accounts : []);
	let account = $state('');
	$effect(() => {
		if (!account && accounts.length > 0) account = accounts[0].id;
	});

	let list = $state<Takeout[] | null>(null);
	let failed = $state('');
	$effect(() => {
		const a = account;
		if (!a) return;
		list = null;
		failed = '';
		getTakeouts(a).then(
			(l) => (list = l),
			(e) => (failed = String(e))
		);
	});

	// "1 Oct, 09:14", "Until 8 Oct": the design's dates.
	const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
	const day = { format: (d: Date) => `${d.getDate()} ${months[d.getMonth()]}` };
	const clock = new Intl.DateTimeFormat('en-GB', { hour: '2-digit', minute: '2-digit' });
	const when = { format: (d: Date) => `${day.format(d)}, ${clock.format(d)}` };

	function holds(t: Takeout) {
		if (!t.known) return text.holdsUnknown;
		return t.year ? text.holdsYear(t.year) : text.holdsAll;
	}
	async function download(t: Takeout) {
		await startGet({ account, export: t.id });
		goto('/bringing');
	}
</script>

<div class="page">
	<header>
		<div>
			<h1>{text.takeoutsTitle}</h1>
			{#if accounts.length === 1}<p class="who">{accounts[0].email}</p>{/if}
		</div>
		<SecondaryButton onclick={() => goto('/choose')}>{text.newTakeout}</SecondaryButton>
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

	{#if failed}
		<p class="err">{failed}</p>
	{:else if list === null}
		<p class="muted">{text.readingTakeouts}</p>
	{:else if list.length === 0}
		<p class="muted">{text.noTakeouts}</p>
	{:else}
		<table>
			<thead>
				<tr><th>{text.takeoutCol}</th><th>{text.askedCol}</th><th>{text.sizeCol}</th><th>{text.statusCol}</th><th></th></tr>
			</thead>
			<tbody>
				{#each list as t (t.id)}
					<tr class:expired={t.status === 'expired'}>
						<td>
							<span class="name"><Archive size={16} /><span><b>{holds(t)}</b><small>{t.id}</small></span></span>
						</td>
						<td>{when.format(new Date(t.Created))}</td>
						<td>{t.Bytes ? bytes(t.Bytes) : '—'}{t.Parts?.length ? ` · ${text.parts(t.Parts.length)}` : ''}</td>
						<td>
							<span class="status {t.status}"><i></i>{t.status === 'ready' ? text.statusReady : t.status === 'preparing' ? text.statusPreparing : text.statusExpired}</span>
							{#if t.Expires && t.status !== 'expired'}<small class="until">{text.until(day.format(new Date(t.Expires)))}</small>{/if}
							{#if t.Expires && t.status === 'expired'}<small class="until">{text.since(day.format(new Date(t.Expires)))}</small>{/if}
						</td>
						<td class="act">
							{#if t.status === 'ready'}
								<SecondaryButton small onclick={() => download(t)}><Download size={14} />{text.download}</SecondaryButton>
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
		{#if list.some((t) => !t.known)}<p class="note">{text.unknownNote}</p>{/if}
	{/if}
</div>

<style>
	.page {
		padding: 32px 40px;
		display: flex;
		flex-direction: column;
		gap: 24px;
	}
	header {
		display: flex;
		align-items: center;
		justify-content: space-between;
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
	table {
		width: 100%;
		border-collapse: collapse;
		border: 1px solid var(--border);
		border-radius: 12px;
	}
	th {
		text-align: left;
		padding: 12px 16px;
		font-size: var(--text-subheadline);
		font-weight: 600;
		letter-spacing: 0.8px;
		color: var(--faint);
		border-bottom: 1px solid var(--line);
	}
	td {
		white-space: nowrap;
		padding: 12px 16px;
		border-bottom: 1px solid var(--line);
		vertical-align: middle;
		color: var(--text);
	}
	tr.expired td {
		color: var(--faint);
	}
	.name {
		display: flex;
		align-items: center;
		gap: 12px;
		color: var(--muted);
	}
	.name b {
		display: block;
		color: var(--fg);
		font-weight: 600;
	}
	small {
		display: block;
		font-family: var(--font-mono);
		font-size: var(--text-subheadline);
		color: var(--faint);
	}
	small.until {
		font-family: var(--font-ui);
		margin-left: 16px;
	}
	/* Half a screen: the table scrolls sideways rather than break its lines. */
	.page:has(table) {
		overflow-x: auto;
	}
	:global(.narrow) .page {
		padding: 24px;
	}
	.status {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		font-weight: 600;
	}
	.status i {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: currentColor;
	}
	.status.ready {
		color: var(--ok);
	}
	.status.preparing {
		color: var(--warn);
	}
	.status.expired {
		color: var(--faint);
	}
	.act {
		text-align: right;
	}
	.note {
		margin-top: -12px;
		font-size: var(--text-callout);
		color: var(--faint);
	}
	.muted {
		color: var(--muted);
	}
	.err {
		color: var(--err);
	}
</style>
