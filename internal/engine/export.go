// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
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
// It is kept once known. The first time, it is read from a download made in
// this profile. If there is none, the app starts one itself: it opens the
// browser on the download address of the export's smallest file, the one
// Google's own page links to. Google asks for the password again before any
// download a program starts (a headless browser following that address ended
// on the password page 14 minutes after signing in, and the download counter
// did not move: 2026-10-02), and typing it is all the user does. The download
// then starts by itself, into a directory of ours, and the id is in its
// address.
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
		link, err := downloadLink(ctx, sess, export)
		if err != nil {
			return "", err
		}
		// What the browser downloads is ours to throw away, not the user's to
		// find among their downloads.
		landing, err := os.MkdirTemp("", "homewend-download-*")
		if err != nil {
			return "", err
		}
		defer os.RemoveAll(landing)
		if err := sess.DownloadsTo(landing); err != nil {
			return "", err
		}
		defer sess.DownloadsTo("")
		err = inWindow(ctx, sess, link, progress.Event{Stage: progress.FirstDownload, Name: Account(sess)}, found, emit)
		if errors.Is(err, errWindowClosed) {
			return "", ErrNoDownload
		}
		if err != nil {
			return "", err
		}
	}
	return id, os.WriteFile(path, []byte(id+"\n"), 0o600)
}

// ErrNoLink means the export's page has no download address on it.
var ErrNoLink = errors.New("the export's page has no download link")

// downloadLink reads, off the export's own page, the address Google gives for
// downloading its smallest file: the manifest's archive. The address carries
// the account's long id, which is on that page and nowhere the app can build
// it from.
func downloadLink(ctx context.Context, sess *session.Session, export takeout.Export) (string, error) {
	page, err := sess.Headless(ctx, takeout.ArchiveURL(export.Job))
	if err != nil {
		return "", err
	}
	var links []string
	err = page.Eval(`[...document.querySelectorAll('a[href]')].map(a => a.href)`, &links)
	page.Close()
	if err != nil {
		return "", err
	}
	if link := pickLink(links, export.Manifest.Index); link != "" {
		return link, nil
	}
	return "", ErrNoLink
}

// pickLink finds, among a page's addresses, the download of the file with
// this index; failing that, of any file of the export.
func pickLink(links []string, index int) string {
	other := ""
	for _, link := range links {
		u, err := url.Parse(link)
		if err != nil || u.Host != "takeout.google.com" || u.Path != "/takeout/download" {
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
