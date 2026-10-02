// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ronalder100/homewend/internal/progress"
	"github.com/ronalder100/homewend/internal/session"
	"github.com/ronalder100/homewend/internal/signin"
	"github.com/ronalder100/homewend/internal/takeout"
)

// ErrNotSignedIn means sign-in did not complete: the window was closed first,
// or the user declined.
var ErrNotSignedIn = errors.New("sign-in did not complete")

// The ways it does not, each one an ErrNotSignedIn: told apart because what
// the user does next differs, and so that a failure can be found.
var (
	// The browser went before Google sent the user back: closed, or stopped.
	ErrSignInClosed = fmt.Errorf("%w: the browser closed before Google sent the user back", ErrNotSignedIn)
	// Google sent the user back without signing them in.
	ErrSignInDeclined = fmt.Errorf("%w: Google sent the user back without signing them in", ErrNotSignedIn)
	// Signed in, but the session never reached the profile on disk.
	ErrSessionNotWritten = fmt.Errorf("%w: the browser closed without writing the session", ErrNotSignedIn)
	// Signed in, but Takeout does not accept the session.
	ErrSessionNotAccepted = fmt.Errorf("%w: Takeout does not accept the new session", ErrNotSignedIn)
)

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
// It reports whether the profile was already signed in and, after a sign-in,
// whose session it is, as Google names the account.
func Login(ctx context.Context, sess *session.Session, emit progress.Func) (account string, already bool, err error) {
	emit.Emit(progress.Event{Stage: progress.Checking})
	if ok, err := signedIn(sess); ok || err != nil {
		return "", ok, err
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
		if err := w.Err(); err != nil {
			return "", false, fmt.Errorf("%w (%v)", ErrSignInClosed, err)
		}
		return "", false, ErrSignInClosed
	case result = <-back.Done():
	}
	if result != nil {
		leave(ctx, sess, w, declinedLeave)
		return "", false, fmt.Errorf("%w (%v)", ErrSignInDeclined, result)
	}
	emit.Emit(progress.Event{Stage: progress.SessionReady})
	leave(ctx, sess, w, signedInLeave)
	if ctx.Err() != nil {
		return "", false, ctx.Err()
	}
	if !cookiesLanded(sess) {
		return "", false, ErrSessionNotWritten
	}
	account, err = openTakeout(ctx, sess, emit)
	if account != "" {
		os.WriteFile(accountFile(sess), []byte(account+"\n"), 0o600)
	}
	return account, false, err
}

// accountFile keeps the address of the account the profile is signed in to.
// It is inside the profile, so that it goes when the profile does.
func accountFile(sess *session.Session) string {
	return filepath.Join(sess.Profile, "homewend-account")
}

// Account says whose session the profile holds, as Google named the account
// at sign-in. It is kept beside the session because asking Google again means
// downloading Takeout's whole page, seconds of it; a profile signed in before
// the address was kept is asked for it once.
func Account(sess *session.Session) string {
	if kept, err := os.ReadFile(accountFile(sess)); err == nil {
		return strings.TrimSpace(string(kept))
	}
	account, _ := takeout.Account(sess)
	if account != "" {
		os.WriteFile(accountFile(sess), []byte(account+"\n"), 0o600)
	}
	return account
}

// How long the browser is given to go by itself: after a sign-in, longer than
// the 30 seconds Chrome takes at most to write its cookies; after one Google
// declined, there is nothing to wait for but the page's goodbye.
const (
	// takeoutHost is where a headless visit to Takeout ends when the session
	// is accepted; one that is not ends at accounts.google.com.
	takeoutHost = "takeout.google.com"

	signedInLeave = 45 * time.Second
	declinedLeave = 5 * time.Second
)

// leave lets the browser go by itself. The page closes its own window, and a
// browser with no window left quits and writes the profile out. It is not
// stopped on the way: stopped three seconds after 60 cookies were set, it lost
// every one of them four times in six, the write left half done; closing its
// own window, it kept them five times in five (Chromium 144, 2026-10-02).
//
// A browser that stays open with no window, as on a Mac, or whose page could
// not close its window, is stopped once the session is on disk, which its own
// timer sees to; and after limit, whatever is there.
func leave(ctx context.Context, sess *session.Session, w *session.Window, limit time.Duration) {
	late := time.After(limit)
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-w.Exited():
			return
		case <-tick.C:
			if !onDisk(sess) {
				continue
			}
		case <-late:
		case <-ctx.Done():
		}
		w.Close()
		return
	}
}

// onDisk reports whether the profile holds the session's cookies.
func onDisk(sess *session.Session) bool {
	_, names, err := sess.Cookies()
	return err == nil && names["SID"] && names["SSID"]
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
	// Where the browser ended up says whether Takeout took the session, and
	// the page it has says whose: asking again from here would cost the user
	// the seconds of a second download of the same page.
	var seen struct {
		Host string `json:"host"`
		Page string `json:"page"`
	}
	err = page.Eval(`({host: location.host, page: document.documentElement.outerHTML})`, &seen)
	page.Close()
	if err != nil {
		return "", err
	}
	if seen.Host != takeoutHost {
		return "", ErrSessionNotAccepted
	}
	return takeout.AccountIn([]byte(seen.Page)), nil
}

// cookiesLanded waits for the session cookies to reach the profile after the
// browser has shut down. Its main process can exit before they are on disk:
// on 2026-10-01 a read right after the exit found none, and the file was
// written 170 ms later.
func cookiesLanded(sess *session.Session) bool {
	for range 50 {
		if onDisk(sess) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// signedIn reports whether the profile holds a session Google accepts.
func signedIn(sess *session.Session) (bool, error) {
	_, names, err := sess.Cookies()
	if err != nil || !names["SID"] || !names["SSID"] {
		return false, nil
	}
	return takeout.SignedIn(sess)
}
