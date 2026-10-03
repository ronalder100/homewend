// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/ronalder100/homewend/internal/session"
)

// An account is one Google account and the browser profile that holds its
// session. Each has its own profile, because a Google session belongs to a
// browser profile, and its own folder in the library, so that two people's
// photos never mix on disk.
type AccountInfo struct {
	// ID names the profile: "1" for the first, which lives where a
	// single-account homewend always kept it, then "2", "3"…
	ID      string `json:"id"`
	Email   string `json:"email"` // empty until signed in
	Profile string `json:"-"`
}

// accountsDir holds every profile after the first.
const accountsDir = "accounts"

// Accounts lists the profiles that exist, first to last. A profile signed out
// is listed with no address.
func Accounts() ([]AccountInfo, error) {
	first, err := DefaultProfile()
	if err != nil {
		return nil, err
	}
	return accountsIn(first)
}

func accountsIn(first string) ([]AccountInfo, error) {
	var list []AccountInfo
	if _, err := os.Stat(first); err == nil {
		list = append(list, AccountInfo{ID: "1", Profile: first})
	}
	entries, err := os.ReadDir(filepath.Join(filepath.Dir(first), accountsDir))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	var more []int
	for _, e := range entries {
		if n, err := strconv.Atoi(e.Name()); err == nil && n > 1 && e.IsDir() {
			more = append(more, n)
		}
	}
	sort.Ints(more)
	for _, n := range more {
		id := strconv.Itoa(n)
		list = append(list, AccountInfo{ID: id, Profile: profileOf(first, id)})
	}
	for i := range list {
		list[i].Email = keptAccount(list[i].Profile)
	}
	return list, nil
}

// profileOf is where account id keeps its browser profile.
func profileOf(first, id string) string {
	if id == "1" {
		return first
	}
	return filepath.Join(filepath.Dir(first), accountsDir, id)
}

// AccountSession opens the session of account id.
func AccountSession(id string) (*session.Session, error) {
	first, err := DefaultProfile()
	if err != nil {
		return nil, err
	}
	if _, err := strconv.Atoi(id); err != nil {
		return nil, ErrNoSuchAccount
	}
	return session.New(profileOf(first, id))
}

// ErrNoSuchAccount is an account id that names no profile.
var ErrNoSuchAccount = errors.New("no such account")

// NextAccount is the id a new account gets: the profile is made when it signs
// in.
func NextAccount() (string, error) {
	list, err := Accounts()
	if err != nil {
		return "", err
	}
	next := 1
	for _, a := range list {
		if n, _ := strconv.Atoi(a.ID); n >= next {
			next = n + 1
		}
	}
	return strconv.Itoa(next), nil
}

// keptAccount is the address Login kept beside the session, without asking
// Google: listing accounts must not cost a request per profile.
func keptAccount(profile string) string {
	kept, err := os.ReadFile(filepath.Join(profile, accountFileName))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(kept))
}

// LibraryOf is the folder of an account's photos inside the library root: the
// address before the @, which is how a person tells the folders apart.
func LibraryOf(root, email string) string {
	name, _, _ := strings.Cut(email, "@")
	return filepath.Join(root, name)
}
