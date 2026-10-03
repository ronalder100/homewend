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

func openLibrary() (*library.Catalog, string, error) {
	root, err := DefaultLibrary()
	if err != nil {
		return nil, "", err
	}
	if root == "" {
		return nil, "", ErrNoLibrary
	}
	if _, err := os.Stat(catalogPath(root)); errors.Is(err, fs.ErrNotExist) {
		return nil, root, nil
	}
	c, err := library.OpenCatalog(catalogPath(root))
	return c, root, err
}

// Photos is a page of the grid, newest first: all of it, a year ("" for no
// date) or an album.
func Photos(year, album string, noDate bool, offset, limit int) ([]GridPhoto, error) {
	c, _, err := openLibrary()
	if err != nil || c == nil {
		return []GridPhoto{}, err
	}
	defer c.Close()
	page, err := c.Page(library.Filter{Year: year, Album: album, NoDate: noDate, Offset: offset, Limit: limit})
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
