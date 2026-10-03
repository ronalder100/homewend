<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- A3, A4, A5: Google preparing, the parts arriving, or paused. -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { Folder, CircleCheck } from '@lucide/svelte';
	import { app, shownJob } from '#lib/app.svelte.js';
	import { startGet, stopJob } from '#lib/api.js';
	import { bytes } from '#lib/format.js';
	import { fraction, headline } from '#lib/job.js';
	import { text } from '#lib/strings.js';
	import PrimaryButton from '#lib/components/PrimaryButton.svelte';
	import SecondaryButton from '#lib/components/SecondaryButton.svelte';

	const shown = $derived(shownJob());
	const job = $derived(shown?.job);
	const waiting = $derived(job?.stage === 'waiting');
	const paused = $derived(job && !job.running && !job.finished && !job.error && job.stage);
	const title = $derived(
		job?.finished ? text.finishedTitle : paused ? text.pausedTitle : waiting ? text.preparingTitle : text.bringingTitle
	);
	const lead = $derived(paused ? text.pausedLead : waiting ? text.preparingLead : text.bringingLead);
</script>

<div class="page">
	{#if !shown || !job}
		<p class="lead">{text.noPhotosDesc}</p>
		<PrimaryButton onclick={() => goto('/choose')}>{text.bringHome}</PrimaryButton>
	{:else}
		<div class="left">
			<span class="pill" class:err={job.error}><i></i>{headline(job)}</span>
			<h1>{title}</h1>
			<p class="lead">{job.error || lead}</p>
			{#if job.total > 0}
				<div class="track"><b style:width="{fraction(job) * 100}%"></b></div>
				<div class="numbers">
					<div><b>{bytes(job.done)}</b><span>of {bytes(job.total)}</span></div>
					<div><b>{job.parts}</b><span>of {job.of} parts</span></div>
				</div>
			{/if}
			<div class="actions">
				{#if job.running}
					<SecondaryButton onclick={() => stopJob(shown.account)}>{text.pause}</SecondaryButton>
				{:else if !job.finished}
					<PrimaryButton onclick={() => startGet({ account: shown.account, year: job.year })}>{text.resume}</PrimaryButton>
				{/if}
				{#if (app.overview?.total ?? 0) > 0}
					<button class="link" onclick={() => goto('/library')}>{text.seeLibrary} →</button>
				{/if}
			</div>
		</div>
		{#if job.years.length > 0}
			<section class="years">
				<header><h2>{text.byYear}</h2><span>{text.byYearLegend}</span></header>
				{#each job.years as y (y.year)}
					{@const done = y.of > 0 && y.arrived >= y.of}
					<div class="year">
						<span class="folder"><Folder size={18} /></span>
						<span class="name">{y.year || text.noDate}</span>
						<span class="count" class:done>{y.arrived.toLocaleString('en')} / {y.of.toLocaleString('en')}</span>
						<span class="state">
							{#if done}<CircleCheck size={18} />{:else}<span class="bar"><b style:width="{y.of ? (y.arrived / y.of) * 100 : 0}%"></b></span>{/if}
						</span>
					</div>
				{/each}
			</section>
		{/if}
	{/if}
</div>

<style>
	.page {
		height: 100%;
		display: flex;
		gap: 40px;
		padding: 64px 40px 40px;
		align-items: flex-start;
	}
	.left {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 16px;
		align-items: flex-start;
	}
	.pill {
		height: 28px;
		padding: 0 12px;
		display: inline-flex;
		align-items: center;
		gap: 8px;
		border-radius: 999px;
		background: var(--accent-soft);
		color: var(--accent);
		font-size: var(--text-callout);
		font-weight: 600;
	}
	.pill i {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: currentColor;
	}
	.pill.err {
		background: var(--err-soft);
		color: var(--err);
	}
	h1 {
		font-size: var(--text-hero);
		font-weight: 700;
		letter-spacing: -2px;
		line-height: 1.05;
		color: var(--fg);
		white-space: pre-line;
	}
	.lead {
		font-size: var(--text-title-2);
		line-height: 1.5;
		color: var(--muted);
		max-width: 560px;
	}
	.track {
		width: 100%;
		height: 8px;
		border-radius: 999px;
		background: var(--line);
		overflow: hidden;
	}
	.track b,
	.bar b {
		display: block;
		height: 100%;
		border-radius: 999px;
		background: var(--accent);
	}
	.numbers {
		display: flex;
		gap: 40px;
	}
	.numbers b {
		display: block;
		font-size: var(--text-title-1);
		font-weight: 700;
		color: var(--fg);
	}
	.numbers span {
		font-size: var(--text-callout);
		color: var(--muted);
	}
	.actions {
		display: flex;
		align-items: center;
		gap: 24px;
	}
	.link {
		color: var(--accent);
		font-weight: 600;
	}
	.years {
		width: 380px;
		flex-shrink: 0;
		max-height: 100%;
		overflow-y: auto;
		padding: 24px;
		border: 1px solid var(--border);
		border-radius: 16px;
	}
	.years header {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		padding-bottom: 12px;
	}
	.years h2 {
		font-size: var(--text-title-2);
		font-weight: 700;
		color: var(--fg);
	}
	.years header span {
		font-size: var(--text-callout);
		color: var(--faint);
	}
	.year {
		height: 44px;
		display: flex;
		align-items: center;
		gap: 12px;
		border-bottom: 1px solid var(--line);
	}
	.folder {
		display: grid;
		color: var(--folder);
	}
	.name {
		flex: 1;
		font-weight: 500;
		color: var(--fg);
	}
	.count {
		color: var(--text);
	}
	.count.done {
		color: var(--ok);
		font-weight: 600;
	}
	.state {
		width: 40px;
		display: flex;
		justify-content: flex-end;
		color: var(--ok);
	}
	.bar {
		width: 40px;
		height: 4px;
		border-radius: 999px;
		background: var(--line);
		overflow: hidden;
	}
</style>
