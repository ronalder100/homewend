// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package session

import (
	"database/sql"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// WriteHistory puts a History file in profile with the given redirect chains,
// in the one table DownloadURLs reads, shaped as Chrome creates it.
func writeHistory(t *testing.T, profile string, chains ...[]string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(profile, "Default"), 0o700); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(profile, "Default", "History"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE downloads_url_chains (id INTEGER NOT NULL, chain_index INTEGER NOT NULL, url LONGVARCHAR NOT NULL, PRIMARY KEY (id, chain_index))`); err != nil {
		t.Fatal(err)
	}
	for id, chain := range chains {
		for i, u := range chain {
			if _, err := db.Exec(`INSERT INTO downloads_url_chains VALUES (?, ?, ?)`, id+1, i, u); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestDownloadURLsAreTheLastLinkOfEachChainNewestFirst(t *testing.T) {
	profile := t.TempDir()
	writeHistory(t, profile,
		[]string{"https://a.example/start", "https://a.example/end"},
		[]string{"https://b.example/start", "https://b.example/middle", "https://b.example/end"},
	)
	s, err := New(profile)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.DownloadURLs()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "https://b.example/end" || got[1] != "https://a.example/end" {
		t.Errorf("got %q", got)
	}
}

func TestAProfileWithNoHistoryHasNoDownloads(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got, err := s.DownloadURLs(); err != nil || got != nil {
		t.Errorf("got %q, %v; want none and no error", got, err)
	}
}

// Chrome keeps History in WAL mode, and a download it has just recorded can be
// in History-wal alone. The browser holds the file open meanwhile, as the
// writer below does until the read is over.
func TestDownloadURLsSeeRowsStillInTheWAL(t *testing.T) {
	profile := t.TempDir()
	writeHistory(t, profile)
	db, err := sql.Open("sqlite", filepath.Join(profile, "Default", "History"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, stmt := range []string{
		`PRAGMA journal_mode=WAL`,
		`PRAGMA wal_autocheckpoint=0`,
		`INSERT INTO downloads_url_chains VALUES (1, 0, 'https://a.example/end')`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(profile, "Default", "History-wal")); err != nil {
		t.Fatal("the row is not in a WAL file:", err)
	}

	s, err := New(profile)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.DownloadURLs()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "https://a.example/end" {
		t.Errorf("got %q", got)
	}
}

// The cookie keys are Chrome's, not ours to change. These are the ones real
// profiles were read with, on Linux and on a Mac, derived then by
// golang.org/x/crypto: whatever does the deriving has to give the same.
func TestTheCookieKeysAreChromes(t *testing.T) {
	for system, want := range map[string]string{
		"linux":  "fd621fe5a2b402539dfa147ca9272778",
		"darwin": "af0f762aaf6d7d11581b7aa8ce7218de",
	} {
		if got := hex.EncodeToString(keyFor(system)); got != want {
			t.Errorf("%s: key %s, want %s", system, got, want)
		}
	}
}
