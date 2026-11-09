// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"errors"
	"os"
	"path/filepath"

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
	return library.PlanImport(dir, folder, catalog, emit)
}

// Import brings dir's photos and videos into the library, into profile:
// copied and read back, or with link given a second name on the library's
// disk. dir is not changed.
func Import(dir, profile string, link bool, emit progress.Func) (library.Organized, error) {
	dir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return library.Organized{}, err
	}
	folder, err := library.ReadFolder(dir)
	if err != nil {
		return library.Organized{}, err
	}
	root, err := ChosenLibrary()
	if err != nil {
		return library.Organized{}, err
	}
	if _, err := os.Stat(root); err != nil {
		return library.Organized{}, LibraryGoneError{Root: root}
	}
	// What would be copied, counted before anything is: a copy that fills
	// the disk halfway helps nobody. The photos already in the library are
	// counted too, as they are not known before they are read.
	if !link {
		var need int64
		for _, item := range folder.Items {
			if info, err := os.Stat(item.Path); err == nil {
				need += info.Size()
			}
		}
		free, err := Free(root)
		if err != nil {
			return library.Organized{}, err
		}
		if need > free {
			return library.Organized{}, NoSpaceError{Need: need, Free: free}
		}
	}
	if err := os.MkdirAll(filepath.Join(root, library.WorkDir), 0o755); err != nil {
		return library.Organized{}, err
	}
	catalog, err := library.OpenCatalog(catalogPath(root))
	if err != nil {
		return library.Organized{}, err
	}
	defer catalog.Close()
	return library.Import(folder, root, catalog, library.Options{Profile: profile, SeparateMessaging: true}, link, emit)
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
