<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- C4: where the photos go, how the window looks, each account's cookies. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { Monitor, Sun, Moon, Trash2, Code } from '@lucide/svelte';
	import { bytes } from '#lib/format.js';
	import { text } from '#lib/strings.js';
	import { theme, setTheme } from '#lib/theme.svelte.js';
	import { app, refresh, sidebarOf } from '#lib/app.svelte.js';
	import { getLibrary, getVersion, logout, putLibrary, type Theme } from '#lib/api.js';
	import SecondaryButton from '#lib/components/SecondaryButton.svelte';
	import Avatar from '#lib/components/Avatar.svelte';

	const choices: { value: Theme; label: string; icon: typeof Sun }[] = [
		{ value: 'system', label: text.themeSystem, icon: Monitor },
		{ value: 'light', label: text.themeLight, icon: Sun },
		{ value: 'dark', label: text.themeDark, icon: Moon }
	];

	let dir = $state('');
	let free = $state(0);
	let version = $state('');
	const accounts = $derived(app.overview ? sidebarOf(app.overview).accounts : []);

	onMount(async () => {
		({ dir, free } = await getLibrary());
		version = (await getVersion()).version;
	});

	async function change() {
		const picked = await window.shell?.chooseFolder(dir);
		if (picked) {
			await putLibrary(picked);
			({ dir, free } = await getLibrary());
		}
		await refresh();
	}

	async function purge(id: string) {
		await logout(id);
		await refresh();
		if ((app.overview?.accounts.length ?? 0) === 0) goto('/');
	}
</script>

<div class="page">
	<h1>{text.settingsTitle}</h1>

	<section>
		<h2>{text.library}</h2>
		<div class="card">
			<div class="item">
				<div class="text">
					<h3>{text.whereTheyGo}</h3>
					<p class="mono">{dir || text.noLibraryYet}</p>
				</div>
				{#if dir && free}<span class="free">{text.free(bytes(free))}</span>{/if}
				<SecondaryButton onclick={change}>{dir ? text.change : text.chooseFolder}</SecondaryButton>
			</div>
		</div>
	</section>

	<section>
		<h2>{text.appearance}</h2>
		<div class="card">
			<div class="item">
				<div class="text">
					<h3>{text.theme}</h3>
					<p>{text.themeDesc}</p>
				</div>
				<div class="segments" role="radiogroup" aria-label={text.theme}>
					{#each choices as c (c.value)}
						<button
							role="radio"
							aria-checked={theme.value === c.value}
							class:on={theme.value === c.value}
							onclick={() => setTheme(c.value)}><c.icon size={14} />{c.label}</button
						>
					{/each}
				</div>
			</div>
		</div>
	</section>

	<section>
		<h2>{text.privacy}</h2>
		<div class="card">
			<div class="item">
				<div class="text">
					<h3>{text.googleCookies}</h3>
					<p>{text.googleCookiesDesc}</p>
				</div>
			</div>
			{#each accounts as a, i (a.id)}
				<div class="item account">
					<Avatar account={a} index={i} />
					<div class="text">
						<h3>{a.email}</h3>
					</div>
					<button class="danger" onclick={() => purge(a.id)}><Trash2 size={14} />{text.purge}</button>
				</div>
			{/each}
		</div>
	</section>

	<section>
		<h2>{text.about}</h2>
		<div class="card">
			<div class="item">
				<div class="text">
					<h3>Homewend {version}</h3>
					<p>{text.aboutDesc}</p>
				</div>
				<a class="link" href="https://github.com/ronalder100/homewend" target="_blank" rel="noreferrer"
					><Code size={14} />{text.sourceCode}</a
				>
			</div>
		</div>
	</section>
</div>

<style>
	:global(.narrow) .page {
		padding: 24px;
	}
	.page {
		padding: 32px 40px 40px;
		display: flex;
		flex-direction: column;
		gap: 32px;
	}
	h1 {
		font-size: var(--text-display);
		font-weight: 700;
		color: var(--fg);
	}
	section {
		width: 720px;
		max-width: 100%;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	h2 {
		font-size: var(--text-subheadline);
		font-weight: 600;
		letter-spacing: 0.8px;
		color: var(--faint);
	}
	.card {
		border: 1px solid var(--border);
		border-radius: 12px;
	}
	/* A narrow window puts the control under its words, instead of squeezing
	   the words into a column. */
	.item {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 12px 24px;
		padding: 16px 24px;
	}
	.item + .item {
		border-top: 1px solid var(--line);
	}
	.item.account {
		gap: 12px;
		padding: 12px 24px;
	}
	.text {
		flex: 1 1 240px;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}
	h3 {
		font-size: var(--text-body);
		font-weight: 600;
		color: var(--fg);
	}
	p {
		font-size: var(--text-callout);
		line-height: 1.5;
		color: var(--muted);
	}
	.mono {
		font-family: var(--font-mono);
		overflow-wrap: anywhere;
		user-select: text;
	}
	.segments {
		display: flex;
		gap: 2px;
		padding: 2px;
		border-radius: 8px;
		background: var(--soft);
		border: 1px solid var(--border);
	}
	.segments button {
		height: 28px;
		padding: 0 12px;
		display: flex;
		align-items: center;
		gap: 6px;
		border-radius: 6px;
		font-size: var(--text-callout);
		font-weight: 500;
		color: var(--text);
	}
	.segments button :global(svg) {
		color: var(--muted);
	}
	.segments button.on {
		background: var(--surface);
		box-shadow: 0 1px 2px #0a0a1a1f;
		color: var(--fg);
		font-weight: 600;
	}
	.segments button.on :global(svg) {
		color: var(--fg);
	}
	.danger,
	.link {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		font-size: var(--text-callout);
		font-weight: 600;
		text-decoration: none;
		white-space: nowrap;
	}
	.danger {
		height: 40px;
		padding: 0 16px;
		border: 1px solid var(--border);
		border-radius: 12px;
		color: var(--err);
	}
	.free {
		color: var(--ok);
		font-size: var(--text-callout);
	}
	.link {
		height: 40px;
		padding: 0 16px;
		border: 1px solid var(--border);
		border-radius: 12px;
		color: var(--accent);
	}
</style>
