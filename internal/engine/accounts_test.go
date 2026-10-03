// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAccountsInOrderWithTheirAddress(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "profile")
	for _, p := range []string{first, profileOf(first, "10"), profileOf(first, "2")} {
		if err := os.MkdirAll(p, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	os.WriteFile(filepath.Join(first, accountFileName), []byte("ann@example.com\n"), 0o600)
	os.WriteFile(filepath.Join(profileOf(first, "2"), accountFileName), []byte("bo@example.com\n"), 0o600)
	os.MkdirAll(filepath.Join(dir, accountsDir, "notes"), 0o700)

	list, err := accountsIn(first)
	if err != nil {
		t.Fatal(err)
	}
	want := []AccountInfo{{ID: "1", Email: "ann@example.com"}, {ID: "2", Email: "bo@example.com"}, {ID: "10"}}
	if len(list) != len(want) {
		t.Fatalf("got %+v", list)
	}
	for i := range want {
		if list[i].ID != want[i].ID || list[i].Email != want[i].Email {
			t.Errorf("%d: got %+v, want %+v", i, list[i], want[i])
		}
	}
}

func TestNoProfileNoAccounts(t *testing.T) {
	list, err := accountsIn(filepath.Join(t.TempDir(), "profile"))
	if err != nil || len(list) != 0 {
		t.Fatalf("got %+v, %v", list, err)
	}
}

func TestLibraryOf(t *testing.T) {
	if got := LibraryOf("/lib", "ann.lee@example.com"); got != filepath.Join("/lib", "ann.lee") {
		t.Fatalf("got %s", got)
	}
}
