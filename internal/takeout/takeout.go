// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package takeout reads /manage, the one Google page this program reads, and
// builds the addresses of the archives an export is made of.
//
// Reading /manage is what makes Google's "your data is ready" email redundant:
// the app knows before the mail does, because it is watching. It fetches a
// status page, at the rate of a person refreshing a tab.
//
// One thing here fills a Google page: the form that asks for an export
// (create.go), pressed as a person would, through the page's own controls.
// Everything else only reads.
package takeout

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	// Opens Takeout with Google Photos as the only thing selected, which
	// removes the worst step of the process: "67 of 67 selected, deselect all,
	// then find Photos among sixty-seven".
	PhotosURL = "https://takeout.google.com/settings/takeout/custom/photos"

	ManageURL = "https://takeout.google.com/manage"

	// The host that serves the bytes. Reaching it directly matters twice:
	// it does not ask for the `rapt` re-auth token, and the "downloaded N of 5
	// times" counter lives on the takeout.google.com redirect rather than here.
	// Going through that redirect spends one of the five even when no byte
	// arrives; building this address spends none. Measured 2026-09-23.
	downloadHost = "takeout-download.usercontent.google.com"
)

// ArchiveURL is the page of one export, where its download buttons are.
func ArchiveURL(job string) string {
	return "https://takeout.google.com/manage/archive/" + job
}

// Getter is the part of a session this package needs: an authenticated GET.
type Getter interface {
	Get(url string, extra map[string]string) (*http.Response, error)
}

// Target identifies one export on one account.
type Target struct {
	Job string // the j= parameter, a UUID
	// The user= parameter of the download host: a short numeric id, and NOT
	// the long one that /manage and Google's own download links carry, which
	// the host answers with 400. See UserFromDownload.
	User string
}

// Part is one downloadable piece of an export.
type Part struct {
	Index    int    // the i= parameter, zero-based
	Filename string // as Google lists it, never constructed
	Size     int64  // bytes, as Google lists it
	// How many of the five allowed downloads Google has already counted.
	// Informational: we do not go through the endpoint that increments it.
	Downloads int
}

// URL is where this part's bytes actually live.
func (p Part) URL(t Target) string {
	return fmt.Sprintf("https://%s/download/%s?j=%s&i=%d&user=%s&authuser=0",
		downloadHost, p.Filename, t.Job, p.Index, t.User)
}

// Export is one export as /manage describes it.
type Export struct {
	Job     string
	Created time.Time
	Bytes   int64 // declared size, manifest excluded
	// Empty until the export is ready to download, and again once it expired.
	Parts []Part
	// archive_browser.html's archive, one index past the last part.
	Manifest Part
	// When Google stops offering it, or stopped; zero while it is prepared.
	Expires time.Time
	// Google no longer offers it. Being prepared looks the same on /manage,
	// with no parts: only this tells the two apart.
	Expired bool
}

// Ready reports whether the export can be downloaded now.
func (e Export) Ready() bool { return len(e.Parts) > 0 && e.Manifest.Filename != "" }

// The fields of an export record, by position. /manage is a JavaScript
// application, but its data is in the HTML, inside the AF_initDataCallback
// blocks Google uses to seed the page, and each export is an array that starts
// with the marker "ac.t.ta". Read on two exports, 2026-09-26, one live and one
// expired:
//
//	[1]  job id
//	[6]  declared bytes
//	[8]  parts, each [filename, bytes, downloads so far, ...]; null once expired
//	[22] created, milliseconds since the epoch (matches the filenames' timestamp)
//	[24] when it expires, while it can be downloaded; null once expired
//	[25] when it expired, once it has; null before
//	[27] the manifest's archive, shaped like a part; null once expired
//
// [24] and [25] read on 2026-10-01, fourteen exports: twelve live, each
// expiring exactly seven days after field [23], and two expired.
//
// The marker matters: a bare UUID picked at random also matches things that
// are not exports (the same page carries one that belongs to YouTube).
const (
	recordMarker   = `["ac.t.ta"`
	fieldJob       = 1
	fieldBytes     = 6
	fieldParts     = 8
	fieldCreated   = 22
	fieldExpires   = 24
	fieldExpired   = 25
	fieldManifest  = 27
	recordMinWidth = fieldManifest + 1
)

// SignedIn reports whether Google accepts the session: /manage answers one it
// does not accept by sending it to sign in. Having the cookies proves nothing —
// a revoked session leaves them in the profile, and /manage then redirects to
// accounts.google.com/ServiceLogin. Measured 2026-09-26.
//
// Any other answer is an error, not a "no": it says nothing about the session.
func SignedIn(g Getter) (bool, error) {
	res, err := g.Get(ManageURL, nil)
	if err != nil {
		return false, fmt.Errorf("checking the session: %w", err)
	}
	res.Body.Close()
	switch {
	case res.StatusCode == http.StatusOK:
		return true, nil
	case res.StatusCode >= 300 && res.StatusCode < 400:
		if to, err := url.Parse(res.Header.Get("Location")); err == nil && to.Host == "accounts.google.com" {
			return false, nil
		}
	}
	return false, fmt.Errorf("checking the session: HTTP %d", res.StatusCode)
}

