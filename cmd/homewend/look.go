// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"image/color"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
)

// look draws the lines every screen is made of: one function per kind of
// line, so that the same thing looks the same everywhere. Off, it writes the
// same words with no colour and no weight, for a file or a pipe.
type look struct{ on bool }

// lookFor is the look of what goes to out.
func lookFor(out *os.File) look { return look{on: coloured(out)} }

// The signs in the first column, one per state (see palette.go).
const (
	signDone    = "✓"
	signFailed  = "×"
	signStopped = "-"
	signWaiting = "*"
	signWarning = "!"
	signAsk     = "?"
	signCursor  = "›"
)

func (l look) paint(s string, c color.Color, bold bool) string {
	if !l.on || s == "" {
		return s
	}
	st := lipgloss.NewStyle().Bold(bold)
	if c != nil {
		st = st.Foreground(c)
	}
	return st.Render(s)
}

// marked draws s in c, with the parts between ** in bold: what to read or to
// type stands out by weight, not by colour.
func (l look) marked(s string, c color.Color) string {
	var b strings.Builder
	for i, part := range strings.Split(s, "**") {
		b.WriteString(l.paint(part, c, i%2 == 1))
	}
	return b.String()
}

// header is the command as typed, the first line of every screen: the screen
// is cleared, and this is what says what runs. Bold and darker than the text,
// it reads as context.
func (l look) header(command string) string { return l.paint(command, titleColour, true) }

// step is a line of work: a sign, what is said, and details beside it.
func (l look) step(sign string, c color.Color, said, details string) string {
	return l.paint(sign, c, false) + " " + l.marked(said, nil) + l.paint(details, mutedColour, false)
}

func (l look) done(said, details string) string { return l.step(signDone, okColour, said, details) }
func (l look) failed(said, details string) string {
	return l.step(signFailed, errColour, said, details)
}
func (l look) stopped(said, details string) string {
	return l.step(signStopped, mutedColour, said, details)
}
func (l look) waiting(said, details string) string {
	return l.step(signWaiting, warnColour, said, details)
}
func (l look) warning(said string) string { return l.step(signWarning, warnColour, said, "") }

// result is the line that matters, the last: bold, with its sign.
func (l look) result(said string) string {
	return l.paint(signDone, okColour, true) + " " + l.paint(said, nil, true)
}

// summary is a last line that is not a success: bold, no sign.
func (l look) summary(said string) string { return l.paint(said, nil, true) }

// failure says what went wrong and nothing more; the fix is a hint.
func (l look) failure(said string) string { return l.paint("error:", errColour, true) + " " + said }

// cause is a detail under an error or a question.
func (l look) cause(said string) string { return "  " + l.paint(said, mutedColour, false) }

// hint says what to do next, the command to type in bold.
func (l look) hint(said string) string {
	return l.paint("hint:", accentColour, true) + " " + l.marked(said, nil)
}

// field is a line of a record: a name, a sign where there is a state, a value
// and details beside it.
func (l look) field(name, sign string, c color.Color, value, details string) string {
	if sign == "" {
		sign = " "
	}
	return "  " + l.paint(pad(name, 11), mutedColour, false) + l.paint(sign, c, false) + " " + value + l.paint(details, mutedColour, false)
}

// heading is the title of a section of a help page.
func (l look) heading(said string) string { return l.paint(said, accentColour, true) }

// pad fills s with spaces to n columns.
func pad(s string, n int) string {
	if w := lipgloss.Width(s); w < n {
		return s + strings.Repeat(" ", n-w)
	}
	return s
}
