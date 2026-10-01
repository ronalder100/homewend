// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"errors"
	"testing"
	"time"

	"github.com/ronalder100/homewend/internal/takeout"
)

var now = time.Date(2026, 10, 1, 16, 0, 0, 0, time.UTC)

func ready(job string, age time.Duration) takeout.Export {
	return takeout.Export{Job: job, Created: now.Add(-age),
		Parts: []takeout.Part{{Filename: "a.zip"}}, Manifest: takeout.Part{Filename: "m.zip"}}
}

// preparing is an export Google has not finished; gone, one it no longer offers.
func preparing(job string, age time.Duration) takeout.Export {
	return takeout.Export{Job: job, Created: now.Add(-age)}
}

func gone(job string, age time.Duration) takeout.Export {
	return takeout.Export{Job: job, Created: now.Add(-age), Expired: true}
}

func TestStatusOf(t *testing.T) {
	day := 24 * time.Hour
	for _, c := range []struct {
		e    takeout.Export
		want Status
	}{
		{ready("a", 5*day), Ready},
		{preparing("b", time.Hour), Preparing},
		{gone("c", 8*day), Expired},
	} {
		if got := StatusOf(c.e); got != c.want {
			t.Errorf("%s: got %s, want %s", c.e.Job, got, c.want)
		}
	}
}

func TestByID(t *testing.T) {
	exports := []takeout.Export{ready("8f6c3233-aaaa", 0), ready("8f6d0000-bbbb", 0)}
	if e, err := byID(exports, "8f6c"); err != nil || e.Job != "8f6c3233-aaaa" {
		t.Errorf("got %v, %v", e.Job, err)
	}
	if _, err := byID(exports, "8f6"); err == nil {
		t.Error("an ambiguous id was accepted")
	}
	if _, err := byID(exports, "ffff"); !errors.Is(err, ErrNoSuchTakeout) {
		t.Errorf("got %v, want ErrNoSuchTakeout", err)
	}
}

func TestLatestForPicksTheNewestStillOffered(t *testing.T) {
	day := 24 * time.Hour
	exports := []takeout.Export{
		ready("old-2025", 3*day),
		ready("new-2025", day),
		ready("2024", 0),
		gone("gone-2025", 10*day),
	}
	notes := []note{
		{Year: 2025, Job: "old-2025"},
		{Year: 2025, Job: "new-2025"},
		{Year: 2024, Job: "2024"},
		{Year: 2025, Job: "gone-2025"},
	}
	if job, _ := latestFor(2025, notes, exports); job != "new-2025" {
		t.Errorf("got %q, want new-2025", job)
	}
	if job, _ := latestFor(2023, notes, exports); job != "" {
		t.Errorf("got %q for a year never asked for, want none", job)
	}
}

func TestLatestForSkipsAnExpiredExport(t *testing.T) {
	exports := []takeout.Export{gone("gone", 8*24*time.Hour)}
	if job, _ := latestFor(2025, []note{{Year: 2025, Job: "gone"}}, exports); job != "" {
		t.Errorf("got %q, want none: the export expired", job)
	}
}

// A run stopped between sending the form and seeing the export listed finds
// it by its time, and keeps the job for next time.
func TestLatestForFindsTheExportOfAnUnfinishedNote(t *testing.T) {
	asked := now.Add(-time.Hour)
	exports := []takeout.Export{preparing("before", 2*time.Hour), preparing("after", 50*time.Minute)}
	job, notes := latestFor(2025, []note{{Year: 2025, Asked: asked}}, exports)
	if job != "after" || notes[0].Job != "after" {
		t.Errorf("got %q, note %q; want after", job, notes[0].Job)
	}
}

func TestNotesSurviveBetweenRuns(t *testing.T) {
	profile := t.TempDir()
	if notes, err := loadNotes(profile); err != nil || notes != nil {
		t.Fatalf("no file: got %v, %v", notes, err)
	}
	want := []note{{Year: 2025, Asked: now, Job: "8f6c3233"}}
	if err := saveNotes(profile, want); err != nil {
		t.Fatal(err)
	}
	got, err := loadNotes(profile)
	if err != nil || len(got) != 1 || got[0].Job != want[0].Job || !got[0].Asked.Equal(now) {
		t.Errorf("got %+v, %v", got, err)
	}
}
