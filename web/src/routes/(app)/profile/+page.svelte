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
	// The address's name is proposed once; the person may clear it.
	let proposed = false;
	$effect(() => {
		if (!proposed && account) {
			name = account.profile;
			proposed = true;
		}
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

<!-- C5: the dialog of "Add a Google account", its next step. -->
<div class="page">
	<form class="dialog" onsubmit={save}>
		<div class="head">
			<h1>{text.askProfile}</h1>
			<p>{text.askProfileLead(account?.email ?? '')}</p>
		</div>
		<label>
			<span>{text.askProfileField}</span>
			<!-- svelte-ignore a11y_autofocus -->
			<input bind:value={name} autofocus spellcheck="false" autocomplete="off" />
			<small class:err={failed}>{failed || (dir ? `${dir}${dir.endsWith(sep) ? '' : sep}${name.trim()}` : '')}</small>
		</label>
		<PrimaryButton>{text.continue}</PrimaryButton>
	</form>
</div>

<style>
	.page {
		height: 100%;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 40px;
	}
	.dialog {
		width: 440px;
		display: flex;
		flex-direction: column;
		gap: 24px;
		padding: 32px;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 16px;
		box-shadow: 0 24px 64px #0a0a1a33;
	}
	.head {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	h1 {
		font-size: var(--text-title-2);
		font-weight: 700;
		letter-spacing: -0.2px;
		color: var(--fg);
	}
	p {
		color: var(--muted);
		line-height: 1.5;
	}
	label {
		display: flex;
		flex-direction: column;
		gap: 8px;
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
		outline: none;
		border-color: var(--accent);
	}
	small {
		font-family: var(--font-mono);
		font-size: var(--text-callout);
		color: var(--muted);
		overflow-wrap: anywhere;
	}
	small.err {
		font-family: inherit;
		color: var(--err);
	}
	.dialog :global(button) {
		width: 100%;
	}
</style>
