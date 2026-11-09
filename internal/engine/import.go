// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/ronalder100/homewend/internal/library"
	"github.com/ronalder100/homewend/internal/progress"
)

// LibraryGoneError is a library folder chosen once and not there now: its
// disk unplugged, its share not mounted. Taken for an empty library, it
// would make everything look new.
type LibraryGoneError struct{ Root string }

func (e LibraryGoneError) Error() string { return "the library folder " + e.Root + " is not there" }

// PlanImport is what importing dir would bring into the library, read and
// never written: dir, a card or the library are left as they are.
func PlanImport(dir string, emit progress.Func) (library.ImportPlan, error) {
	// A link to a folder is walked as the folder.
	dir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return library.ImportPlan{}, err
	}
	folder, err := library.ReadFolder(dir)
	if err != nil {
		return library.ImportPlan{}, err
	}
	catalog, err := lookAtLibrary()
	if err != nil {
		return library.ImportPlan{}, err
	}
	if catalog != nil {
		defer catalog.Close()
	}
	return library.PlanImport(dir, folder, catalog, time.Local, emit)
}

// lookAtLibrary opens the chosen library's catalog to read it, or nil when
// no library was chosen or nothing is in it yet.
func lookAtLibrary() (*library.Catalog, error) {
	root, err := DefaultLibrary()
	if err != nil || root == "" {
		return nil, err
	}
	if _, err := os.Stat(root); err != nil {
		return nil, LibraryGoneError{Root: root}
	}
	if _, err := os.Stat(catalogPath(root)); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return library.OpenCatalogReadOnly(catalogPath(root))
}
