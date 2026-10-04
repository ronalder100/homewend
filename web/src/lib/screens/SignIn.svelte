<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<script lang="ts">
	import { LogIn, ListChecks, HardDriveDownload } from '@lucide/svelte';
	import Titlebar from '#lib/components/Titlebar.svelte';
	import GoogleButton from '#lib/components/GoogleButton.svelte';
	import Mark from '#lib/components/Mark.svelte';
	import { text } from '#lib/strings.js';
	import { goto } from '$app/navigation';
	import { login } from '#lib/api.js';
	import { app, refresh } from '#lib/app.svelte.js';

	let waiting = $state(false);
	let failed = $state('');

	async function signIn() {
		waiting = true;
		failed = '';
		try {
			const a = await login();
			await refresh();
			// A new account is asked, once, whose photos it holds.
			goto(app.settings.profiles?.[a.id] ? '/choose' : `/profile?account=${encodeURIComponent(a.id)}`);
		} catch (e) {
			failed = String(e);
		} finally {
			waiting = false;
		}
	}

	const steps = [
		{ icon: LogIn, title: text.stepSignIn, desc: text.stepSignInDesc },
		{ icon: ListChecks, title: text.stepChoose, desc: text.stepChooseDesc },
		{ icon: HardDriveDownload, title: text.stepBringHome, desc: text.stepBringHomeDesc }
	];
</script>

<div class="window">
	<Titlebar menu={false} />
	<main>
		<div class="intro">
			<Mark size={48} />
			<h1>{text.signInTitle}</h1>
			<p class="lead">{text.signInLead}</p>
		</div>

		<section class="action">
			<div>
				<h2>{text.signInCall}</h2>
				<p class="sub" class:err={failed}>
					{failed || (waiting ? text.signingIn : text.signInCallSub)}
				</p>
			</div>
			<GoogleButton {waiting} onclick={signIn} />
		</section>

		<ol class="steps">
			{#each steps as step (step.title)}
				<li>
					<step.icon size={20} />
					<div>
						<h3>{step.title}</h3>
						<p>{step.desc}</p>
					</div>
				</li>
			{/each}
		</ol>
	</main>
</div>

<style>
	.window {
		height: 100%;
		display: flex;
		flex-direction: column;
	}
	main {
		flex: 1;
		overflow: auto;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 40px;
		padding: 80px;
	}
	/* A laptop's 768 lines leave less than the design's 800: the margins
	   give way before anything has to scroll. */
	@media (max-height: 760px) {
		main {
			padding: 40px;
		}
	}
	.intro,
	.action,
	.steps {
		width: 720px;
		max-width: 100%;
	}
	.intro {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 16px;
		text-align: center;
	}
	.intro :global(svg) {
		margin-bottom: 8px;
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
	}
	.action {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 24px;
		padding: 32px 40px;
		background: var(--soft);
		border: 1px solid var(--border);
		border-radius: 16px;
	}
	h2 {
		font-size: var(--text-title-2);
		font-weight: 700;
		letter-spacing: -0.6px;
		color: var(--fg);
	}
	.sub {
		margin-top: 4px;
		color: var(--muted);
	}
	.sub.err {
		color: var(--err);
	}
	.steps {
		list-style: none;
		display: flex;
		gap: 24px;
		padding: 24px 40px 0;
		border-top: 1px solid var(--line);
	}
	.steps li {
		flex: 1;
		display: flex;
		gap: 12px;
		color: var(--accent);
	}
	h3 {
		font-size: var(--text-body);
		font-weight: 600;
		color: var(--fg);
	}
	.steps p {
		margin-top: 4px;
		font-size: var(--text-callout);
		line-height: 1.4;
		color: var(--muted);
	}
</style>
