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
// and albums of those shown, summed.
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
	Shown bool `json:"shown"`
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
	accounts, err := Accounts()
	if err != nil {
		return Overview{}, err
	}
	return overview(root, accounts, s.Hidden)
}

func overview(root string, accounts []AccountInfo, hidden []string) (Overview, error) {
	o := Overview{Years: []YearCount{}, Albums: []AlbumCount{}, Accounts: []ShownAccount{}}
	// No library chosen yet: the accounts are there, their photos are not.
	years := map[string]int{}
	for _, a := range accounts {
		// A profile that never finished signing in is nobody yet.
		if a.Email == "" {
			continue
		}
		shown := !slices.Contains(hidden, a.ID)
		o.Accounts = append(o.Accounts, ShownAccount{AccountInfo: a, Shown: shown})
		if notes, err := loadNotes(a.Profile); err == nil {
			o.Takeouts += len(notes)
		}
		if !shown || root == "" {
			continue
		}
		path := catalogPath(LibraryOf(root, a.Email))
		if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
			continue
		}
		c, err := library.OpenCatalog(path)
		if err != nil {
			return o, err
		}
		byYear, err := c.CountsByYear()
		if err == nil {
			var albums map[string]int
			albums, err = c.AlbumNames()
			for name, n := range albums {
				o.Albums = append(o.Albums, AlbumCount{Account: a.ID, Name: name, Count: n})
			}
		}
		c.Close()
		if err != nil {
			return o, err
		}
		for y, n := range byYear {
			years[y] += n
			o.Total += n
		}
	}
	for y, n := range years {
		o.Years = append(o.Years, YearCount{Year: y, Count: n})
	}
	// Newest first; the photos with no date last.
	sort.Slice(o.Years, func(i, j int) bool {
		a, b := o.Years[i].Year, o.Years[j].Year
		if a == "" || b == "" {
			return b == ""
		}
		return a > b
	})
	sort.SliceStable(o.Albums, func(i, j int) bool { return o.Albums[i].Name < o.Albums[j].Name })
	return o, nil
}
