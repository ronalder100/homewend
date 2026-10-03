<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- Before the first photo arrives: Apple Photos' empty state. -->
<script lang="ts">
	import { Images } from '@lucide/svelte';
	import { text } from '#lib/strings.js';
</script>

<div class="empty">
	<Images size={40} strokeWidth={1.5} />
	<h2>{text.noPhotos}</h2>
	<p>{text.noPhotosDesc}</p>
</div>

<style>
	.empty {
		height: 100%;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 8px;
		padding: 40px;
		text-align: center;
		color: var(--faint);
	}
	h2 {
		font-size: var(--text-title-2);
		font-weight: 600;
		color: var(--muted);
	}
	p {
		color: var(--muted);
	}
</style>
