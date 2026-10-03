<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<script lang="ts">
	import '#lib/app.css';
	import { text } from '#lib/strings.js';

	let { children } = $props();
</script>

<svelte:head>
	<title>{text.brand}</title>
</svelte:head>

{@render children()}
