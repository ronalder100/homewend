// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package sessiontest makes the test binary stand in for the browser, so that
// the tests that start one run alike on every system: a shell script does not
// run on Windows.
package sessiontest

import (
	"encoding/json"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"
)

// env carries what the stand-in is to do, from the test to the binary it
// starts as the browser.
const env = "HOMEWEND_STAND_IN"

// Browser is what the stand-in does once started. Each file is left out when
// its name is "".
type Browser struct {
	Args  string        // written with the arguments it was started with, one per line
	Asked string        // written when it is asked to shut down (SIGTERM)
	Ready string        // written once it listens for that
	Stay  time.Duration // how long it runs if nobody asks it to go
}

// Use makes the test binary the browser for the rest of the test. The
// package's TestMain must call Main first.
func Use(t testing.TB, b Browser) {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("BROWSER_BIN", self)
	t.Setenv(env, string(raw))
}

// Main plays the browser and exits when the binary was started as one, and
// returns at once otherwise.
func Main() {
	raw := os.Getenv(env)
	if raw == "" {
		return
	}
	var b Browser
	if err := json.Unmarshal([]byte(raw), &b); err != nil {
		os.Exit(2)
	}
	asked := make(chan os.Signal, 1)
	signal.Notify(asked, syscall.SIGTERM)
	write(b.Ready, "")
	write(b.Args, strings.Join(os.Args[1:], "\n")+"\n")
	select {
	case <-asked:
		write(b.Asked, "")
	case <-time.After(b.Stay):
	}
	os.Exit(0)
}

// write puts the file in place whole, so that a test that sees it sees all of
// it.
func write(name, content string) {
	if name == "" {
		return
	}
	if os.WriteFile(name+".tmp", []byte(content), 0o600) != nil || os.Rename(name+".tmp", name) != nil {
		os.Exit(2)
	}
}
