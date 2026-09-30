// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The cases below are the ones that actually happen in a Takeout, and each one
// is a way a naive organiser loses or duplicates a photograph.

func TestSamePhotoInYearAndAlbumIsStoredOnce(t *testing.T) {
	src := t.TempDir()
	photo := []byte("the same bytes")
	write(t, src, "Photos from 2019/IMG_1.jpg", photo)
	sidecar(t, src, "Photos from 2019/IMG_1.jpg", 1562345678)
	write(t, src, "Greece 2019/IMG_1.jpg", photo)
	sidecar(t, src, "Greece 2019/IMG_1.jpg", 1562345678)

	out := t.TempDir()
	result, err := organize(t, src, out)
	if err != nil {
		t.Fatal(err)
	}

	if result.Placed != 1 {
		t.Errorf("stored %d copies of one photo, want 1", result.Placed)
	}
	if result.Skipped != 1 {
		t.Errorf("recognised %d duplicates, want 1", result.Skipped)
	}
	if result.Linked+result.Copied != 1 {
		t.Errorf("album entries %d, want 1", result.Linked+result.Copied)
	}
	mustExist(t, filepath.Join(out, "2019", "07", "IMG_1.jpg"))
	mustExist(t, filepath.Join(out, "albums", "Greece 2019", "IMG_1.jpg"))
}

// The sidecar holds what the photo often does not — description, place,
// people — so it is kept with the photo's library path, never thrown away.
func TestTheSidecarIsKeptBesideThePhotosPath(t *testing.T) {
	src := t.TempDir()
	write(t, src, "Photos from 2019/IMG_1.jpg", []byte("a photo"))
	sidecar(t, src, "Photos from 2019/IMG_1.jpg", 1562345678)

	out := t.TempDir()
	if _, err := organize(t, src, out); err != nil {
		t.Fatal(err)
	}
	mustExist(t, filepath.Join(out, WorkDir, "metadata", "2019", "07", "IMG_1.jpg.json"))
}

