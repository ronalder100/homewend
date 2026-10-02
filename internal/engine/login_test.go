// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ronalder100/homewend/internal/session"
	"github.com/ronalder100/homewend/internal/takeout"
)

// standIn opens a stand-in browser that runs for the given seconds and
// writes down whether it was asked to stop before that.
func standIn(t *testing.T, seconds string) (sess *session.Session, w *session.Window, asked string) {
	t.Helper()
	dir := t.TempDir()
	asked = filepath.Join(dir, "asked")
	browser := filepath.Join(dir, "browser")
	script := "#!/bin/sh\ntrap 'touch " + asked + "; exit 0' TERM\nsleep " + seconds + " &\nwait $!\n"
	if err := os.WriteFile(browser, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BROWSER_BIN", browser)
	sess, err := session.New(filepath.Join(dir, "profile"))
	if err != nil {
		t.Fatal(err)
	}
	if w, err = sess.Open("about:blank"); err != nil {
		t.Fatal(err)
	}
	return sess, w, asked
}

// A browser that goes by itself is left to: stopping it is what loses the
// session.
func TestLeaveLetsTheBrowserGoByItself(t *testing.T) {
	sess, w, asked := standIn(t, "0.3")
	leave(context.Background(), sess, w, 30*time.Second)
	select {
	case <-w.Exited():
	default:
		t.Error("leave returned with the browser still running")
	}
	if _, err := os.Stat(asked); err == nil {
		t.Error("the browser was asked to stop")
	}
}

// One that stays is stopped when its time is up.
func TestLeaveStopsABrowserThatStays(t *testing.T) {
	sess, w, asked := standIn(t, "30")
	start := time.Now()
	leave(context.Background(), sess, w, 300*time.Millisecond)
	if time.Since(start) > 10*time.Second {
		t.Error("leave waited for the browser to end by itself")
	}
	if _, err := os.Stat(asked); err != nil {
		t.Error("the browser was not asked to stop")
	}
}

// The account kept at sign-in is read from the profile: Google is not asked,
// and this profile, with no session, could not ask.
func TestAccountIsTheOneKeptAtSignIn(t *testing.T) {
	sess, err := session.New(filepath.Join(t.TempDir(), "profile"))
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(accountFile(sess), []byte("someone@example.com\n"), 0o600)
	if got := Account(sess); got != "someone@example.com" {
		t.Errorf("got %q", got)
	}
}

// The fallback opens the browser on Takeout, and a browser closed before
// anyone signed in is not a sign-in.
func TestLoginAtTakeoutOpensTakeout(t *testing.T) {
	dir := t.TempDir()
	args := filepath.Join(dir, "args")
	browser := filepath.Join(dir, "browser")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + args + "\n"
	if err := os.WriteFile(browser, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BROWSER_BIN", browser)

	sess, err := session.New(filepath.Join(dir, "profile"))
	if err != nil {
		t.Fatal(err)
	}
	_, already, err := LoginAtTakeout(context.Background(), sess, nil)
	if already || !errors.Is(err, ErrSignInClosed) || !errors.Is(err, ErrNotSignedIn) {
		t.Errorf("got %v, %v; want false, ErrSignInClosed", already, err)
	}
	got, _ := os.ReadFile(args)
	lines := strings.Split(strings.TrimSpace(string(got)), "\n")
	if lines[len(lines)-1] != takeout.PhotosURL {
		t.Errorf("browser started with %q", lines)
	}
}

// The address to open is Google's own, for the export's smallest file: the
// one with the manifest's index; any other of the export's when that one is
// not there; none when the page has no download on it.
func TestPickLinkTakesTheManifestsDownload(t *testing.T) {
	part := "https://takeout.google.com/takeout/download?j=job&i=0&user=123456789012345678901"
	manifest := "https://takeout.google.com/takeout/download?j=job&i=1&user=123456789012345678901"
	page := []string{"https://takeout.google.com/manage", manifest, part, "https://support.google.com/download"}
	if got := pickLink(page, 1); got != manifest {
		t.Errorf("got %q, want the manifest's", got)
	}
	if got := pickLink(page, 7); got != manifest {
		t.Errorf("no such index: got %q, want the first download", got)
	}
	if got := pickLink(page[:1], 1); got != "" {
		t.Errorf("a page with no download: got %q", got)
	}
}
