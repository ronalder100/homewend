// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package session

import (
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// On Windows a browser is not on PATH: it is in the registry, where its
// installer leaves an open command under Software\Clients\StartMenuInternet,
// per user or per machine. Chrome and Edge were measured there on Windows
// Server 2025 (docs/architecture.md, "Windows"). The default browser is the
// ProgId Windows keeps for https, whose own open command names the
// executable.

// matches reports whether def — the default browser's executable — is c.
func matches(c chromium, def string) bool {
	return strings.EqualFold(filepath.Base(def), c.exe)
}

// installed prefers the default browser's own executable: Chrome and Chromium
// are both chrome.exe, and a Chromium default must not launch a Chrome that
// happens to be registered first.
func installed(c chromium) string {
	for _, path := range append([]string{defaultBrowser()}, registeredBrowsers()...) {
		if path != "" && strings.EqualFold(filepath.Base(path), c.exe) {
			return path
		}
	}
	return ""
}

func defaultBrowser() string {
	id := regValue(registry.CURRENT_USER,
		`Software\Microsoft\Windows\Shell\Associations\UrlAssociations\https\UserChoice`, "ProgId")
	if id == "" {
		return ""
	}
	return commandPath(regValue(registry.CLASSES_ROOT, id+`\shell\open\command`, ""))
}

// registeredBrowsers lists the executables of the browsers registered with
// Windows, the user's own first.
func registeredBrowsers() []string {
	var paths []string
	for _, root := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
		const clients = `Software\Clients\StartMenuInternet`
		key, err := registry.OpenKey(root, clients, registry.ENUMERATE_SUB_KEYS)
		if err != nil {
			continue
		}
		names, _ := key.ReadSubKeyNames(-1)
		key.Close()
		for _, name := range names {
			if path := commandPath(regValue(root, clients+`\`+name+`\shell\open\command`, "")); path != "" {
				paths = append(paths, path)
			}
		}
	}
	return paths
}

func regValue(root registry.Key, path, name string) string {
	key, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer key.Close()
	value, _, err := key.GetStringValue(name)
	if err != nil {
		return ""
	}
	return value
}
