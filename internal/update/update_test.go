// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ronalder100/homewend/internal/progress"
)

// github stands in for GitHub: the redirect that names the latest release,
// its binary for this machine and its checksums. It counts what it is asked.
func github(t *testing.T, version string, binary []byte, declared string) *int {
	t.Helper()
	asked := 0
	name := "homewend_" + runtime.GOOS + "_" + runtime.GOARCH
	mux := http.NewServeMux()
	mux.HandleFunc("/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		asked++
		http.Redirect(w, r, "/releases/tag/v"+version, http.StatusFound)
	})
	mux.HandleFunc("/releases/latest/download/"+name, func(w http.ResponseWriter, r *http.Request) { w.Write(binary) })
	mux.HandleFunc("/releases/latest/download/homewend_checksums.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("0000  homewend_other_arch\n" + declared + "  " + name + "\n"))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	was := latest
	latest = server.URL + "/releases/latest"
	t.Cleanup(func() { latest = was })
	return &asked
}

func sum(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func TestLatestReadsTheRedirect(t *testing.T) {
	cache(t)
	github(t, "0.1.3", nil, "")
	if got, err := Latest(context.Background()); got != "0.1.3" || err != nil {
		t.Errorf("got %q, %v", got, err)
	}
}

func TestInstallReplacesTheProgram(t *testing.T) {
	binary := []byte("the new homewend")
	github(t, "0.1.3", binary, sum(binary))
	target := filepath.Join(t.TempDir(), "homewend")
	os.WriteFile(target, []byte("the old one"), 0o755)

	var last progress.Event
	if err := Install(context.Background(), target, func(e progress.Event) { last = e }); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(target)
	info, _ := os.Stat(target)
	if string(got) != string(binary) || info.Mode().Perm() != 0o755 {
		t.Errorf("installed %q, mode %v", got, info.Mode())
	}
	if last.Stage != progress.Update || last.Done != int64(len(binary)) {
		t.Errorf("last event %+v", last)
	}
	if left, _ := filepath.Glob(filepath.Join(filepath.Dir(target), ".homewend-*")); len(left) != 0 {
		t.Errorf("left behind: %v", left)
	}
}

// A download that does not match its checksum installs nothing and leaves
// nothing behind.
func TestInstallRefusesADamagedDownload(t *testing.T) {
	github(t, "0.1.3", []byte("damaged on the way"), sum([]byte("what was released")))
	target := filepath.Join(t.TempDir(), "homewend")
	os.WriteFile(target, []byte("the old one"), 0o755)

	if err := Install(context.Background(), target, nil); !errors.Is(err, ErrDamaged) {
		t.Fatalf("got %v, want ErrDamaged", err)
	}
	if got, _ := os.ReadFile(target); string(got) != "the old one" {
		t.Errorf("the program was replaced with %q", got)
	}
	if left, _ := filepath.Glob(filepath.Join(filepath.Dir(target), ".homewend-*")); len(left) != 0 {
		t.Errorf("left behind: %v", left)
	}
}

// cache points the user's cache directory at a temporary one.
func cache(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", dir)
	t.Setenv("HOME", dir)
	got, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(got, "homewend")
}

// GitHub is asked once a day at most; the running release and a build that is
// not a release are told nothing.
func TestNewerAsksOnceADay(t *testing.T) {
	asked := github(t, "0.1.3", nil, "")
	cache(t)
	ctx := context.Background()
	for range 2 {
		if got := Newer(ctx, "0.1.2"); got != "0.1.3" {
			t.Errorf("got %q, want 0.1.3", got)
		}
	}
	if *asked != 1 {
		t.Errorf("asked %d times, want once", *asked)
	}
	if got := Newer(ctx, "0.1.3"); got != "" {
		t.Errorf("the running release: got %q", got)
	}
	cache(t)
	if got := Newer(ctx, "dev"); got != "" || *asked != 1 {
		t.Errorf("a dev build: got %q, asked %d times", got, *asked)
	}
}

// No network is no news, and nothing is kept for tomorrow to believe.
func TestNewerWithNoAnswer(t *testing.T) {
	was := latest
	latest = "http://127.0.0.1:1/releases/latest"
	t.Cleanup(func() { latest = was })
	dir := cache(t)
	if got := Newer(context.Background(), "0.1.2"); got != "" {
		t.Errorf("got %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "latest")); err == nil {
		t.Error("an answer was kept")
	}
}

// What update is told replaces what was kept: an answer from before the
// releases changed must not go on being told for a day.
func TestLatestReplacesTheAnswerKept(t *testing.T) {
	dir := cache(t)
	os.MkdirAll(dir, 0o700)
	os.WriteFile(filepath.Join(dir, "latest"), []byte("0.1.4\n"), 0o600)
	ctx := context.Background()
	if got := Newer(ctx, "0.1.1"); got != "0.1.4" {
		t.Fatalf("the answer kept: got %q", got)
	}
	github(t, "0.1.1", nil, "")
	if _, err := Latest(ctx); err != nil {
		t.Fatal(err)
	}
	if got := Newer(ctx, "0.1.1"); got != "" {
		t.Errorf("after asking again: told of %q", got)
	}
}
