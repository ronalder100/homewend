// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ronalder100/homewend/internal/session/sessiontest"
)

func TestMain(m *testing.M) {
	sessiontest.Main()
	os.Exit(m.Run())
}

// A stand-in browser that, like Chrome, shuts down when asked with SIGTERM
// and writes down that it was asked.
func TestCloseAsksTheBrowserToShutDown(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("taskkill asks the browser's windows to close, and the stand-in has none")
	}
	dir := t.TempDir()
	asked := filepath.Join(dir, "asked")
	ready := filepath.Join(dir, "ready")
	// Ready is written once the stand-in listens for SIGTERM, so the test can
	// wait for that instead of guessing how long it takes.
	sessiontest.Use(t, sessiontest.Browser{Asked: asked, Ready: ready, Stay: time.Minute})

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
	sessiontest.Use(t, sessiontest.Browser{})

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

// Where downloads go is set in the preferences, beside what is already
// there, and given back by setting nowhere.

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
	sessiontest.Use(t, sessiontest.Browser{Args: args})

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
