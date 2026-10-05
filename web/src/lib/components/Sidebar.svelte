<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<script lang="ts">
	import {
		Images,
		Archive,
		Settings as SettingsIcon,
		CircleCheck,
		Loader,
		Check,
		ChevronDown,
		ChevronRight
	} from '@lucide/svelte';
	import Avatar from './Avatar.svelte';
	import ContextMenu, { folderItems } from './ContextMenu.svelte';
	import { folderOf } from '#lib/api.js';
	import { Opener } from '#lib/menu.svelte.js';
	import { text } from '#lib/strings.js';
	import { FOLD, samePlace, type Place, type Sidebar } from '#lib/library.js';

	let {
		data,
		place,
		rail = false,
		onplace,
		ontoggle,
		onexpand
	}: {
		data: Sidebar;
		place: Place;
		/** Only the rows that have an icon, as a 56px column. */
		rail?: boolean;
		onplace?: (p: Place) => void;
		ontoggle?: (account: string) => void;
		/** A click anywhere on the rail opens the full sidebar instead. */
		onexpand?: () => void;
	} = $props();

	let allAccounts = $state(false);
	let allYears = $state(false);

	// Right-click on an account, a year or an album: open its folder, or copy
	// where it is.
	const opener = new Opener<{ path: string; key: string }>();
	function context(e: MouseEvent, key: string, where: { account?: string; year?: string; album?: string; noDate?: boolean }) {
		if (rail) return;
		opener.open(e, async () => ({ path: (await folderOf(where)).path, key }));
	}
	const years = $derived(allYears ? data.years : data.years.slice(0, FOLD.years));
	let folded = $state<Record<string, boolean>>({});
	let allAlbums = $state<Record<string, boolean>>({});

	const many = $derived(data.accounts.length > 1);
	const accounts = $derived(
		allAccounts ? data.accounts : data.accounts.slice(0, FOLD.accounts)
	);
	// Albums by account, in the accounts' order; only shown accounts have any.
	// Albums the library does not tie to an account form one group.
	const groups = $derived(
		data.albums.some((al) => al.account)
			? data.accounts
					.filter((a) => a.shown)
					.map((a) => ({ account: a, albums: data.albums.filter((al) => al.account === a.id) }))
					.filter((g) => g.albums.length > 0)
			: data.albums.length > 0
				? [{ account: null, albums: data.albums }]
				: []
	);
	const is = (p: Place) => samePlace(p, place);
	// On the rail a row opens the sidebar and its page; an account only opens
	// the sidebar, where its checkbox is.
	const go = (p: Place) => {
		if (rail) onexpand?.();
		onplace?.(p);
	};
	const toggle = (id: string) => (rail ? onexpand?.() : many && ontoggle?.(id));
</script>

