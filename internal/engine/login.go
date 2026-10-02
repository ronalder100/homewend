// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/ronalder100/homewend/internal/progress"
	"github.com/ronalder100/homewend/internal/session"
	"github.com/ronalder100/homewend/internal/signin"
	"github.com/ronalder100/homewend/internal/takeout"
)

// ErrNotSignedIn means sign-in did not complete: the window was closed first,
// or the user declined.
var ErrNotSignedIn = errors.New("sign-in did not complete")

// DefaultProfile is where the browser profile lives unless told otherwise: in
// the user's config directory, because it is settings and a session, not data.
func DefaultProfile() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "homewend", "profile"), nil
}

// Login makes sure the profile holds a Google session. If it does not, it
// opens Google's sign-in in a small browser window, waits until Google sends
// the user back to our page, and closes the window.
//
// It reports whose session it is, as Google names the account, and whether the
// profile was already signed in.
func Login(ctx context.Context, sess *session.Session, emit progress.Func) (account string, already bool, err error) {
	if account, ok, err := signedIn(sess); ok || err != nil {
		return account, ok, err
	}
	back, err := signin.Listen()
	if err != nil {
		return "", false, err
	}
	defer back.Close()
	w, err := sess.OpenSignIn(back.URL())
	if err != nil {
		return "", false, err
	}
	emit.Emit(progress.Event{Stage: progress.SignIn})

	var result error
	select {
	case <-ctx.Done():
		w.Close()
		return "", false, ctx.Err()
	case <-w.Exited():
		return "", false, ErrNotSignedIn
	case result = <-back.Done():
	}
	// Long enough for the page's animation to play out, so the user sees how
	// it ended before the window goes.
	select {
	case <-time.After(3 * time.Second):
	case <-w.Exited():
	}
	// Closing writes the profile out: the browser takes cookies to disk on
	// shutdown, not only on its 30-second timer.
	w.Close()
	if result != nil {
		return "", false, ErrNotSignedIn
	}
	if !cookiesLanded(sess) {
		return "", false, ErrNotSignedIn
	}
	account, err = openTakeout(ctx, sess, emit)
	return account, false, err
}

// openTakeout opens Takeout once, out of sight, so the profile holds
// Takeout's own cookies. Signing in gives the account's cookies but not a
// service's: OSID and __Secure-OSID on takeout.google.com come only from
// opening Takeout signed in, and without them Takeout sends the session to
// sign in (measured 2026-10-01). The old sign-in started at Takeout and got
// them on the way. Closing the page waits for them to reach the disk.
func openTakeout(ctx context.Context, sess *session.Session, emit progress.Func) (account string, err error) {
	emit.Emit(progress.Event{Stage: progress.SessionReady})
	page, err := sess.Headless(ctx, takeout.ManageURL)
	if err != nil {
		return "", err
	}
	page.Close()
	account, ok, err := signedIn(sess)
	if err == nil && !ok {
		err = ErrNotSignedIn
	}
	return account, err
}

// cookiesLanded waits for the session cookies to reach the profile after the
// browser has shut down. Its main process can exit before they are on disk:
// on 2026-10-01 a read right after the exit found none, and the file was
// written 170 ms later.
func cookiesLanded(sess *session.Session) bool {
	for range 50 {
		if _, names, err := sess.Cookies(); err == nil && names["SID"] && names["SSID"] {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// signedIn reports whether the profile holds a session Google accepts, and
// whose.
func signedIn(sess *session.Session) (account string, ok bool, err error) {
	_, names, err := sess.Cookies()
	if err != nil || !names["SID"] || !names["SSID"] {
		return "", false, nil
	}
	return takeout.Account(sess)
}
