// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
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

// The window opens the catalog for several requests at once: the first
// opening after an update turns the dates once, and none of them fails.
func TestAnOlderCatalogOpenedManyTimesAtOnceTurnsOnce(t *testing.T) {
	rome, err := time.LoadLocation("Europe/Rome")
	if err != nil {
		t.Skip(err)
	}
	was := time.Local
	time.Local = rome
	t.Cleanup(func() { time.Local = was })

	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.db")
	catalog, err := OpenCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	instant := time.Date(2019, 7, 14, 12, 0, 0, 0, time.UTC)
	for i := range 200 {
		name := fmt.Sprintf("%d.jpg", i)
		if err := catalog.Put(Photo{Hash: name, Path: "2019/07/" + name, Name: name, Taken: instant, Source: FromSidecar}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := catalog.db.Exec(`PRAGMA user_version = 0`); err != nil {
		t.Fatal(err)
	}
	catalog.Close()

	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, err := OpenCatalog(path)
			if err != nil {
				errs <- err
				return
			}
			c.Close()
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("an opening failed: %v", err)
	}
	catalog, err = OpenCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	p, err := catalog.photo("7.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if want := asClock(instant.In(rome)); !p.Taken.Equal(want) {
		t.Errorf("date %v, want %v: turned once", p.Taken, want)
	}
	copies, _ := filepath.Glob(path + ".before-clocks*")
	if len(copies) != 1 || copies[0] != path+".before-clocks" {
		t.Errorf("copies kept: %v, want the one", copies)
	}
}

// The one statement that turns an older catalog gives each instant what
// asClock gives it: across summer time, and at the very second it changes.
func TestTheOffsetsAreAsClocksAre(t *testing.T) {
	rome, err := time.LoadLocation("Europe/Rome")
	if err != nil {
		t.Skip(err)
	}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	first := time.Date(2003, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	last := time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC).Unix()
	expr := strings.ReplaceAll(offsets(first, last, rome), "taken", "?1")
	check := func(at int64) {
		var got int64
		if err := db.QueryRow(`SELECT ?1 + `+expr, at).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if want := asClock(time.Unix(at, 0).In(rome)).Unix(); got != want {
			t.Errorf("%v: %d, want %d", time.Unix(at, 0).UTC(), got, want)
		}
	}
	for at := first; at <= last; at += 86400*3 + 3607 {
		check(at)
	}
	// A date far in the future, a sidecar's mistake, ends: the spans after
	// 2037 must still move on.
	if far := offsets(first, time.Date(2045, 1, 1, 0, 0, 0, 0, time.UTC).Unix(), rome); !strings.HasPrefix(far, "CASE") {
		t.Errorf("to 2045: %.40s", far)
	}
	// The change to summer time on 30 March 2025, 01:00 UTC.
	change := time.Date(2025, 3, 30, 1, 0, 0, 0, time.UTC).Unix()
	for _, at := range []int64{change - 1, change, change + 1, first, last} {
		check(at)
	}
}
