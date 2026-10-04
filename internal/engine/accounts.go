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

// An account is the session of one service: for Google, a browser profile
// that holds its cookies. Accounts live in the user's config directory, one
// folder per service and address: accounts/google/<email>. The profile an
// account belongs to is whose photos they are, a folder in the library; it is
// kept in the settings (ProfileFor).
type AccountInfo struct {
	// ID is "<service>/<email>", or "<service>/new-<n>" for a sign-in that
	// has not finished: what the shells name an account by.
	ID      string `json:"id"`
	Service string `json:"service"`
	Email   string `json:"email"` // empty until the sign-in finishes
	Profile string `json:"profile"`
	Dir     string `json:"-"`
}

// Google is the only service so far.
const Google = "google"

// configDir is homewend's folder in the user's config directory.
func configDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "homewend"), nil
}

func accountsDir() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "accounts"), nil
}

// Accounts lists the accounts, signed in or not, by service and address. The
// folders an older homewend kept are moved into place first.
func Accounts() ([]AccountInfo, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}
	if err := moveOldAccounts(dir); err != nil {
		return nil, err
	}
	s, err := LoadSettings()
	if err != nil {
		return nil, err
	}
	return accountsIn(filepath.Join(dir, "accounts"), s)
}

func accountsIn(dir string, s Settings) ([]AccountInfo, error) {
	var list []AccountInfo
	services, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	for _, svc := range services {
		if !svc.IsDir() {
			continue
		}
		entries, err := os.ReadDir(filepath.Join(dir, svc.Name()))
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			id := svc.Name() + "/" + e.Name()
			if !e.IsDir() || !accountID.MatchString(id) {
				continue
			}
			a := AccountInfo{ID: id, Service: svc.Name(), Dir: filepath.Join(dir, svc.Name(), e.Name())}
			if strings.Contains(e.Name(), "@") {
				a.Email = e.Name()
				a.Profile = s.ProfileFor(id)
			}
			list = append(list, a)
		}
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	return list, nil
}

// accountID is "<service>/<email>" or "<service>/new-<n>": nothing that could
// leave the accounts folder.
var accountID = regexp.MustCompile(`^[a-z]+/([^/\\]+@[^/\\]+|new-[0-9]+)$`)

// AccountDir is the folder of the account id.
func AccountDir(id string) (string, error) {
	if !accountID.MatchString(id) || strings.Contains(id, "..") {
		return "", ErrNoSuchAccount
	}
	dir, err := accountsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, filepath.FromSlash(id)), nil
}

// AccountSession opens the session of the account id.
func AccountSession(id string) (*session.Session, error) {
	dir, err := AccountDir(id)
	if err != nil {
		return nil, err
	}
	return session.New(dir)
}

// ErrNoSuchAccount is an id that names no account.
var ErrNoSuchAccount = errors.New("no such account")

// NextAccount is where a new Google account signs in: a folder with no
// address yet, new-1, new-2…, which Adopt names once Google says whose it is.
func NextAccount() (string, error) {
	dir, err := accountsDir()
	if err != nil {
		return "", err
	}
	for n := 1; ; n++ {
		id := Google + "/new-" + strconv.Itoa(n)
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(id))); errors.Is(err, os.ErrNotExist) {
			return id, nil
		}
	}
}

// Adopt names the account id after the address it signed in to: a new
// sign-in's folder becomes accounts/<service>/<email>. The browser must be
// closed. An id that already has its name is returned as it is; an address
// signed in twice keeps the newer session.
func Adopt(id string) (AccountInfo, error) {
	dir, err := AccountDir(id)
	if err != nil {
		return AccountInfo{}, err
	}
	sess, err := session.New(dir)
	if err != nil {
		return AccountInfo{}, err
	}
	email := Account(sess)
	service, _, _ := strings.Cut(id, "/")
	if email == "" || strings.HasSuffix(id, "/"+email) {
		return AccountInfo{ID: id, Service: service, Email: email, Dir: dir}, nil
	}
	named := service + "/" + email
	to := filepath.Join(filepath.Dir(dir), email)
	if err := os.RemoveAll(to); err != nil {
		return AccountInfo{}, err
	}
	if err := os.Rename(dir, to); err != nil {
		return AccountInfo{}, err
	}
	s, err := LoadSettings()
	if err != nil {
		return AccountInfo{}, err
	}
	return AccountInfo{ID: named, Service: service, Email: email, Profile: s.ProfileFor(named), Dir: to}, nil
}

// oldAccount is a folder an older homewend kept a Google session in:
// "profile", "profile-<x>", or "accounts/<n>".
var oldAccount = regexp.MustCompile(`^(profile(-[^/\\]+)?|accounts/[0-9]+)$`)

// moveOldAccounts moves the sessions an older homewend kept into
// accounts/google/<email>. A folder whose address is not known is a sign-in
// that never finished, and is left where it is.
func moveOldAccounts(dir string) error {
	var old []string
	for _, parent := range []string{"", "accounts"} {
		entries, err := os.ReadDir(filepath.Join(dir, parent))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		for _, e := range entries {
			name := filepath.ToSlash(filepath.Join(parent, e.Name()))
			if e.IsDir() && oldAccount.MatchString(name) {
				old = append(old, filepath.Join(dir, parent, e.Name()))
			}
		}
	}
	for _, from := range old {
		email := addressOf(from)
		if email == "" {
			continue
		}
		to := filepath.Join(dir, "accounts", Google, email)
		if _, err := os.Stat(to); err == nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(to), 0o700); err != nil {
			return err
		}
		if err := os.Rename(from, to); err != nil {
			return err
		}
	}
	return nil
}

// addressOf is the address of the account a folder is signed in to: the one
// Login kept, or Google's answer, asked only when the folder holds Google's
// session cookies, so that a signed-out folder costs no request.
func addressOf(dir string) string {
	if kept, err := os.ReadFile(filepath.Join(dir, accountFileName)); err == nil {
		return strings.TrimSpace(string(kept))
	}
	sess, err := session.New(dir)
	if err != nil {
		return ""
	}
	if _, names, err := sess.Cookies(); err != nil || !names["SID"] || !names["SSID"] {
		return ""
	}
	return Account(sess)
}
