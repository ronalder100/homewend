// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ronalder100/homewend/internal/library"
	"github.com/ronalder100/homewend/internal/progress"
	"github.com/ronalder100/homewend/internal/session"
	"github.com/ronalder100/homewend/internal/takeout"
)

// ErrExportGone means the export asked for is no longer on /manage.
var ErrExportGone = errors.New("the export asked for is no longer listed by Google")

// Get is one year of Google Photos, or all of it, to bring into one library,
// from asking Google for it to the last photo checked.
type Get struct {
	Year    int // 0: everything
	Library string
	// Takeout picks an export by its id, or the first characters of it, as
	// the takeouts command lists them; Year is then not looked at.
	Takeout string
	// New asks Google for a new export even when one of Year is still offered.
	New bool
	// Profile is whose photos they are: the library's folder they go in.
	Profile string
}

// waitPoll is how often /manage is read while Google prepares the export: the
// pace of a person refreshing a tab.
const waitPoll = time.Minute

// Run downloads the export of Year that Homewend asked for last, if Google
// still offers it or is preparing it, and asks for a new one only when there
// is none; then it waits for it, fetches it and verifies it.
//
// It can be stopped at any point and run again: an export already asked for
// is never asked for twice, and the fetch resumes as Fetch.Run does. Into
// another library, the same export is downloaded again. When the network
// fails it runs itself again (see Patiently).
func (g Get) Run(ctx context.Context, sess *session.Session, emit progress.Func) (result Result, err error) {
	err = Patiently(ctx, emit, func() error {
		result, err = g.run(ctx, sess, emit)
		return err
	})
	return result, err
}

// Found is the export Run would download rather than ask Google for a new one.
type Found struct {
	Takeout
	// This library already holds some of it: Run carries on where it was.
	Started bool
}

// Existing reports the export Run would download, or nil when it would ask
// Google for a new one, so that a person can be asked first: about a new
// export, which takes Google hours, or about one already there, which may not
// be the one they want. The profile must be signed in.
func (g Get) Existing(sess *session.Session) (*Found, error) {
	export, err := g.pick(sess)
	if err != nil || export == nil {
		return nil, err
	}
	_, err = os.Stat(filepath.Join(g.Library, library.WorkDir, export.Job))
	return &Found{
		Takeout: Takeout{Export: *export, ID: ShortID(export.Job), Status: StatusOf(*export), Year: g.Year, Known: g.Takeout == ""},
		Started: err == nil,
	}, nil
}

// pick is the export Run downloads, or nil when it has to ask for a new one.
func (g Get) pick(sess *session.Session) (*takeout.Export, error) {
	exports, err := takeout.Exports(sess)
	if err != nil {
		return nil, err
	}
	if g.Takeout != "" {
		e, err := byID(exports, g.Takeout)
		if err != nil {
			return nil, err
		}
		if StatusOf(e) == Expired {
			return nil, fmt.Errorf("%w: %s", ErrExpired, g.Takeout)
		}
		return &e, nil
	}
	if g.New {
		return nil, nil
	}
	notes, err := loadNotes(sess.Profile)
	if err != nil {
		return nil, err
	}
	// A library from before the notes moved to the profile kept its own.
	if old, err := loadRequest(g.libraryNote()); err == nil && !old.Asked.IsZero() && !noted(notes, old.Asked) {
		notes = append(notes, note{Year: old.Year, Asked: old.Asked, Job: old.Job})
	}
	job, notes := latestFor(g.Year, notes, exports)
	if err := saveNotes(sess.Profile, notes); err != nil || job == "" {
		return nil, err
	}
	return &exports[indexOf(exports, job)], nil
}

func noted(notes []note, asked time.Time) bool {
	for _, n := range notes {
		if n.Asked.Equal(asked) {
			return true
		}
	}
	return false
}

