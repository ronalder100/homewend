// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/ronalder100/homewend/internal/library"
)

// ErrLibraryBusy is another run placing photos in the library: an import,
// or a download, in this program or in another one (the window and the
// command line are two). Two at once would each move files the other is
// counting on.
var ErrLibraryBusy = errors.New("another import or download is placing photos in this library")

// lockLibrary takes the library's turn to place photos, or says at once that
// another run has it. The turn is a lock on a file in the library, held by
// the system: it goes with a program that dies.
func lockLibrary(root string) (unlock func(), err error) {
	dir := filepath.Join(root, library.WorkDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(filepath.Join(dir, "lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	switch err := tryLock(file); {
	case errors.Is(err, errLocked):
		file.Close()
		return nil, ErrLibraryBusy
	case err != nil:
		// A disk that cannot lock at all (some network shares): the run
		// goes on unguarded rather than not at all.
		file.Close()
		return func() {}, nil
	}
	return func() { unlockFile(file); file.Close() }, nil
}

// waitLibrary waits for the library's turn, asking again every few seconds,
// or for ctx to end. A download in the background waits for an import to
// finish placing rather than failing.
func waitLibrary(ctx context.Context, root string) (unlock func(), err error) {
	for {
		unlock, err := lockLibrary(root)
		if !errors.Is(err, ErrLibraryBusy) {
			return unlock, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

// errLocked is a lock another run holds.
var errLocked = errors.New("locked")
