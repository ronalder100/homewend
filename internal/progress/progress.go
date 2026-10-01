// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package progress is the one way every stage says what it is doing. The CLI,
// the MCP server and the interface read the same events, so none of them has
// to know how a stage works to show it.
package progress

// The stages an event can come from.
const (
	SignIn        = "sign-in"        // the browser is open, waiting for the user to sign in
	FirstDownload = "first-download" // the browser is open on the export, for the user to download one part, once per account
	Request       = "request"        // an export is being asked for
	Waiting       = "waiting"        // Google is preparing the export; Name is its job
	Ready         = "ready"          // the export is ready and its download starts; Of counts its parts, manifest included, Total their bytes
	Download      = "download"       // a part starts or resumes; Done and Total are bytes
	Receiving     = "receiving"      // bytes of a part are arriving, at most once a second; Done and Total are bytes
	Downloaded    = "downloaded"     // a part is complete on disk
	Short         = "short"          // a transfer stopped early; the bytes on disk are kept
	Retry         = "retry"          // the network did not answer and will be tried again; N counts, Note says why
	Damaged       = "damaged"        // a part failed its check on unpacking and is downloaded again
	Unpack        = "unpack"         // a part is being opened
	Place         = "place"          // a photo is being placed in the library
)

// Event is one thing that happened. Fields that do not apply are left empty.
type Event struct {
	Stage string `json:"stage"`
	N     int    `json:"n,omitempty"`
	Of    int    `json:"of,omitempty"`
	Name  string `json:"name,omitempty"`
	Done  int64  `json:"done,omitempty"`
	Total int64  `json:"total,omitempty"`
	Note  string `json:"note,omitempty"`
}

// Func receives events. A nil Func is valid and receives nothing.
type Func func(Event)

// Emit sends e, if anyone is listening.
func (f Func) Emit(e Event) {
	if f != nil {
		f(e)
	}
}
