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