<nav class:rail>
	<div class="scroll">
		<div class="group">
			{#each accounts as account (account.id)}
				<button
					class="row"
					title={rail ? account.name : undefined}
					aria-pressed={many ? account.shown : undefined}
					aria-label={many ? text.showAccount(account.name) : account.name}
					onclick={() => toggle(account.id)}
					oncontextmenu={(e) => context(e, 'a:' + account.id, { account: account.id })}
					class:target={opener.menu?.key === 'a:' + account.id}
				>
					<span class="lead"><Avatar {account} index={data.accounts.indexOf(account)} /></span>
					<span class="label">{account.name}</span>
					{#if many}
						<span class="check" class:on={account.shown}>
							{#if account.shown}<Check size={12} strokeWidth={3.5} />{/if}
						</span>
					{/if}
				</button>
			{/each}
			{#if data.accounts.length > FOLD.accounts && !rail}
				<button class="row more" onclick={() => (allAccounts = !allAccounts)}>
					<span class="lead"><ChevronDown size={16} /></span>
					<span class="label"
						>{allAccounts ? text.fewer : text.more(data.accounts.length - FOLD.accounts)}</span
					>
				</button>
			{/if}
		</div>

		<hr />

		{#if !rail}
			<button class="row" class:selected={is({ kind: 'all' })} onclick={() => go({ kind: 'all' })}>
				<span class="lead"><Images size={20} /></span>
				<span class="label">{text.allPhotos}</span>
				<span class="count">{data.total.toLocaleString('en')}</span>
			</button>

			<hr />

			<div class="group">
				<h2>{text.years}</h2>
				{#each years as year (year.label)}
					<button
						class="row"
						class:selected={is({ kind: 'year', label: year.label })}
						class:target={opener.menu?.key === 'y:' + year.label}
						onclick={() => go({ kind: 'year', label: year.label })}
						oncontextmenu={(e) =>
							context(e, 'y:' + year.label, year.label === text.noDate ? { noDate: true } : { year: year.label })}
					>
						<span class="label">{year.label}</span>
						<span class="count">{year.count.toLocaleString('en')}</span>
						{#if year.state === 'complete'}
							<span class="state ok"><CircleCheck size={16} /></span>
						{:else if year.state === 'arriving'}
							<span class="state arriving"><Loader size={16} /></span>
						{/if}
					</button>
				{/each}
				{#if data.years.length > FOLD.years}
					<button class="row folded" onclick={() => (allYears = !allYears)}>
						<span class="label">{allYears ? text.fewer : text.more(data.years.length - FOLD.years)}</span>
						<span class="count">—</span>
					</button>
				{/if}
			</div>

			{#if groups.length > 0}
				<hr />
				<div class="group">
					<h2>{text.albums}</h2>
					{#each groups as g (g.account?.id ?? '')}
						{@const key = g.account?.id ?? ''}
						{@const open = !folded[key]}
						{@const list = allAlbums[key] ? g.albums : g.albums.slice(0, FOLD.albums)}
						{#if g.account}
							<button
								class="owner"
								aria-expanded={open}
								onclick={() => (folded[key] = open)}
							>
								{#if open}<ChevronDown size={12} />{:else}<ChevronRight size={12} />{/if}
								{g.account.name}
							</button>
						{/if}
						{#if open}
							{#each list as album (album.name)}
								<button
									class="row"
									class:indent={!!g.account}
									class:selected={is({ kind: 'album', account: album.account, name: album.name })}
									class:target={opener.menu?.key === 'al:' + album.name}
									title={album.name}
									onclick={() => go({ kind: 'album', account: album.account, name: album.name })}
									oncontextmenu={(e) => context(e, 'al:' + album.name, { account: album.account, album: album.name })}
								>
									<span class="label">{album.name}</span>
									<span class="count">{album.count.toLocaleString('en')}</span>
								</button>
							{/each}
							{#if g.albums.length > FOLD.albums && !allAlbums[key]}
								<button
									class="row more"
									class:indent={!!g.account}
									onclick={() => (allAlbums[key] = true)}
								>
									<span class="label">{text.more(g.albums.length - FOLD.albums)}</span>
								</button>
							{/if}
						{/if}
					{/each}
				</div>
			{/if}
		{/if}
	</div>

	<div class="pinned">
		<button
			class="row"
			class:selected={is({ kind: 'takeouts' })}
			title={rail ? text.takeouts : undefined}
			onclick={() => go({ kind: 'takeouts' })}
		>
			<span class="lead"><Archive size={20} /></span>
			<span class="label">{text.takeouts}</span>
			{#if data.takeouts > 0}<span class="count">{data.takeouts}</span>{/if}
		</button>
		<button
			class="row"
			class:selected={is({ kind: 'settings' })}
			title={rail ? text.settings : undefined}
			onclick={() => go({ kind: 'settings' })}
		>
			<span class="lead"><SettingsIcon size={20} /></span>
			<span class="label">{text.settings}</span>
		</button>
	</div>
</nav>

{#if opener.menu}<ContextMenu items={folderItems(opener.menu.path)} anchor={opener.menu.anchor} onclose={opener.close} />{/if}

<style>
	nav {
		width: 248px;
		height: 100%;
		flex-shrink: 0;
		display: flex;
		flex-direction: column;
		background: var(--soft);
		border-right: 1px solid var(--line);
	}
	.scroll {
		flex: 1;
		min-height: 0;
		overflow-y: auto;
		padding: 8px;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	/* The column scrolls rather than squeeze: without this a long list
	   shrinks the 1px lines to nothing. */
	.scroll > * {
		flex-shrink: 0;
	}
	.group {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}
	hr {
		border: 0;
		height: 1px;
		margin: 0 12px;
		background: var(--line);
	}
	h2,
	.owner {
		padding: 0 12px 4px;
		font-size: var(--text-subheadline);
		font-weight: 600;
		letter-spacing: 0.8px;
		color: var(--faint);
	}
	.owner {
		height: 24px;
		padding-bottom: 0;
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: var(--text-footnote);
		text-transform: uppercase;
	}
	.row {
		height: 36px;
		flex-shrink: 0;
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 0 12px;
		border-radius: 999px;
		text-align: left;
		color: var(--text);
		font-size: var(--text-body);
	}
	.row:has(.lead) {
		padding-left: 8px;
	}
	.row:hover {
		background: var(--surface);
	}
	.row.selected {
		background: var(--line);
		color: var(--fg);
		font-weight: 600;
	}
	.row.target {
		background: var(--accent-soft);
	}
	.row.folded,
	.row.folded .count {
		color: var(--faint);
	}
	.row.indent {
		padding-left: 32px;
	}
	.row.more {
		color: var(--accent);
		font-weight: 500;
	}
	.lead {
		width: 24px;
		height: 24px;
		flex-shrink: 0;
		display: grid;
		place-items: center;
		color: var(--muted);
	}
	.label {
		flex: 1;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.count {
		color: var(--muted);
		font-size: var(--text-callout);
		font-weight: 400;
	}
	.state {
		display: grid;
	}
	.ok {
		color: var(--ok);
	}
	.arriving {
		color: var(--accent);
	}
	.check {
		width: 16px;
		height: 16px;
		flex-shrink: 0;
		display: grid;
		place-items: center;
		border-radius: 4px;
		border: 1.5px solid var(--faint);
		color: var(--on-accent);
	}
	.check.on {
		border: 0;
		background: var(--accent);
	}
	.pinned {
		padding: 8px;
		display: flex;
		flex-direction: column;
		gap: 4px;
		border-top: 1px solid var(--line);
	}

	/* The rail: the same rows, labels hidden, so nothing moves between the two. */
	.rail {
		width: 56px;
	}
	.rail .label,
	.rail .count,
	.rail .check {
		display: none;
	}
	.rail .row {
		width: 40px;
	}
</style>
