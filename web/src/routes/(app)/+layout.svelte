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

	// Said once each, when it happens: Google asking for the password, and
	// the photos being home.
	let said = '';
	$effect(() => {
		const j = job?.job;
		const email = app.overview?.accounts.find((a) => a.id === job?.account)?.email ?? '';
		const now = j?.stage === 'first-download' && j.running ? 'password' : j?.finished ? 'finished' : '';
		if (!now || now === said) return;
		said = now;
		if (now === 'password') window.shell?.notify(text.notifyPassword, text.bannerPassword(email));
		else window.shell?.notify(text.notifyFinished, email);
	});
</script>

{#snippet bar()}
	{#if job}<StatusBar account={job.account} job={job.job} ondetails={() => goto('/bringing')} />{/if}
{/snippet}

{#snippet strip()}
	{#if job}
		<Banner
			account={job.account}
			email={app.overview?.accounts.find((a) => a.id === job.account)?.email ?? ''}
			job={job.job}
		/>
	{/if}
{/snippet}

{#if app.overview}
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
