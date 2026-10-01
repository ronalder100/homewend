// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build !windows

package session

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"howett.net/plist"
)

// matches reports whether def — a .desktop id on Linux, a bundle id on macOS —
// names c.
func matches(c chromium, def string) bool {
	if runtime.GOOS == "darwin" {
		return strings.EqualFold(def, c.bundle)
	}
	for _, name := range c.commands {
		if def == name+".desktop" {
			return true
		}
	}
	return false
}

func installed(c chromium) string {
	if runtime.GOOS == "darwin" {
		if _, err := os.Stat(c.app); err == nil {
			return c.app
		}
		return ""
	}
	for _, name := range c.commands {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return ""
}

// defaultBrowser names the system's default browser, or returns "" when it
// cannot tell: not knowing only means falling back to what is installed.
func defaultBrowser() string {
	if runtime.GOOS == "darwin" {
		return macDefaultBrowser()
	}
	// The freedesktop way, and what xdg-open itself consults.
	out, err := exec.Command("xdg-settings", "get", "default-web-browser").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// macDefaultBrowser reads the https handler from LaunchServices' preferences,
// the file behind System Settings' "Default web browser". Never tested on a
// Mac yet.
func macDefaultBrowser() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(home, "Library", "Preferences",
		"com.apple.LaunchServices", "com.apple.launchservices.secure.plist"))
	if err != nil {
		return ""
	}
	var prefs struct {
		LSHandlers []struct {
			LSHandlerURLScheme string
			LSHandlerRoleAll   string
		}
	}
	if _, err := plist.Unmarshal(data, &prefs); err != nil {
		return ""
	}
	for _, h := range prefs.LSHandlers {
		if h.LSHandlerURLScheme == "https" {
			return h.LSHandlerRoleAll
		}
	}
	return ""
}
