// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"errors"
	"io/fs"
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

// readFolder is what is under dir, a link to a folder followed. The folder
// is made whole: a name given as the shell completes it, from here or from
// /, is found below it. A folder not there is a card taken out: the window
// asks for a card it saw a moment before.
func readFolder(dir string) (string, library.Folder, error) {
	dir, err := library.RealPath(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return "", library.Folder{}, FolderGoneError{Dir: dir}
	}
	if err != nil {
		return "", library.Folder{}, err
	}
	folder, err := library.ReadFolder(dir)
	return dir, folder, err
}

// PlanImport is what importing dir would bring into the library, read and
// never written: dir, a card or the library are left as they are. ctx stops
// it between two files.
func PlanImport(ctx context.Context, dir string, opt library.PlanOptions, emit progress.Func) (library.ImportPlan, error) {
	if opt.Profile != "" && !ValidProfile(opt.Profile) {
		return library.ImportPlan{}, ErrBadProfile
	}
	dir, folder, err := readFolder(dir)
	if err != nil {
		return library.ImportPlan{}, err
	}
	// A photo the catalog lists and the disk lacks was deleted: the plan
	// looks for it under the library's folder.
	var catalog *library.Catalog
	opt.Root, catalog, err = lookAtLibrary()
	if err != nil {
		return library.ImportPlan{}, err
	}
	if catalog != nil {
		defer catalog.Close()
	}
	plan, err := library.PlanImport(ctx, dir, folder, catalog, opt, emit)
	// A card taken out while it is read is said as during the copy: its
	// files read after are only unreadable, not an error.
	if _, gone := os.Stat(dir); gone != nil && !errors.Is(err, context.Canceled) {
		return plan, FolderGoneError{Dir: dir}
	}
	return plan, err
}

// Import brings dir's photos and videos into the library, into profile:
// copied and read back, or with how.Link given a second name on the
// library's disk. dir is not changed. ctx stops it between two files. The
// room on the library's disk is checked here: how.Fits is not the caller's.
func Import(ctx context.Context, dir, profile string, how library.ImportOptions, emit progress.Func) (library.Imported, error) {
	// A profile is a folder of the library: a name like "../x" would put
	// photos outside it, or into the folder being imported.
	if !ValidProfile(profile) {
		return library.Imported{}, ErrBadProfile
	}
	// The photos' badge: the card's name, or the folder's, as the person
	// put it in or chose it.
	source := sourceOf(dir)
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
	unlock, err := lockLibrary(root)
	if err != nil {
		return library.Imported{}, err
	}
	defer unlock()
	catalog, err := library.OpenCatalog(catalogPath(root))
	if err != nil {
		return library.Imported{}, err
	}
	defer catalog.Close()
	// Asked before each copy, of the new photos only: a copy that fills
	// the disk helps nobody, and a card mostly in the library already needs
	// room for what is new on it.
	how.Fits = func(size int64) error {
		free, err := Free(root)
		if err != nil {
			return err
		}
		if size > free {
			return NoSpaceError{Need: size, Free: free}
		}
		return nil
	}
	opt := library.Options{Profile: profile, SeparateMessaging: true, Account: source}
	done, err := library.Import(ctx, folder, root, catalog, opt, how, emit)
	// A card taken out halfway, during a copy or between two, is not a run
	// that ended or failed: it is one to resume. A stop asked for is said
	// as a stop.
	if _, gone := os.Stat(dir); gone != nil && !errors.Is(err, context.Canceled) {
		return done, FolderGoneError{Dir: dir}
	}
	return done, err
}

// sourceOf is the badge of the photos imported from dir, "local/<name>":
// a card's name as its system shows it, or the folder's. The catalog keeps
// it as it keeps a Google account's, and the window names the photo by it.
func sourceOf(dir string) string {
	if c, ok := cardAt(dir); ok && c.Name != "" {
		return library.LocalSource + c.Name
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	return library.LocalSource + filepath.Base(abs)
}

// FolderGoneError is the folder being imported gone before the end: a card
// taken out, a disk unplugged. What was in stays in.
type FolderGoneError struct{ Dir string }

func (e FolderGoneError) Error() string { return e.Dir + " was taken out before the end" }

// lookAtLibrary opens the chosen library's catalog to read it, or nil when
// no library was chosen or nothing is in it yet.
func lookAtLibrary() (root string, catalog *library.Catalog, err error) {
	root, err = DefaultLibrary()
	if err != nil || root == "" {
		return "", nil, err
	}
	if _, err := os.Stat(root); err != nil {
		return "", nil, LibraryGoneError{Root: root}
	}
	if _, err := os.Stat(catalogPath(root)); errors.Is(err, os.ErrNotExist) {
		return root, nil, nil
	}
	catalog, err = library.OpenCatalogReadOnly(catalogPath(root))
	return root, catalog, err
}
