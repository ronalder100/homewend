// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// A stand-in browser that, like Chrome, shuts down when asked with SIGTERM
// and writes down that it was asked.
func TestCloseAsksTheBrowserToShutDown(t *testing.T) {
	dir := t.TempDir()
	asked := filepath.Join(dir, "asked")
	ready := filepath.Join(dir, "ready")
	browser := filepath.Join(dir, "browser")
	// Touch ready right after the trap is installed, so the test can wait
	// for the trap instead of guessing how long that takes.
	script := "#!/bin/sh\ntrap 'touch " + asked + "; exit 0' TERM\ntouch " + ready + "\nwhile :; do sleep 0.05; done\n"
	if err := os.WriteFile(browser, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BROWSER_BIN", browser)

	sess, err := New(filepath.Join(dir, "profile"))
	if err != nil {
		t.Fatal(err)
	}
	w, err := sess.Open("about:blank")
	if err != nil {
		t.Fatal(err)
	}
	waitForFile(t, ready, 5*time.Second) // wait for the trap to be set, rather than guessing

	closed := make(chan struct{})
	go func() { w.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not return")
	}
	if _, err := os.Stat(asked); err != nil {
		t.Error("the browser was not asked to shut down")
	}
}

// The browser is told not to offer to save passwords before it starts, in a
// new profile and in one that has preferences already, whose others are kept.
func TestOpenNeverOffersToSavePasswords(t *testing.T) {
	dir := t.TempDir()
	browser := filepath.Join(dir, "browser")
	if err := os.WriteFile(browser, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BROWSER_BIN", browser)

	for _, before := range []string{"", `{"credentials_enable_service":true,"other":7}`} {
		sess, err := New(filepath.Join(t.TempDir(), "profile"))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(sess.Profile, "Default", "Preferences")
		if before != "" {
			os.MkdirAll(filepath.Dir(path), 0o700)
			os.WriteFile(path, []byte(before), 0o600)
		}
		w, err := sess.Open("about:blank")
		if err != nil {
			t.Fatal(err)
		}
		<-w.Exited()
		raw, _ := os.ReadFile(path)
		var prefs map[string]any
		if err := json.Unmarshal(raw, &prefs); err != nil || prefs["credentials_enable_service"] != false {
			t.Errorf("after %q: preferences %s", before, raw)
		}
		if before != "" && prefs["other"] != float64(7) {
			t.Errorf("the other preferences were lost: %s", raw)
		}
	}
}

// waitForFile polls for path to appear, failing the test if it does not show
// up within timeout.
func waitForFile(t *testing.T, path string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if _, err := os.Stat(path); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s never appeared", path)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// OpenSignIn lets the page open its popup, and opens the address last.
func TestOpenSignInAllowsThePopup(t *testing.T) {
	dir := t.TempDir()
	args := filepath.Join(dir, "args")
	browser := filepath.Join(dir, "browser")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + args + ".tmp && mv " + args + ".tmp " + args + "\n"
	if err := os.WriteFile(browser, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BROWSER_BIN", browser)

	sess, err := New(filepath.Join(dir, "profile"))
	if err != nil {
		t.Fatal(err)
	}
	w, err := sess.OpenSignIn("http://127.0.0.1:1/")
	if err != nil {
		t.Fatal(err)
	}
	<-w.Exited()
	waitForFile(t, args, 5*time.Second)
	got, _ := os.ReadFile(args)
	lines := strings.Split(strings.TrimSpace(string(got)), "\n")
	if lines[0] != "--user-data-dir="+sess.Profile || !slices.Contains(lines, "--disable-popup-blocking") || lines[len(lines)-1] != "http://127.0.0.1:1/" {
		t.Errorf("browser started with %q", lines)
	}
}
