// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
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

// readFolder is what is under dir, a link to a folder followed.
func readFolder(dir string) (string, library.Folder, error) {
	dir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", library.Folder{}, err
	}
	folder, err := library.ReadFolder(dir)
	return dir, folder, err
}

// PlanImport is what importing dir would bring into the library, read and
// never written: dir, a card or the library are left as they are.
func PlanImport(dir string, emit progress.Func) (library.ImportPlan, error) {
	dir, folder, err := readFolder(dir)
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
// disk. dir is not changed. ctx stops it between two files.
func Import(ctx context.Context, dir, profile string, link bool, emit progress.Func) (library.Imported, error) {
	// A profile is a folder of the library: a name like "../x" would put
	// photos outside it, or into the folder being imported.
	if !ValidProfile(profile) {
		return library.Imported{}, ErrBadProfile
	}
	dir, folder, err := readFolder(dir)
	if err != nil {
		return library.Imported{}, err
	}
	root, err := ChosenLibrary()
	if err != nil {
		return library.Imported{}, err
	}
	if _, err := os.Stat(root); err != nil {
		return library.Imported{}, LibraryGoneError{Root: root}
	}
	if err := os.MkdirAll(filepath.Join(root, library.WorkDir), 0o755); err != nil {
		return library.Imported{}, err
	}
	catalog, err := library.OpenCatalog(catalogPath(root))
	if err != nil {
		return library.Imported{}, err
	}
	defer catalog.Close()
	// Asked before each copy, of the new photos only: a copy that fills
	// the disk helps nobody, and a card mostly in the library already needs
	// room for what is new on it.
	fits := func(size int64) error {
		free, err := Free(root)
		if err != nil {
			return err
		}
		if size > free {
			return NoSpaceError{Need: size, Free: free}
		}
		return nil
	}
	done, err := library.Import(ctx, folder, root, catalog, library.Options{Profile: profile, SeparateMessaging: true}, link, fits, emit)
	// A card taken out halfway, during a copy or between two, is not a run
	// that ended or failed: it is one to resume. A stop asked for is said
	// as a stop.
	if _, gone := os.Stat(dir); gone != nil && !errors.Is(err, context.Canceled) {
		return done, FolderGoneError{Dir: dir}
	}
	return done, err
}

// FolderGoneError is the folder being imported gone before the end: a card
// taken out, a disk unplugged. What was in stays in.
type FolderGoneError struct{ Dir string }

func (e FolderGoneError) Error() string { return e.Dir + " was taken out before the end" }

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
