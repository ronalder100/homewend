// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/ronalder100/homewend/internal/library"
	"github.com/ronalder100/homewend/internal/session"
)

// libraryFile is where an older homewend kept the library folder, before it
// moved into the settings.
const libraryFile = "library"

// DefaultLibrary is the library folder chosen once, or "" when none was.
func DefaultLibrary() (string, error) {
	s, err := LoadSettings()
	if err != nil || s.Library != "" {
		return s.Library, err
	}
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(dir, libraryFile))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	old := strings.TrimSpace(string(data))
	if err := SetDefaultLibrary(old); err != nil {
		return "", err
	}
	return old, os.Remove(filepath.Join(dir, libraryFile))
}

// SetDefaultLibrary keeps dir as the library folder from now on.
func SetDefaultLibrary(dir string) error {
	s, err := LoadSettings()
	if err != nil {
		return err
	}
	s.Library = dir
	return SaveSettings(s)
}

// State is where things stand, as the status command shows it.
type State struct {
	SignedIn bool   `json:"signed_in"`
	Account  string `json:"account,omitempty"`
	// Export is the one Homewend asked for last, nil when there is none or
	// the profile is not signed in.
	Export  *Takeout `json:"export,omitempty"`
	Library string   `json:"library,omitempty"`
}

// Now reads the state without changing it: the session is checked, not
// renewed, and no window opens.
func Now(sess *session.Session) (State, error) {
	var st State
	var err error
	if st.Library, err = DefaultLibrary(); err != nil {
		return st, err
	}
	if st.SignedIn, err = signedIn(sess); err != nil || !st.SignedIn {
		return st, err
	}
	st.Account = Account(sess)
	list, err := Takeouts(sess)
	if err != nil {
		return st, err
	}
	for _, t := range list {
		if t.Known {
			st.Export = &t
			break
		}
	}
	return st, nil
}

// settle moves a library filed before profiles into them, once: the photos
// of each source to its account's profile, those of no source to the first
// account's.
func settle(root string) error {
	if root == "" || !library.Filed(root) {
		return nil
	}
	accounts, err := Accounts()
	if err != nil {
		return err
	}
	s, err := LoadSettings()
	if err != nil {
		return err
	}
	fallback := ""
	for _, a := range accounts {
		if a.Profile != "" {
			fallback = a.Profile
			break
		}
	}
	if fallback == "" {
		return nil // nobody to file them under yet
	}
	catalog, err := library.OpenCatalog(catalogPath(root))
	if err != nil {
		return err
	}
	defer catalog.Close()
	return library.Relayout(root, catalog, s.ProfileFor, fallback, nil)
}
