<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<script lang="ts">
	import { Monitor, Sun, Moon } from '@lucide/svelte';
	import { text } from '#lib/strings.js';
	import { theme, setTheme } from '#lib/theme.svelte.js';
	import type { Theme } from '#lib/api.js';

	const choices: { value: Theme; label: string; icon: typeof Sun }[] = [
		{ value: 'system', label: text.themeSystem, icon: Monitor },
		{ value: 'light', label: text.themeLight, icon: Sun },
		{ value: 'dark', label: text.themeDark, icon: Moon }
	];
</script>

<div class="page">
	<h1>{text.settingsTitle}</h1>
	<section>
		<h2>{text.appearance}</h2>
		<div class="card">
			<div class="item">
				<div class="text">
					<h3>{text.theme}</h3>
					<p>{text.themeDesc}</p>
				</div>
				<div class="segments" role="radiogroup" aria-label={text.theme}>
					{#each choices as c (c.value)}
						<button
							role="radio"
							aria-checked={theme.value === c.value}
							class:on={theme.value === c.value}
							onclick={() => setTheme(c.value)}><c.icon size={14} />{c.label}</button
						>
					{/each}
				</div>
			</div>
		</div>
	</section>
</div>

<style>
	.page {
		padding: 32px 40px;
		display: flex;
		flex-direction: column;
		gap: 32px;
	}
	h1 {
		font-size: var(--text-display);
		font-weight: 700;
		color: var(--fg);
	}
	section {
		width: 720px;
		max-width: 100%;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	h2 {
		font-size: var(--text-subheadline);
		font-weight: 600;
		letter-spacing: 0.8px;
		color: var(--faint);
	}
	.card {
		border: 1px solid var(--border);
		border-radius: 12px;
	}
	.item {
		display: flex;
		align-items: center;
		gap: 24px;
		padding: 16px 24px;
	}
	.text {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}
	h3 {
		font-size: var(--text-body);
		font-weight: 600;
		color: var(--fg);
	}
	p {
		font-size: var(--text-callout);
		color: var(--muted);
	}
	.segments {
		display: flex;
		gap: 2px;
		padding: 2px;
		border-radius: 8px;
		background: var(--soft);
		border: 1px solid var(--border);
	}
	.segments button {
		height: 28px;
		padding: 0 12px;
		display: flex;
		align-items: center;
		gap: 6px;
		border-radius: 6px;
		font-size: var(--text-callout);
		font-weight: 500;
		color: var(--text);
	}
	.segments button :global(svg) {
		color: var(--muted);
	}
	.segments button.on {
		background: var(--surface);
		box-shadow: 0 1px 2px #0a0a1a1f;
		color: var(--fg);
		font-weight: 600;
	}
	.segments button.on :global(svg) {
		color: var(--fg);
	}
</style>
