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

// Counts are what a look and its copy both count of a folder, and how it
// ends: one set, so that the two count and say it alike.
type Counts struct {
	// Already in the library, found by content: nothing copied.
	InLibrary int `json:"in_library"`
	// The same photo twice in the folder: brought in once.
	Twins int `json:"twins"`
	// Moved to a date given, new or already in.
	Redated int `json:"redated"`
	// Given a date but filed in another profile, whose date it is to give.
	NotRedated int `json:"not_redated"`
	// Photos the library had and someone deleted from it; they come in
	// again only when BroughtBack.
	Deleted     int  `json:"deleted"`
	BroughtBack bool `json:"brought_back,omitempty"`
	// Files and folders that could not be read, passed over.
	Unreadable int `json:"unreadable"`
	// How it ends, for every screen and line that says it.
	Outcome Outcome `json:"outcome"`
}

// end decides the Outcome: fresh is what comes in, leftOut whether groups
// of the folder stay out.
func (c *Counts) end(fresh int, leftOut bool) {
	stayedOut := c.Deleted
	if c.BroughtBack {
		stayedOut = 0
	}
	c.Outcome = outcome(fresh, c.Redated, leftOut, stayedOut, c.InLibrary+c.Twins, c.Unreadable)
}

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
