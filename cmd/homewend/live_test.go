// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"strings"
	"testing"
	"time"

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
	for _, want := range []string{"[1/2] ", "10 MiB of 100 MiB", "10 MiB/s", "9s left"} {
		if !strings.Contains(line, want) {
			t.Errorf("status line %q lacks %q", line, want)
		}
	}
	s.show(progress.Event{Stage: progress.Downloaded, N: 1, Of: 2, Name: "a.zip"}, start.Add(10*time.Second))
	if line := s.line(start.Add(10 * time.Second)); line != "" {
		t.Errorf("after the part: status line %q, want none", line)
	}
}
