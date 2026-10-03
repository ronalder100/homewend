<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<script lang="ts">
	import { text } from '#lib/strings.js';

	let { onclick }: { onclick?: () => void } = $props();
</script>

<!-- Google's "G" as Google's sign-in branding draws it, on a white tile. -->
<button {onclick}>
	<span class="tile">
		<svg width="20" height="20" viewBox="0 0 48 48" aria-hidden="true">
			<path
				fill="#EA4335"
				d="M24 9.5c3.54 0 6.71 1.22 9.21 3.6l6.85-6.85C35.9 2.38 30.47 0 24 0 14.62 0 6.51 5.38 2.56 13.22l7.98 6.19C12.43 13.72 17.74 9.5 24 9.5z"
			/>
			<path
				fill="#4285F4"
				d="M46.98 24.55c0-1.57-.15-3.09-.38-4.55H24v9.02h12.94c-.58 2.96-2.26 5.48-4.78 7.18l7.73 6c4.51-4.18 7.09-10.36 7.09-17.65z"
			/>
			<path
				fill="#FBBC05"
				d="M10.53 28.59c-.48-1.45-.76-2.99-.76-4.59s.27-3.14.76-4.59l-7.98-6.19C.92 16.46 0 20.12 0 24c0 3.88.92 7.54 2.56 10.78l7.97-6.19z"
			/>
			<path
				fill="#34A853"
				d="M24 48c6.48 0 11.93-2.13 15.89-5.81l-7.73-6c-2.15 1.45-4.92 2.3-8.16 2.3-6.26 0-11.57-4.22-13.47-9.91l-7.98 6.19C6.51 42.62 14.62 48 24 48z"
			/>
		</svg>
	</span>
	{text.signInButton}
</button>

<style>
	button {
		height: 52px;
		padding: 0 24px 0 8px;
		display: inline-flex;
		align-items: center;
		gap: 16px;
		border-radius: 10px;
		background: var(--accent);
		color: var(--on-accent);
		box-shadow:
			inset 0 1px 0 #ffffff2e,
			0 1px 2px #0a0a1a26;
		font-size: var(--text-title-3);
		font-weight: 600;
		letter-spacing: -0.1px;
		white-space: nowrap;
	}
	button:hover {
		filter: brightness(1.06);
	}
	.tile {
		width: 36px;
		height: 36px;
		display: grid;
		place-items: center;
		border-radius: 6px;
		background: #ffffff;
	}
</style>
