<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- Asked once, after a new account signs in: whose photos it holds, the
     folder of the library they go in. The address's name is proposed. -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import { getLibrary, putProfile } from '#lib/api.js';
	import { app, refresh } from '#lib/app.svelte.js';
	import { text } from '#lib/strings.js';
	import PrimaryButton from '#lib/components/PrimaryButton.svelte';

	const id = page.url.searchParams.get('account') ?? '';
	const account = $derived(app.overview?.accounts.find((a) => a.id === id));
	let name = $state('');
	let dir = $state('');
	let failed = $state('');

	onMount(() => {
		getLibrary().then((l) => (dir = l.dir), () => {});
	});
	$effect(() => {
		if (!name && account) name = account.profile;
	});
	// As the engine keeps it: one name, a folder, no separators.
	const sep = $derived(dir.includes('\\') ? '\\' : '/');

	async function save(e: SubmitEvent) {
		e.preventDefault();
		failed = '';
		try {
			await putProfile(id, name.trim());
			await refresh();
			goto('/choose', { replaceState: true });
		} catch {
			failed = text.badProfile;
		}
	}
</script>

<form class="page" onsubmit={save}>
	<h1>{text.askProfile}</h1>
	<p><b>{account?.email ?? ''}</b><br />{text.askProfileLead}</p>
	<label>
		<span>{text.askProfileField}</span>
		<!-- svelte-ignore a11y_autofocus -->
		<input bind:value={name} autofocus spellcheck="false" autocomplete="off" />
	</label>
	<p class="where" class:err={failed}>
		{failed || (dir ? `${dir}${dir.endsWith(sep) ? '' : sep}${name.trim()}` : '')}
	</p>
	<PrimaryButton>{text.continue}</PrimaryButton>
</form>

<style>
	.page {
		height: 100%;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 16px;
		padding: 40px;
		text-align: center;
	}
	h1 {
		font-size: var(--text-large-title);
		font-weight: 700;
		color: var(--fg);
	}
	p {
		color: var(--muted);
		max-width: 420px;
	}
	p b {
		color: var(--fg);
		font-weight: 600;
	}
	label {
		width: 320px;
		display: flex;
		flex-direction: column;
		gap: 8px;
		margin-top: 8px;
		text-align: left;
	}
	label span {
		font-size: var(--text-callout);
		font-weight: 600;
		color: var(--fg);
	}
	input {
		height: 40px;
		padding: 0 12px;
		border: 1px solid var(--border);
		border-radius: 8px;
		background: var(--surface);
		color: var(--fg);
		font: inherit;
		user-select: text;
	}
	input:focus {
		outline: 3px solid var(--accent-soft);
		border-color: var(--accent);
	}
	.where {
		font-family: var(--font-mono);
		font-size: var(--text-callout);
	}
	.where.err {
		font-family: inherit;
		color: var(--err);
	}
</style>
