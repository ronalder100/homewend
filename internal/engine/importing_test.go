// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ronalder100/homewend/internal/library"
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
	root := t.TempDir()
	if err := SetDefaultLibrary(root); err != nil {
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
	if err := im.Plan(ImportAsk{Dir: card, Profile: "ron"}); err != nil {
		t.Fatal(err)
	}
	s := importEnded(t, &im)
	if s.Error != "" || s.Plan == nil || s.Plan.Photos != 2 {
		t.Fatalf("plan: %+v", s)
	}
	// The card and the library share the test's disk: a link is offered,
	// and the probe that found it out is not left behind.
	if !s.CanLink {
		t.Error("a folder on the library's disk is not offered a link")
	}
	if probes, _ := filepath.Glob(filepath.Join(root, ".homewend", "link-probe-*")); len(probes) > 0 {
		t.Errorf("probes left: %v", probes)
	}
	if err := im.Import(ImportAsk{Dir: card, Profile: "../out"}); !errors.Is(err, ErrBadProfile) {
		t.Errorf("a profile outside the library: %v", err)
	}
	if err := im.Import(ImportAsk{Dir: card, Profile: "ron"}); err != nil {
		t.Fatal(err)
	}
	if err := im.Import(ImportAsk{Dir: card, Profile: "ron"}); err != nil && !errors.Is(err, ErrImportRunning) {
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
	im := Importing{state: ImportState{ImportAsk: ImportAsk{Dir: "/media/EOS_DIGITAL"}, Kind: "import", Problem: "taken-out", Error: "taken out"}}
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
	_, err := PlanImport(ctx, card, library.PlanOptions{Profile: "ron"}, func(e progress.Event) {
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
	_, err := PlanImport(context.Background(), card, library.PlanOptions{Profile: "ron"}, func(e progress.Event) {
		if e.N == 3 {
			os.RemoveAll(card)
		}
	})
	var gone FolderGoneError
	if !errors.As(err, &gone) {
		t.Errorf("taken out while read: %v", err)
	}
	if _, err := PlanImport(context.Background(), card, library.PlanOptions{Profile: "ron"}, nil); !errors.As(err, &gone) {
		t.Errorf("gone before: %v", err)
	}
	if _, err := Import(context.Background(), card, "ron", library.ImportOptions{}, nil); !errors.As(err, &gone) {
		t.Errorf("gone before the copy: %v", err)
	}
}

// Carried on with the card still out, a copy stays a card's, and what came
// in before it went out is still what the screen shows.
func TestACopyCarriedOnWithTheCardOutKeepsWhatCameIn(t *testing.T) {
	config(t)
	if err := SetDefaultLibrary(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	gone := filepath.Join(t.TempDir(), "EOS_DIGITAL")
	before := &library.Imported{Placed: 300}
	im := Importing{state: ImportState{ImportAsk: ImportAsk{Dir: gone}, Kind: "import", Card: true, Imported: before, Problem: "taken-out", Error: "taken out"}}
	if err := im.Import(ImportAsk{Dir: gone, Profile: "ron"}); err != nil {
		t.Fatal(err)
	}
	s := importEnded(t, &im)
	if s.Problem != "taken-out" || !s.Card {
		t.Errorf("state %+v, want a card still taken out", s)
	}
	if s.Imported == nil || s.Imported.Placed != 300 {
		t.Errorf("came in: %+v, want the 300 from before", s.Imported)
	}
	// A new look at it has no plan but its own.
	im.state.Plan = &library.ImportPlan{Photos: 7}
	if err := im.Plan(ImportAsk{Dir: gone, Profile: "ron"}); err != nil {
		t.Fatal(err)
	}
	if s := importEnded(t, &im); s.Plan != nil {
		t.Errorf("an old plan shown for a new look: %+v", s.Plan)
	}
}

// A photo of the card looked at has its thumbnail; a path the look did not
// read has none, whatever it names.
func TestTheWindowShowsACardsPhotosSmall(t *testing.T) {
	config(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	if err := SetDefaultLibrary(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	card := t.TempDir()
	var b bytes.Buffer
	if err := jpeg.Encode(&b, image.NewRGBA(image.Rect(0, 0, 64, 48)), nil); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(card, "DCIM"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(card, "DCIM", "IMG_0001.JPG"), b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	var im Importing
	if err := im.Plan(ImportAsk{Dir: card, Profile: "ron"}); err != nil {
		t.Fatal(err)
	}
	if s := importEnded(t, &im); s.Plan == nil || len(s.Plan.Files) != 1 || s.Plan.Files[0].Path != "DCIM/IMG_0001.JPG" {
		t.Fatalf("files: %+v", s.Plan)
	}
	path, err := im.Thumbnail(context.Background(), "DCIM/IMG_0001.JPG")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("no thumbnail made: %v", err)
	}
	// Through a link to the folder, as a home on Fedora Silverblue or a
	// Mac's /tmp is, the same photo is found.
	via := filepath.Join(t.TempDir(), "via")
	if err := os.Symlink(card, via); err != nil {
		t.Fatal(err)
	}
	if err := im.Plan(ImportAsk{Dir: via, Profile: "ron"}); err != nil {
		t.Fatal(err)
	}
	importEnded(t, &im)
	if _, err := im.Thumbnail(context.Background(), "DCIM/IMG_0001.JPG"); err != nil {
		t.Errorf("through a link: %v", err)
	}
	for _, other := range []string{"../../etc/passwd", "DCIM/IMG_0002.JPG"} {
		if _, err := im.Thumbnail(context.Background(), other); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s: %v, want none", other, err)
		}
	}
}

// The copy counts what came in by month, as the window shows it.
func TestTheWindowCountsACopyByMonth(t *testing.T) {
	config(t)
	if err := SetDefaultLibrary(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	card := filepath.Join(t.TempDir(), "card")
	if err := os.MkdirAll(filepath.Join(card, "DCIM"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"A.JPG", "B.JPG"} {
		if err := os.WriteFile(filepath.Join(card, "DCIM", name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var im Importing
	given := []library.Redate{{Names: []string{"A.JPG"}, When: time.Date(2025, 9, 14, 10, 0, 0, 0, time.UTC)}}
	if err := im.Import(ImportAsk{Dir: card, Profile: "ron", Redates: given}); err != nil {
		t.Fatal(err)
	}
	s := importEnded(t, &im)
	if s.Filed["2025"][9] != 1 || s.Filed[""][0] != 1 {
		t.Errorf("filed %v, want one in 2025-09 and one undated", s.Filed)
	}
}
