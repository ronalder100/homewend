// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"
)

// exifPhoto is the smallest file the date reader takes for a camera's JPEG:
// an EXIF block whose DateTimeOriginal is when, then bytes of its own.
func exifPhoto(when, body string) []byte {
	tiff := "II*\x00\x08\x00\x00\x00" + // little-endian TIFF, first IFD at 8
		"\x01\x00" + // one entry
		"\x03\x90\x02\x00\x14\x00\x00\x00\x1a\x00\x00\x00" + // DateTimeOriginal, ASCII, 20 bytes, at 26
		"\x00\x00\x00\x00" + // no next IFD
		when + "\x00"
	return []byte("\xff\xd8\xff\xe1\x00\x40Exif\x00\x00" + tiff + body)
}

// What a camera's card read on a Mac holds, measured 09/11: the photos under
// DCIM, the camera's own files beside them, and the Mac's in hidden folders.
func cameraCard(t *testing.T) string {
	card := t.TempDir()
	write(t, card, "DCIM/100_FUJI/DSCF0001.JPG", exifPhoto("2026:08:15 18:23:12", "one"))
	write(t, card, "DCIM/100_FUJI/DSCF0002.JPG", exifPhoto("2018:01:01 00:01:36", "two"))
	write(t, card, "DCIM/100_FUJI/DSCF0003.MOV", []byte("a video, no date the reader knows"))
	write(t, card, "DCIM/100_FUJI/._DSCF0001.JPG", []byte("the Mac's twin of a photo"))
	write(t, card, ".Spotlight-V100/Store-V2/store.db", []byte("the Mac's index"))
	write(t, card, "FWUP0013.DAT", []byte("firmware"))
	write(t, card, "FFDB/FFXFER.DAT", []byte("the camera's database"))
	return card
}

func TestAFolderListsItsMediaAndWhatElseIsThere(t *testing.T) {
	card := cameraCard(t)
	folder, err := ReadFolder(card)
	if err != nil {
		t.Fatal(err)
	}
	if len(folder.Items) != 3 {
		t.Errorf("%d photos and videos, want 3: %v", len(folder.Items), folder.Items)
	}
	want := []string{filepath.Join(card, "FFDB", "FFXFER.DAT"), filepath.Join(card, "FWUP0013.DAT")}
	if len(folder.Others) != len(want) || folder.Others[0] != want[0] || folder.Others[1] != want[1] {
		t.Errorf("others %v, want %v: the system's files are no one's", folder.Others, want)
	}
}

