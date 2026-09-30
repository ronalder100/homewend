// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ronalder/homewend/engine/internal/takeout"
)

func TestAskedSinceFindsTheExportMadeAfterAsking(t *testing.T) {
	asked := time.Date(2026, 9, 26, 17, 58, 0, 0, time.UTC)
	exports := []takeout.Export{
		{Job: "older", Created: asked.Add(-time.Hour)},
		{Job: "ours", Created: asked.Add(30 * time.Second)},
		// Google's clock a little behind ours.
		{Job: "skewed", Created: asked.Add(-20 * time.Second)},
	}
	if got := askedSince(exports, asked); got != "ours" {
		t.Errorf("got %q, want the newest made after asking", got)
	}
	if got := askedSince(exports[:1], asked); got != "" {
		t.Errorf("got %q, want none: the only export predates the request", got)
	}
}

func TestARequestSurvivesBetweenRuns(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".homewend", "request-2025.json")
	if req, err := loadRequest(path); err != nil || req.Job != "" || !req.Asked.IsZero() {
		t.Fatalf("no file: got %+v, %v; want an empty request", req, err)
	}
	want := request{Year: 2025, Asked: time.Date(2026, 9, 26, 17, 58, 0, 0, time.UTC), Job: "34255969"}
	if err := writeJSON(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := loadRequest(path)
	if err != nil || got.Year != want.Year || !got.Asked.Equal(want.Asked) || got.Job != want.Job {
		t.Errorf("got %+v, %v; want %+v", got, err, want)
	}
}
