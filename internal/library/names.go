// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"regexp"
	"runtime"
	"strings"
)

// Windows refuses some names that every other system takes: the characters
// below, control characters, a trailing space or period, and the names of
// its old devices with or without an extension (Microsoft, "Naming Files,
// Paths, and Namespaces", read 2026-10-01). Google writes a zip's entries
// with whatever the user called their albums.
var (
	windowsReserved = regexp.MustCompile(`[<>:"|?*\x00-\x1f]`)
	windowsDevices  = regexp.MustCompile(`(?i)^(CON|PRN|AUX|NUL|(COM|LPT)[1-9¹²³])(\..*)?$`)
)

// fileName makes one path component acceptable to the system this runs on.
// Elsewhere than Windows every name is returned as it came: what a user named
// a thing is theirs, and changed only where the disk leaves no choice.
func fileName(name string) string {
	if runtime.GOOS != "windows" {
		return name
	}
	return windowsFileName(name)
}

func windowsFileName(name string) string {
	name = windowsReserved.ReplaceAllString(name, "_")
	name = strings.TrimRight(name, " .")
	if name == "" {
		return "_"
	}
	if windowsDevices.MatchString(name) {
		return "_" + name
	}
	return name
}
