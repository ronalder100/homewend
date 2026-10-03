// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/ronalder100/homewend/internal/session"
)

// libraryFile keeps the library folder chosen once, beside the profile's
// folder in the user's config directory: a setting, like the profile.
const libraryFile = "library"

func libraryPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "homewend", libraryFile), nil
}

// DefaultLibrary is the library folder chosen once, or "" when none was.
func DefaultLibrary() (string, error) {
	path, err := libraryPath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	return strings.TrimSpace(string(data)), err
}

// SetDefaultLibrary keeps dir as the library folder from now on.
func SetDefaultLibrary(dir string) error {
	path, err := libraryPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(dir+"\n"), 0o600)
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
