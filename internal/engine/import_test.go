// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ronalder100/homewend/internal/library"
	"github.com/ronalder100/homewend/internal/progress"
)

// A profile is a folder of the library: one that would leave it is refused
// before anything is read or copied.
func TestAnImportRefusesAProfileOutsideTheLibrary(t *testing.T) {
	for _, profile := range []string{"../escaped", ".homewend", "a/b", ""} {
		if _, err := Import(context.Background(), t.TempDir(), profile, library.ImportOptions{}, nil); !errors.Is(err, ErrBadProfile) {
			t.Errorf("profile %q: %v, want ErrBadProfile", profile, err)
		}
	}
}

// A card taken out halfway is a run to resume, not one that ended: what
// was copied stays, and the error says so.
func TestACardTakenOutHalfwayIsSaid(t *testing.T) {
	config(t)
	root := t.TempDir()
	if err := SetDefaultLibrary(root); err != nil {
		t.Fatal(err)
	}
	card := filepath.Join(t.TempDir(), "card")
	for _, name := range []string{"DSCF0001.JPG", "DSCF0002.JPG", "DSCF0003.JPG"} {
		if err := os.MkdirAll(filepath.Join(card, "DCIM"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(card, "DCIM", name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out := func(e progress.Event) {
		if e.Stage == progress.Copy && e.N == 2 {
			os.RemoveAll(card) // taken out
		}
	}
	done, err := Import(context.Background(), card, "ron", library.ImportOptions{}, out)
	var gone FolderGoneError
	if !errors.As(err, &gone) {
		t.Fatalf("err %v, want the folder said gone", err)
	}
	if done.Placed != 1 {
		t.Errorf("placed %d, want the one copied before", done.Placed)
	}
}

// A folder's photos carry its name as their source, and stay in the grid
// when every Google account is hidden.
func TestAnImportedPhotoIsNamedByItsFolder(t *testing.T) {
	config(t)
	root := t.TempDir()
	if err := SetDefaultLibrary(root); err != nil {
		t.Fatal(err)
	}
	card := filepath.Join(t.TempDir(), "old-laptop")
	if err := os.MkdirAll(filepath.Join(card, "DCIM"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(card, "DCIM", "A.JPG"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Import(context.Background(), card, "ron", library.ImportOptions{}, nil); err != nil {
		t.Fatal(err)
	}
	catalog, err := library.OpenCatalog(catalogPath(root))
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	shown, err := catalog.Page(library.Filter{Accounts: []string{}, Limit: 10})
	if err != nil || len(shown) != 1 {
		t.Fatalf("%d shown with every account hidden, %v; want the imported one", len(shown), err)
	}
	_, sources, err := catalog.About(shown[0].Hash)
	if err != nil || len(sources) != 1 || sources[0] != "local/old-laptop" {
		t.Errorf("sources %v, %v; want local/old-laptop", sources, err)
	}
}

// A folder is named by the last part of its path; a drive's root by
// itself, never by a lone separator.
func TestTheTopIsNamedAfterTheFolder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "old-laptop")
	if got := TopName(dir); got != "old-laptop" {
		t.Errorf("%s: %q", dir, got)
	}
	root := string(filepath.Separator)
	if got := TopName(root); got != root {
		t.Errorf("the root: %q, want %q", got, root)
	}
}

// A library typed that does not exist yet is made when it is chosen: the
// first import into it is not told its disk is away.
func TestAChosenLibraryIsMade(t *testing.T) {
	config(t)
	root := filepath.Join(t.TempDir(), "new", "lib")
	if err := ChooseLibrary(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(root); err != nil {
		t.Errorf("not made: %v", err)
	}
	if got, _ := DefaultLibrary(); got != root {
		t.Errorf("kept %q, want %q", got, root)
	}
}
