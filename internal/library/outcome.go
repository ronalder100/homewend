// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

// Outcome is how a look or a copy of a folder ends, said once here so that
// the command line and the window, a look and its copy, never tell the
// same folder apart. They only put it in words.
type Outcome string

const (
	OutcomeNew        Outcome = "new"         // something comes in
	OutcomeRedated    Outcome = "redated"     // nothing new; photos already in move to a date given
	OutcomeLeftOut    Outcome = "left-out"    // nothing new by itself; folders stay out, new or not
	OutcomeDeleted    Outcome = "deleted"     // nothing new; what was deleted from the library stays out
	OutcomeKnownNoted Outcome = "known-noted" // all read is in already; some could not be read
	OutcomeKnown      Outcome = "known"       // every photo and video is in already
	OutcomeUnread     Outcome = "unread"      // nothing could be read
	OutcomeEmpty      Outcome = "empty"       // no photo or video at all
)

// outcome decides it, in this order: what comes in, what moves, the
// folders to look at, the deleted, what is in, what could not be read.
func outcome(fresh, redated int, leftOut bool, deleted, known, unreadable int) Outcome {
	switch {
	case fresh > 0:
		return OutcomeNew
	case redated > 0:
		return OutcomeRedated
	case leftOut:
		return OutcomeLeftOut
	case deleted > 0:
		return OutcomeDeleted
	case known > 0 && unreadable > 0:
		return OutcomeKnownNoted
	case known > 0:
		return OutcomeKnown
	case unreadable > 0:
		return OutcomeUnread
	}
	return OutcomeEmpty
}