func TestThePlanCountsByYearAndFindsWhatIsAlreadyThere(t *testing.T) {
	card := cameraCard(t)
	// The library already holds the photo of 2026, under another name.
	lib := t.TempDir()
	write(t, lib, "2026/08/renamed.jpg", []byte("the library's file"))
	catalog, err := OpenCatalog(filepath.Join(lib, "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	hash, err := hashOf(filepath.Join(card, "DCIM", "100_FUJI", "DSCF0001.JPG"))
	if err != nil {
		t.Fatal(err)
	}
	if err := catalog.Put(Photo{Hash: hash, Path: "2026/08/renamed.jpg", Name: "renamed.jpg"}); err != nil {
		t.Fatal(err)
	}
	// And the card holds one photo twice.
	write(t, card, "DCIM/101_FUJI/DSCF0002.JPG", exifPhoto("2018:01:01 00:01:36", "two"))

	folder, err := ReadFolder(card)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanImport(context.Background(), card, folder, catalog, PlanOptions{Root: lib}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.InLibrary != 1 {
		t.Errorf("%d already in the library, want 1: found by content, not name", plan.InLibrary)
	}
	if plan.Twins != 1 {
		t.Errorf("%d twins, want 1", plan.Twins)
	}
	if plan.Photos != 1 || plan.Videos != 1 {
		t.Errorf("%d photos and %d videos to copy, want 1 and 1", plan.Photos, plan.Videos)
	}
	// Each year with its months; the undated have none.
	want := []ImportYear{{Year: "2018", Photos: 1, Months: []ImportMonth{{Month: 1, Photos: 1}}}, {Year: "", Videos: 1}}
	if !reflect.DeepEqual(plan.Years, want) {
		t.Errorf("years %+v, want %+v: newest first, the undated last", plan.Years, want)
	}
	if len(plan.Others) != 2 || plan.Others[0].Name != "FFDB/" || plan.Others[1].Name != "FWUP0013.DAT" {
		t.Errorf("others %+v, want FFDB/ and FWUP0013.DAT, as at the card's top", plan.Others)
	}
}

func TestAnEmptyLibraryTakesEverything(t *testing.T) {
	card := cameraCard(t)
	folder, err := ReadFolder(card)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanImport(context.Background(), card, folder, nil, PlanOptions{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Photos != 2 || plan.Videos != 1 || plan.InLibrary != 0 {
		t.Errorf("plan %+v, want 2 photos, 1 video, none in the library", plan)
	}
}

// A card with one damaged file still says what the rest holds.
func TestAFileThatCannotBeReadIsPassedOver(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads every file")
	}
	card := cameraCard(t)
	locked := filepath.Join(card, "DCIM", "100_FUJI", "DSCF0009.JPG")
	write(t, card, "DCIM/100_FUJI/DSCF0009.JPG", exifPhoto("2026:08:16 10:00:00", "nine"))
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o644) })

	folder, err := ReadFolder(card)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanImport(context.Background(), card, folder, nil, PlanOptions{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Unreadable != 1 || plan.Photos != 2 || plan.Videos != 1 {
		t.Errorf("plan %+v, want 1 unreadable and the other 3 counted", plan)
	}
}

// A folder copied out of a Takeout is not read by Takeout's rules: a folder
// named for a year is not a date.
func TestAFolderNameIsNotADate(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "Photos from 2019/IMG_1.jpg", []byte("no camera date"))
	folder, err := ReadFolder(dir)
	if err != nil {
		t.Fatal(err)
	}
	// With no camera's date it is unsure, and comes in when ticked.
	choice := &Choice{In: []string{"Photos from 2019"}}
	plan, err := PlanImport(context.Background(), dir, folder, nil, PlanOptions{Asked: Asked{Choice: choice}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Years) != 1 || plan.Years[0].Year != "" {
		t.Errorf("years %+v, want only the undated", plan.Years)
	}
}

// A download stopped by closing its terminal leaves its last rows in the
// -wal beside the catalog: a look must see them.
func TestALookSeesRowsStillInTheWal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog.db")
	writer, err := OpenCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	if err := writer.Put(Photo{Hash: "late", Path: "2026/08/late.jpg", Name: "late.jpg"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + "-wal"); err != nil {
		t.Skip("the row went straight into the catalog file")
	}
	look, err := OpenCatalogReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	defer look.Close()
	if _, known, err := look.PathOf("late"); err != nil || !known {
		t.Errorf("known %v, %v: a row in the -wal is in the library", known, err)
	}
}

// Looking at the library before an import writes nothing into it.
func TestTheCatalogIsOnlyRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog.db")
	catalog, err := OpenCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := catalog.Put(Photo{Hash: "h", Path: "2026/08/a.jpg", Name: "a.jpg"}); err != nil {
		t.Fatal(err)
	}
	catalog.Close()
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	look, err := OpenCatalogReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, known, err := look.PathOf("h"); err != nil || !known {
		t.Errorf("known %v, %v: the photo should be found", known, err)
	}
	if err := look.Put(Photo{Hash: "x", Path: "x", Name: "x"}); err == nil {
		t.Error("a write went through a catalog opened to read")
	}
	look.Close()
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Error("the catalog changed while it was only looked at")
	}
	for _, beside := range []string{"-shm", "-wal"} {
		if _, err := os.Stat(path + beside); err == nil {
			t.Errorf("%s left beside the catalog by a look", beside)
		}
	}
}

// A second look at the same card, with dates given, reads no file again: a
// file's content is taken from the first look while it has not changed.
// The files are listed with their camera's dates.
func TestASecondLookReadsNothingAgain(t *testing.T) {
	card := cameraCard(t)
	folder, err := ReadFolder(card)
	if err != nil {
		t.Fatal(err)
	}
	hashes := Hashes{}
	plan, err := PlanImport(context.Background(), card, folder, nil, PlanOptions{Hashes: hashes}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(hashes) != 3 || len(plan.Files) != 3 {
		t.Fatalf("%d read, %d listed; want the 3 photos and video", len(hashes), len(plan.Files))
	}
	one := ImportFile{Path: "DCIM/100_FUJI/DSCF0001.JPG", Taken: time.Date(2026, 8, 15, 18, 23, 12, 0, time.UTC)}
	if !slices.ContainsFunc(plan.Files, func(f ImportFile) bool { return f.Path == one.Path && f.Taken.Equal(one.Taken) }) {
		t.Errorf("files %+v, want %+v among them", plan.Files, one)
	}
	// The library holds what the first look said one file is: the second
	// look finds it there without reading the file.
	path := filepath.Join(card, "DCIM", "100_FUJI", "DSCF0001.JPG")
	known := hashes[path]
	known.hash = "said by the first look"
	hashes[path] = known
	lib := t.TempDir()
	write(t, lib, "2026/08/DSCF0001.JPG", []byte("the library's file"))
	catalog, err := OpenCatalog(filepath.Join(lib, "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	if err := catalog.Put(Photo{Hash: known.hash, Path: "2026/08/DSCF0001.JPG", Name: "DSCF0001.JPG"}); err != nil {
		t.Fatal(err)
	}
	plan, err = PlanImport(context.Background(), card, folder, catalog, PlanOptions{Hashes: hashes, Root: lib}, nil)
	if err != nil || plan.InLibrary != 1 {
		t.Errorf("second look: %d in the library, %v; want the one the first look read", plan.InLibrary, err)
	}
}

// What stays out is named ten at most; the rest are counted.
func TestWhatStaysOutIsNamedTenAtMost(t *testing.T) {
	dir := t.TempDir()
	for i := range 12 {
		write(t, dir, fmt.Sprintf("doc%02d.pdf", i), []byte("not a photo"))
	}
	folder, err := ReadFolder(dir)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanImport(context.Background(), dir, folder, nil, PlanOptions{}, nil)
	if err != nil || len(plan.Others) != MaxOthers || plan.OthersMore != 2 || plan.OtherFiles != 12 {
		t.Errorf("%d named, %d more, %d files, %v; want %d, 2 and all 12", len(plan.Others), plan.OthersMore, plan.OtherFiles, err, MaxOthers)
	}
}
