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

// OutGroup is a group of the folder that stays out, and how many files it
// holds.
type OutGroup struct {
	Group
	Files int `json:"files"`
}

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
	// Photos the library had and someone deleted from it: those that stay
	// out, and those brought back, counted as new too.
	Deleted     int `json:"deleted"`
	BroughtBack int `json:"brought_back"`
	// Those that stay out by their path below the folder, MaxOthers at
	// most, and how many more: a person knows which photo they deleted when
	// they see its name, and brings it back by it — by the path, which no
	// other file of a card with two cameras shares.
	DeletedNames []string `json:"deleted_names,omitempty"`
	DeletedMore  int      `json:"deleted_more,omitempty"`
	// Files and folders that could not be read, passed over.
	Unreadable int `json:"unreadable"`
	// The groups that stay out, by their kind or as chosen: said, so that
	// no photo stays out unseen. A look's Groups hold them too, among all,
	// for the window's table; this list is what a look and its copy share,
	// for the end and the command line.
	LeftOut []OutGroup `json:"left_out,omitempty"`
	// How it ends, for every screen and line that says it.
	Outcome Outcome `json:"outcome"`
}

// deleted counts a photo deleted from the library, path being below root:
// brought back, or left out by its path.
func (c *Counts) deleted(root, path string, back bool) {
	if back {
		c.BroughtBack++
		return
	}
	c.Deleted++
	c.DeletedNames = append(c.DeletedNames, relativeTo(root, path))
}

// end decides the Outcome, fresh being what comes in, in this order: what
// comes in, what moves, the folders to look at, the deleted that stayed
// out, what is in, what could not be read. The deleted are named a few.
func (c *Counts) end(fresh int) {
	c.DeletedNames, c.DeletedMore = fewNames(c.DeletedNames)
	known := c.InLibrary + c.Twins
	switch {
	case fresh > 0:
		c.Outcome = OutcomeNew
	case c.Redated > 0:
		c.Outcome = OutcomeRedated
	case len(c.LeftOut) > 0:
		c.Outcome = OutcomeLeftOut
	case c.Deleted > 0:
		c.Outcome = OutcomeDeleted
	case known > 0 && c.Unreadable > 0:
		c.Outcome = OutcomeKnownNoted
	case known > 0:
		c.Outcome = OutcomeKnown
	case c.Unreadable > 0:
		c.Outcome = OutcomeUnread
	default:
		c.Outcome = OutcomeEmpty
	}
}
