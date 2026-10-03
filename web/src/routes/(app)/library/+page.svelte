<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<script lang="ts">
	import { app } from '#lib/app.svelte.js';
	import EmptyLibrary from '#lib/screens/EmptyLibrary.svelte';
</script>

{#if app.overview && app.overview.total === 0}
	<EmptyLibrary />
{/if}
