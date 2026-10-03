// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"errors"
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
