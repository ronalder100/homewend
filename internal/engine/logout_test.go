// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLogoutDeletesTheProfile(t *testing.T) {
	profile := filepath.Join(t.TempDir(), "profile")
	os.MkdirAll(filepath.Join(profile, "Default"), 0o700)
	os.WriteFile(filepath.Join(profile, "Local State"), []byte("{}"), 0o600)
	if was, err := Logout(profile); err != nil || !was {
		t.Fatalf("got %v, %v; want true", was, err)
	}
	if _, err := os.Stat(profile); !os.IsNotExist(err) {
		t.Error("the profile is still there")
	}
}

func TestLogoutWithNoProfile(t *testing.T) {
	if was, err := Logout(filepath.Join(t.TempDir(), "none")); err != nil || was {
		t.Errorf("got %v, %v; want false, nil", was, err)
	}
}

// A folder that is not a browser profile is left alone.
func TestLogoutLeavesOtherFoldersAlone(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "photo.jpg"), []byte("x"), 0o600)
	if _, err := Logout(dir); err == nil {
		t.Error("no error for a folder that is not a profile")
	}
	if _, err := os.Stat(filepath.Join(dir, "photo.jpg")); err != nil {
		t.Error("the folder's file is gone")
	}
}

// Signing out takes the session and leaves the notes of what each export
// holds; signing out again, with only the notes there, has nothing to do.
func TestLogoutKeepsTheNotesOfTheExports(t *testing.T) {
	profile := filepath.Join(t.TempDir(), "profile")
	os.MkdirAll(filepath.Join(profile, "Default"), 0o700)
	os.WriteFile(filepath.Join(profile, "Local State"), []byte("{}"), 0o600)
	notes := []byte(`[{"year":2025,"asked":"2026-10-01T15:42:58Z","job":"8f6c3233"}]`)
	os.WriteFile(filepath.Join(profile, notesFile), notes, 0o600)

	if was, err := Logout(profile); err != nil || !was {
		t.Fatalf("got %v, %v; want true", was, err)
	}
	if kept, _ := os.ReadFile(filepath.Join(profile, notesFile)); string(kept) != string(notes) {
		t.Errorf("the notes are %q", kept)
	}
	if _, err := os.Stat(filepath.Join(profile, "Local State")); !os.IsNotExist(err) {
		t.Error("the session is still there")
	}
	if was, err := Logout(profile); err != nil || was {
		t.Errorf("a second time: got %v, %v; want false, nil", was, err)
	}
}