// libraryNote is where a library kept its request before the notes moved to
// the profile.
func (g Get) libraryNote() string {
	name := "request-all.json"
	if g.Year != 0 {
		name = fmt.Sprintf("request-%d.json", g.Year)
	}
	return filepath.Join(g.Library, library.WorkDir, name)
}

func (g Get) run(ctx context.Context, sess *session.Session, emit progress.Func) (Result, error) {
	if _, _, err := Login(ctx, sess, emit); err != nil {
		return Result{}, err
	}
	picked, err := g.pick(sess)
	if err != nil {
		return Result{}, err
	}
	job := ""
	if picked != nil {
		job = picked.Job
	}
	if job == "" {
		// The note is written before the form is sent: a run stopped in
		// between finds the export by its time, and does not ask twice.
		notes, err := loadNotes(sess.Profile)
		if err != nil {
			return Result{}, err
		}
		notes = append(notes, note{Year: g.Year, Asked: time.Now()})
		if err := saveNotes(sess.Profile, notes); err != nil {
			return Result{}, err
		}
		if job, err = requestExport(ctx, sess, g.Year, emit); err != nil {
			return Result{}, err
		}
		notes[len(notes)-1].Job = job
		if err := saveNotes(sess.Profile, notes); err != nil {
			return Result{}, err
		}
	}

	export, err := waitReady(ctx, sess, job, emit)
	if err != nil {
		return Result{}, err
	}
	user, err := User(ctx, sess, export, emit)
	if err != nil {
		return Result{}, err
	}
	return Fetch{
		Target:  takeout.Target{Job: export.Job, User: user},
		Export:  export,
		Library: g.Library,
		Account: Google + "/" + Account(sess),
		Profile: g.Profile,
	}.Run(ctx, sess, emit)
}

// request is what a library kept between runs before the notes moved to the
// profile, one file per year asked for, and one for everything.
type request struct {
	Year  int       `json:"year"`
	Asked time.Time `json:"asked,omitzero"` // just before the form was sent
	Job   string    `json:"job,omitempty"`  // once /manage lists the export
}

func loadRequest(path string) (request, error) {
	var req request
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return req, nil
	}
	if err != nil {
		return req, err
	}
	if err := json.Unmarshal(data, &req); err != nil {
		return req, fmt.Errorf("reading %s: %w", path, err)
	}
	return req, nil
}

// askedSince is the newest export created after asked, or "" if there is
// none. Google's clock and ours can differ by a little: a minute's grace.
func askedSince(exports []takeout.Export, asked time.Time) string {
	var job string
	var newest time.Time
	for _, e := range exports {
		if e.Created.After(asked.Add(-time.Minute)) && e.Created.After(newest) {
			job, newest = e.Job, e.Created
		}
	}
	return job
}

// waitReady reads /manage until the export with job can be downloaded.
func waitReady(ctx context.Context, sess *session.Session, job string, emit progress.Func) (takeout.Export, error) {
	for waiting := false; ; {
		exports, err := takeout.Exports(sess)
		if err != nil {
			return takeout.Export{}, err
		}
		i := indexOf(exports, job)
		if i < 0 {
			return takeout.Export{}, fmt.Errorf("%w: %s", ErrExportGone, job)
		}
		switch StatusOf(exports[i]) {
		case Ready:
			return exports[i], nil
		case Expired:
			return takeout.Export{}, fmt.Errorf("%w: %s", ErrExpired, job)
		}
		if !waiting {
			emit.Emit(progress.Event{Stage: progress.Waiting, Name: job})
			waiting = true
		}
		select {
		case <-ctx.Done():
			return takeout.Export{}, ctx.Err()
		case <-time.After(waitPoll):
		}
	}
}

func indexOf(exports []takeout.Export, job string) int {
	for i, e := range exports {
		if e.Job == job {
			return i
		}
	}
	return -1
}

// writeJSON writes v to path whole or not at all: a crash mid-write must never
// leave a file that says less, or more, than what happened.
func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path+".tmp", data, 0o644); err != nil {
		return err
	}
	return os.Rename(path+".tmp", path)
}
