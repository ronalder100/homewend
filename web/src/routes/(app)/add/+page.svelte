<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- C2: another Google account, signed in in the system browser like the first. -->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { login } from '#lib/api.js';
	import { app, refresh } from '#lib/app.svelte.js';
	import { text } from '#lib/strings.js';
	import GoogleButton from '#lib/components/GoogleButton.svelte';

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
</script>

<div class="page">
	<h1>{text.addAccount}</h1>
	<p class:err={failed}>{failed || (waiting ? text.signingIn : text.signInCallSub)}</p>
	<GoogleButton {waiting} onclick={signIn} />
</div>

<style>
	.page {
		height: 100%;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 16px;
		padding: 40px;
		text-align: center;
	}
	h1 {
		font-size: var(--text-large-title);
		font-weight: 700;
		color: var(--fg);
	}
	p {
		color: var(--muted);
		max-width: 420px;
	}
	p.err {
		color: var(--err);
	}
</style>
