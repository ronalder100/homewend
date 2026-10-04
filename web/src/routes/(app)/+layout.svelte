<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- Every screen after sign-in sits in the window frame, with the sidebar and,
     while photos come home, the status bar. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import Frame from '#lib/components/Frame.svelte';
	import StatusBar from '#lib/components/StatusBar.svelte';
	import Opening from '#lib/components/Opening.svelte';
	import Banner from '#lib/components/Banner.svelte';
	import UpdateBanner from '#lib/components/UpdateBanner.svelte';
	import Titlebar from '#lib/components/Titlebar.svelte';
	import Steps from '#lib/components/Steps.svelte';
	import { app, refresh, shownJob, sidebarOf, toggleAccount, watchJobs } from '#lib/app.svelte.js';
	import { placeOf, urlOf } from '#lib/places.js';
	import { text } from '#lib/strings.js';

	let { children } = $props();

	onMount(() => {
		let stop = () => {};
		refresh().then(() => (stop = watchJobs()));
		return () => stop();
	});
	const place = $derived(placeOf(page.url));
	const job = $derived(shownJob());

	// The first run, before any photo: Choose and the download are three steps
	// of their own, without the library around them (A2–A5).
	const firstRun = $derived(
		(app.overview?.total ?? 0) === 0 && (page.url.pathname === '/choose' || page.url.pathname === '/bringing')
	);
	const chose = $derived(
		job?.job.year ? text.choseYear(job.job.year) : job ? text.choseEverything : text.stepChooseLabel
	);

	// Said once each, when it happens, in D2's words.
	let said = '';
	$effect(() => {
		const j = job?.job;
		const email = app.overview?.accounts.find((a) => a.id === job?.account)?.email ?? '';
		const now =
			j?.stage === 'first-download' && j.running
				? 'password'
				: j?.problem === 'signed-out'
					? 'signed-out'
					: j?.finished
						? 'finished'
						: j?.stage === 'ready'
							? 'ready'
							: '';
		if (!now || now === said) return;
		said = now;
		const n = (j?.years ?? []).reduce((s, y) => s + y.arrived, 0).toLocaleString('en');
		const say = {
			password: [text.notifyPassword, text.notifyPasswordBody(email)],
			'signed-out': [text.notifySignedOut, text.notifySignedOutBody(email)],
			finished: [text.notifyFinished, text.notifyFinishedBody(n, email)],
			ready: [text.notifyReady, text.notifyReadyBody(email)]
		}[now];
		if (say) window.shell?.notify(say[0], say[1]);
	});
</script>

{#snippet bar()}
	{#if job}<StatusBar account={job.account} job={job.job} ondetails={() => goto('/bringing')} />{/if}
{/snippet}

{#snippet strip()}
	<UpdateBanner />
	{#if job}
		<Banner
			account={job.account}
			email={app.overview?.accounts.find((a) => a.id === job.account)?.email ?? ''}
			job={job.job}
		/>
	{/if}
{/snippet}

{#if app.overview && firstRun}
	<div class="first">
		<Titlebar menu={false} />
		{@render strip()}
		<Steps
			steps={[text.stepSignIn, page.url.pathname === '/choose' ? text.stepChooseLabel : chose, text.stepBringHome]}
			at={page.url.pathname === '/choose' ? 1 : 2}
		/>
		<div class="body">{@render children()}</div>
	</div>
{:else if app.overview}
	<Frame
		sidebar={sidebarOf(app.overview)}
		{place}
		onplace={(p) => goto(urlOf(p))}
		ontoggle={toggleAccount}
		status={job ? bar : undefined}
		banner={strip}
	>
		{@render children()}
	</Frame>
{:else}
	<Opening />
{/if}

<style>
	.first {
		height: 100%;
		display: flex;
		flex-direction: column;
	}
	.body {
		flex: 1;
		min-height: 0;
		overflow-y: auto;
	}
</style>
