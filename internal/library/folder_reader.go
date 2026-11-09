// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// The files a library takes in. Anything else in a folder — a camera's
// firmware, its database, a project's code — stays where it is and is said.
var (
	photoExtensions = map[string]bool{
		".jpg": true, ".jpeg": true, ".heic": true, ".heif": true, ".png": true, ".gif": true,
		".webp": true, ".tif": true, ".tiff": true, ".bmp": true, ".avif": true,
		".dng": true, ".cr2": true, ".cr3": true, ".nef": true, ".arw": true, ".raf": true,
		".orf": true, ".rw2": true, ".pef": true, ".srw": true,
	}
	videoExtensions = map[string]bool{
		".mp4": true, ".mov": true, ".m4v": true, ".3gp": true, ".avi": true, ".mkv": true,
		".webm": true, ".mts": true, ".m2ts": true,
	}
)

// IsVideo is whether the file is a video, by its extension.
func IsVideo(path string) bool { return videoExtensions[strings.ToLower(filepath.Ext(path))] }

// isMedia is whether the file is a photo or a video, by its extension.
func isMedia(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return photoExtensions[ext] || videoExtensions[ext]
}

// systemNames are what an operating system writes on a disk for itself: a
// card read on a Mac carries .Spotlight-V100, .fseventsd and a "._" twin of
// every file (measured 09/11, 94 of 131 files on a camera's card). They are
// nobody's files, so they are not listed at all.
var systemNames = map[string]bool{
	"System Volume Information": true,
	"$RECYCLE.BIN":              true,
}

func isSystem(name string) bool { return strings.HasPrefix(name, ".") || systemNames[name] }

// ReadFolder lists the photos and videos under dir as items for Organize,
// and the other files, so that what stays out can be said.
//
// A folder's own conventions are not Takeout's: there is no sidecar, and no
// folder is an album unless the person says so.
func ReadFolder(dir string) (items []Item, others []string, err error) {
	err = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path != dir && isSystem(entry.Name()) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if !isMedia(path) {
			others = append(others, path)
			return nil
		}
		items = append(items, Item{Path: path, Capture: DateOf(path, filepath.Base(filepath.Dir(path)))})
		return nil
	})
	// A stable order: two runs plan the same thing.
	sort.Slice(items, func(i, j int) bool { return items[i].Path < items[j].Path })
	sort.Strings(others)
	return items, others, err
}
