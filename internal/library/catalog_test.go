// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"path/filepath"
	"testing"
	"time"
)

func TestCountsByYear(t *testing.T) {
	c, err := OpenCatalog(filepath.Join(t.TempDir(), "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	put := func(hash string, taken time.Time) {
		if err := c.Put(Photo{Hash: hash, Path: hash, Name: hash, Taken: taken}); err != nil {
			t.Fatal(err)
		}
	}
	put("a", time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC))
	put("b", time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC))
	put("c", time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC))
	put("d", time.Time{})

	got, err := c.CountsByYear()
	if err != nil {
		t.Fatal(err)
	}
	if got["2024"] != 2 || got["2019"] != 1 || got[""] != 1 || len(got) != 3 {
		t.Fatalf("got %v", got)
	}
}
