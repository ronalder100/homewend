// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ronalder100/homewend/internal/progress"
)

// Asking takes a minute or two, and counts them; the wait for Google takes
// hours, and does not.
func TestOnlyTheRequestCountsItsTime(t *testing.T) {
	s := newStatus()
	start := time.Now()
	s.show(progress.Event{Stage: progress.Request}, start)
	if line := s.line(start.Add(63 * time.Second)); !strings.Contains(line, "1m3s") {
		t.Errorf("asking: status line %q, want the time", line)
	}
	s.show(progress.Event{Stage: progress.Waiting, Name: "job"}, start)
	if line := s.line(start.Add(time.Hour)); !strings.Contains(line, text["waiting status"]) || strings.Contains(line, "·") {
		t.Errorf("waiting: status line %q, want no time", line)
	}
}

// What to do in the browser is asked on the status line, apart from the log,
// and goes once it is done; setting the account up then shows as work going
// on, with no time to count and no word of how it is done.
func TestSignInIsAskedThenGoes(t *testing.T) {
	s := newStatus()
	start := time.Now()
	s.show(progress.Event{Stage: progress.SignIn}, start)
	if line := s.line(start); !strings.HasPrefix(line, "\n") || !strings.Contains(line, text["sign in"]) {
		t.Errorf("status line %q, want an empty line and what to do", line)
	}
	s.show(progress.Event{Stage: progress.SessionReady}, start)
	line := s.line(start.Add(12 * time.Second))
	if !strings.HasPrefix(line, "\n") || !strings.Contains(line, "●") || !strings.Contains(line, text["session status"]) {
		t.Errorf("status line %q, want the dot and the wait", line)
	}
	if strings.Contains(line, text["sign in"]) || strings.Contains(line, "12s") || strings.Contains(line, "Takeout") {
		t.Errorf("status line %q, want nothing of the sign-in, the time or Takeout", line)
	}
}

// A command that begins by asking Google about the session says so at once:
// a second of empty screen reads as a program that is stuck.
func TestCheckingTheSessionShowsAtOnce(t *testing.T) {
	s := newStatus()
	s.show(progress.Event{Stage: progress.Checking}, time.Now())
	if line := s.line(time.Now()); !strings.Contains(line, "●") || !strings.Contains(line, text["checking status"]) {
		t.Errorf("status line %q, want the dot and the check", line)
	}
}

// What to do in the browser is in the middle of the window while it is
// asked, and only then: the wait that follows is a line at the top again.
func TestTheBrowserPromptIsInTheMiddle(t *testing.T) {
	var m tea.Model = newStatus()
	for _, msg := range []tea.Msg{centre{}, tea.WindowSizeMsg{Width: 80, Height: 24}, progress.Event{Stage: progress.Checking}} {
		m, _ = m.Update(msg)
	}
	if rows := strings.Split(m.(status).View().Content, "\n"); len(rows) > 2 {
		t.Errorf("checking the session takes %d rows, want a line", len(rows))
	}
	m, _ = m.Update(progress.Event{Stage: progress.SignIn})
	rows := strings.Split(ansi.Strip(m.(status).View().Content), "\n")
	at := -1
	for i, row := range rows {
		if strings.Contains(row, text["sign in"]) {
			at = i
		}
	}
	if len(rows) != 23 || at < 10 || at > 12 {
		t.Fatalf("%d rows, the words on row %d", len(rows), at)
	}
	before := len(rows[at]) - len(strings.TrimLeft(rows[at], " "))
	if want := (80 - len(text["sign in"])) / 2; before < want-1 || before > want+1 {
		t.Errorf("%d spaces before the words, want about %d", before, want)
	}
	m, _ = m.Update(progress.Event{Stage: progress.SessionReady})
	if rows := strings.Split(m.(status).View().Content, "\n"); len(rows) > 2 {
		t.Errorf("setting up takes %d rows, want a line", len(rows))
	}
}

// Once the export is ready the wait is over, and the spinner with it.
func TestReadyEndsTheWait(t *testing.T) {
	s := newStatus()
	start := time.Now()
	s.show(progress.Event{Stage: progress.Waiting}, start)
	s.show(progress.Event{Stage: progress.Ready, Of: 4, Total: 1 << 30}, start.Add(time.Hour))
	if line := s.line(start.Add(time.Hour)); line != "" {
		t.Errorf("status line %q after ready, want none", line)
	}
}

func TestADownloadShowsItsSpeedAndTimeLeft(t *testing.T) {
	s := newStatus()
	start := time.Now()
	const mib = 1 << 20
	s.show(progress.Event{Stage: progress.Download, N: 1, Of: 2, Name: "a.zip", Total: 100 * mib}, start)
	s.show(progress.Event{Stage: progress.Receiving, N: 1, Of: 2, Name: "a.zip", Done: 10 * mib, Total: 100 * mib}, start.Add(time.Second))
	line := s.line(start.Add(time.Second))
	for _, want := range []string{spinner.Dot.Frames[0], "[1/2] ", "10 MiB of 100 MiB", "10 MiB/s", "9s left"} {
		if !strings.Contains(line, want) {
			t.Errorf("status line %q lacks %q", line, want)
		}
	}
	s.show(progress.Event{Stage: progress.Downloaded, N: 1, Of: 2, Name: "a.zip"}, start.Add(10*time.Second))
	if line := s.line(start.Add(10 * time.Second)); line != "" {
		t.Errorf("after the part: status line %q, want none", line)
	}
}

// Colour is for a terminal; anywhere else the line stays plain text.
func TestANoticeIsPlainOutsideATerminal(t *testing.T) {
	if got := (printer{}).notice(text["restart"]); got != text["restart"] {
		t.Errorf("got %q", got)
	}
}
