// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package session

import "strings"

// commandPath is the executable of a Windows open command: the quoted first
// word of `"C:\...\chrome.exe" --single-argument %1`, or, where the installer
// did not quote it, everything up to the .exe.
func commandPath(command string) string {
	command = strings.TrimSpace(command)
	if strings.HasPrefix(command, `"`) {
		if end := strings.Index(command[1:], `"`); end >= 0 {
			return command[1 : end+1]
		}
		return ""
	}
	if at := strings.Index(strings.ToLower(command), ".exe"); at >= 0 {
		return command[:at+len(".exe")]
	}
	return ""
}
