// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
)

// Filed reports whether root holds photos filed before profiles: years,
// albums or undated straight under it.
func Filed(root string) bool {
	entries, err := os.ReadDir(root)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() && (yearDir.MatchString(e.Name()) || oldTopDirs[e.Name()]) {
			return true
		}
	}
	return false
}

var (
	yearDir    = regexp.MustCompile(`^[0-9]{4}$`)
	oldTopDirs = map[string]bool{"albums": true, "undated": true, "messaging": true}
)

// Relayout moves a library filed before profiles into them: each photo to
// its first source's profile, by year and month, a link to it in the
// profile of every other source, and its albums rebuilt beside it. A photo
// with no source goes in fallback's. Nothing is copied and nothing is lost:
// files are renamed on the same disk, and an old album entry is removed only
// once it is the same file as a photo now in place.
func Relayout(root string, catalog *Catalog, profileOf func(source string) string, fallback string, loc *time.Location) error {
	photos, err := catalog.Page(Filter{})
	if err != nil {
		return err
	}
	// The photos now in place, by name: an old album entry keeps its photo's
	// name, and os.SameFile says whether it is that photo.
	moved := map[string][]fs.FileInfo{}
	for _, p := range photos {
		old := filepath.Join(root, filepath.FromSlash(p.Path))
		if !exists(old) {
			continue
		}
		albums, sources, err := catalog.About(p.Hash)
		if err != nil {
			return err
		}
		profiles := profilesOf(sources, profileOf, fallback)
		opt := Options{Location: loc, Profile: profiles[0], SeparateMessaging: true}
		home := old
		if !within(old, profileRoot(root, opt)) {
			if home, err = destination(old, root, p.capture(), opt, catalog); err != nil {
				return err
			}
			if err := move(old, home); err != nil {
				return err
			}
			if err := catalog.setPath(p.Hash, relativeTo(root, home), filepath.Base(home)); err != nil {
				return err
			}
			sidecar := filepath.Join(root, WorkDir, "metadata", filepath.FromSlash(p.Path)+".json")
			if exists(sidecar) {
				if err := move(sidecar, filepath.Join(root, WorkDir, "metadata", relativeTo(root, home)+".json")); err != nil {
					return err
				}
			}
		}
		if info, err := os.Lstat(home); err == nil {
			moved[filepath.Base(home)] = append(moved[filepath.Base(home)], info)
		}
		for i, profile := range profiles {
			o := opt
			o.Profile = profile
			if i > 0 {
				there, err := destination(home, root, p.capture(), o, catalog)
				if err != nil {
					return err
				}
				if _, err := linkInto(home, there); err != nil {
					return err
				}
			}
			for _, album := range albums {
				if _, err := linkInto(home, filepath.Join(profileRoot(root, o), "albums", album, filepath.Base(home))); err != nil {
					return err
				}
			}
		}
	}
	return clearOldTops(root, moved)
}

// profilesOf are the profiles of a photo's sources, the first source's first.
func profilesOf(sources []string, profileOf func(string) string, fallback string) []string {
	sort.Strings(sources)
	var out []string
	seen := map[string]bool{}
	for _, s := range sources {
		if p := profileOf(s); p != "" && !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		out = []string{fallback}
	}
	return out
}

// clearOldTops removes what is left of the old layout: album entries that are
// the same file as a photo now filed, then the folders left empty. Anything
// else stays where it is.
func clearOldTops(root string, moved map[string][]fs.FileInfo) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() || !(yearDir.MatchString(e.Name()) || oldTopDirs[e.Name()]) {
			continue
		}
		top := filepath.Join(root, e.Name())
		err := filepath.WalkDir(top, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			info, err := os.Lstat(path)
			if err != nil {
				return err
			}
			for _, m := range moved[d.Name()] {
				if os.SameFile(info, m) {
					return os.Remove(path)
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("clearing %s: %w", e.Name(), err)
		}
		removeEmpty(top)
	}
	return nil
}

// removeEmpty removes dir and the folders under it that hold nothing.
func removeEmpty(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			removeEmpty(filepath.Join(dir, e.Name()))
		}
	}
	os.Remove(dir) // a folder that still holds something stays
}
