// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"errors"
	"os"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// A question in the flow of a command: ? and the question in bold, a detail
// under it, the choices, the keys at the bottom. The choice under the cursor
// is marked › and bold, the others are secondary text: no band, which belongs
// to full-screen lists. huh, Charm's forms, lays its key help out one key per
// binding, which cannot read "↑/↓ move"; these two questions are small enough
// to draw here, on bubbletea.

// choose asks question and returns the index of the choice taken. Ctrl-C
// stops the command, as anywhere else.
func choose(ctx context.Context, question, detail string, choices []string) (int, error) {
	m, err := ask(ctx, &chooser{question: question, detail: detail, choices: choices, out: lookFor(os.Stdout)})
	if err != nil {
		return 0, err
	}
	return m.(*chooser).at, nil
}

// field asks question with value proposed, and returns what was given.
func field(ctx context.Context, question, detail, value string) (string, error) {
	in := textinput.New()
	in.Prompt = ""
	in.SetValue(value)
	in.CursorEnd()
	st := textinput.DefaultDarkStyles()
	st.Focused.Text = lipgloss.NewStyle().Bold(true)
	in.SetStyles(st)
	in.Focus()
	m, err := ask(ctx, &input{question: question, detail: detail, in: in, out: lookFor(os.Stdout)})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(m.(*input).in.Value()), nil
}

// asked is a question's model once answered or abandoned.
type asked interface {
	tea.Model
	stopped() bool
}

func ask(ctx context.Context, m asked) (tea.Model, error) {
	final, err := tea.NewProgram(m, tea.WithContext(ctx), tea.WithoutSignalHandler()).Run()
	if errors.Is(err, tea.ErrProgramKilled) || err == nil && final.(asked).stopped() {
		return nil, context.Canceled
	}
	return final, err
}

// keys draws the key help: the key dim, what it does faint.
func (l look) keys(pairs ...string) string {
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		parts = append(parts, l.paint(pairs[i], mutedColour, false)+" "+l.paint(pairs[i+1], faintColour, false))
	}
	return "  " + strings.Join(parts, l.paint(" · ", faintColour, false))
}

// question draws ? and the question, and the detail under it.
func (l look) question(said, detail string) string {
	s := l.paint(signAsk, accentColour, true) + " " + l.paint(unmarked(said), nil, true)
	if detail != "" {
		s += "\n" + l.cause(detail)
	}
	return s
}

type chooser struct {
	question, detail string
	choices          []string
	at               int
	done, quit       bool
	out              look
}

func (c *chooser) Init() tea.Cmd { return nil }

func (c *chooser) stopped() bool { return c.quit }

func (c *chooser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "up", "k":
			c.at = max(0, c.at-1)
		case "down", "j":
			c.at = min(len(c.choices)-1, c.at+1)
		case "enter":
			c.done = true
			return c, tea.Quit
		case "ctrl+c":
			c.quit = true
			return c, tea.Quit
		}
	}
	return c, nil
}

func (c *chooser) View() tea.View {
	if c.done || c.quit {
		return tea.NewView("")
	}
	l := c.out
	lines := []string{l.question(c.question, c.detail), ""}
	for i, choice := range c.choices {
		if i == c.at {
			lines = append(lines, "  "+l.paint(signCursor, accentColour, true)+" "+l.paint(choice, nil, true))
		} else {
			lines = append(lines, "    "+l.paint(choice, mutedColour, false))
		}
	}
	lines = append(lines, "", l.keys("↑/↓", text["key move"], "enter", text["key select"], "ctrl+c", text["key quit"]))
	return tea.NewView(strings.Join(lines, "\n"))
}

type input struct {
	question, detail string
	in               textinput.Model
	done, quit       bool
	out              look
}

func (i *input) Init() tea.Cmd { return textinput.Blink }

func (i *input) stopped() bool { return i.quit }

func (i *input) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "enter":
			i.done = true
			return i, tea.Quit
		case "ctrl+c":
			i.quit = true
			return i, tea.Quit
		}
	}
	var cmd tea.Cmd
	i.in, cmd = i.in.Update(msg)
	return i, cmd
}

func (i *input) View() tea.View {
	if i.done || i.quit {
		return tea.NewView("")
	}
	l := i.out
	lines := []string{l.question(i.question, i.detail), "",
		"  " + l.paint(signCursor, accentColour, true) + " " + i.in.View(), "",
		l.keys("enter", text["key accept"], "ctrl+c", text["key quit"])}
	return tea.NewView(strings.Join(lines, "\n"))
}
