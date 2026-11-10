// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsDefaultWhenMissing(t *testing.T) {
	s, err := loadSettings(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil || s.Theme != "" || s.Hidden != nil {
		t.Fatalf("got %+v, %v; want the defaults", s, err)
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "homewend", "settings.json")
	if err := saveSettings(path, Settings{Theme: "dark"}); err != nil {
		t.Fatal(err)
	}
	s, err := loadSettings(path)
	if err != nil || s.Theme != "dark" {
		t.Fatalf("got %+v, %v; want dark", s, err)
	}
}

func TestSettingsRefuseUnknownTheme(t *testing.T) {
	err := saveSettings(filepath.Join(t.TempDir(), "settings.json"), Settings{Theme: "blue"})
	if !errors.Is(err, ErrBadTheme) {
		t.Fatalf("got %v, want ErrBadTheme", err)
	}
}

func TestSetProfileOnlyForAnAccountThatIsHere(t *testing.T) {
	cfg := config(t)
	if err := SetProfile("google/ann@example.com", "ann"); !errors.Is(err, ErrNoSuchAccount) {
		t.Fatalf("no account yet: got %v, want ErrNoSuchAccount", err)
	}
	if err := os.MkdirAll(filepath.Join(cfg, "accounts", "google", "ann@example.com"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := SetProfile("google/ann@example.com", "ann"); err != nil {
		t.Fatal(err)
	}
	if s, _ := LoadSettings(); s.Profiles["google/ann@example.com"] != "ann" {
		t.Errorf("profiles %v", s.Profiles)
	}
}

// A profile a card was imported into, with no account, is a profile the
// next import finds by itself; the library's work folder and its files are
// not.
func TestTheLibrarysFoldersAreProfiles(t *testing.T) {
	lib := t.TempDir()
	for _, dir := range []string{"ron", ".homewend"} {
		if err := os.Mkdir(filepath.Join(lib, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(lib, "notes.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	got := ProfileNames([]AccountInfo{{Profile: "anna"}}, Settings{Library: lib})
	if len(got) != 2 || got[0] != "anna" || got[1] != "ron" {
		t.Errorf("%q, want anna and ron", got)
	}
}
