// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"path/filepath"
	"testing"
	"time"
)

// An older catalog kept Google's dates as instants in UTC: opened, they
// become the clock of this computer's zone, once, with a copy kept first.
func TestAnOlderCatalogsGoogleDatesBecomeClocks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog.db")
	catalog, err := OpenCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	instant := time.Date(2019, 12, 31, 23, 30, 0, 0, time.UTC)
	if err := catalog.Put(Photo{Hash: "g", Path: "2019/12/a.jpg", Name: "a.jpg", Taken: instant, Source: FromSidecar}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.Put(Photo{Hash: "e", Path: "2019/12/b.jpg", Name: "b.jpg", Taken: instant, Source: FromEXIF}); err != nil {
		t.Fatal(err)
	}
	// As before this version.
	if _, err := catalog.db.Exec(`PRAGMA user_version = 0`); err != nil {
		t.Fatal(err)
	}
	catalog.Close()

	for run := 0; run < 2; run++ {
		catalog, err := OpenCatalog(path)
		if err != nil {
			t.Fatal(err)
		}
		google, err := catalog.photo("g")
		if err != nil {
			t.Fatal(err)
		}
		if want := asClock(instant.In(time.Local)); !google.Taken.Equal(want) {
			t.Errorf("run %d: Google's date %v, want the clock %v", run, google.Taken, want)
		}
		camera, err := catalog.photo("e")
		if err != nil {
			t.Fatal(err)
		}
		if !camera.Taken.Equal(instant) {
			t.Errorf("run %d: the camera's clock moved to %v", run, camera.Taken)
		}
		catalog.Close()
	}
	if !exists(path + ".before-clocks") {
		t.Error("no copy of the catalog as it was")
	}
}

// A download keeps Google's date as the clock its folder is chosen by, and
// a date read back from the catalog is not moved a second time.
func TestGooglesDateIsKeptAsItsFoldersClock(t *testing.T) {
	src := t.TempDir()
	write(t, src, "Photos from 2019/IMG_1.jpg", []byte("a photo"))
	// 22:30 UTC on 31 December is 23:30 in Rome: still 2019.
	sidecar(t, src, "Photos from 2019/IMG_1.jpg", time.Date(2019, 12, 31, 22, 30, 0, 0, time.UTC).Unix())
	rome, err := time.LoadLocation("Europe/Rome")
	if err != nil {
		t.Skip("no timezone database")
	}
	out := t.TempDir()
	catalog, err := OpenCatalog(filepath.Join(t.TempDir(), "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	items, err := ReadTakeout(src)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Organize(items, out, catalog, Options{Location: rome}, nil); err != nil {
		t.Fatal(err)
	}
	mustExist(t, filepath.Join(out, "2019", "12", "IMG_1.jpg"))
	hash, err := hashOf(filepath.Join(out, "2019", "12", "IMG_1.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	photo, err := catalog.photo(hash)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2019, 12, 31, 23, 30, 0, 0, time.UTC); !photo.Taken.Equal(want) {
		t.Errorf("kept %v, want the clock %v", photo.Taken, want)
	}
	if got := photo.capture().Local(rome); !got.Equal(photo.Taken) {
		t.Errorf("read back and moved again: %v", got)
	}
}
