// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"fmt"
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
// terminal: what the user is asked to do, a pulsing dot while Google works, a
// spinner and a bar while a part downloads.
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

// idle clears the status line, so a command's last words are not followed by
// a stale bar.
type idle struct{}

// centre puts the status in the middle of the window: for a command that has
// one thing to say at a time and no log above it, like login.
type centre struct{}

// last is a command's last word, when the status is in the middle of the
// window: it takes the status's place and stays there when the program ends.
type last string

type status struct {
	// In the middle of a window this wide and tall, when centred; the last
	// word, once said.
	centred       bool
	width, height int
	last          string

	// Two ways of saying "not stuck": a dot that pulses while there is only
	// waiting to do, a spinner while files are coming down.
	dot  spinner.Model
	spin spinner.Model
	bar  bar.Model

	// What the user is asked to do in the browser. It is here, not in the
	// log, so that it goes once they have done it.
	asks string

	// Beside the dot while Google works, and since when; no time while
	// Google prepares the export, where hours on a counter read as a hang,
	// nor while the account is set up, which is ours to wait for, not theirs
	// to count.
	waiting string
	since   time.Time

	// Sign-in stands apart from the log above it, by an empty line.
	apart bool

	// The part downloading, or the photos being placed; zero when neither.
	current progress.Event

	// Bytes per second, smoothed over the reports of the current part.
	rate     float64
	lastAt   time.Time
	lastDone int64
}

func newStatus() status {
	return status{
		dot:  spinner.New(spinner.WithSpinner(pulse())),
		spin: spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(lipgloss.NewStyle().Foreground(accentColour))),
		bar:  bar.New(bar.WithDefaultBlend(), bar.WithWidth(18)),
	}
}

// pulse is one dot breathing from all but off to accent and back, slowly: a
// breath every two and a half seconds. Each frame carries its colour and its
// trailing space.
func pulse() spinner.Spinner {
	shades := lipgloss.Blend1D(30, lineColour, accentColour, lineColour)
	frames := make([]string, len(shades))
	for i, shade := range shades {
		frames[i] = lipgloss.NewStyle().Foreground(shade).Render("●") + " "
	}
	return spinner.Spinner{Frames: frames, FPS: time.Second / 12}
}

func (s status) Init() tea.Cmd { return tea.Batch(s.dot.Tick, s.spin.Tick) }

func (s status) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		// Each takes its own ticks and lets the other's pass.
		var dot, spin tea.Cmd
		s.dot, dot = s.dot.Update(msg)
		s.spin, spin = s.spin.Update(msg)
		return s, tea.Batch(dot, spin)
	case idle:
		return newStatus(), nil
	case centre:
		s.centred = true
	case tea.WindowSizeMsg:
		s.width, s.height = msg.Width, msg.Height
	case last:
		s.last = string(msg)
	case progress.Event:
		s.show(msg, time.Now())
	}
	return s, nil
}

// show takes in one event from the engine.
func (s *status) show(e progress.Event, now time.Time) {
	if e.Stage != progress.Receiving {
		s.asks, s.apart = "", false
	}
	switch e.Stage {
	case progress.Checking:
		s.waiting, s.since, s.current, s.apart = text["checking status"], time.Time{}, progress.Event{}, true
	case progress.SignIn:
		s.asks, s.apart = text["sign in"], true
		s.waiting, s.current = "", progress.Event{}
	case progress.SessionReady:
		s.waiting, s.since, s.current, s.apart = text["session status"], time.Time{}, progress.Event{}, true
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
	line := s.line(time.Now())
	if s.last != "" {
		line = s.last
	}
	if !s.centred || s.width == 0 || line == "" {
		return tea.NewView(line)
	}
	// A line longer than the window breaks between words, and each piece is
	// centred. One row short of the window, so that nothing scrolls.
	block := lipgloss.NewStyle().Width(min(s.width-4, 64)).Align(lipgloss.Center).Render(strings.TrimPrefix(line, "\n"))
	return tea.NewView(lipgloss.Place(s.width, s.height-1, lipgloss.Center, lipgloss.Center, block))
}

func (s status) line(now time.Time) string {
	e := s.current
	switch e.Stage {
	case progress.Download, progress.Receiving:
		// The name is in the log above; the line has to fit a narrow terminal.
		// The spinner's frames carry their own trailing space.
		line := s.spin.View() + fmt.Sprintf(text["download status"], e.N, e.Of,
			s.bar.ViewAs(fraction(e.Done, e.Total)), size(e.Done), size(e.Total))
		if s.rate > 0 {
			left := time.Duration(float64(e.Total-e.Done) / s.rate * float64(time.Second))
			line += fmt.Sprintf(text["download rate"], size(int64(s.rate)), left.Round(time.Second))
		}
		return line
	case progress.Place:
		return fmt.Sprintf(text["place status"], s.bar.ViewAs(fraction(int64(e.N), int64(e.Of))), e.N, e.Of)
	}
	// Both in the muted grey: they say what is going on, not what was done.
	muted := lipgloss.NewStyle().Foreground(mutedColour)
	var line string
	switch {
	case s.asks != "":
		line = muted.Render(s.asks)
	case s.waiting != "":
		waiting := s.waiting
		if !s.since.IsZero() {
			waiting += " · " + now.Sub(s.since).Round(time.Second).String()
		}
		line = s.dot.View() + muted.Render(waiting)
	}
	if line != "" && s.apart {
		line = "\n" + line
	}
	return line
}

func fraction(done, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(done) / float64(total)
}

func size(n int64) string { return humanize.IBytes(uint64(n)) }
