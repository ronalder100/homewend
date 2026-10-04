// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// config makes a config directory of its own for the test, and returns
// homewend's folder in it.
func config(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	t.Setenv("HOME", t.TempDir())
	dir, err := configDir()
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestAccountsAreByServiceAndAddress(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "accounts")
	for _, name := range []string{"google/sam@example.com", "google/new-1", "google/not-an-account", "notes"} {
		os.MkdirAll(filepath.Join(dir, name), 0o700)
	}
	list, err := accountsIn(dir, Settings{Profiles: map[string]string{"google/sam@example.com": "family"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != "google/new-1" || list[0].Email != "" ||
		list[1].ID != "google/sam@example.com" || list[1].Email != "sam@example.com" ||
		list[1].Service != Google || list[1].Profile != "family" {
		t.Fatalf("got %+v", list)
	}
}

func TestNoAccountsFolderNoAccounts(t *testing.T) {
	list, err := accountsIn(filepath.Join(t.TempDir(), "missing"), Settings{})
	if err != nil || len(list) != 0 {
		t.Fatalf("got %+v, %v", list, err)
	}
}

func TestAccountDirRefusesOtherFolders(t *testing.T) {
	config(t)
	for _, id := range []string{"", "library", "../profile", "google/../x", "google/a@b/../c", "profile"} {
		if _, err := AccountDir(id); err != ErrNoSuchAccount {
			t.Errorf("%q: got %v", id, err)
		}
	}
	if _, err := AccountDir("google/sam@example.com"); err != nil {
		t.Errorf("an address: %v", err)
	}
}

// The sessions an older homewend kept are moved under their address; one
// whose address is not known stays where it was.
func TestOldAccountsAreMoved(t *testing.T) {
	dir := config(t)
	for name, email := range map[string]string{"profile": "ron@example.com", "profile-sam": "sam@example.com", "accounts/3": "kim@example.com", "profile-unknown": ""} {
		os.MkdirAll(filepath.Join(dir, name), 0o700)
		if email != "" {
			os.WriteFile(filepath.Join(dir, name, accountFileName), []byte(email+"\n"), 0o600)
		}
	}
	list, err := Accounts()
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, a := range list {
		ids = append(ids, a.ID+"→"+a.Profile)
	}
	want := "[google/kim@example.com→kim google/ron@example.com→ron google/sam@example.com→sam]"
	if got := fmt.Sprint(ids); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
	if _, err := os.Stat(filepath.Join(dir, "profile-unknown")); err != nil {
		t.Error("a folder of no known address was moved")
	}
}

func TestNextAccountIsAFolderWithNoAddressYet(t *testing.T) {
	dir := config(t)
	if id, err := NextAccount(); err != nil || id != "google/new-1" {
		t.Fatalf("got %q %v", id, err)
	}
	os.MkdirAll(filepath.Join(dir, "accounts", "google", "new-1"), 0o700)
	if id, _ := NextAccount(); id != "google/new-2" {
		t.Errorf("got %q", id)
	}
}

// After the sign-in the folder takes the address's name.
func TestAdoptNamesTheFolder(t *testing.T) {
	dir := config(t)
	pending := filepath.Join(dir, "accounts", "google", "new-1")
	os.MkdirAll(pending, 0o700)
	os.WriteFile(filepath.Join(pending, accountFileName), []byte("ron@example.com\n"), 0o600)
	a, err := Adopt("google/new-1")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "google/ron@example.com" || a.Profile != "ron" {
		t.Errorf("got %+v", a)
	}
	if _, err := os.Stat(filepath.Join(dir, "accounts", "google", "ron@example.com", accountFileName)); err != nil {
		t.Error("the folder was not renamed")
	}
}

func TestProfiles(t *testing.T) {
	config(t)
	if err := SetProfile("google/sam@example.com", "../x"); err != ErrBadProfile {
		t.Errorf("a path as a profile: %v", err)
	}
	if err := SetProfile("google/sam@example.com", "family"); err != nil {
		t.Fatal(err)
	}
	s, _ := LoadSettings()
	if s.ProfileFor("google/sam@example.com") != "family" || s.ProfileFor("google/ron@example.com") != "ron" {
		t.Errorf("got %+v", s.Profiles)
	}
}

// The library folder an older homewend kept in a file of its own moves into
// the settings.
func TestTheLibraryFileMovesIntoTheSettings(t *testing.T) {
	dir := config(t)
	os.MkdirAll(dir, 0o700)
	os.WriteFile(filepath.Join(dir, libraryFile), []byte("/photos\n"), 0o600)
	if got, err := DefaultLibrary(); err != nil || got != "/photos" {
		t.Fatalf("got %q %v", got, err)
	}
	if s, _ := LoadSettings(); s.Library != "/photos" {
		t.Errorf("settings %+v", s)
	}
	if _, err := os.Stat(filepath.Join(dir, libraryFile)); !os.IsNotExist(err) {
		t.Error("the old file is still there")
	}
}
