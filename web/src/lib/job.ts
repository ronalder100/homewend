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
			return text.checking;
		case 'session-ready':
			return text.settingUpAccount;
		case 'prepare':
			return text.settingUp;
		case 'request':
			return text.asking;
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
		case 'not-offered':
			return text.notOfferedLead([...(j.offered ?? [])].sort((a, b) => b - a));
	}
	return text.failedLead;
}

/** A wait on Google or on Homewend with nothing to count yet: it turns and
 *  says "please wait", as the command line does. Not the password, which is
 *  the person's, nor Google preparing, hours long with a screen of its own. */
export function busy(j: JobState): boolean {
	return j.running && !j.retry && !['first-download', 'sign-in', 'waiting', 'download', 'receiving'].includes(j.stage ?? '');
}

/** Photos of this download are in the folder: the ones there before are
 *  not "just arrived". */
export function arrived(j: JobState): boolean {
	return (j.years ?? []).some((y) => y.arrived > 0);
}

export function fraction(j: JobState): number {
	return j.total > 0 ? Math.min(1, j.done / j.total) : 0;
}
