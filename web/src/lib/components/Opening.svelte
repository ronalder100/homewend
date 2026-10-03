<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- While the engine reads the accounts and the library: the mark, breathing. -->
<script lang="ts">
	import Mark from './Mark.svelte';
</script>

<div class="opening"><Mark size={48} /></div>

<style>
	.opening {
		height: 100%;
		display: grid;
		place-items: center;
		animation: breathe 1.6s ease-in-out infinite;
	}
	@keyframes breathe {
		50% {
			opacity: 0.35;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.opening {
			animation: none;
		}
	}
</style>
