// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAccountsAreTheProfilesInTheConfigDir(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"profile", "profile-sam", "library-not-a-profile", "profiles"} {
		os.MkdirAll(filepath.Join(dir, name), 0o700)
	}
	os.WriteFile(filepath.Join(dir, "profile-sam", accountFileName), []byte("sam@example.com\n"), 0o600)
	os.WriteFile(filepath.Join(dir, "library"), []byte("/photos\n"), 0o600)

	list, err := accountsIn(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != "profile" || list[0].Email != "" ||
		list[1].ID != "profile-sam" || list[1].Email != "sam@example.com" {
		t.Fatalf("got %+v", list)
	}
}

func TestNoConfigDirNoAccounts(t *testing.T) {
	list, err := accountsIn(filepath.Join(t.TempDir(), "missing"))
	if err != nil || len(list) != 0 {
		t.Fatalf("got %+v, %v", list, err)
	}
}

func TestAccountSessionRefusesOtherFolders(t *testing.T) {
	for _, id := range []string{"", "library", "../profile", "profile/../x"} {
		if _, err := AccountSession(id); err != ErrNoSuchAccount {
			t.Errorf("%q: got %v", id, err)
		}
	}
}
