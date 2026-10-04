// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/ronalder100/homewend/internal/download"
	"github.com/ronalder100/homewend/internal/library"
	"github.com/ronalder100/homewend/internal/session"
	"github.com/ronalder100/homewend/internal/takeout"
)

// Google does not say what a takeout holds. Its manifest does: one small
// archive, the first a download fetches anyway. A takeout made on Google can
// be read that way without downloading its photos.

// ErrNoUserYet is reading a takeout before the account's first download:
// the address of its files needs the user id that download gives.
var ErrNoUserYet = errors.New("the account has not downloaded anything yet")

// manifestYears are the years a takeout's manifest lists, when it has been
// fetched into the library, oldest first.
func manifestYears(root, job string) ([]string, bool) {
	work := filepath.Join(root, library.WorkDir, job)
	st, err := loadState(work)
	if err != nil || st.Manifest == "" {
		return nil, false
	}
	m, err := library.ParseManifestZip(filepath.Join(work, st.Manifest))
	if err != nil {
		return nil, false
	}
	_, years := m.ByYear()
	return years, true
}

// ReadContents fetches the manifest of the export whose id starts with id
// into the library, where a later download finds it, and says its years.
func ReadContents(ctx context.Context, sess *session.Session, root, id string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(sess.Profile, userFile))
	user := strings.TrimSpace(string(data))
	if err != nil || user == "" {
		return nil, ErrNoUserYet
	}
	exports, err := takeout.Exports(sess)
	if err != nil {
		return nil, err
	}
	e, err := byID(exports, id)
	if err != nil {
		return nil, err
	}
	if !e.Ready() {
		return nil, ErrExpired
	}
	if years, ok := manifestYears(root, e.Job); ok {
		return years, nil
	}
	work := filepath.Join(root, library.WorkDir, e.Job)
	if err := os.MkdirAll(work, 0o755); err != nil {
		return nil, err
	}
	target := takeout.Target{Job: e.Job, User: user}
	if err := download.Part(ctx, sess, target, e.Manifest, 1, len(e.Parts)+1, work, nil); err != nil {
		return nil, err
	}
	st, err := loadState(work)
	if err != nil {
		return nil, err
	}
	st.Manifest = e.Manifest.Filename
	if err := st.save(work); err != nil {
		return nil, err
	}
	years, _ := manifestYears(root, e.Job)
	return years, nil
}
