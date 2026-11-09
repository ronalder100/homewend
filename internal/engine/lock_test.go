// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"errors"
	"testing"
	"time"
)

// One run places photos in a library at a time: a second is told at once,
// and has its turn as soon as the first is done.
func TestOneRunPlacesPhotosAtATime(t *testing.T) {
	root := t.TempDir()
	unlock, err := lockLibrary(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lockLibrary(root); !errors.Is(err, ErrLibraryBusy) {
		t.Errorf("second run: %v, want ErrLibraryBusy", err)
	}
	unlock()
	again, err := lockLibrary(root)
	if err != nil {
		t.Errorf("after the first: %v, want the turn", err)
	} else {
		again()
	}
}

// A download waits for its turn, or for its stop.
func TestADownloadWaitsForItsTurn(t *testing.T) {
	root := t.TempDir()
	unlock, err := lockLibrary(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := waitLibrary(ctx, root, nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("waiting while held: %v, want the stop", err)
	}
	go func() { time.Sleep(200 * time.Millisecond); unlock() }()
	turn, err := waitLibrary(context.Background(), root, nil)
	if err != nil {
		t.Fatalf("waiting for the turn: %v", err)
	}
	turn()
}
