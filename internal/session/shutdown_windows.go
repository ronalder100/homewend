// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package session

import (
	"os/exec"
	"strconv"
	"time"
)

// shutDown stops a browser we launched and waits until exited is closed.
//
// There is no SIGTERM here. What taskkill sends without /F is a close message
// to the browser's windows, which Chrome answers by shutting down cleanly:
// a profile closed that way kept its cookies, one killed had none (Chrome
// 154 and Edge 153 on Windows Server 2025, 2026-10-01).
// A browser that does not go — no window to close, or one that will not —
// is killed after a while rather than waited on forever.
//
// taskkill names the process by its number, which Windows gives to the next
// process once this one is gone: a browser that has already exited, as one
// closed through DevTools has, is not looked for again.
func shutDown(cmd *exec.Cmd, exited <-chan struct{}) {
	select {
	case <-exited:
		return
	default:
	}
	if err := exec.Command("taskkill", "/PID", strconv.Itoa(cmd.Process.Pid)).Run(); err != nil {
		cmd.Process.Kill()
	}
	select {
	case <-exited:
	case <-time.After(30 * time.Second):
		cmd.Process.Kill()
		<-exited
	}
}
