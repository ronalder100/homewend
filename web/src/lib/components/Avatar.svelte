<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<script lang="ts">
	import { initials, type Account } from '#lib/library.js';

	let { account, index = 0, size = 24 }: { account: Account; index?: number; size?: number } =
		$props();

	// Accounts are told apart by colour, taken in order from the theme.
	const colours = ['--accent', '--fg', '--ok', '--warn', '--err', '--folder', '--chip'];
	const colour = $derived(`var(${colours[index % colours.length]})`);
</script>

<span
	class="avatar"
	style:width="{size}px"
	style:height="{size}px"
	style:background={colour}
	title={account.email}>{initials(account.name)}</span
>

<style>
	.avatar {
		border-radius: 50%;
		display: grid;
		place-items: center;
		flex-shrink: 0;
		color: var(--on-accent);
		font-size: 9px;
		font-weight: 600;
	}
</style>
