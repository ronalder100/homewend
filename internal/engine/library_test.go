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

func TestOverviewOfTheLibrary(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, library.WorkDir), 0o755)
	c, err := library.OpenCatalog(catalogPath(root))
	if err != nil {
		t.Fatal(err)
	}
	y := func(year int) time.Time { return time.Date(year, 6, 1, 0, 0, 0, 0, time.UTC) }
	for _, p := range []library.Photo{
		{Hash: "a", Path: "a", Name: "a", Taken: y(2024)},
		{Hash: "b", Path: "b", Name: "b", Taken: y(2024)},
		{Hash: "c", Path: "c", Name: "c", Taken: y(2019)},
		{Hash: "d", Path: "d", Name: "d"},
	} {
		c.Put(p)
	}
	c.Close()
	accounts := []AccountInfo{
		{ID: "profile", Profile: t.TempDir()}, // signed out
		{ID: "profile-sam", Email: "sam@example.com", Profile: t.TempDir()},
	}

	o, err := overview(root, accounts, []string{"profile-sam"})
	if err != nil {
		t.Fatal(err)
	}
	if len(o.Accounts) != 1 || o.Accounts[0].Shown || o.Total != 4 {
		t.Fatalf("got %+v", o)
	}
	want := []YearCount{{"2024", 2}, {"2019", 1}, {"", 1}}
	for i, w := range want {
		if o.Years[i] != w {
			t.Fatalf("years %+v, want %+v", o.Years, want)
		}
	}
}

func TestOverviewWithoutALibrary(t *testing.T) {
	o, err := overview("", nil, nil)
	if err != nil || o.Total != 0 {
		t.Fatalf("got %+v, %v", o, err)
	}
}
