// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/ronalder/homewend/engine/internal/download"
	"github.com/ronalder/homewend/engine/internal/progress"
)

// Patiently runs fn again for as long as it fails on the network, with a
// pause that grows to five minutes. Every step fn can take resumes where it
// was, so running it again is the whole of the recovery. On a NAS run the
// line to Google was down for more than an hour at a time (2026-09-29): a
// program that gives up then needs a person to start it again.
//
// Any other error — a session Google no longer accepts, a full disk — is
// returned at once: waiting does not cure it.
func Patiently(ctx context.Context, emit progress.Func, fn func() error) error {
	for attempt := 1; ; attempt++ {
		err := fn()
		if err == nil || !transient(err) {
			return err
		}
		emit.Emit(progress.Event{Stage: progress.Retry, N: attempt, Note: err.Error()})
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-after(min(time.Duration(attempt)*30*time.Second, 5*time.Minute)):
		}
	}
}

// transient reports whether err came from the network: a connection that
// failed, timed out or was reset, or a part that stopped bringing bytes.
func transient(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) || errors.Is(err, download.ErrIncomplete)
}

// after is time.After, replaced in tests.
var after = time.After
