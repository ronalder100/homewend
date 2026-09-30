// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"errors"
	"time"

	"github.com/ronalder100/homewend/internal/progress"
	"github.com/ronalder100/homewend/internal/session"
)

// errWindowClosed means the user closed the browser before doing what it was
// opened for.
var errWindowClosed = errors.New("the browser was closed")

// inWindow opens the browser at url for the user to do one thing there, waits
// until done reports it done, and closes the browser: once it has what it was
// opened for, it has no reason to stay. It stops when ctx is done or the user
// closes the browser first.
//
// done reads the profile, not the page, and Chrome writes the profile in
// batches: cookies every 30 seconds or 512 changes
// (net/extras/sqlite/sqlite_persistent_cookie_store.cc), history every 10
// seconds (components/history/core/browser/history_backend.cc). Polling every
// two seconds adds nothing to that.
func inWindow(ctx context.Context, sess *session.Session, url, stage string, done func() bool, emit progress.Func) error {
	w, err := sess.Open(url)
	if err != nil {
		return err
	}
	emit.Emit(progress.Event{Stage: stage})

	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-w.Exited():
			// Closing the browser writes the profile out: look one last time.
			if done() {
				return nil
			}
			return errWindowClosed
		case <-tick.C:
			if done() {
				w.Close()
				return nil
			}
		}
	}
}
