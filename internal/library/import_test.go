// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"os"
	"path/filepath"
	"testing"
)

func importCard(t *testing.T, card, root string, link bool) Organized {
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
	done, err := Import(folder, root, catalog, Options{Profile: "ruvido"}, link, nil)
	if err != nil {
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

	if done.Placed != 3 || done.Skipped != 1 || done.Undated != 1 {
		t.Errorf("placed %d, skipped %d, undated %d; want 3, 1 (the twin), 1 (the video)", done.Placed, done.Skipped, done.Undated)
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
	if again.Placed != 0 || again.Skipped != 3 {
		t.Errorf("second run placed %d and skipped %d, want 0 and 3", again.Placed, again.Skipped)
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
