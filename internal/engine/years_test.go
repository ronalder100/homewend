// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A count a week old is said stale, to be made again; a fresh one is not.
func TestAnOldCountIsStale(t *testing.T) {
	config(t)
	const account = "google/ron@example.com"
	dir, err := AccountDir(account)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		age   time.Duration
		stale bool
	}{{time.Hour, false}, {8 * 24 * time.Hour, true}} {
		if err := writeJSON(filepath.Join(dir, yearsFile), YearCounts{Counted: time.Now().Add(-c.age)}); err != nil {
			t.Fatal(err)
		}
		got, err := LoadYearCounts(account)
		if err != nil || got.Stale != c.stale {
			t.Errorf("counted %v ago: stale %v, %v; want %v", c.age, got.Stale, err, c.stale)
		}
	}
}
