// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Settings are the choices a person makes in the window. A missing file is
// every default.
type Settings struct {
	// Theme is "system", "light" or "dark"; empty is "system".
	Theme string `json:"theme,omitempty"`
	// Hidden are the accounts, by id, whose photos the window leaves out.
	Hidden []string `json:"hidden,omitempty"`
	// Library is the library folder, chosen once.
	Library string `json:"library,omitempty"`
	// Profiles names the profile of each account, by id: the folder of the
	// library its photos go in. An account not named here goes in ProfileFor's.
	Profiles map[string]string `json:"profiles,omitempty"`
}

// ProfileFor is the profile of the account id: the one chosen, or the
// address's name before the @.
func (s Settings) ProfileFor(id string) string {
	if p := s.Profiles[id]; p != "" {
		return p
	}
	_, email, _ := strings.Cut(id, "/")
	name, _, _ := strings.Cut(email, "@")
	return name
}

// ProfileNames are the profiles there are: those of the accounts, and those
// named in the settings.
func ProfileNames(accounts []AccountInfo, s Settings) []string {
	seen := map[string]bool{}
	var names []string
	add := func(n string) {
		if n != "" && !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}
	for _, a := range accounts {
		add(a.Profile)
	}
	for _, p := range s.Profiles {
		add(p)
	}
	slices.Sort(names)
	return names
}

// SetProfile puts the account id in the profile named profile.
func SetProfile(id, profile string) error {
	if !validProfile(profile) {
		return ErrBadProfile
	}
	// Only an account that is here: the settings keep no profile for one
	// that is not.
	dir, err := AccountDir(id)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dir); err != nil {
		return ErrNoSuchAccount
	}
	s, err := LoadSettings()
	if err != nil {
		return err
	}
	if s.Profiles == nil {
		s.Profiles = map[string]string{}
	}
	s.Profiles[id] = profile
	return SaveSettings(s)
}

// ErrBadProfile is a profile name that cannot be a folder.
var ErrBadProfile = errors.New("a profile is a name, without / or \\, and not . or ..")

func validProfile(p string) bool {
	return p != "" && p != "." && p != ".." && !strings.ContainsAny(p, "/\\") && !strings.HasPrefix(p, ".")
}

// Themes are the values Theme takes.
var Themes = []string{"system", "light", "dark"}

// ErrBadTheme is a theme that is not one of Themes.
var ErrBadTheme = errors.New("theme must be system, light or dark")

func settingsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "homewend", "settings.json"), nil
}

// LoadSettings reads the settings, or the defaults when none were saved.
func LoadSettings() (Settings, error) {
	path, err := settingsPath()
	if err != nil {
		return Settings{}, err
	}
	return loadSettings(path)
}

func loadSettings(path string) (Settings, error) {
	var s Settings
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s, fmt.Errorf("reading %s: %w", path, err)
	}
	return s, nil
}

// SaveSettings checks the settings and writes them.
func SaveSettings(s Settings) error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	return saveSettings(path, s)
}

func saveSettings(path string, s Settings) error {
	if s.Theme != "" && !validTheme(s.Theme) {
		return ErrBadTheme
	}
	return writeJSON(path, s)
}

func validTheme(t string) bool { return slices.Contains(Themes, t) }

// Free is how many bytes the disk of dir has free, or of the nearest folder
// above it that exists: a library not created yet has its disk too.
func Free(dir string) (int64, error) {
	for {
		if _, err := os.Stat(dir); err == nil {
			return freeBytes(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return freeBytes(dir)
		}
		dir = parent
	}
}
