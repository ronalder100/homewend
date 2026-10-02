// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Logout signs out by deleting the browser profile: the Google session lives
// in it and nowhere else. It reports whether there was one to delete.
//
// Only a directory a Chromium-family browser made is deleted, known by the
// "Local State" file it writes at its top: a --profile pointed somewhere else
// by mistake must not cost the user a folder.
//
// What Homewend noted of the exports it asked for stays behind, alone in the
// folder. Google does not say what an export holds, so those notes are the
// only record of which year each one is, and the exports outlive a session:
// signing out and in again must not turn them all into unknowns.
func Logout(profile string) (bool, error) {
	entries, err := os.ReadDir(profile)
	if errors.Is(err, os.ErrNotExist) || err == nil && len(entries) == 0 {
		return false, os.RemoveAll(profile)
	}
	if err != nil {
		return false, err
	}
	if len(entries) == 1 && entries[0].Name() == notesFile {
		return false, nil
	}
	if _, err := os.Stat(filepath.Join(profile, "Local State")); err != nil {
		return false, fmt.Errorf("%s does not look like a browser profile: not deleted", profile)
	}
	notes, err := os.ReadFile(filepath.Join(profile, notesFile))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	if err := os.RemoveAll(profile); err != nil || notes == nil {
		return true, err
	}
	if err := os.MkdirAll(profile, 0o700); err != nil {
		return true, err
	}
	return true, os.WriteFile(filepath.Join(profile, notesFile), notes, 0o600)
}
