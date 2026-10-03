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
	import { app, refresh, shownJob, sidebarOf, toggleAccount, watchJobs } from '#lib/app.svelte.js';
	import { placeOf, urlOf } from '#lib/places.js';

	let { children } = $props();

	onMount(() => {
		let stop = () => {};
		refresh().then(() => (stop = watchJobs()));
		return () => stop();
	});
	const place = $derived(placeOf(page.url));
	const job = $derived(shownJob());
</script>

{#snippet bar()}
	{#if job}<StatusBar account={job.account} job={job.job} ondetails={() => goto('/bringing')} />{/if}
{/snippet}

{#if app.overview}
	<Frame
		sidebar={sidebarOf(app.overview)}
		{place}
		onplace={(p) => goto(urlOf(p))}
		ontoggle={toggleAccount}
		status={job ? bar : undefined}
	>
		{@render children()}
	</Frame>
{/if}
