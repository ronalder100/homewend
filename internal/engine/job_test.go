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
	if err := j.start("a", Get{}, slow(&ended)); err != nil {
		t.Fatal(err)
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
