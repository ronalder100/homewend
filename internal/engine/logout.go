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
func Logout(profile string) (bool, error) {
	entries, err := os.ReadDir(profile)
	if errors.Is(err, os.ErrNotExist) || err == nil && len(entries) == 0 {
		return false, os.RemoveAll(profile)
	}
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(filepath.Join(profile, "Local State")); err != nil {
		return false, fmt.Errorf("%s does not look like a browser profile: not deleted", profile)
	}
	return true, os.RemoveAll(profile)
}
