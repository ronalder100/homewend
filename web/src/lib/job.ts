// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// A job's state in words, the same in the status bar and on its own screen.
import type { JobState } from './api.js';
import { bytes } from './format.js';
import { text } from './strings.js';

export function headline(j: JobState): string {
	if (j.error) return text.failed;
	if (j.finished) return text.finished;
	// Paused before any part was known, Google preparing it: no part to name.
	if (!j.running) return j.of > 0 ? text.pausedAt(j.parts + 1, j.of) : text.paused;
	switch (j.stage) {
		case 'first-download':
			return text.waitingForPassword;
		case 'checking':
		case 'prepare':
		case 'request':
			return text.checking;
		case 'waiting':
			return text.waitingForGoogle;
		case 'unpack':
		case 'place':
		case 'unassigned':
		case 'year':
			return text.sorting;
	}
	if (j.of > 0) return text.bringingPart(Math.min(j.parts + 1, j.of), j.of);
	return text.checking;
}

/** Why a job stopped, said as what the person can do. */
export function failure(j: JobState): string {
	switch (j.problem) {
		case 'signed-out':
			return text.bannerSignedOutDesc(j.parts + 1);
		case 'no-space':
			return text.bannerNoSpaceDesc(bytes(j.need ?? 0), bytes(j.free ?? 0));
		case 'expired':
			return text.bannerExpiredDesc;
	}
	return text.failedLead;
}

export function fraction(j: JobState): number {
	return j.total > 0 ? Math.min(1, j.done / j.total) : 0;
}
