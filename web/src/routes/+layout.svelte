<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<script lang="ts">
	import '#lib/app.css';
	import { onMount } from 'svelte';
	import { text } from '#lib/strings.js';
	import { loadTheme } from '#lib/theme.svelte.js';
	import { app } from '#lib/app.svelte.js';

	let { children } = $props();

	onMount(() => {
		loadTheme();
		window.shell?.onUpdate((v) => {
			app.update = v;
			window.shell?.announceUpdate(v, text.updateAvailable(v), text.updateAvailableBody);
		});
	});
</script>

<svelte:head>
	<title>{text.brand}</title>
</svelte:head>

{@render children()}
