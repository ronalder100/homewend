// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"errors"
	"testing"

	"github.com/ronalder100/homewend/internal/progress"
	"github.com/ronalder100/homewend/internal/takeout"
)

func TestJobRecordsWhereItStands(t *testing.T) {
	jb := &job{state: JobState{Years: []YearProgress{}}}
	for _, e := range []progress.Event{
		{Stage: progress.Ready, Of: 3, Total: 300},
		{Stage: progress.Receiving, Done: 60, Total: 100},
		{Stage: progress.Downloaded},
		{Stage: progress.Receiving, Done: 40, Total: 100},
		{Stage: progress.Year, Name: "2019", N: 5, Of: 10},
		{Stage: progress.Year, Name: "2024", N: 1, Of: 2},
		{Stage: progress.Year, Name: "2019", N: 7, Of: 10},
		{Stage: progress.Retry, Note: "no network"},
	} {
		jb.record(e)
	}
	s := jb.state
	if s.Of != 3 || s.Total != 300 || s.Parts != 1 || s.Done != 100 {
		t.Fatalf("parts and bytes: %+v", s)
	}
	if len(s.Years) != 2 || s.Years[0].Year != "2024" || s.Years[1].Arrived != 7 {
		t.Fatalf("years: %+v", s.Years)
	}
	if s.Retry != "no network" || s.Stage != progress.Year {
		t.Fatalf("retry: %+v", s)
	}
}

func TestProblemNamesWhatAPersonCanFix(t *testing.T) {
	for err, want := range map[error]string{
		ErrSignInClosed:                 "signed-out",
		takeout.ErrSignedOut:            "signed-out",
		NoSpaceError{Need: 10, Free: 1}: "no-space",
		ErrExpired:                      "expired",
		errors.New("something else"):    "",
	} {
		var s JobState
		problem(err, &s)
		if s.Problem != want {
			t.Errorf("%v: got %q, want %q", err, s.Problem, want)
		}
	}
}
