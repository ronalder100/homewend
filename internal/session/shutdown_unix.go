// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build !windows

package session

import (
	"os/exec"
	"syscall"
)

// shutDown stops a browser we launched and waits until exited is closed.
// SIGTERM, not a kill: Chrome takes it as a request to shut down
// (chrome/browser/chrome_browser_main_posix.cc). It does not wait for the
// cookies of the last seconds: four times in six they were lost, the write
// left half done (Chromium 144, 2026-10-02). Whoever needs them on disk sees
// them there before closing.
func shutDown(cmd *exec.Cmd, exited <-chan struct{}) {
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		cmd.Process.Kill()
	}
	<-exited
}
