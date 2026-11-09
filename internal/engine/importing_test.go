// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ronalder100/homewend/internal/progress"
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

// A copy waiting for its card, ended by the person, is stopped: the card,
// or another with its name, coming back does not carry it on.
func TestEndingACopyWaitingForItsCardStopsIt(t *testing.T) {
	im := Importing{state: ImportState{Dir: "/media/EOS_DIGITAL", Kind: "import", Problem: "taken-out", Error: "taken out"}}
	<-im.Stop()
	s := im.State()
	if s.Problem != "" || s.Error != "" || !s.Stopped {
		t.Errorf("after the end: %+v", s)
	}
}

// A stop is heard while a card is read, between two files.
func TestALookAtACardStops(t *testing.T) {
	config(t)
	if err := SetDefaultLibrary(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	card := t.TempDir()
	for i := range 50 {
		if err := os.WriteFile(filepath.Join(card, fmt.Sprintf("IMG_%04d.JPG", i)), []byte{byte(i)}, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	_, err := PlanImport(ctx, card, "ron", nil, func(e progress.Event) {
		if e.N == 3 {
			cancel()
		}
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err %v, want the look stopped", err)
	}
}

// A card taken out while it is read is said as taken out, not as a plan of
// unreadable files; and one already gone when asked for, too.
func TestACardTakenOutWhileReadIsSaid(t *testing.T) {
	config(t)
	if err := SetDefaultLibrary(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	card := filepath.Join(t.TempDir(), "card")
	if err := os.MkdirAll(card, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := range 20 {
		if err := os.WriteFile(filepath.Join(card, fmt.Sprintf("IMG_%04d.JPG", i)), []byte{byte(i)}, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, err := PlanImport(context.Background(), card, "ron", nil, func(e progress.Event) {
		if e.N == 3 {
			os.RemoveAll(card)
		}
	})
	var gone FolderGoneError
	if !errors.As(err, &gone) {
		t.Errorf("taken out while read: %v", err)
	}
	if _, err := PlanImport(context.Background(), card, "ron", nil, nil); !errors.As(err, &gone) {
		t.Errorf("gone before: %v", err)
	}
	if _, err := Import(context.Background(), card, "ron", false, nil, nil); !errors.As(err, &gone) {
		t.Errorf("gone before the copy: %v", err)
	}
}
