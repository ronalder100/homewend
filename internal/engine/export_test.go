// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/ronalder100/homewend/internal/session"
	"github.com/ronalder100/homewend/internal/takeout"
)

func TestUserIsReadFromADownloadAndKept(t *testing.T) {
	profile := t.TempDir()
	if err := os.MkdirAll(filepath.Join(profile, "Default"), 0o700); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(profile, "Default", "History"))
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`CREATE TABLE downloads_url_chains (id INTEGER NOT NULL, chain_index INTEGER NOT NULL, url LONGVARCHAR NOT NULL)`,
		`INSERT INTO downloads_url_chains VALUES (1, 0, 'https://takeout.google.com/takeout/download?j=x&i=0&user=98765432109876543210')`,
		`INSERT INTO downloads_url_chains VALUES (1, 1, 'https://takeout-download.usercontent.google.com/download/a.zip?j=x&i=0&user=123456&authuser=0')`,
		`INSERT INTO downloads_url_chains VALUES (2, 0, 'https://example.com/unrelated.pdf')`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()

	sess, err := session.New(profile)
	if err != nil {
		t.Fatal(err)
	}
	// No browser is opened: the id is already in the history. BROWSER_BIN
	// points nowhere so that opening one would fail the test.
	t.Setenv("BROWSER_BIN", filepath.Join(profile, "no-browser"))
	id, err := User(context.Background(), sess, "x", nil)
	if err != nil || id != "123456" {
		t.Fatalf("got %q, %v; want 123456", id, err)
	}

	// Kept: found again with the history gone.
	os.Remove(filepath.Join(profile, "Default", "History"))
	if id, err := User(context.Background(), sess, "x", nil); err != nil || id != "123456" {
		t.Errorf("second time: got %q, %v; want 123456", id, err)
	}
}

func TestSpaceNeededCountsOnlyWhatIsLeft(t *testing.T) {
	f := Fetch{Export: takeout.Export{
		Parts: []takeout.Part{
			{Filename: "a.zip", Size: 100},
			{Filename: "b.zip", Size: 300},
			{Filename: "c.zip", Size: 200},
		},
		Manifest: takeout.Part{Filename: "m.zip", Size: 7},
	}}

	fresh := state{Unpacked: map[string]bool{}}
	// Every part, the largest twice, the manifest.
	if got := f.spaceNeeded(fresh); got != 100+300+200+300+7 {
		t.Errorf("fresh: %d", got)
	}

	resumed := state{Unpacked: map[string]bool{"b.zip": true}, Manifest: "m.zip"}
	if got := f.spaceNeeded(resumed); got != 100+200+200 {
		t.Errorf("resumed: %d", got)
	}
}
