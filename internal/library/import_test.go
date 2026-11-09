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
	done, err := Import(ctx, folder, root, catalog, Options{Profile: "ruvido"}, ImportOptions{Link: link, Fits: fits}, nil)
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
	mustExist(t, filepath.Join(root, "ruvido", "2026", "08", "DSCF0001.JPG"))
	mustExist(t, filepath.Join(root, "ruvido", "2018", "01", "DSCF0002.JPG"))
	mustExist(t, filepath.Join(root, "ruvido", "undated", "DSCF0003.MOV"))
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
	content, err := os.ReadFile(filepath.Join(root, "ruvido", "2026", "08", "DSCF0001.JPG"))
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
	b, err := os.Stat(filepath.Join(root, "ruvido", "2019", "07", "IMG_1.JPG"))
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
	info, err := os.Lstat(filepath.Join(root, "ruvido", "2019", "07", "IMG_2.JPG"))
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

// A photo deleted from the library and brought in again is new: placed in
// the profile it is brought into, and counted.
func TestAPhotoDeletedFromTheLibraryComesBackAsNew(t *testing.T) {
	card := cameraCard(t)
	root := t.TempDir()
	importCard(t, card, root, false)
	gone := filepath.Join(root, "ruvido", "2018", "01", "DSCF0002.JPG")
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
	done, err := Import(context.Background(), folder, root, catalog, Options{Profile: "anna"}, ImportOptions{Link: false, Fits: func(int64) error { return nil }}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if done.Placed != 1 {
		t.Errorf("%+v, want the deleted photo placed again", done)
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
	if _, err := Import(context.Background(), folder, root, catalog, Options{Profile: "ruvido"}, ImportOptions{Link: false, Fits: func(int64) error { return nil }}, emit); err != nil {
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
	how := ImportOptions{Fits: func(int64) error { return nil }, Redates: []Redate{{Names: []string{"DSCF0004.JPG", "DSCF0005.JPG"}, When: when}}}
	done, err := Import(context.Background(), folder, root, catalog, Options{Profile: "ruvido"}, how, nil)
	if err != nil {
		t.Fatal(err)
	}
	if done.Redated != 2 {
		t.Errorf("redated %d, want 2", done.Redated)
	}
	five := filepath.Join(root, "ruvido", "2026", "09", "DSCF0005.JPG")
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
	how := ImportOptions{Fits: func(int64) error { return nil }, Redates: []Redate{{Names: []string{"DSCF0099.JPG"}, When: time.Now()}}}
	done, err := Import(context.Background(), folder, root, catalog, Options{Profile: "ruvido"}, how, nil)
	var unknown UnknownNameError
	if !errors.As(err, &unknown) || unknown.Name != "DSCF0099.JPG" || done.Placed != 0 {
		t.Errorf("%v, %+v; want DSCF0099.JPG said and nothing copied", err, done)
	}
}
