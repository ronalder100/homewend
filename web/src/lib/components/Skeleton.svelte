<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- Rows that shimmer while the engine asks Google: the window is working,
     not stuck. -->
<script lang="ts">
	let { rows = 3, height = 56, label }: { rows?: number; height?: number; label: string } = $props();
</script>

<div class="skeleton" role="status" aria-label={label}>
	<p>{label}</p>
	{#each Array(rows) as _, i (i)}<div class="bar" style:height="{height}px"></div>{/each}
</div>

<style>
	.skeleton {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	p {
		color: var(--muted);
		margin-bottom: 4px;
	}
	.bar {
		border-radius: 12px;
		background: linear-gradient(90deg, var(--soft) 0%, var(--line) 50%, var(--soft) 100%);
		background-size: 200% 100%;
		animation: shimmer 1.4s ease-in-out infinite;
	}
	@keyframes shimmer {
		from {
			background-position: 100% 0;
		}
		to {
			background-position: -100% 0;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.bar {
			animation: none;
		}
	}
</style>
