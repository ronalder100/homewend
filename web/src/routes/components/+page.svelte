<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- Every component on one page, to set beside the design's Components frame. -->
<script lang="ts">
	import { FolderOpen } from '@lucide/svelte';
	import Brand from '#lib/components/Brand.svelte';
	import Titlebar from '#lib/components/Titlebar.svelte';
	import PrimaryButton from '#lib/components/PrimaryButton.svelte';
	import SecondaryButton from '#lib/components/SecondaryButton.svelte';
	import GoogleButton from '#lib/components/GoogleButton.svelte';
</script>

<div class="sheet">
	<div class="row">
		<Brand />
		<PrimaryButton>Continue</PrimaryButton>
		<SecondaryButton><FolderOpen size={16} />Show in folder</SecondaryButton>
		<GoogleButton />
	</div>
	<div class="bar"><Titlebar /></div>
</div>

<style>
	.sheet {
		height: 100%;
		overflow: auto;
		padding: 40px;
		display: flex;
		flex-direction: column;
		gap: 32px;
		background: var(--soft);
	}
	.row {
		display: flex;
		align-items: center;
		gap: 40px;
	}
	.bar {
		border: 1px solid var(--border);
	}
</style>
