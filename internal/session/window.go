// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package session

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// Window is a browser the user drives, open on our profile.
type Window struct {
	cmd    *exec.Cmd
	exited chan struct{}
	err    error
}

// Open launches the browser on the session's profile, at url, in a window the
// user can see. Sign-in must happen there: a hidden browser asking for a
// password is exactly what this product exists to avoid.
//
// No debug port, and nothing else that lets a program drive the browser:
// Google refuses sign-in in a browser launched with --remote-debugging-port
// ("This browser or app may not be secure"), and accepts the same browser
// without it. Measured on Chromium 144, 2026-09-26.
func (s *Session) Open(url string) (*Window, error) { return s.launch(url) }

// OpenSignIn is Open for a page that opens a popup as soon as it loads, with
// no click to allow it: the sign-in page, which hands over to Google in a
// popup and closes itself (see package signin). The popup has to come from an
// ordinary window: one opened from an --app window shows no address, and the
// address is why it is a popup (measured on Chromium 144, 2026-10-01).
func (s *Session) OpenSignIn(url string) (*Window, error) {
	return s.launch("--disable-popup-blocking", url)
}

// launch starts the browser on the profile, with what to open last.
func (s *Session) launch(open ...string) (*Window, error) {
	browser, err := findBrowser()
	if err != nil {
		return nil, err
	}
	if err := s.neverSavePasswords(); err != nil {
		return nil, err
	}
	args := []string{
		"--user-data-dir=" + s.Profile,
		// A fixed password for the cookie key, so that we can read the cookies
		// back (see derivedKey): the first on Linux, the second on macOS. Each
		// browser ignores the one that is not for its system.
		"--password-store=basic",
		"--use-mock-keychain",
		"--no-first-run", "--no-default-browser-check", "--disable-sync",
		// A browser stopped by an interrupted run looks like a crash to Chrome,
		// which then offers to restore the session: noise, next time round.
		"--disable-session-crashed-bubble", "--hide-crash-restore-bubble",
	}
	cmd := exec.Command(browser, append(args, open...)...)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting %s: %w", browser, err)
	}
	w := &Window{cmd: cmd, exited: make(chan struct{})}
	go func() {
		w.err = cmd.Wait()
		close(w.exited)
	}()
	return w, nil
}

// neverSavePasswords tells the browser not to offer to save a password in
// this profile. The profile's cookie key is a fixed one (see derivedKey), and
// so is the key of anything else it stores: a password saved here would be as
// good as written in the clear. It also takes one bubble off the sign-in.
//
// The preference is credentials_enable_service
// (components/password_manager/core/common/password_manager_pref_names.h:
// "When it is false, it doesn't ask if you want to save passwords"). Written
// into Default/Preferences before the browser starts, it is kept, in a new
// profile and in one the browser made itself: Chromium 144, 2026-10-02.
func (s *Session) neverSavePasswords() error {
	path := filepath.Join(s.Profile, "Default", "Preferences")
	prefs := map[string]any{}
	if raw, err := os.ReadFile(path); err == nil {
		// A file that does not parse is the browser's to repair, not ours.
		if json.Unmarshal(raw, &prefs) != nil {
			return nil
		}
	}
	if prefs["credentials_enable_service"] == false {
		return nil
	}
	prefs["credentials_enable_service"] = false
	raw, err := json.Marshal(prefs)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

// Err is how the browser ended, once Exited is closed: nil when it exited
// cleanly, as it does when the user closes it.
func (w *Window) Err() error { return w.err }

// Exited is closed when the user closes the browser.
func (w *Window) Exited() <-chan struct{} { return w.exited }

// Close closes the browser once it has served its purpose, and waits for it
// to go.
func (w *Window) Close() { shutDown(w.cmd, w.exited) }

// shutDown stops a browser we launched and waits until exited is closed.
// SIGTERM, not a kill: Chrome takes it as a request to shut down cleanly and
// writes the profile out first (chrome/browser/chrome_browser_main_posix.cc).
// Where there is no SIGTERM, on Windows, it is killed.
func shutDown(cmd *exec.Cmd, exited <-chan struct{}) {
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		cmd.Process.Kill()
	}
	<-exited
}
