// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"fmt"
	"os"
	"time"

	bar "charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
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

// idle clears the status line, so a command's last words are not followed by
// a stale bar.
type idle struct{}

type status struct {
	spin spinner.Model
	bar  bar.Model

	// Beside the spinner while Google works, and since when.
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
		spin: spinner.New(spinner.WithSpinner(spinner.Dot)),
		bar:  bar.New(bar.WithDefaultBlend(), bar.WithWidth(30)),
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
	case progress.Request:
		s.waiting, s.since, s.current = text["asking status"], now, progress.Event{}
	case progress.Waiting:
		s.waiting, s.since, s.current = text["waiting status"], now, progress.Event{}
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
	case progress.Downloaded, progress.Short, progress.Unpack:
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
		line := fmt.Sprintf(text["download status"], e.N, e.Of, e.Name,
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
		return fmt.Sprintf("%s %s · %s", s.spin.View(), s.waiting, now.Sub(s.since).Round(time.Second))
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
