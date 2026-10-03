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
	for _, a := range accounts {
		// A profile that is signed out is nobody yet.
		if a.Email == "" {
			continue
		}
		o.Accounts = append(o.Accounts, ShownAccount{AccountInfo: a, Shown: !slices.Contains(hidden, a.ID)})
		if notes, err := loadNotes(a.Profile); err == nil {
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
	years, err := c.CountsByYear()
	if err != nil {
		return o, err
	}
	albums, err := c.AlbumNames()
	if err != nil {
		return o, err
	}
	for y, n := range years {
		o.Years = append(o.Years, YearCount{Year: y, Count: n})
		o.Total += n
	}
	for name, n := range albums {
		o.Albums = append(o.Albums, AlbumCount{Name: name, Count: n})
	}
	// Newest first; the photos with no date last.
	sort.Slice(o.Years, func(i, j int) bool {
		a, b := o.Years[i].Year, o.Years[j].Year
		if a == "" || b == "" {
			return b == ""
		}
		return a > b
	})
	sort.Slice(o.Albums, func(i, j int) bool { return o.Albums[i].Name < o.Albums[j].Name })
	return o, nil
}
