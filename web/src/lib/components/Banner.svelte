<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- D1: when Homewend needs the person, one strip under the titlebar. -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { KeyRound, LogIn, HardDrive, CalendarX, WifiOff, CircleCheck } from '@lucide/svelte';
	import type { JobState } from '#lib/api.js';
	import { login } from '#lib/api.js';
	import { bytes } from '#lib/format.js';
	import { refresh } from '#lib/app.svelte.js';
	import { text } from '#lib/strings.js';

	let { account, email, job }: { account: string; email: string; job: JobState } = $props();
	let dismissed = $state('');

	const kind = $derived(
		job.stage === 'first-download' && job.running
			? 'password'
			: job.problem === 'signed-out'
				? 'signed-out'
				: job.problem === 'no-space'
					? 'no-space'
					: job.problem === 'expired'
						? 'expired'
						: job.retry
							? 'network'
							: job.finished
								? 'finished'
								: ''
	);

	async function signInAgain() {
		await login(account);
		await refresh();
	}
	async function chooseFolder() {
		const picked = await window.shell?.chooseFolder();
		if (picked) goto('/settings');
	}
</script>

{#if kind && dismissed !== kind}
	<div class="banner {kind}">
		<span class="icon">
			{#if kind === 'password'}<KeyRound size={18} />
			{:else if kind === 'signed-out'}<LogIn size={18} />
			{:else if kind === 'no-space'}<HardDrive size={18} />
			{:else if kind === 'expired'}<CalendarX size={18} />
			{:else if kind === 'network'}<WifiOff size={18} />
			{:else}<CircleCheck size={18} />{/if}
		</span>
		<div class="text">
			{#if kind === 'password'}
				<b>{text.bannerPassword(email)}</b><span>{text.bannerPasswordDesc}</span>
			{:else if kind === 'signed-out'}
				<b>{text.bannerSignedOut(email)}</b><span>{text.bannerSignedOutDesc}</span>
			{:else if kind === 'no-space'}
				<b>{text.bannerNoSpace}</b><span>{text.bannerNoSpaceDesc(bytes(job.need ?? 0), bytes(job.free ?? 0))}</span>
			{:else if kind === 'expired'}
				<b>{text.bannerExpired}</b><span>{text.bannerExpiredDesc}</span>
			{:else if kind === 'network'}
				<b>{text.bannerNetwork}</b><span>{text.bannerNetworkDesc}</span>
			{:else}
				<b>{text.bannerFinished(job.years.reduce((n, y) => n + y.arrived, 0).toLocaleString('en'))}</b><span>{text.bannerFinishedDesc}</span>
			{/if}
		</div>
		{#if kind === 'signed-out'}
			<button class="act" onclick={signInAgain}>{text.signInAgain}</button>
		{:else if kind === 'no-space'}
			<button class="act" onclick={chooseFolder}>{text.chooseAnotherFolder}</button>
		{:else if kind === 'expired'}
			<button class="act" onclick={() => goto('/choose')}>{text.newTakeoutAction}</button>
		{:else if kind === 'finished'}
			<button class="act" onclick={() => ((dismissed = kind), goto('/library'))}>{text.openLibrary}</button>
		{/if}
	</div>
{/if}

<style>
	.banner {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px 24px;
		border-bottom: 1px solid var(--line);
		background: var(--accent-soft);
	}
	.signed-out,
	.expired {
		background: var(--err-soft);
	}
	.no-space,
	.network {
		background: var(--warn-soft);
	}
	.finished {
		background: var(--ok-soft);
	}
	.icon {
		display: grid;
		color: var(--accent);
	}
	.signed-out .icon,
	.expired .icon {
		color: var(--err);
	}
	.no-space .icon,
	.network .icon {
		color: var(--warn);
	}
	.finished .icon {
		color: var(--ok);
	}
	.text {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}
	b {
		color: var(--fg);
		font-weight: 600;
	}
	.text span {
		font-size: var(--text-callout);
		color: var(--muted);
	}
	.act {
		height: 32px;
		padding: 0 12px;
		border-radius: 8px;
		background: var(--accent);
		color: var(--on-accent);
		font-size: var(--text-callout);
		font-weight: 600;
		white-space: nowrap;
	}
</style>
