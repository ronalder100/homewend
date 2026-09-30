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
}

// waitPoll is how often /manage is read while Google prepares the export: the
// pace of a person refreshing a tab.
const waitPoll = time.Minute

// Run asks Google for the export, waits for it, fetches it and verifies it.
//
// It can be stopped at any point and run again. What it has done is kept in
// the library (see request): an export already asked for is never asked for
// twice, and the fetch resumes as Fetch.Run does. When the network fails it
// runs itself again (see Patiently).
func (g Get) Run(ctx context.Context, sess *session.Session, emit progress.Func) (result Result, err error) {
	err = Patiently(ctx, emit, func() error {
		result, err = g.run(ctx, sess, emit)
		return err
	})
	return result, err
}

func (g Get) run(ctx context.Context, sess *session.Session, emit progress.Func) (Result, error) {
	if _, err := Login(ctx, sess, emit); err != nil {
		return Result{}, err
	}
	name := "request-all.json"
	if g.Year != 0 {
		name = fmt.Sprintf("request-%d.json", g.Year)
	}
	path := filepath.Join(g.Library, library.WorkDir, name)
	req, err := loadRequest(path)
	if err != nil {
		return Result{}, err
	}
	req.Year = g.Year

	if req.Job == "" && !req.Asked.IsZero() {
		// Stopped between sending the form and seeing the export listed:
		// Google may have it. Asking again would make a second one.
		exports, err := takeout.Exports(sess)
		if err != nil {
			return Result{}, err
		}
		req.Job = askedSince(exports, req.Asked)
	}
	if req.Job == "" {
		req.Asked = time.Now()
		if err := writeJSON(path, req); err != nil {
			return Result{}, err
		}
		if req.Job, err = requestExport(ctx, sess, g.Year, emit); err != nil {
			return Result{}, err
		}
	}
	if err := writeJSON(path, req); err != nil {
		return Result{}, err
	}

	export, err := waitReady(ctx, sess, req.Job, emit)
	if err != nil {
		return Result{}, err
	}
	user, err := User(ctx, sess, export.Job, emit)
	if err != nil {
		return Result{}, err
	}
	return Fetch{
		Target:  takeout.Target{Job: export.Job, User: user},
		Export:  export,
		Library: g.Library,
	}.Run(ctx, sess, emit)
}

// request is what Get keeps between runs, one file per year asked for, and
// one for everything.
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
		if exports[i].Ready() {
			return exports[i], nil
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
