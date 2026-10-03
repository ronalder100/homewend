// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
// It is kept once known. The first time, it is read from a download made in
// this profile; if there is none, the app starts one itself, of the export's
// smallest file, by the address Google's own page links to. Minutes after a
// sign-in Google lets it through, and it all happens out of sight. Later
// Google asks for the password first, and only then does a window open, on
// Google's password page: typing it is all the user does. The download
// follows by itself, into a directory of ours, and the window closes; the
// Takeout page Google passes through on the way is never on screen.
func User(ctx context.Context, sess *session.Session, export takeout.Export, emit progress.Func) (string, error) {
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
		emit.Emit(progress.Event{Stage: progress.Prepare, Name: Account(sess)})
		// What the browser downloads is ours to throw away, not the user's to
		// find among their downloads.
		landing, err := os.MkdirTemp("", "homewend-download-*")
		if err != nil {
			return "", err
		}
		defer os.RemoveAll(landing)
		link, from, err := quietly(ctx, sess, export, landing)
		if err != nil {
			return "", err
		}
		// Out of sight it worked: Google had no question, and nobody saw a thing.
		if from == "" {
			// Google wants the password, and only the user has it.
			emit.Emit(progress.Event{Stage: progress.FirstDownload, Name: Account(sess)})
			if from, err = withPassword(ctx, sess, link, landing); err != nil {
				return "", err
			}
		}
		if id = takeout.UserFromDownload(from); id == "" {
			return "", fmt.Errorf("the download came from an address with no user id: %s", from)
		}
	}
	return id, os.WriteFile(path, []byte(id+"\n"), 0o600)
}

// ErrNoLink means the export's page has no download address on it.
var ErrNoLink = errors.New("the export's page has no download link")

// quietWait is how long the download is given to begin out of sight: it
// began 3.7 seconds after its address was followed (2026-10-02).
const quietWait = 15 * time.Second

// quietly reads, off the export's own page, the address Google gives for
// downloading its smallest file, the manifest's archive, and follows it out
// of sight. It returns that address, and the one the download came from, or
// "" when Google asked for the password instead. The address read carries the
// account's long id, which is on that page and nowhere the app can build it
// from.
func quietly(ctx context.Context, sess *session.Session, export takeout.Export, landing string) (link, from string, err error) {
	page, err := sess.Headless(ctx, takeout.ArchiveURL(export.Job))
	if err != nil {
		return "", "", err
	}
	defer page.Close()
	var links []string
	if err := page.Eval(`[...document.querySelectorAll('a[href]')].map(a => a.href)`, &links); err != nil {
		return "", "", err
	}
	if link = pickLink(links, export.Manifest.Index); link == "" {
		return "", "", ErrNoLink
	}
	from, err = page.Download(link, landing, time.After(quietWait))
	return link, from, err
}

// withPassword follows link in a window the user sees, for the password
// Google asks for, and returns the address the download came from. The window
// opens blank and goes to Google's password page; once the password is typed
// Google goes through the export's Takeout page, which starts the download
// three seconds later, and is kept off screen. The window is closed as soon as
// the file is down: the manifest's archive, small (see Page.Download).
func withPassword(ctx context.Context, sess *session.Session, link, landing string) (string, error) {
	page, err := sess.Watched(ctx, takeoutHost)
	if err != nil {
		return "", err
	}
	defer page.Close()
	from, err := page.Download(link, landing, nil)
	if errors.Is(err, session.ErrExited) {
		return "", ErrNoDownload
	}
	return from, err
}

// pickLink finds, among a page's addresses, the download of the file with
// this index; failing that, of any file of the export.
func pickLink(links []string, index int) string {
	other := ""
	for _, link := range links {
		u, err := url.Parse(link)
		if err != nil || u.Host != takeoutHost || u.Path != "/takeout/download" {
			continue
		}
		if u.Query().Get("i") == strconv.Itoa(index) {
			return link
		}
		if other == "" {
			other = link
		}
	}
	return other
}
