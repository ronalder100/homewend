<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- A newer Homewend, said in the window as well as by the system, which
     may have nobody to show its notifications. Nothing is downloaded until
     the person asks. -->
<script lang="ts">
	import { RefreshCw } from '@lucide/svelte';
	import { app } from '#lib/app.svelte.js';
	import { text } from '#lib/strings.js';
	import Strip from './Strip.svelte';

	function update() {
		app.updating = true;
		window.shell?.installUpdate();
	}
</script>

{#if app.update}
	<Strip
		title={text.updateAvailable(app.update)}
		detail={app.updating ? text.updating : text.updateDesc}
		action={app.updating ? undefined : text.update}
		onaction={update}
	>
		{#snippet icon()}<RefreshCw size={18} />{/snippet}
	</Strip>
{/if}
