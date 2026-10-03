<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- The download, in one line under every screen of the library. -->
<script lang="ts">
	import { Download, ChevronUp, Pause, Play, CircleAlert } from '@lucide/svelte';
	import type { JobState } from '#lib/api.js';
	import { stopJob, startGet } from '#lib/api.js';
	import { bytes } from '#lib/format.js';
	import { headline, fraction } from '#lib/job.js';
	import { text } from '#lib/strings.js';

	let { account, job, ondetails }: { account: string; job: JobState; ondetails?: () => void } =
		$props();
</script>

<span class="icon" class:err={job.error}>
	{#if job.error}<CircleAlert size={16} />{:else}<Download size={16} />{/if}
</span>
<span class="what">{headline(job)}</span>
{#if job.total > 0}
	<span class="track"><b style:width="{fraction(job) * 100}%"></b></span>
	<span class="numbers">{text.gbOf(bytes(job.done), bytes(job.total))}</span>
{/if}
{#if job.retry}<span class="note">{text.retrying(job.retry)}</span>{/if}
{#if job.error}<span class="note err" title={job.error}>{job.error}</span>{/if}
<span class="spacer"></span>
<button class="details" onclick={ondetails}>{text.details}<ChevronUp size={14} /></button>
{#if job.running}
	<button class="ctl" aria-label={text.pause} onclick={() => stopJob(account)}><Pause size={16} /></button>
{:else if !job.finished}
	<button class="ctl" aria-label={text.resume} onclick={() => startGet({ account, year: job.year })}
		><Play size={16} /></button
	>
{/if}

<style>
	.icon {
		display: grid;
		color: var(--accent);
	}
	.icon.err {
		color: var(--err);
	}
	.what {
		font-weight: 600;
		color: var(--fg);
		white-space: nowrap;
	}
	.track {
		width: 160px;
		height: 8px;
		flex-shrink: 0;
		border-radius: 999px;
		background: var(--line);
		overflow: hidden;
	}
	.track b {
		display: block;
		height: 100%;
		border-radius: 999px;
		background: var(--accent);
	}
	.numbers {
		color: var(--muted);
		white-space: nowrap;
	}
	.note {
		color: var(--faint);
		font-size: var(--text-callout);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		min-width: 0;
	}
	.note.err {
		color: var(--err);
	}
	.spacer {
		flex: 1;
	}
	.details {
		display: flex;
		align-items: center;
		gap: 4px;
		color: var(--accent);
		font-weight: 600;
	}
	.ctl {
		display: grid;
		color: var(--muted);
	}
</style>
