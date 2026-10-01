// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"fmt"
	"image/color"
	"os"
	"strings"
	"time"

	bar "charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dustin/go-humanize"
	"golang.org/x/term"

	"github.com/ronalder100/homewend/internal/progress"
)

// startLive shows a status line under the log while a command runs in a
// terminal: a spinner while Google works, a bar while a part downloads.
// Hours of silence look like a hang; this says what is being waited for.
// Nil when stdout is not a terminal, where a redrawn line is only noise.
func startLive() *tea.Program {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return nil
	}
	// No input and no signal handler: the terminal stays as it was, so
	// Ctrl-C still reaches the command and stops it the usual way.
	p := tea.NewProgram(newStatus(), tea.WithInput(nil), tea.WithoutSignalHandler())
	go p.Run()
	return p
}

// The spinner's colour, as in the demo in docs/demo.
var spinColour = lipgloss.Color("#7571F9")

// States in homewend.app's dark-theme colours: --ok for what is done, --err
// for what failed or is missing, --accent for what to type, --faint for what
// is over.
var (
	okColour     = lipgloss.Color("#9ECE6A")
	errColour    = lipgloss.Color("#F7768E")
	accentColour = lipgloss.Color("#7AA2F7")
	faintColour  = lipgloss.Color("#565F89")
)

// paint colours s when out is a terminal, and leaves it plain anywhere else.
func paint(out *os.File, c color.Color, s string) string {
	if !term.IsTerminal(int(out.Fd())) {
		return s
	}
	return lipgloss.NewStyle().Foreground(c).Render(s)
}

// The one line a person must not skim past runs from the site's dark-theme
// grey to its blue, homewend.app's --faint and --accent.
var noticeFrom, noticeTo = lipgloss.Color("#565F89"), lipgloss.Color("#7AA2F7")

// blended colours s letter by letter from noticeFrom to noticeTo.
func blended(s string) string {
	letters := []rune(s)
	colours := lipgloss.Blend1D(len(letters), noticeFrom, noticeTo)
	var b strings.Builder
	for i, r := range letters {
		b.WriteString(lipgloss.NewStyle().Foreground(colours[i]).Render(string(r)))
	}
	return b.String()
}

// idle clears the status line, so a command's last words are not followed by
// a stale bar.
type idle struct{}

type status struct {
	spin spinner.Model
	bar  bar.Model

	// Beside the spinner while Google works, and since when; no time while
	// Google prepares the export, where hours on a counter read as a hang.
	waiting string
	since   time.Time

	// The part downloading, or the photos being placed; zero when neither.
	current progress.Event

	// Bytes per second, smoothed over the reports of the current part.
	rate     float64
	lastAt   time.Time
	lastDone int64
}

func newStatus() status {
	return status{
		spin: spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(lipgloss.NewStyle().Foreground(spinColour))),
		bar:  bar.New(bar.WithDefaultBlend(), bar.WithWidth(18)),
	}
}

func (s status) Init() tea.Cmd { return s.spin.Tick }

func (s status) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		s.spin, cmd = s.spin.Update(msg)
		return s, cmd
	case idle:
		return newStatus(), nil
	case progress.Event:
		s.show(msg, time.Now())
	}
	return s, nil
}

// show takes in one event from the engine.
func (s *status) show(e progress.Event, now time.Time) {
	switch e.Stage {
	case progress.SessionReady:
		s.waiting, s.since, s.current = text["session status"], now, progress.Event{}
	case progress.Request:
		s.waiting, s.since, s.current = text["asking status"], now, progress.Event{}
	case progress.Waiting:
		s.waiting, s.since, s.current = text["waiting status"], time.Time{}, progress.Event{}
	case progress.Download:
		s.waiting, s.current = "", e
		s.rate, s.lastAt, s.lastDone = 0, now, e.Done
	case progress.Receiving:
		if dt := now.Sub(s.lastAt).Seconds(); dt > 0 {
			seen := float64(e.Done-s.lastDone) / dt
			if s.rate == 0 {
				s.rate = seen
			} else {
				s.rate = 0.7*s.rate + 0.3*seen
			}
		}
		s.current, s.lastAt, s.lastDone = e, now, e.Done
	case progress.Place:
		s.waiting, s.current = "", e
	case progress.Ready, progress.InLibrary, progress.Downloaded, progress.Short, progress.Damaged, progress.Unpack:
		s.waiting = ""
		s.current = progress.Event{}
	}
}

func (s status) View() tea.View {
	return tea.NewView(s.line(time.Now()))
}

func (s status) line(now time.Time) string {
	e := s.current
	switch e.Stage {
	case progress.Download, progress.Receiving:
		// The name is in the log above; the line has to fit a narrow terminal.
		line := fmt.Sprintf(text["download status"], e.N, e.Of,
			s.bar.ViewAs(fraction(e.Done, e.Total)), size(e.Done), size(e.Total))
		if s.rate > 0 {
			left := time.Duration(float64(e.Total-e.Done) / s.rate * float64(time.Second))
			line += fmt.Sprintf(text["download rate"], size(int64(s.rate)), left.Round(time.Second))
		}
		return line
	case progress.Place:
		return fmt.Sprintf(text["place status"], s.bar.ViewAs(fraction(int64(e.N), int64(e.Of))), e.N, e.Of)
	}
	if s.waiting != "" {
		// The spinner's frames carry their own trailing space.
		line := s.spin.View() + s.waiting
		if !s.since.IsZero() {
			line += " · " + now.Sub(s.since).Round(time.Second).String()
		}
		return line
	}
	return ""
}

func fraction(done, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(done) / float64(total)
}

func size(n int64) string { return humanize.IBytes(uint64(n)) }
