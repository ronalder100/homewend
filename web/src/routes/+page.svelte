<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- The first screen: sign in when there is no account, the library otherwise. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import SignIn from '#lib/screens/SignIn.svelte';
	import { app, refresh } from '#lib/app.svelte.js';

	let ready = $state(false);
	onMount(async () => {
		await refresh();
		if (app.overview && app.overview.accounts.length > 0) goto('/library');
		else ready = true;
	});
</script>

{#if ready}<SignIn />{/if}
