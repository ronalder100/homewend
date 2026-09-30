// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ronalder/homewend/engine/internal/progress"
	"github.com/ronalder/homewend/engine/internal/session"
	"github.com/ronalder/homewend/engine/internal/takeout"
)

// ErrNotRequested means the form was sent but no new export appeared on
// /manage.
var ErrNotRequested = errors.New("Google did not list a new export")

// requestExport asks Google for an export of the Google Photos of one year,
// or of all of them when year is 0, and returns its job id once /manage lists it. The form is filled in a
// browser of our own, out of sight (see session.Headless); Google takes a
// minute or two before it lets a year be chosen (see takeout.PickerWait).
func requestExport(ctx context.Context, sess *session.Session, year int, emit progress.Func) (string, error) {
	before, err := takeout.Exports(sess)
	if err != nil {
		return "", err
	}
	known := map[string]bool{}
	for _, e := range before {
		known[e.Job] = true
	}

	emit.Emit(progress.Event{Stage: progress.Request})
	page, err := sess.Headless(ctx, takeout.PhotosURL)
	if err != nil {
		return "", err
	}
	defer page.Close()
	if err := takeout.CreateExport(page, year); err != nil {
		return "", err
	}

	// The page stays open meanwhile: closing it could cut the request short.
	// Two minutes is generous for a form Google answers in seconds.
	for deadline := time.Now().Add(2 * time.Minute); time.Now().Before(deadline); {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(5 * time.Second):
		}
		now, err := takeout.Exports(sess)
		if err != nil {
			return "", err
		}
		for _, e := range now {
			if !known[e.Job] {
				return e.Job, nil
			}
		}
	}
	var where string
	if err := page.Eval(`location.href`, &where); err == nil {
		return "", fmt.Errorf("%w; the form was left at %s", ErrNotRequested, where)
	}
	return "", ErrNotRequested
}
