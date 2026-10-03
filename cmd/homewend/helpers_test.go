// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

var timeZero = time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)

// keyPress is the key named k, as bubbletea sends it.
func keyPress(k string) tea.KeyPressMsg {
	switch k {
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	}
	return tea.KeyPressMsg{Code: rune(k[0]), Text: k}
}
