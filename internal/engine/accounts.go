// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ronalder100/homewend/internal/session"
)

// An account is one Google account and the browser profile that holds its
// session: a Google session belongs to a browser profile. The profiles live
// side by side in the user's config directory, as login --profile makes
// them: "profile", the default, and "profile-<anything>" beside it.
type AccountInfo struct {
	// ID is the profile's folder name.
	ID      string `json:"id"`
	Email   string `json:"email"` // empty while signed out
	Profile string `json:"-"`
}

var profileName = regexp.MustCompile(`^profile(-[^/\\]+)?$`)

// Accounts lists the profiles in the config directory, the default first.
func Accounts() ([]AccountInfo, error) {
	first, err := DefaultProfile()
	if err != nil {
		return nil, err
	}
	return accountsIn(filepath.Dir(first))
}

func accountsIn(dir string) ([]AccountInfo, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var list []AccountInfo
	for _, e := range entries {
		if e.IsDir() && profileName.MatchString(e.Name()) {
			list = append(list, AccountInfo{ID: e.Name(), Profile: filepath.Join(dir, e.Name())})
		}
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	for i := range list {
		list[i].Email = addressOf(list[i].Profile)
	}
	return list, nil
}

// addressOf is the address of the account a profile is signed in to: the one
// Login kept, or Google's answer, asked only when the profile holds Google's
// session cookies, so that a signed-out profile costs no request.
func addressOf(profile string) string {
	if kept, err := os.ReadFile(filepath.Join(profile, accountFileName)); err == nil {
		return strings.TrimSpace(string(kept))
	}
	sess, err := session.New(profile)
	if err != nil {
		return ""
	}
	if _, names, err := sess.Cookies(); err != nil || !names["SID"] || !names["SSID"] {
		return ""
	}
	return Account(sess)
}

// AccountSession opens the session of the account whose profile is id.
func AccountSession(id string) (*session.Session, error) {
	if !profileName.MatchString(id) {
		return nil, ErrNoSuchAccount
	}
	first, err := DefaultProfile()
	if err != nil {
		return nil, err
	}
	return session.New(filepath.Join(filepath.Dir(first), id))
}

// ErrNoSuchAccount is an id that names no profile.
var ErrNoSuchAccount = errors.New("no such account")

// NextAccount is the profile a new account signs in to: the default while it
// is free, then profile-2, profile-3…
func NextAccount() (string, error) {
	list, err := Accounts()
	if err != nil {
		return "", err
	}
	taken := map[string]bool{}
	for _, a := range list {
		if a.Email != "" {
			taken[a.ID] = true
		}
	}
	if !taken["profile"] {
		return "profile", nil
	}
	for n := 2; ; n++ {
		if id := "profile-" + strconv.Itoa(n); !taken[id] {
			return id, nil
		}
	}
}
