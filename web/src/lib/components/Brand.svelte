<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<script lang="ts">
	import Mark from './Mark.svelte';
	import { text } from '#lib/strings.js';
</script>

<span class="brand"><Mark />{text.brand}</span>

<style>
	.brand {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		font-size: var(--text-title-3);
		font-weight: 700;
		color: var(--fg);
	}
</style>
