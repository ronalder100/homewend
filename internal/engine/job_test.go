// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ronalder100/homewend/internal/progress"
	"github.com/ronalder100/homewend/internal/takeout"
)

func TestJobRecordsWhereItStands(t *testing.T) {
	jb := &job{state: JobState{Years: []YearProgress{}}}
	for _, e := range []progress.Event{
		{Stage: progress.Ready, Of: 3, Total: 300},
		{Stage: progress.Receiving, Done: 60, Total: 100},
		{Stage: progress.Downloaded, Done: 100, Total: 100},
		{Stage: progress.Receiving, Done: 40, Total: 100},
		{Stage: progress.Year, Name: "2019", N: 5, Of: 10},
		{Stage: progress.Year, Name: "2024", N: 1, Of: 2},
		{Stage: progress.Year, Name: "2019", N: 7, Of: 10},
		{Stage: progress.Retry, Note: "no network"},
	} {
		jb.record(e)
	}
	s := jb.state
	if s.Of != 3 || s.Total != 300 || s.Parts != 1 || s.Done != 140 {
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
		ErrSignInClosed:                      "signed-out",
		takeout.ErrSignedOut:                 "signed-out",
		NoSpaceError{Need: 10, Free: 1}:      "no-space",
		ErrExpired:                           "expired",
		&takeout.NotOfferedError{Year: 2024}: "not-offered",
		errors.New("something else"):         "",
	} {
		var s JobState
		problem(err, &s)
		if s.Problem != want {
			t.Errorf("%v: got %q, want %q", err, s.Problem, want)
		}
	}
}

// A year Google has no photos of is said with the years it has.
func TestNotOfferedSaysTheYearsGoogleHas(t *testing.T) {
	var s JobState
	// "EgUyAwjpDw" is Google's id for 2025 (see takeout.YearOf); "Trip" an album.
	problem(&takeout.NotOfferedError{Year: 2024, Offered: []takeout.Entry{{ID: "EgUyAwjpDw"}, {ID: "Trip"}}}, &s)
	if s.Problem != "not-offered" || len(s.Offered) != 1 || s.Offered[0] != 2025 {
		t.Fatalf("got %q %v, want not-offered [2025]", s.Problem, s.Offered)
	}
}

// A job that carries on is told of the parts a run before brought, and starts
// from them; a part found damaged is taken off the count until it comes again.
func TestJobCountsThePartsAlreadyHere(t *testing.T) {
	jb := &job{state: JobState{Years: []YearProgress{}}}
	for _, e := range []progress.Event{
		{Stage: progress.Ready, Name: "job", Of: 3, Total: 300},
		{Stage: progress.Already, N: 1, Of: 3, Done: 10, Total: 10},
		{Stage: progress.Already, N: 2, Of: 3, Done: 100, Total: 100},
		{Stage: progress.Downloaded, N: 3, Of: 3, Done: 200, Total: 200},
		{Stage: progress.Damaged, N: 3, Of: 3, Total: 200},
		{Stage: progress.Receiving, N: 3, Of: 3, Done: 50, Total: 200},
	} {
		jb.record(e)
	}
	if s := jb.state; s.Export != "job" || s.Parts != 2 || s.Done != 160 {
		t.Fatalf("%+v, want job, 2 parts and 160 bytes", s)
	}
}

// slow is a job that takes a while to let go once stopped, as one closing
// its browser does; ended says when it has.
func slow(ended *bool) func(context.Context, progress.Func) error {
	return func(ctx context.Context, _ progress.Func) error {
		<-ctx.Done()
		time.Sleep(50 * time.Millisecond)
		*ended = true
		return ctx.Err()
	}
}

func TestStopWaitsForTheJobToEnd(t *testing.T) {
	var j Jobs
	var ended bool
	if err := j.start("a", Get{Takeout: "abc"}, slow(&ended)); err != nil {
		t.Fatal(err)
	}
	if s := j.State("a"); s.Export != "abc" {
		t.Errorf("export %q, want the takeout asked for from the start", s.Export)
	}
	<-j.Stop("a")
	if !ended {
		t.Fatal("Stop's channel closed before the job ended")
	}
	if s := j.State("a"); s.Running || s.Error != "" {
		t.Errorf("stopped: %+v", s)
	}
	select {
	case <-j.Stop("none"):
	default:
		t.Error("stopping no job does not end at once")
	}
}

func TestStopAllWaitsForEveryJob(t *testing.T) {
	var j Jobs
	ended := make([]bool, 2)
	for i, account := range []string{"a", "b"} {
		if err := j.start(account, Get{}, slow(&ended[i])); err != nil {
			t.Fatal(err)
		}
	}
	j.StopAll()
	if !ended[0] || !ended[1] {
		t.Fatalf("StopAll returned before every job ended: %v", ended)
	}
}