// Exports lists the exports /manage shows, newest first.
//
// An error is never an empty list: a failed read must never look like "there
// are no exports". One means sign in again, the other means keep waiting, and
// confusing them is how a watcher gives up on a job that was about to be ready.
func Exports(g Getter) ([]Export, error) {
	res, err := g.Get(ManageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("reading the export list: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reading the export list: HTTP %d", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading the export list: %w", err)
	}
	return parseExports(string(body))
}

func parseExports(page string) ([]Export, error) {
	seen := map[string]bool{}
	var exports []Export
	for rest := page; ; {
		at := strings.Index(rest, recordMarker)
		if at < 0 {
			break
		}
		rest = rest[at:]
		// The record is a JSON array; the decoder stops where it ends.
		var record []json.RawMessage
		if err := json.NewDecoder(strings.NewReader(rest)).Decode(&record); err != nil {
			return nil, fmt.Errorf("reading an export record: %w", err)
		}
		rest = rest[len(recordMarker):]

		e, err := parseRecord(record)
		if err != nil {
			return nil, err
		}
		// The page repeats each record several times.
		if !seen[e.Job] {
			seen[e.Job] = true
			exports = append(exports, e)
		}
	}
	sort.Slice(exports, func(i, j int) bool { return exports[i].Created.After(exports[j].Created) })
	return exports, nil
}

func parseRecord(record []json.RawMessage) (Export, error) {
	if len(record) < recordMinWidth {
		return Export{}, fmt.Errorf("an export record has %d fields, expected at least %d", len(record), recordMinWidth)
	}
	var e Export
	var created int64
	for _, f := range []struct {
		index int
		into  any
	}{{fieldJob, &e.Job}, {fieldBytes, &e.Bytes}, {fieldCreated, &created}} {
		if err := json.Unmarshal(record[f.index], f.into); err != nil {
			return Export{}, fmt.Errorf("export record field %d: %w", f.index, err)
		}
	}
	if e.Job == "" {
		return Export{}, errors.New("an export record has no id")
	}
	e.Created = time.UnixMilli(created)

	var expires, expired *int64
	if err := json.Unmarshal(record[fieldExpires], &expires); err != nil {
		return Export{}, fmt.Errorf("export %s, expiry: %w", e.Job, err)
	}
	if err := json.Unmarshal(record[fieldExpired], &expired); err != nil {
		return Export{}, fmt.Errorf("export %s, expired: %w", e.Job, err)
	}
	switch {
	case expired != nil:
		e.Expires, e.Expired = time.UnixMilli(*expired), true
	case expires != nil:
		e.Expires = time.UnixMilli(*expires)
	}

	var parts [][]json.RawMessage
	if err := json.Unmarshal(record[fieldParts], &parts); err != nil {
		return Export{}, fmt.Errorf("export %s, parts: %w", e.Job, err)
	}
	for i, raw := range parts {
		p, err := parsePart(raw, i)
		if err != nil {
			return Export{}, fmt.Errorf("export %s, part %d: %w", e.Job, i, err)
		}
		e.Parts = append(e.Parts, p)
	}

	var manifest []json.RawMessage
	if err := json.Unmarshal(record[fieldManifest], &manifest); err != nil {
		return Export{}, fmt.Errorf("export %s, manifest: %w", e.Job, err)
	}
	if manifest != nil {
		// One index past the last part: parts i=0…158 and the manifest at 159
		// on the large export, i=0 and 1 on the small one, where i=2 answers
		// 500. Measured 2026-09-23.
		m, err := parsePart(manifest, len(e.Parts))
		if err != nil {
			return Export{}, fmt.Errorf("export %s, manifest: %w", e.Job, err)
		}
		e.Manifest = m
	}
	return e, nil
}

func parsePart(raw []json.RawMessage, index int) (Part, error) {
	p := Part{Index: index}
	if len(raw) < 3 {
		return p, fmt.Errorf("%d fields, expected at least 3", len(raw))
	}
	for i, into := range []any{&p.Filename, &p.Size, &p.Downloads} {
		if err := json.Unmarshal(raw[i], into); err != nil {
			return p, err
		}
	}
	if p.Filename == "" || strings.ContainsAny(p.Filename, `/\`) {
		return p, fmt.Errorf("unusable filename %q", p.Filename)
	}
	return p, nil
}

// UserFromDownload reads the short user id out of the address the download
// host was reached at, or returns "" if the address is not one.
//
// That short id appears on no page and in no cookie. Google's own download
// link carries the long id, and the short one only shows up at the end of its
// redirect, which asks for the password again. So it is read from a download
// the user made in the browser, once, and kept: see session.DownloadURLs.
func UserFromDownload(address string) string {
	u, err := url.Parse(address)
	if err != nil || u.Host != downloadHost {
		return ""
	}
	return u.Query().Get("user")
}
