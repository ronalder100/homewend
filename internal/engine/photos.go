// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"time"

	"github.com/ronalder100/homewend/internal/library"
	"github.com/ronalder100/homewend/internal/thumb"
)

// GridPhoto is one tile of the window's grid.
type GridPhoto struct {
	Hash  string    `json:"hash"`
	Name  string    `json:"name"`
	Taken time.Time `json:"taken,omitzero"`
	Video bool      `json:"video,omitempty"`
}

// ErrNoLibrary is asking for photos before a library folder was chosen.
var ErrNoLibrary = errors.New("no library folder chosen yet")

// ChosenLibrary is the library folder, or ErrNoLibrary before one is chosen.
func ChosenLibrary() (string, error) {
	root, err := DefaultLibrary()
	if err == nil && root == "" {
		err = ErrNoLibrary
	}
	return root, err
}

func openLibrary() (*library.Catalog, string, error) {
	root, err := ChosenLibrary()
	if err != nil {
		return nil, "", err
	}
	if _, err := os.Stat(catalogPath(root)); errors.Is(err, fs.ErrNotExist) {
		return nil, root, nil
	}
	c, err := library.OpenCatalog(catalogPath(root))
	return c, root, err
}

// Photos is a page of the grid, newest first: all of it, a year ("" for no
// date) or an album.
// Only the photos of the accounts shown are in it.
func Photos(s Settings, year, album string, noDate bool, offset, limit int) ([]GridPhoto, error) {
	c, _, err := openLibrary()
	if err != nil || c == nil {
		return []GridPhoto{}, err
	}
	defer c.Close()
	accounts, err := Accounts()
	if err != nil {
		return nil, err
	}
	page, err := c.Page(library.Filter{Year: year, Album: album, NoDate: noDate, Offset: offset, Limit: limit,
		Accounts: shownAddresses(accounts, s.Hidden)})
	if err != nil {
		return nil, err
	}
	out := make([]GridPhoto, len(page))
	for i, p := range page {
		out[i] = GridPhoto{Hash: p.Hash, Name: p.Name, Taken: p.Taken, Video: isVideo(p.Name)}
	}
	return out, nil
}

func isVideo(name string) bool {
	switch filepath.Ext(name) {
	case ".mp4", ".MP4", ".mov", ".MOV", ".m4v", ".M4V", ".3gp", ".avi", ".mkv", ".webm":
		return true
	}
	return false
}

// thumbSlots keeps thumbnails from taking every core while a grid scrolls.
var thumbSlots = make(chan struct{}, max(1, runtime.NumCPU()-1))

// Thumbnail returns the file of a photo's thumbnail, making it the first time.
func Thumbnail(ctx context.Context, hash string) (string, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dst := thumb.Path(filepath.Join(cache, "homewend", "thumbs"), hash)
	if _, err := os.Stat(dst); err == nil {
		return dst, nil
	}
	c, root, err := openLibrary()
	if err != nil {
		return "", err
	}
	if c == nil {
		return "", fs.ErrNotExist
	}
	rel, ok, err := c.PathOf(hash)
	c.Close()
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fs.ErrNotExist
	}
	select {
	case thumbSlots <- struct{}{}:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	defer func() { <-thumbSlots }()
	if _, err := os.Stat(dst); err == nil { // made while this one waited
		return dst, nil
	}
	return dst, thumb.Make(ctx, filepath.Join(root, rel), dst)
}

// Original is the file of a photo in the library.
func Original(hash string) (string, error) {
	c, root, err := openLibrary()
	if err != nil {
		return "", err
	}
	if c == nil {
		return "", fs.ErrNotExist
	}
	defer c.Close()
	rel, ok, err := c.PathOf(hash)
	if err != nil || !ok {
		return "", errors.Join(err, fs.ErrNotExist)
	}
	return filepath.Join(root, rel), nil
}

// Folder is where a place of the library is on disk: a profile, a year, the
// photos with no date, or an album, as Organize files them. The profile is
// the account's, when one is named, else the one profile the window shows;
// with several shown and none named, a place is in each, and the library
// itself is the folder.
func Folder(s Settings, account, year, album string, noDate bool) (string, error) {
	root, err := ChosenLibrary()
	if err != nil {
		return "", err
	}
	accounts, err := Accounts()
	if err != nil {
		return "", err
	}
	return folder(root, accounts, s.Hidden, account, year, album, noDate), nil
}

func folder(root string, accounts []AccountInfo, hidden []string, account, year, album string, noDate bool) string {
	var profiles []string
	for _, a := range accounts {
		named := account != "" && a.ID == account
		shown := account == "" && a.Email != "" && !slices.Contains(hidden, a.ID)
		if (named || shown) && !slices.Contains(profiles, a.Profile) {
			profiles = append(profiles, a.Profile)
		}
	}
	if len(profiles) != 1 {
		return root
	}
	home := filepath.Join(root, profiles[0])
	switch {
	case album != "":
		return filepath.Join(home, "albums", album)
	case noDate:
		return filepath.Join(home, "undated")
	case year != "":
		return filepath.Join(home, year)
	}
	return home
}

// ProfileFolder is the folder of a profile in the library.
func ProfileFolder(profile string) (string, error) {
	if !validProfile(profile) {
		return "", ErrBadProfile
	}
	root, err := ChosenLibrary()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, profile), nil
}

// Latest are the n photos that arrived last.
func Latest(n int) ([]GridPhoto, error) {
	c, _, err := openLibrary()
	if err != nil || c == nil {
		return []GridPhoto{}, err
	}
	defer c.Close()
	list, err := c.Latest(n)
	out := make([]GridPhoto, len(list))
	for i, p := range list {
		out[i] = GridPhoto{Hash: p.Hash, Name: p.Name, Video: isVideo(p.Name)}
	}
	return out, err
}

// PhotoAbout is what the viewer says beside a photo's name.
type PhotoAbout struct {
	Albums   []string `json:"albums"`
	Accounts []string `json:"accounts"`
}

// About is a photo's albums and the accounts it came from.
func About(hash string) (PhotoAbout, error) {
	c, _, err := openLibrary()
	if err != nil || c == nil {
		return PhotoAbout{Albums: []string{}, Accounts: []string{}}, err
	}
	defer c.Close()
	albums, accounts, err := c.About(hash)
	return PhotoAbout{Albums: albums, Accounts: accounts}, err
}
