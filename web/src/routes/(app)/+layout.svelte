<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- Every screen after sign-in sits in the window frame, with the sidebar. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import Frame from '#lib/components/Frame.svelte';
	import { app, refresh, sidebarOf, toggleAccount } from '#lib/app.svelte.js';
	import { placeOf, urlOf } from '#lib/places.js';

	let { children } = $props();

	onMount(refresh);
	const place = $derived(placeOf(page.url));
</script>

{#if app.overview}
	<Frame
		sidebar={sidebarOf(app.overview)}
		{place}
		onplace={(p) => goto(urlOf(p))}
		ontoggle={toggleAccount}
	>
		{@render children()}
	</Frame>
{/if}