// Google handed the 344 GB export the same 8.64 GB video twice, as two separate
// parts, and said nothing about it. Content hashing is what catches that.
func TestTheSameVideoDeliveredTwiceCostsSpaceOnce(t *testing.T) {
	src := t.TempDir()
	video := []byte("one very large video")
	write(t, src, "Photos from 2018/FXT28182.MOV", video)
	sidecar(t, src, "Photos from 2018/FXT28182.MOV", 1531000000)
	write(t, src, "Photos from 2019/FXT28182.MOV", video)
	sidecar(t, src, "Photos from 2019/FXT28182.MOV", 1531000000)

	result, err := organize(t, src, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if result.Placed != 1 || result.Skipped != 1 {
		t.Errorf("placed %d skipped %d, want 1 and 1", result.Placed, result.Skipped)
	}
	if want := int64(len(video)); result.Bytes != want {
		t.Errorf("counted %d bytes, want %d — a duplicate must cost nothing", result.Bytes, want)
	}
}

// Two cameras both make IMG_0001.jpg. Same name, different picture: keeping one
// would be losing the other.
func TestSameNameDifferentPhotoKeepsBoth(t *testing.T) {
	src := t.TempDir()
	write(t, src, "Photos from 2019/IMG_0001.jpg", []byte("from the phone"))
	sidecar(t, src, "Photos from 2019/IMG_0001.jpg", 1562345678)
	write(t, src, "Greece 2019/IMG_0001.jpg", []byte("from the camera"))
	sidecar(t, src, "Greece 2019/IMG_0001.jpg", 1562345678)

	out := t.TempDir()
	result, err := organize(t, src, out)
	if err != nil {
		t.Fatal(err)
	}
	if result.Placed != 2 {
		t.Fatalf("placed %d, want 2 — both photos must survive", result.Placed)
	}
	mustExist(t, filepath.Join(out, "2019", "07", "IMG_0001.jpg"))
	mustExist(t, filepath.Join(out, "2019", "07", "IMG_0001-2.jpg"))
}

// No sidecar, no EXIF: the year folder is Google's own filing, and it is better
// than nothing — but the month is not known and the path says so.
func TestNoSidecarNoExifFallsBackToTheYearFolder(t *testing.T) {
	src := t.TempDir()
	write(t, src, "Photos from 2016/scan.png", []byte("not a jpeg, no exif here"))

	out := t.TempDir()
	result, err := organize(t, src, out)
	if err != nil {
		t.Fatal(err)
	}
	if result.BySource[FromFolder] != 1 {
		t.Errorf("date sources %v, want one from the folder", result.BySource)
	}
	if result.Undated != 0 {
		t.Errorf("undated %d, want 0 — the year was known", result.Undated)
	}
	mustExist(t, filepath.Join(out, "2016", "unknown-month", "scan.png"))
}

// Nothing at all: no sidecar, no EXIF, not even a year. The photo is kept and
// the program says it does not know, rather than inventing a date.
func TestNothingKnownIsPlacedAndAdmitted(t *testing.T) {
	src := t.TempDir()
	write(t, src, "Untitled/mystery.png", []byte("no metadata anywhere"))

	out := t.TempDir()
	result, err := organize(t, src, out)
	if err != nil {
		t.Fatal(err)
	}
	if result.Undated != 1 {
		t.Errorf("undated %d, want 1", result.Undated)
	}
	mustExist(t, filepath.Join(out, "undated", "mystery.png"))
	// It is in an album folder, so it must still appear under that album.
	mustExist(t, filepath.Join(out, "albums", "Untitled", "mystery.png"))
}

// The sidecar wins over everything: it is Google's own answer, and it survives
// files whose metadata was stripped on upload.
func TestSidecarBeatsTheYearFolder(t *testing.T) {
	src := t.TempDir()
	// Filed under 2019 by Takeout, but the sidecar says July 2007.
	write(t, src, "Photos from 2019/old.jpg", []byte("scanned long after it was taken"))
	sidecar(t, src, "Photos from 2019/old.jpg", 1183000000)

	out := t.TempDir()
	result, err := organize(t, src, out)
	if err != nil {
		t.Fatal(err)
	}
	if result.BySource[FromSidecar] != 1 {
		t.Errorf("date sources %v, want one from the sidecar", result.BySource)
	}
	mustExist(t, filepath.Join(out, "2007", "06", "old.jpg"))
}

// Takeout truncates the sidecar suffix when the whole name would be too long.
// A program that builds "<file>.supplemental-metadata.json" and stops there
// misses exactly those, which are the files with the longest names.
func TestTruncatedSidecarIsStillFound(t *testing.T) {
	src := t.TempDir()
	name := "a-very-long-original-filename-from-a-messaging-app.jpg"
	write(t, src, "Photos from 2019/"+name, []byte("photo"))
	writeJSON(t, src, "Photos from 2019/"+name+".supplemental-met.json", 1562345678)

	result, err := organize(t, src, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if result.BySource[FromSidecar] != 1 {
		t.Errorf("date sources %v, want one from the sidecar", result.BySource)
	}
}

// The modification time is the one source that is always available and always
// wrong: Takeout stamps every file with the day it built the archive.
func TestModificationTimeIsNeverUsedAsADate(t *testing.T) {
	src := t.TempDir()
	write(t, src, "Untitled/no-metadata.png", []byte("mtime is today, the photo is not"))

	capture := DateOf(filepath.Join(src, "Untitled", "no-metadata.png"), "Untitled")
	if capture.Source != NoDate {
		t.Errorf("date source %q, want %q — mtime must not be trusted", capture.Source, NoDate)
	}
}

// A run that stops between recording a photo and moving it must finish the
// move next time, at the place it promised, and not file the photo twice.
func TestAnInterruptedMoveIsFinishedWhereItWasPromised(t *testing.T) {
	src := t.TempDir()
	write(t, src, "Photos from 2019/IMG_1.jpg", []byte("the photo"))
	sidecar(t, src, "Photos from 2019/IMG_1.jpg", 1562345678)
	root := t.TempDir()

	catalog, err := OpenCatalog(filepath.Join(t.TempDir(), "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	items, err := ReadTakeout(src)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := hashOf(items[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	// What the first run managed before it stopped: the record, not the move.
	if err := catalog.Put(Photo{Hash: hash, Path: "2019/07/IMG_1.jpg", Name: "IMG_1.jpg"}); err != nil {
		t.Fatal(err)
	}

	if _, err := Organize(items, root, catalog, Options{Location: time.UTC}, nil); err != nil {
		t.Fatal(err)
	}
	mustExist(t, filepath.Join(root, "2019", "07", "IMG_1.jpg"))
	if _, err := os.Stat(filepath.Join(root, "2019", "07", "IMG_1-2.jpg")); err == nil {
		t.Error("the photo was filed twice")
	}
	if exists(items[0].Path) {
		t.Error("the photo was left behind instead of moved")
	}
}

// organize runs a Takeout tree through the organiser the way the engine does.
func organize(t *testing.T, src, root string) (Organized, error) {
	t.Helper()
	catalog, err := OpenCatalog(filepath.Join(t.TempDir(), "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	items, err := ReadTakeout(src)
	if err != nil {
		t.Fatal(err)
	}
	return Organize(items, root, catalog, Options{Location: time.UTC}, nil)
}

func write(t *testing.T, dir, relative string, content []byte) {
	t.Helper()
	path := filepath.Join(dir, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func sidecar(t *testing.T, dir, mediaRelative string, timestamp int64) {
	t.Helper()
	writeJSON(t, dir, mediaRelative+".supplemental-metadata.json", timestamp)
}

func writeJSON(t *testing.T, dir, relative string, timestamp int64) {
	t.Helper()
	body := fmt.Sprintf(`{"photoTakenTime":{"timestamp":"%d"}}`, timestamp)
	write(t, dir, relative, []byte(body))
}

func mustExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected %s to exist: %v", path, err)
	}
}
