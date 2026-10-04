// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A library filed before profiles moves into the profile of its photos'
// source: the photo, its sidecar, its album; nothing is left behind and
// nothing is copied.
func TestAnOldLibraryMovesIntoItsProfile(t *testing.T) {
	src, root := t.TempDir(), t.TempDir()
	write(t, src, "Photos from 2019/IMG_1.jpg", []byte("a photo"))
	sidecar(t, src, "Photos from 2019/IMG_1.jpg", 1562345678)
	write(t, src, "Greece 2019/IMG_1.jpg", []byte("a photo"))
	sidecar(t, src, "Greece 2019/IMG_1.jpg", 1562345678)
	if err := os.MkdirAll(filepath.Join(root, WorkDir), 0o755); err != nil {
		t.Fatal(err)
	}
	catalog, err := OpenCatalog(filepath.Join(root, WorkDir, "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { catalog.Close() }()
	items, err := ReadTakeout(src)
	if err != nil {
		t.Fatal(err)
	}
	// As an older homewend filed it: at the root, by year and month.
	if _, err := Organize(items, root, catalog, Options{Location: time.UTC, Account: "ron@example.com"}, nil); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(root, "2019", "07", "IMG_1.jpg")
	if err := os.MkdirAll(filepath.Join(root, "2019", "07"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(old, filepath.Join(root, "2019", "07", "IMG_1.jpg")); err != nil {
		t.Fatal(err)
	}
	os.Remove(filepath.Dir(old))
	if err := catalog.setPath(mustHash(t, filepath.Join(root, "2019", "07", "IMG_1.jpg")), "2019/07/IMG_1.jpg", "IMG_1.jpg"); err != nil {
		t.Fatal(err)
	}
	if !Filed(root) {
		t.Fatal("an old library is not seen as one")
	}

	// Reopened, as a new homewend does: the sources get their service.
	catalog.Close()
	if catalog, err = OpenCatalog(filepath.Join(root, WorkDir, "catalog.db")); err != nil {
		t.Fatal(err)
	}
	profileOf := func(source string) string {
		if source == "google/ron@example.com" {
			return "ron"
		}
		return ""
	}
	if err := Relayout(root, catalog, profileOf, "ron", time.UTC); err != nil {
		t.Fatal(err)
	}
	mustExist(t, filepath.Join(root, "ron", "2019", "07", "IMG_1.jpg"))
	mustExist(t, filepath.Join(root, "ron", "albums", "Greece 2019", "IMG_1.jpg"))
	for _, gone := range []string{"2019", "albums"} {
		if _, err := os.Stat(filepath.Join(root, gone)); !os.IsNotExist(err) {
			t.Errorf("%s is still there", gone)
		}
	}
	if Filed(root) {
		t.Error("the library is still seen as old")
	}
	if p, _, _ := catalog.PathOf(mustHash(t, filepath.Join(root, "ron", "2019", "07", "IMG_1.jpg"))); p != "ron/2019/07/IMG_1.jpg" {
		t.Errorf("the catalog says %q", p)
	}
}
