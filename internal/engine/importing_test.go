// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// importEnded waits for the import going on to end, and is where it stands.
func importEnded(t *testing.T, im *Importing) ImportState {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for im.State().Running {
		if time.Now().After(deadline) {
			t.Fatal("the import did not end")
		}
		time.Sleep(10 * time.Millisecond)
	}
	return im.State()
}

// The window's import: a look at the card, then the copy, the plan kept
// beside what came in for the screen that shows both.
func TestTheWindowPlansThenImports(t *testing.T) {
	config(t)
	if err := SetDefaultLibrary(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	card := filepath.Join(t.TempDir(), "card")
	if err := os.MkdirAll(filepath.Join(card, "DCIM"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"DSCF0001.JPG", "DSCF0002.JPG"} {
		if err := os.WriteFile(filepath.Join(card, "DCIM", name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var im Importing
	if err := im.Plan(card, "ron", nil); err != nil {
		t.Fatal(err)
	}
	s := importEnded(t, &im)
	if s.Error != "" || s.Plan == nil || s.Plan.Photos != 2 {
		t.Fatalf("plan: %+v", s)
	}
	if err := im.Import(card, "../out", false, nil); !errors.Is(err, ErrBadProfile) {
		t.Errorf("a profile outside the library: %v", err)
	}
	if err := im.Import(card, "ron", false, nil); err != nil {
		t.Fatal(err)
	}
	if err := im.Import(card, "ron", false, nil); err != nil && !errors.Is(err, ErrImportRunning) {
		t.Errorf("a second import while one runs: %v", err)
	}
	s = importEnded(t, &im)
	if s.Error != "" || s.Imported == nil || s.Imported.Placed != 2 {
		t.Fatalf("import: %+v", s)
	}
	if s.Plan == nil || s.Plan.Photos != 2 {
		t.Errorf("the plan of the same card was not kept: %+v", s.Plan)
	}
}

// A card gone is named, for the screen that waits for it to come back.
func TestTheWindowNamesACardTakenOut(t *testing.T) {
	if p, _, _ := importProblem(FolderGoneError{Dir: "/media/card"}); p != "taken-out" {
		t.Errorf("problem %q", p)
	}
	if p, need, _ := importProblem(NoSpaceError{Need: 5, Free: 1}); p != "no-space" || need != 5 {
		t.Errorf("problem %q, need %d", p, need)
	}
}
