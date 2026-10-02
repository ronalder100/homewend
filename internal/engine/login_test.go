// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ronalder100/homewend/internal/session"
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
