// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/ronalder100/homewend/internal/progress"
	"github.com/ronalder100/homewend/internal/session"
	"github.com/ronalder100/homewend/internal/takeout"
)

// ErrNoDownload means the browser was closed before a download started, so
// the short user id is still unknown.
var ErrNoDownload = errors.New("the browser was closed before a download started")

// userFile keeps the short user id beside the profile it belongs to. It is per
// account and has not changed between two exports; Chrome's own history of
// downloads, where it is found, is not kept forever.
const userFile = "homewend-user-id"

// User returns the account's short user id, the one the download host wants.
//
// It is kept once known. The first time, it is read from a download the user
// made in this profile; if there is none, the browser opens on the export
// with job and waits for the user to download one part — which is when Google
// asks for the password again, and why this is theirs to do.
func User(ctx context.Context, sess *session.Session, job string, emit progress.Func) (string, error) {
	path := filepath.Join(sess.Profile, userFile)
	if data, err := os.ReadFile(path); err == nil {
		if id := strings.TrimSpace(string(data)); id != "" {
			return id, nil
		}
	}

	var id string
	found := func() bool {
		urls, err := sess.DownloadURLs()
		if err != nil {
			return false
		}
		for _, u := range urls {
			if id = takeout.UserFromDownload(u); id != "" {
				return true
			}
		}
		return false
	}
	if !found() {
		err := inWindow(ctx, sess, takeout.ArchiveURL(job), progress.FirstDownload, found, emit)
		if errors.Is(err, errWindowClosed) {
			return "", ErrNoDownload
		}
		if err != nil {
			return "", err
		}
	}
	return id, os.WriteFile(path, []byte(id+"\n"), 0o600)
}
