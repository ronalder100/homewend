// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package session

import (
	"os/exec"
	"testing"
	"time"
)

// A process with no window to close is killed rather than waited on.
func TestShutDownKillsWhatWillNotClose(t *testing.T) {
	cmd := exec.Command("cmd", "/c", "ping -n 60 127.0.0.1 >nul")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan struct{})
	go func() {
		cmd.Wait()
		close(exited)
	}()
	done := make(chan struct{})
	go func() { shutDown(cmd, exited); close(done) }()
	select {
	case <-done:
	case <-time.After(45 * time.Second):
		t.Fatal("shutDown did not return")
	}
}
