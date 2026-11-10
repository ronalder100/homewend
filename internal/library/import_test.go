// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ronalder100/homewend/internal/progress"
)

func importCard(t *testing.T, card, root string, link bool) Imported {
	t.Helper()
	return importWith(t, card, root, link, context.Background(), func(int64) error { return nil })
}

func importWith(t *testing.T, card, root string, link bool, ctx context.Context, fits func(int64) error) Imported {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	catalog, err := OpenCatalog(filepath.Join(root, "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	folder, err := ReadFolder(card)
	if err != nil {
		t.Fatal(err)
	}
	done, err := Import(ctx, folder, root, catalog, Options{Profile: "ron"}, ImportOptions{Link: link, Fits: fits}, nil)
	if err != nil && ctx.Err() == nil && !errors.Is(err, errFull) {
		t.Fatal(err)
	}
	return done
}

func TestACardIsCopiedByDateAndLeftAsItWas(t *testing.T) {
	card := cameraCard(t)
	write(t, card, "DCIM/101_FUJI/DSCF0002.JPG", exifPhoto("2018:01:01 00:01:36", "two")) // the same photo again
	before := snapshot(t, card)
	root := t.TempDir()

	done := importCard(t, card, root, false)

	if done.Placed != 3 || done.Twins != 1 || done.InLibrary != 0 {
		t.Errorf("%+v, want 3 placed and 1 twin", done)
	}
	mustExist(t, filepath.Join(root, "ron", "2026", "08", "DSCF0001.JPG"))
	mustExist(t, filepath.Join(root, "ron", "2018", "01", "DSCF0002.JPG"))
	mustExist(t, filepath.Join(root, "ron", "undated", "DSCF0003.MOV"))
	if after := snapshot(t, card); len(after) != len(before) {
		t.Errorf("the card changed: %d files before, %d after", len(before), len(after))
	} else {
		for path, content := range before {
			if after[path] != content {
				t.Errorf("%s changed on the card", path)
			}
		}
	}
	if entries, _ := os.ReadDir(arrivals(root)); len(entries) != 0 {
		t.Errorf("%d left waiting in the arrivals", len(entries))
	}
}

func TestACardBroughtInTwiceAddsNothing(t *testing.T) {
	card := cameraCard(t)
	root := t.TempDir()
	importCard(t, card, root, false)
	again := importCard(t, card, root, false)
	if again.Placed != 0 || again.InLibrary != 3 {
		t.Errorf("second run %+v, want nothing placed and 3 in the library", again)
	}
}

// A run stopped halfway — the card taken out — leaves a half copy in the
// arrivals: the next run starts from what is in the library, not from it.
func TestWhatAStoppedRunLeftIsNotKept(t *testing.T) {
	card := cameraCard(t)
	root := t.TempDir()
	write(t, arrivals(root), "0/DSCF0001.JPG", []byte("half a photo"))
	done := importCard(t, card, root, false)
	if done.Placed != 3 {
		t.Errorf("placed %d, want 3", done.Placed)
	}
	content, err := os.ReadFile(filepath.Join(root, "ron", "2026", "08", "DSCF0001.JPG"))
	if err != nil || string(content) != string(exifPhoto("2026:08:15 18:23:12", "one")) {
		t.Errorf("the photo in the library is not the card's: %q, %v", content, err)
	}
}

func TestALinkIsTheSameFile(t *testing.T) {
	dir := t.TempDir()
	card := filepath.Join(dir, "old-laptop")
	write(t, card, "Pictures/IMG_1.JPG", exifPhoto("2019:07:14 18:22:05", "one"))
	root := filepath.Join(dir, "library")
	importCard(t, card, root, true)
	a, err := os.Stat(filepath.Join(card, "Pictures", "IMG_1.JPG"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.Stat(filepath.Join(root, "ron", "2019", "07", "IMG_1.JPG"))
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(a, b) {
		t.Error("the library holds a copy, want a second name of the same file")
	}
}

// snapshot is every file under dir and its content.
func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		content, err := os.ReadFile(path)
		files[path] = string(content)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

var errFull = errors.New("full")

// A photo the library holds is not copied again: only the new one asks for
// room, and a nearly full disk takes a card already brought in.
func TestOnlyWhatIsNewIsCopied(t *testing.T) {
	card := cameraCard(t)
	root := t.TempDir()
	importCard(t, card, root, false)
	write(t, card, "DCIM/100_FUJI/DSCF0010.JPG", exifPhoto("2026:08:20 09:00:00", "ten"))
	asked := 0
	done := importWith(t, card, root, false, context.Background(), func(int64) error { asked++; return nil })
	if asked != 1 || done.Placed != 1 || done.InLibrary != 3 {
		t.Errorf("room asked %d times, %+v; want once, 1 placed, 3 in the library", asked, done)
	}
}

// A disk too full stops the import before the copy that would not fit.
func TestAFullDiskStopsBeforeTheCopy(t *testing.T) {
	card := cameraCard(t)
	root := t.TempDir()
	done := importWith(t, card, root, false, context.Background(), func(int64) error { return errFull })
	if done.Placed != 0 {
		t.Errorf("placed %d with no room", done.Placed)
	}
	if entries, _ := os.ReadDir(arrivals(root)); len(entries) != 0 {
		t.Error("something was left in the arrivals")
	}
}

// A stop asked between two files keeps what was in and copies no more.
func TestAStopKeepsWhatIsIn(t *testing.T) {
	card := cameraCard(t)
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	n := 0
	done := importWith(t, card, root, false, ctx, func(int64) error {
		n++
		if n == 1 {
			cancel()
		}
		return nil
	})
	if done.Placed != 1 {
		t.Errorf("placed %d, want the 1 copied before the stop", done.Placed)
	}
}

// A folder imported with links into the library reaches the folder's file,
// not a link to it.
func TestALinkToALinkIsTheFile(t *testing.T) {
	dir := t.TempDir()
	card := filepath.Join(dir, "old-laptop")
	write(t, dir, "elsewhere/IMG_2.JPG", exifPhoto("2019:07:14 18:22:05", "two"))
	if err := os.MkdirAll(filepath.Join(card, "Pictures"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../elsewhere/IMG_2.JPG", filepath.Join(card, "Pictures", "IMG_2.JPG")); err != nil {
		t.Skip("no symbolic links here")
	}
	root := filepath.Join(dir, "library")
	importCard(t, card, root, true)
	info, err := os.Lstat(filepath.Join(root, "ron", "2019", "07", "IMG_2.JPG"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("the library holds a symbolic link, want the file")
	}
}

// A suffix counts up the file's name, never a folder above it: under a
// folder named Disk-2, a second DSCF0001 stays in the library.
func TestASuffixIsTheNamesOnly(t *testing.T) {
	for _, c := range []struct {
		in   string
		n    int
		want string
	}{
		{"/media/Disk-2/lib/2018/01/DSCF0001.JPG", 2, "/media/Disk-2/lib/2018/01/DSCF0001-2.JPG"},
		{"/media/Disk-2/lib/2018/01/DSCF0001.JPG", 3, "/media/Disk-2/lib/2018/01/DSCF0001-3.JPG"},
		{"/home/u/Photos-2024/IMG_1.JPG", 2, "/home/u/Photos-2024/IMG_1-2.JPG"},
		{"/lib/2018/01/2018-01-05.jpg", 2, "/lib/2018/01/2018-01-05-2.jpg"},
	} {
		if got := withSuffix(filepath.FromSlash(c.in), c.n); got != filepath.FromSlash(c.want) {
			t.Errorf("withSuffix(%s, %d) = %s, want %s", c.in, c.n, got, c.want)
		}
	}
}

// A card brought into a second profile, then again, gives each photo one
// name there, not one more each time.
func TestASecondProfileGetsOneNameEach(t *testing.T) {
	card := cameraCard(t)
	root := t.TempDir()
	importCard(t, card, root, false)
	for run := 0; run < 2; run++ {
		catalog, err := OpenCatalog(filepath.Join(root, "catalog.db"))
		if err != nil {
			t.Fatal(err)
		}
		folder, err := ReadFolder(card)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Import(context.Background(), folder, root, catalog, Options{Profile: "anna"}, ImportOptions{Link: false, Fits: func(int64) error { return nil }}, nil); err != nil {
			t.Fatal(err)
		}
		catalog.Close()
	}
	var names []string
	filepath.Walk(filepath.Join(root, "anna"), func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			names = append(names, path)
		}
		return nil
	})
	if len(names) != 3 {
		t.Errorf("%d names in the second profile, want 3: %v", len(names), names)
	}
}

// A photo deleted from the library stays out of the next import, counted:
// the card still holds it. Brought back, it is new: placed in the profile
// it is brought into.
func TestAPhotoDeletedFromTheLibraryStaysOutUnlessBroughtBack(t *testing.T) {
	card := cameraCard(t)
	root := t.TempDir()
	importCard(t, card, root, false)
	gone := filepath.Join(root, "ron", "2018", "01", "DSCF0002.JPG")
	if err := os.Remove(gone); err != nil {
		t.Fatal(err)
	}
	catalog, err := OpenCatalog(filepath.Join(root, "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	folder, err := ReadFolder(card)
	if err != nil {
		t.Fatal(err)
	}
	fits := func(int64) error { return nil }
	plan, err := PlanImport(context.Background(), card, folder, catalog, PlanOptions{Root: root}, nil)
	if err != nil || plan.Deleted != 1 || plan.Photos != 0 {
		t.Errorf("look: %+v, %v; want the deleted photo said, not to copy", plan, err)
	}
	done, err := Import(context.Background(), folder, root, catalog, Options{Profile: "anna"}, ImportOptions{Fits: fits}, nil)
	if err != nil || done.Deleted != 1 || done.Placed != 0 {
		t.Errorf("%+v, %v; want the deleted photo left out and said", done, err)
	}
	done, err = Import(context.Background(), folder, root, catalog, Options{Profile: "anna"}, ImportOptions{Asked: Asked{BringBack: true}, Fits: fits}, nil)
	if err != nil || done.Placed != 1 {
		t.Errorf("%+v, %v; want the deleted photo placed again", done, err)
	}
	mustExist(t, filepath.Join(root, "anna", "2018", "01", "DSCF0002.JPG"))
}

// On a disk with no hard links, a photo's name in a second profile is a
// copy: the next run must know it for the same photo, not add another.
func TestACopyIsTheSamePhoto(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a/DSCF0001.JPG", []byte("one photo"))
	write(t, dir, "b/DSCF0001.JPG", []byte("one photo"))
	write(t, dir, "c/DSCF0001.JPG", []byte("another!!"))
	if !same(filepath.Join(dir, "a", "DSCF0001.JPG"), filepath.Join(dir, "b", "DSCF0001.JPG")) {
		t.Error("two copies of one photo were taken for two photos")
	}
	if same(filepath.Join(dir, "a", "DSCF0001.JPG"), filepath.Join(dir, "c", "DSCF0001.JPG")) {
		t.Error("two photos of the same size were taken for one")
	}
}

// A file being copied tells its bytes, ending with all of them.
func TestACopyTellsItsBytes(t *testing.T) {
	card := cameraCard(t)
	root := t.TempDir()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	catalog, err := OpenCatalog(filepath.Join(root, "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	folder, err := ReadFolder(card)
	if err != nil {
		t.Fatal(err)
	}
	whole := map[string]bool{}
	emit := func(e progress.Event) {
		if e.Stage == progress.Copy && e.Total > 0 && e.Done == e.Total {
			whole[e.Name] = true
		}
	}
	if _, err := Import(context.Background(), folder, root, catalog, Options{Profile: "ron"}, ImportOptions{Link: false, Fits: func(int64) error { return nil }}, emit); err != nil {
		t.Fatal(err)
	}
	if len(whole) != len(folder.Items) {
		t.Errorf("%d files told whole, want %d: %v", len(whole), len(folder.Items), whole)
	}
}

// Photos given a date go where that date says, keep their distance from
// each other, and carry it in a .xmp beside the copy: the photo itself is
// the card's, byte for byte.
func TestADateGivenGoesWithThePhoto(t *testing.T) {
	card := t.TempDir()
	write(t, card, "DCIM/100_FUJI/DSCF0004.JPG", exifPhoto("2018:01:01 00:01:36", "four"))
	write(t, card, "DCIM/100_FUJI/DSCF0005.JPG", exifPhoto("2018:01:01 00:02:46", "five"))
	root := t.TempDir()
	catalog, err := OpenCatalog(filepath.Join(root, "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	folder, err := ReadFolder(card)
	if err != nil {
		t.Fatal(err)
	}
	when := time.Date(2026, 9, 14, 10, 20, 0, 0, time.UTC)
	how := ImportOptions{Fits: func(int64) error { return nil }, Asked: Asked{Redates: []Redate{{Names: []string{"DSCF0004.JPG", "DSCF0005.JPG"}, When: when}}}}
	done, err := Import(context.Background(), folder, root, catalog, Options{Profile: "ron"}, how, nil)
	if err != nil {
		t.Fatal(err)
	}
	if done.Redated != 2 {
		t.Errorf("redated %d, want 2", done.Redated)
	}
	five := filepath.Join(root, "ron", "2026", "09", "DSCF0005.JPG")
	mustExist(t, five)
	content, err := os.ReadFile(five)
	if err != nil || string(content) != string(exifPhoto("2018:01:01 00:02:46", "five")) {
		t.Error("the photo was changed: the date goes beside it, not in it")
	}
	xmp, err := os.ReadFile(five + ".xmp")
	if err != nil {
		t.Fatal(err)
	}
	// Seventy seconds after the first, as on the card, with an offset.
	if !strings.Contains(string(xmp), `exif:DateTimeOriginal="2026-09-14T10:21:10`) || !regexp.MustCompile(`10:21:10[+-]\d\d:\d\d"`).Match(xmp) {
		t.Errorf("xmp says:\n%s", xmp)
	}
}

// A name given a date that the folder does not have is said before any
// copy: a typing mistake, not a photo to skip in silence.
func TestADateForANameNotThereIsSaid(t *testing.T) {
	card := cameraCard(t)
	root := t.TempDir()
	catalog, err := OpenCatalog(filepath.Join(root, "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	folder, err := ReadFolder(card)
	if err != nil {
		t.Fatal(err)
	}
	how := ImportOptions{Fits: func(int64) error { return nil }, Asked: Asked{Redates: []Redate{{Names: []string{"DSCF0099.JPG"}, When: time.Now()}}}}
	done, err := Import(context.Background(), folder, root, catalog, Options{Profile: "ron"}, how, nil)
	var unknown UnknownNameError
	if !errors.As(err, &unknown) || unknown.Name != "DSCF0099.JPG" || done.Placed != 0 {
		t.Errorf("%v, %+v; want DSCF0099.JPG said and nothing copied", err, done)
	}
}

func importDated(t *testing.T, card, root string, redates []Redate) (Imported, error) {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	catalog, err := OpenCatalog(filepath.Join(root, "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	folder, err := ReadFolder(card)
	if err != nil {
		t.Fatal(err)
	}
	return Import(context.Background(), folder, root, catalog, Options{Profile: "ron"}, ImportOptions{Asked: Asked{Redates: redates}, Fits: func(int64) error { return nil }}, nil)
}

// The likeliest use: a first import shows 2018 in the library, and the
// date is given on the next. The photo already in moves to its date.
func TestADateGivenToAPhotoAlreadyInMovesIt(t *testing.T) {
	card := t.TempDir()
	write(t, card, "DCIM/100_FUJI/DSCF0004.JPG", exifPhoto("2018:01:01 00:01:36", "four"))
	root := t.TempDir()
	if _, err := importDated(t, card, root, nil); err != nil {
		t.Fatal(err)
	}
	mustExist(t, filepath.Join(root, "ron", "2018", "01", "DSCF0004.JPG"))
	done, err := importDated(t, card, root, []Redate{{Names: []string{"dscf0004.jpg"}, When: time.Date(2026, 9, 14, 10, 20, 0, 0, time.UTC)}})
	if err != nil {
		t.Fatal(err)
	}
	if done.Redated != 1 || done.InLibrary != 1 {
		t.Errorf("%+v, want 1 redated, 1 in the library", done)
	}
	mustExist(t, filepath.Join(root, "ron", "2026", "09", "DSCF0004.JPG"))
	mustExist(t, filepath.Join(root, "ron", "2026", "09", "DSCF0004.JPG.xmp"))
	if exists(filepath.Join(root, "ron", "2018")) {
		t.Error("2018 is still there, empty or not")
	}
}

// Two cameras both name a photo DSCF0004.JPG: the name alone is refused,
// its path chooses, and the other photo keeps its date.
func TestANameInTwoFoldersIsChosenByItsPath(t *testing.T) {
	card := t.TempDir()
	write(t, card, "DCIM/100_FUJI/DSCF0004.JPG", exifPhoto("2018:01:01 00:01:36", "reset"))
	write(t, card, "DCIM/101_FUJI/DSCF0004.JPG", exifPhoto("2025:06:01 12:00:00", "right"))
	when := time.Date(2026, 9, 14, 10, 20, 0, 0, time.UTC)
	var twice AmbiguousNameError
	if _, err := importDated(t, card, t.TempDir(), []Redate{{Names: []string{"DSCF0004.JPG"}, When: when}}); !errors.As(err, &twice) || len(twice.Paths) != 2 {
		t.Fatalf("%v, want the name said twice", err)
	}
	root := t.TempDir()
	if _, err := importDated(t, card, root, []Redate{{Names: []string{"DCIM/100_FUJI/DSCF0004.JPG"}, When: when}}); err != nil {
		t.Fatal(err)
	}
	mustExist(t, filepath.Join(root, "ron", "2026", "09", "DSCF0004.JPG"))
	mustExist(t, filepath.Join(root, "ron", "2025", "06", "DSCF0004.JPG"))
}

// The XMP begins with the byte-order mark its packet header names, and a
// time the clock skipped keeps the time given.
func TestTheXMPIsWellFormed(t *testing.T) {
	photo := filepath.Join(t.TempDir(), "IMG_1.JPG")
	if err := writeXMP(photo, time.Date(2026, 3, 29, 2, 30, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	xmp, err := os.ReadFile(photo + ".xmp")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(xmp), "begin=\"\ufeff\"") {
		t.Error("the packet header has no byte-order mark")
	}
	if !strings.Contains(string(xmp), `"2026-03-29T02:30:00`) {
		t.Errorf("the time given is not kept:\n%s", xmp)
	}
}

// A dry run counts photos where the dates given put them.
func TestADryRunCountsTheDatesGiven(t *testing.T) {
	card := t.TempDir()
	write(t, card, "DCIM/DSCF0004.JPG", exifPhoto("2018:01:01 00:01:36", "four"))
	folder, err := ReadFolder(card)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanImport(context.Background(), card, folder, nil, PlanOptions{Asked: Asked{Redates: []Redate{{Names: []string{"DSCF0004.JPG"}, When: time.Date(2026, 9, 14, 10, 20, 0, 0, time.UTC)}}}}, nil)
	if err != nil || len(plan.Years) != 1 || plan.Years[0].Year != "2026" {
		t.Errorf("%+v, %v; want 2026", plan.Years, err)
	}
}

// A date given from a second profile does not take the photo away from the
// one that brought it first: it is said, and nothing moves.
func TestADateFromAnotherProfileLeavesThePhoto(t *testing.T) {
	card := t.TempDir()
	write(t, card, "DCIM/DSCF0004.JPG", exifPhoto("2018:01:01 00:01:36", "four"))
	root := t.TempDir()
	if _, err := importDated(t, card, root, nil); err != nil {
		t.Fatal(err)
	}
	catalog, err := OpenCatalog(filepath.Join(root, "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	folder, err := ReadFolder(card)
	if err != nil {
		t.Fatal(err)
	}
	how := ImportOptions{Fits: func(int64) error { return nil }, Asked: Asked{Redates: []Redate{{Names: []string{"DSCF0004.JPG"}, When: time.Date(2026, 9, 14, 10, 20, 0, 0, time.UTC)}}}}
	done, err := Import(context.Background(), folder, root, catalog, Options{Profile: "ann"}, how, nil)
	catalog.Close()
	if err != nil {
		t.Fatal(err)
	}
	if done.NotRedated != 1 || done.Redated != 0 {
		t.Errorf("%+v, want it said as another profile's", done)
	}
	mustExist(t, filepath.Join(root, "ron", "2018", "01", "DSCF0004.JPG"))
	mustExist(t, filepath.Join(root, "ann", "2018", "01", "DSCF0004.JPG"))
}

// Dating a profile's only undated photo leaves the profile's own folder.
func TestTheProfilesFolderStays(t *testing.T) {
	card := t.TempDir()
	write(t, card, "clip.mov", []byte("a video with no header"))
	root := t.TempDir()
	if _, err := importDated(t, card, root, nil); err != nil {
		t.Fatal(err)
	}
	mustExist(t, filepath.Join(root, "ron", "undated", "clip.mov"))
	if _, err := importDated(t, card, root, []Redate{{Names: []string{"clip.mov"}, When: time.Date(2026, 9, 14, 10, 20, 0, 0, time.UTC)}}); err != nil {
		t.Fatal(err)
	}
	mustExist(t, filepath.Join(root, "ron", "2026", "09", "clip.mov"))
	mustExist(t, filepath.Join(root, "ron"))
}

// A dry run says the photos already in that a date would move; a name is
// also taken as the shell completes it, whole or from ./.
func TestADryRunSaysWhatADateWouldMove(t *testing.T) {
	card := t.TempDir()
	write(t, card, "DCIM/DSCF0004.JPG", exifPhoto("2018:01:01 00:01:36", "four"))
	root := t.TempDir()
	if _, err := importDated(t, card, root, nil); err != nil {
		t.Fatal(err)
	}
	catalog, err := OpenCatalog(filepath.Join(root, "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	folder, err := ReadFolder(card)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{filepath.Join(card, "DCIM", "DSCF0004.JPG"), "./DCIM/DSCF0004.JPG"} {
		plan, err := PlanImport(context.Background(), card, folder, catalog, PlanOptions{Asked: Asked{Redates: []Redate{{Names: []string{name}, When: time.Now()}}}, Root: root}, nil)
		if err != nil || plan.Redated != 1 {
			t.Errorf("%s: %+v, %v; want 1 that would move", name, plan, err)
		}
	}
}

// A path completed by the shell from where the command runs names the
// file below the folder too; and a dry run from another profile says the
// date is not its to give, as the import does.
func TestADryRunTakesAPathFromHereAndKnowsTheProfile(t *testing.T) {
	card := t.TempDir()
	write(t, card, "DCIM/DSCF0004.JPG", exifPhoto("2018:01:01 00:01:36", "four"))
	root := t.TempDir()
	if _, err := importDated(t, card, root, nil); err != nil {
		t.Fatal(err)
	}
	catalog, err := OpenCatalog(filepath.Join(root, "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	folder, err := ReadFolder(card)
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(filepath.Dir(card))
	name := filepath.Join(filepath.Base(card), "DCIM", "DSCF0004.JPG")
	dates := []Redate{{Names: []string{name}, When: time.Now()}}
	if plan, err := PlanImport(context.Background(), card, folder, catalog, PlanOptions{Asked: Asked{Redates: dates}, Profile: "ron", Root: root}, nil); err != nil || plan.Redated != 1 {
		t.Errorf("%s from here: %+v, %v; want 1 that would move", name, plan, err)
	}
	if plan, err := PlanImport(context.Background(), card, folder, catalog, PlanOptions{Asked: Asked{Redates: dates}, Profile: "bob", Root: root}, nil); err != nil || plan.Redated != 0 || plan.NotRedated != 1 {
		t.Errorf("from another profile: %+v, %v; want 1 left to its own", plan, err)
	}
	// Run from inside the folder, a path below it still names it.
	t.Chdir(filepath.Join(card, "DCIM"))
	dates = []Redate{{Names: []string{"DCIM/DSCF0004.JPG"}, When: time.Now()}}
	if plan, err := PlanImport(context.Background(), card, folder, catalog, PlanOptions{Asked: Asked{Redates: dates}, Profile: "ron", Root: root}, nil); err != nil || plan.Redated != 1 {
		t.Errorf("from inside the folder: %+v, %v; want 1 that would move", plan, err)
	}
}

// A bare name is the file at the folder's top before another of the same
// name further down: A.JPG beside edited/A.JPG.
func TestABareNameIsTheTopFileFirst(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "A.JPG", []byte("top"))
	write(t, dir, "edited/A.JPG", []byte("edited"))
	folder, err := ReadFolder(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := named("a.jpg", folder); err != nil || got != filepath.Join(dir, "A.JPG") {
		t.Errorf("a.jpg: %s, %v; want the top one", got, err)
	}
}

// A photo already in the library, met again on a card, has the card for a
// source too; a deleted photo's twin given a date stays out with it.
func TestAKnownPhotoTakesTheCardAndADeletedTwinStaysOut(t *testing.T) {
	card := t.TempDir()
	write(t, card, "DCIM/A.JPG", exifPhoto("2018:01:01 00:01:36", "a"))
	write(t, card, "DCIM/B.JPG", exifPhoto("2019:07:14 10:00:00", "b"))
	write(t, card, "DCIM/sub/B.JPG", exifPhoto("2019:07:14 10:00:00", "b"))
	root := t.TempDir()
	catalog, err := OpenCatalog(filepath.Join(root, "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	folder, err := ReadFolder(card)
	if err != nil {
		t.Fatal(err)
	}
	fits := func(int64) error { return nil }
	if _, err := Import(context.Background(), folder, root, catalog, Options{Profile: "ron", Account: "google/ron@example.com"}, ImportOptions{Fits: fits}, nil); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "ron", "2019", "07", "B.JPG")); err != nil {
		t.Fatal(err)
	}
	given := []Redate{{Names: []string{"DCIM/sub/B.JPG"}, When: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}}
	done, err := Import(context.Background(), folder, root, catalog, Options{Profile: "ron", Account: "local/EOS_DIGITAL"}, ImportOptions{Asked: Asked{Redates: given}, Fits: fits}, nil)
	if err != nil || done.Deleted != 2 {
		t.Fatalf("%+v, %v; want B and its twin left out as deleted", done, err)
	}
	hash, err := hashOf(filepath.Join(card, "DCIM", "A.JPG"))
	if err != nil {
		t.Fatal(err)
	}
	_, sources, err := catalog.About(hash)
	if err != nil || len(sources) != 2 {
		t.Errorf("sources %v, %v; want Google and the card", sources, err)
	}
}
