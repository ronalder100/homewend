// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"image/color"
	"os"

	"charm.land/lipgloss/v2"
	"golang.org/x/term"
)

// Every colour the CLI uses, and the only place one is written down: Tokyo
// Night, its night variant (tokyonight.nvim, lua/tokyonight/colors). Weight
// makes the hierarchy and colour marks state, on a sign or a label, never on
// running text.
var (
	titleColour = lipgloss.Color("#737AA2") // dark5: the command as typed, bold, darker than the text
	mutedColour = lipgloss.Color("#A9B1D6") // fg_dark: details beside what is said
	faintColour = lipgloss.Color("#565F89") // comment: chrome, and what is over
	lineColour  = lipgloss.Color("#292E42") // bg_highlight: what is left of a bar

	okColour     = lipgloss.Color("#9ECE6A") // green: done
	errColour    = lipgloss.Color("#F7768E") // red: failed
	warnColour   = lipgloss.Color("#E0AF68") // yellow: waiting, and warnings
	accentColour = lipgloss.Color("#7AA2F7") // blue: work going on, and your move
)

// paint colours s when out is a terminal, and leaves it plain anywhere else.
func paint(out *os.File, c color.Color, s string) string {
	if !coloured(out) {
		return s
	}
	return lipgloss.NewStyle().Foreground(c).Render(s)
}

// coloured is whether out is a terminal that wants colour: not when NO_COLOR
// is set (no-color.org), nor when the output is a file or a pipe.
func coloured(out *os.File) bool {
	return os.Getenv("NO_COLOR") == "" && term.IsTerminal(int(out.Fd()))
}
