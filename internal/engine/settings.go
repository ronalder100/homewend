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
)

// Settings are the choices a person makes in the window. A missing file is
// every default.
type Settings struct {
	// Theme is "system", "light" or "dark"; empty is "system".
	Theme string `json:"theme,omitempty"`
	// Hidden are the accounts, by id, whose photos the window leaves out.
	Hidden []string `json:"hidden,omitempty"`
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
