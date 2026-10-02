// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// Unassigned is the folder of a library that shows, while a download is going
// on, the photos that have arrived and cannot be filed yet: the ones still
// waiting for the part that brings their date. Hours into a download, a
// library with nothing in it looks broken; one with a folder that says how
// much is waiting does not.
const Unassigned = "unassigned"

// ShowUnassigned makes <root>/unassigned hold exactly items, each under the
// folder Takeout had it in, and removes the folder when there are none. It
// returns how many are shown.
//
// They are second names for the files where they are, not copies: no bytes,
// and filing a photo later is still one rename. A disk with no hard links
// shows none, rather than paying for every waiting photo twice.
func ShowUnassigned(root string, items []Item) (int, error) {
	dir := filepath.Join(root, Unassigned)
	want := make(map[string]string, len(items))
	for _, item := range items {
		want[filepath.Join(filepath.Base(filepath.Dir(item.Path)), filepath.Base(item.Path))] = item.Path
	}

	// What is shown and no longer waits goes: it was filed, and the name here
	// would otherwise keep a second one alive.
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relative, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if _, waiting := want[relative]; waiting {
			delete(want, relative)
			return nil
		}
		return os.Remove(path)
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return 0, err
	}
	for relative, source := range want {
		shown := filepath.Join(dir, relative)
		if err := os.MkdirAll(filepath.Dir(shown), 0o755); err != nil {
			return 0, err
		}
		if err := os.Link(source, shown); err != nil {
			// No hard links here: nothing is shown, and nothing is copied.
			return 0, os.RemoveAll(dir)
		}
	}
	if len(items) == 0 {
		return 0, os.RemoveAll(dir)
	}
	// A folder Takeout had, with nothing of it waiting any more.
	folders, _ := os.ReadDir(dir)
	for _, folder := range folders {
		os.Remove(filepath.Join(dir, folder.Name())) // fails, rightly, when it is not empty
	}
	return len(items), nil
}
