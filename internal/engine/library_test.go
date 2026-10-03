// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ronalder100/homewend/internal/library"
)

func catalogWith(t *testing.T, dir string, photos ...library.Photo) {
	t.Helper()
	os.MkdirAll(filepath.Join(dir, library.WorkDir), 0o755)
	c, err := library.OpenCatalog(catalogPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	for _, p := range photos {
		if err := c.Put(p); err != nil {
			t.Fatal(err)
		}
	}
}

func TestOverviewSumsShownAccounts(t *testing.T) {
	root := t.TempDir()
	y := func(year int) time.Time { return time.Date(year, 6, 1, 0, 0, 0, 0, time.UTC) }
	catalogWith(t, LibraryOf(root, "ann@example.com"),
		library.Photo{Hash: "a1", Path: "a1", Name: "a1", Taken: y(2024)},
		library.Photo{Hash: "a2", Path: "a2", Name: "a2"})
	catalogWith(t, LibraryOf(root, "bo@example.com"),
		library.Photo{Hash: "b1", Path: "b1", Name: "b1", Taken: y(2024)},
		library.Photo{Hash: "b2", Path: "b2", Name: "b2", Taken: y(2019)})
	accounts := []AccountInfo{
		{ID: "1", Email: "ann@example.com", Profile: t.TempDir()},
		{ID: "2", Email: "bo@example.com", Profile: t.TempDir()},
		{ID: "3", Profile: t.TempDir()}, // not signed in yet
	}

	o, err := overview(root, accounts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if o.Total != 4 || len(o.Years) != 3 || o.Years[0] != (YearCount{"2024", 2}) ||
		o.Years[1] != (YearCount{"2019", 1}) || o.Years[2] != (YearCount{"", 1}) {
		t.Fatalf("both shown: %+v", o)
	}

	o, err = overview(root, accounts, []string{"2"})
	if err != nil {
		t.Fatal(err)
	}
	if o.Total != 2 || len(o.Accounts) != 2 || !o.Accounts[0].Shown || o.Accounts[1].Shown {
		t.Fatalf("bo hidden: %+v", o)
	}
}
