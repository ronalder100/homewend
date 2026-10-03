// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// A job's state in words, the same in the status bar and on its own screen.
import type { JobState } from './api.js';
import { text } from './strings.js';

export function headline(j: JobState): string {
	if (j.error) return text.failed;
	if (!j.running && j.of > 0 && !j.finished) return text.pausedAt(j.parts + 1, j.of);
	switch (j.stage) {
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

export function fraction(j: JobState): number {
	return j.total > 0 ? Math.min(1, j.done / j.total) : 0;
}
