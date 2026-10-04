// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"github.com/ronalder100/homewend/internal/library"
)

// Overview is what the window's sidebar shows: the accounts, and the years
// and albums of the library folder chosen once (DefaultLibrary).
type Overview struct {
	Accounts []ShownAccount `json:"accounts"`
	Total    int            `json:"total"`
	Years    []YearCount    `json:"years"`
	Albums   []AlbumCount   `json:"albums"`
	Takeouts int            `json:"takeouts"`
}

// ShownAccount is an account and whether the window shows its photos.
type ShownAccount struct {
	AccountInfo
	Shown  bool `json:"shown"`
	Photos int  `json:"photos"` // in the library, brought by this account
}

// YearCount is a year, newest first, or "" for photos with no date.
type YearCount struct {
	Year  string `json:"year"`
	Count int    `json:"count"`
}

// AlbumCount is an album of one account.
type AlbumCount struct {
	Account string `json:"account"`
	Name    string `json:"name"`
	Count   int    `json:"count"`
}

// catalogPath is where an account's library keeps its catalog.
func catalogPath(dir string) string {
	return filepath.Join(dir, library.WorkDir, "catalog.db")
}

// LibraryOverview reads the catalog of every shown account. An account with no
// library yet counts nothing; that is not an error.
func LibraryOverview(s Settings) (Overview, error) {
	root, err := DefaultLibrary()
	if err != nil {
		return Overview{}, err
	}
	if err := settle(root); err != nil {
		return Overview{}, err
	}
	accounts, err := Accounts()
	if err != nil {
		return Overview{}, err
	}
	return overview(root, accounts, s.Hidden)
}

// shownAddresses are the addresses of the accounts shown, or nil when every
// account is: the catalog then needs no filter.
func shownAddresses(accounts []AccountInfo, hidden []string) []string {
	if len(hidden) == 0 {
		return nil
	}
	shown := []string{}
	for _, a := range accounts {
		if a.Email != "" && !slices.Contains(hidden, a.ID) {
			shown = append(shown, a.ID)
		}
	}
	return shown
}

func overview(root string, accounts []AccountInfo, hidden []string) (Overview, error) {
	o := Overview{Years: []YearCount{}, Albums: []AlbumCount{}, Accounts: []ShownAccount{}}
	idOf := map[string]string{}
	for _, a := range accounts {
		// A profile that is signed out is nobody yet.
		if a.Email == "" {
			continue
		}
		idOf[a.ID] = a.ID
		o.Accounts = append(o.Accounts, ShownAccount{AccountInfo: a, Shown: !slices.Contains(hidden, a.ID)})
		if notes, err := loadNotes(a.Dir); err == nil {
			o.Takeouts += len(notes)
		}
	}
	// No library chosen yet, or nothing in it yet: the accounts are there,
	// their photos are not.
	if root == "" {
		return o, nil
	}
	if _, err := os.Stat(catalogPath(root)); errors.Is(err, fs.ErrNotExist) {
		return o, nil
	}
	c, err := library.OpenCatalog(catalogPath(root))
	if err != nil {
		return o, err
	}
	defer c.Close()
	shown := shownAddresses(accounts, hidden)
	years, err := c.CountsByYear(shown)
	if err != nil {
		return o, err
	}
	albums, err := c.AlbumsByOwner()
	if err != nil {
		return o, err
	}
	owned, err := c.CountsByOwner()
	if err != nil {
		return o, err
	}
	for i := range o.Accounts {
		o.Accounts[i].Photos = owned[o.Accounts[i].ID]
	}
	for y, n := range years {
		o.Years = append(o.Years, YearCount{Year: y, Count: n})
		o.Total += n
	}
	for _, al := range albums {
		id := idOf[al.Account]
		if al.Account != "" && (id == "" || slices.Contains(hidden, id)) {
			continue
		}
		o.Albums = append(o.Albums, AlbumCount{Account: id, Name: al.Name, Count: al.Count})
	}
	// Newest first; the photos with no date last.
	sort.Slice(o.Years, func(i, j int) bool {
		a, b := o.Years[i].Year, o.Years[j].Year
		if a == "" || b == "" {
			return b == ""
		}
		return a > b
	})
	return o, nil
}
