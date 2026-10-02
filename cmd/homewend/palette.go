// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"image/color"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"golang.org/x/term"
)

// Every colour the CLI uses, and the only place one is written down. The
// names and values are homewend.app's dark-theme tokens, so the terminal and
// the site are one system: a neutral scale from the text a person reads to
// what is over, and one colour per state.
var (
	mutedColour = lipgloss.Color("#9AA5CE") // --muted: what to do next, said quietly
	faintColour = lipgloss.Color("#565F89") // --faint: what is over

	okColour     = lipgloss.Color("#9ECE6A") // --ok: done
	errColour    = lipgloss.Color("#F7768E") // --err: failed, or missing
	accentColour = lipgloss.Color("#7AA2F7") // --accent: what to type, and work going on
)

// paint colours s when out is a terminal, and leaves it plain anywhere else.
func paint(out *os.File, c color.Color, s string) string {
	if !term.IsTerminal(int(out.Fd())) {
		return s
	}
	return lipgloss.NewStyle().Foreground(c).Render(s)
}

// blended colours s letter by letter from faint to accent: the one line a
// person must not skim past.
func blended(s string) string {
	letters := []rune(s)
	colours := lipgloss.Blend1D(len(letters), faintColour, accentColour)
	var b strings.Builder
	for i, r := range letters {
		b.WriteString(lipgloss.NewStyle().Foreground(colours[i]).Render(string(r)))
	}
	return b.String()
}
