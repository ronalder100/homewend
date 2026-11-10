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
