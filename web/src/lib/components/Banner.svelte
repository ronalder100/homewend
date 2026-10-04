<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- D1: when Homewend needs the person, one strip under the titlebar. -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import Strip from './Strip.svelte';
	import { KeyRound, HardDrive, CalendarX, WifiOff, CircleCheck } from '@lucide/svelte';
	import type { JobState } from '#lib/api.js';
	import { login } from '#lib/api.js';
	import { bytes } from '#lib/format.js';
	import { refresh } from '#lib/app.svelte.js';
	import { text } from '#lib/strings.js';

	import { onMount } from 'svelte';
	import { getLibrary } from '#lib/api.js';

	let { account, email, job }: { account: string; email: string; job: JobState } = $props();
	let dir = $state('');
	onMount(async () => (dir = (await getLibrary()).dir));
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
	{@const tone = kind === 'no-space' || kind === 'expired' ? 'err' : kind === 'signed-out' || kind === 'network' ? 'warn' : kind === 'finished' ? 'ok' : 'accent'}
	{@const total = job.years.reduce((n, y) => n + y.arrived, 0).toLocaleString('en')}
	{@const [title, detail] =
		kind === 'password'
			? [text.bannerPassword(email), text.bannerPasswordDesc]
			: kind === 'signed-out'
				? [text.bannerSignedOut(email), text.bannerSignedOutDesc(job.parts + 1)]
				: kind === 'no-space'
					? [text.bannerNoSpace, text.bannerNoSpaceDesc(bytes(job.need ?? 0), bytes(job.free ?? 0))]
					: kind === 'expired'
						? [text.bannerExpired(email), text.bannerExpiredDesc]
						: kind === 'network'
							? [text.bannerNetwork, text.bannerNetworkDesc]
							: [text.bannerFinished(email, total), text.bannerFinishedDesc(dir)]}
	{@const [action, onaction] =
		kind === 'signed-out'
			? [text.signInAgain, signInAgain]
			: kind === 'no-space'
				? [text.chooseAnotherFolder, chooseFolder]
				: kind === 'expired'
					? [text.newTakeoutAction, () => goto('/choose')]
					: kind === 'finished'
						? [text.openLibrary, () => ((dismissed = kind), goto('/library'))]
						: [undefined, undefined]}
	<Strip {tone} {title} {detail} {action} {onaction}>
		{#snippet icon()}
			{#if kind === 'password' || kind === 'signed-out'}<KeyRound size={18} />
			{:else if kind === 'no-space'}<HardDrive size={18} />
			{:else if kind === 'expired'}<CalendarX size={18} />
			{:else if kind === 'network'}<WifiOff size={18} />
			{:else}<CircleCheck size={18} />{/if}
		{/snippet}
	</Strip>
{/if}
