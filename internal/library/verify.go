// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// Verification is the answer to "is it all there?", said in the terms the user
// thinks in rather than the terms Google split the export into.
type Verification struct {
	Declared int      `json:"declared"` // distinct media files the manifest lists
	Present  int      `json:"present"`  // of those, found on disk
	Missing  []string `json:"missing"`
	// Files on disk that the manifest never mentioned. Not an error — an
	// interrupted run leaves partial files, and the user may have put things
	// there — but worth showing, because a surprise in either direction is
	// worth a sentence.
	Unexpected []string `json:"unexpected"`

	Years []YearCount `json:"years"`
}

// YearCount is one line of the report the user actually reads.
type YearCount struct {
	Year     string `json:"year"`
	Declared int    `json:"declared"`
	Present  int    `json:"present"`
}

// Complete reports whether every file the manifest declared is on disk.
func (v Verification) Complete() bool { return len(v.Missing) == 0 }

// Verify walks dir and compares what is there against the manifest.
//
// The comparison is by filename, not by count. Two numbers that agree can still
// hide a swap — one file lost and another duplicated — and the whole promise
// rests on this check, so it is made the way that cannot agree by accident.
func Verify(dir string, manifest Manifest) (Verification, error) {
	onDisk, err := mediaOnDisk(dir)
	if err != nil {
		return Verification{}, err
	}

	declared := manifest.Names()
	result := Verification{Declared: len(declared)}

	for name := range declared {
		if onDisk[name] {
			result.Present++
		} else {
			result.Missing = append(result.Missing, name)
		}
	}
	for name := range onDisk {
		if !declared[name] {
			result.Unexpected = append(result.Unexpected, name)
		}
	}
	sort.Strings(result.Missing)
	sort.Strings(result.Unexpected)

	counts, years := manifest.ByYear()
	perYear := perYearOnDisk(manifest, onDisk)
	for _, year := range years {
		result.Years = append(result.Years, YearCount{
			Year:     year,
			Declared: counts[year],
			Present:  perYear[year],
		})
	}
	return result, nil
}

// perYearOnDisk counts, for each year, how many of that year's declared files
// are present. A file is attributed to the year the manifest put it in, not to
// wherever it ended up on disk: the question is whether 2019 is complete, and
// the answer must not change because the organiser moved things.
func perYearOnDisk(manifest Manifest, onDisk map[string]bool) map[string]int {
	found := map[string]map[string]bool{}
	for _, entry := range manifest.Media {
		match := yearFolder.FindStringSubmatch(entry.Folder)
		if match == nil || !onDisk[entry.Name] {
			continue
		}
		if found[match[1]] == nil {
			found[match[1]] = map[string]bool{}
		}
		found[match[1]][entry.Name] = true
	}
	counts := make(map[string]int, len(found))
	for year, names := range found {
		counts[year] = len(names)
	}
	return counts
}

// WorkDir is the directory inside a library where the app keeps its own files:
// the catalog, and each export's parts while they are being processed. It sits
// on the library's disk so that placing a photo is a rename, not a copy.
const WorkDir = ".homewend"

// mediaOnDisk is every non-sidecar file under dir, by base name.
//
// By base name because the manifest names files, not paths, and because the
// same photo legitimately exists in several places once albums are rebuilt.
func mediaOnDisk(dir string) (map[string]bool, error) {
	names := map[string]bool{}
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == WorkDir {
				return filepath.SkipDir // our own working files, not the library
			}
			if path == filepath.Join(dir, Unassigned) {
				return filepath.SkipDir // arrived, not filed: not in the library yet
			}
			return nil
		}
		name := entry.Name()
		switch {
		case strings.HasSuffix(name, ".json"):
			return nil // sidecars are metadata, not media
		case strings.HasSuffix(name, ".html"):
			return nil // the manifest itself
		case strings.HasSuffix(name, ".zip"):
			return nil // an archive we have not unpacked, or have kept
		}
		names[name] = true
		return nil
	})
	return names, err
}
