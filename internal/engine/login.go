// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/ronalder100/homewend/internal/progress"
	"github.com/ronalder100/homewend/internal/session"
	"github.com/ronalder100/homewend/internal/takeout"
)

// ErrNotSignedIn means the browser was closed before sign-in completed.
var ErrNotSignedIn = errors.New("the browser was closed before sign-in completed")

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
// opens the browser on Takeout, which sends the user through Google's sign-in,
// waits for them, and closes it once they are in.
//
// It reports whether the profile was already signed in.
func Login(ctx context.Context, sess *session.Session, emit progress.Func) (already bool, err error) {
	if ok, err := signedIn(sess); ok || err != nil {
		return ok, err
	}
	err = inWindow(ctx, sess, takeout.PhotosURL, progress.SignIn, signInDone(sess), emit)
	if errors.Is(err, errWindowClosed) {
		return false, ErrNotSignedIn
	}
	return false, err
}

// signedIn reports whether the profile holds a session Google accepts.
func signedIn(sess *session.Session) (bool, error) {
	_, names, err := sess.Cookies()
	if err != nil || !names["SID"] || !names["SSID"] {
		return false, nil
	}
	return takeout.SignedIn(sess)
}

// signInDone is the check the sign-in window polls. It asks Google only when
// the cookies have changed since it last asked: the window looks every two
// seconds, a revoked session keeps its old cookies until the user signs in
// again, and a person does not refresh a page that often. A failed check
// counts as not yet, and is tried again when the cookies next change.
func signInDone(sess *session.Session) func() bool {
	var asked string
	return func() bool {
		cookies, names, err := sess.Cookies()
		if err != nil || !names["SID"] || !names["SSID"] || cookies == asked {
			return false
		}
		asked = cookies
		ok, _ := takeout.SignedIn(sess)
		return ok
	}
}
