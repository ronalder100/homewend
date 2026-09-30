// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package library turns archives into a library, and answers the only question
// the user really has: is it all there?
//
// The answer is a comparison against `archive_browser.html`, the file Google
// puts one index past the last declared part. It lists every file with the
// folder it belongs to, which makes the check nominal — this file, by name,
// arrived — instead of a count that can agree by accident.
//
// What it cannot do is close the other gap. The manifest says what Google put
// in the export; it cannot say whether Google put everything. So the promise is
// "everything in the export arrived, and here it is file by file", and not
// "you have all your photos". The difference is the whole of our honesty.
package library

import (
	"archive/zip"
	"fmt"
	"html"
	"io"
	"regexp"
	"sort"
	"strings"
)

// Manifest is what Google declared an export contains.
type Manifest struct {
	// Every media file, in the order the manifest lists it, with its folder.
	// Sidecars (*.json) are left out: they are metadata about the media, not
	// media, and counting them is how "102,790 files" gets mistaken for a
	// number of photos.
	Media []Entry
}

// Entry is one file as the manifest lists it.
type Entry struct {
	Folder string
	Name   string
}

var (
	rowPattern = regexp.MustCompile(
		`<div class="extracted-(folder|file)-name">([^<]*)</div>`)
	yearFolder = regexp.MustCompile(`^Photos from (\d{4})$`)
)

// ParseManifest reads archive_browser.html.
func ParseManifest(r io.Reader) (Manifest, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return Manifest{}, fmt.Errorf("reading the manifest: %w", err)
	}

	var manifest Manifest
	folder := ""
	for _, row := range rowPattern.FindAllStringSubmatch(string(body), -1) {
		name := strings.TrimSpace(html.UnescapeString(row[2]))
		if row[1] == "folder" {
			folder = name
			continue
		}
		if name == "" || strings.HasSuffix(name, ".json") {
			continue
		}
		manifest.Media = append(manifest.Media, Entry{Folder: folder, Name: name})
	}
	if len(manifest.Media) == 0 {
		return Manifest{}, fmt.Errorf("no files in the manifest: is this archive_browser.html?")
	}
	return manifest, nil
}

// ParseManifestZip reads archive_browser.html out of the zip Google delivers it in.
func ParseManifestZip(path string) (Manifest, error) {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("opening %s: %w", path, err)
	}
	defer archive.Close()

	for _, file := range archive.File {
		if !strings.HasSuffix(file.Name, "archive_browser.html") {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			return Manifest{}, err
		}
		defer reader.Close()
		return ParseManifest(reader)
	}
	return Manifest{}, fmt.Errorf("no archive_browser.html inside %s", path)
}

// Names is every distinct filename the manifest lists.
//
// Distinct matters: the same photo appears under every album it belongs to as
// well as under its year, so the rows are far more numerous than the photos.
// On a real export: 53,771 media rows, 38,584 distinct names.
func (m Manifest) Names() map[string]bool {
	names := make(map[string]bool, len(m.Media))
	for _, entry := range m.Media {
		names[entry.Name] = true
	}
	return names
}

// ByYear counts distinct files per year folder, and returns the years in order.
//
// This is the unit the user thinks in. Google decides how an export is split —
// into 159 arbitrary archives — but nobody can check an archive. A year they
// can: "2019: 8,662 declared, 8,662 here". The export unit is Google's; the
// verification unit is ours.
func (m Manifest) ByYear() (counts map[string]int, years []string) {
	perYear := map[string]map[string]bool{}
	for _, entry := range m.Media {
		match := yearFolder.FindStringSubmatch(entry.Folder)
		if match == nil {
			continue
		}
		if perYear[match[1]] == nil {
			perYear[match[1]] = map[string]bool{}
		}
		perYear[match[1]][entry.Name] = true
	}

	counts = make(map[string]int, len(perYear))
	for year, names := range perYear {
		counts[year] = len(names)
		years = append(years, year)
	}
	sort.Strings(years)
	return counts, years
}

// AlbumOnly is the distinct files that live in an album and in no year folder.
//
// They matter because a check that only walks the year folders would call them
// missing, and a library that only rebuilds the years would lose them.
func (m Manifest) AlbumOnly() []string {
	inYears, inAlbums := map[string]bool{}, map[string]bool{}
	for _, entry := range m.Media {
		if yearFolder.MatchString(entry.Folder) {
			inYears[entry.Name] = true
		} else {
			inAlbums[entry.Name] = true
		}
	}

	var only []string
	for name := range inAlbums {
		if !inYears[name] {
			only = append(only, name)
		}
	}
	sort.Strings(only)
	return only
}

// Albums counts distinct files per album, ignoring the year folders.
func (m Manifest) Albums() map[string]int {
	perAlbum := map[string]map[string]bool{}
	for _, entry := range m.Media {
		if entry.Folder == "" || yearFolder.MatchString(entry.Folder) {
			continue
		}
		if perAlbum[entry.Folder] == nil {
			perAlbum[entry.Folder] = map[string]bool{}
		}
		perAlbum[entry.Folder][entry.Name] = true
	}
	counts := make(map[string]int, len(perAlbum))
	for album, names := range perAlbum {
		counts[album] = len(names)
	}
	return counts
}
