// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// DateSource says where a capture date came from. The user is shown it, because
// "we think this is from 2019" and "Google says this is from 2019" are different
// claims and only one of them is ours.
type DateSource string

const (
	FromSidecar DateSource = "sidecar" // Google's own answer
	FromEXIF    DateSource = "exif"    // the camera's answer
	FromFolder  DateSource = "folder"  // the year Takeout filed it under
	NoDate      DateSource = "none"    // and we do not invent one
)

// Capture is when a photo was taken, how we know, and what made it.
type Capture struct {
	When   time.Time
	Source DateSource
	Origin Origin
}

// Known reports whether there is a real date, as opposed to a year we inferred
// or nothing at all.
func (c Capture) Known() bool { return c.Source == FromSidecar || c.Source == FromEXIF }

// DateOf answers "when was this taken?" in the order of how much the answer can
// be trusted, and stops at the first real one.
//
// **The file's modification time is never consulted, and that is deliberate.**
// Takeout sets it to the moment it built the archive — every file in the export
// we downloaded carries 24 September 2026 — so a program that trusts mtime files
// a whole life of photographs under the day it downloaded them. It is the most
// obvious source and the only one that is certainly wrong.
func DateOf(mediaPath string, folder string) Capture {
	when, packageName, ok := sidecarInfo(mediaPath)
	origin := OriginOf(packageName, filepath.Base(mediaPath))
	if ok {
		return Capture{When: when, Source: FromSidecar, Origin: origin}
	}
	if when, ok := exifDate(mediaPath); ok {
		return Capture{When: when, Source: FromEXIF, Origin: origin}
	}
	// "Photos from 2019" is not a date, but it is Google's own filing and it is
	// better than nothing: the photo lands in the right year, in a month folder
	// that says it is unknown.
	if match := yearFolder.FindStringSubmatch(folder); match != nil {
		year, err := strconv.Atoi(match[1])
		if err == nil {
			return Capture{
				When:   time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC),
				Source: FromFolder,
				Origin: origin,
			}
		}
	}
	return Capture{Source: NoDate, Origin: origin}
}

// Takeout writes one sidecar per media file. The name is usually
// "<file>.supplemental-metadata.json", but it has changed over the years and it
// gets truncated when the whole name would be too long — which is exactly the
// case where a program that builds the name instead of looking for it fails.
func sidecarInfo(mediaPath string) (time.Time, string, bool) {
	for _, candidate := range sidecarCandidates(mediaPath) {
		file, err := os.Open(candidate)
		if err != nil {
			continue
		}
		var meta struct {
			PhotoTakenTime struct {
				Timestamp string `json:"timestamp"`
			} `json:"photoTakenTime"`
			CreationTime struct {
				Timestamp string `json:"timestamp"`
			} `json:"creationTime"`
			// The app that produced the file, as an Android package name.
			// Google's own answer to "where did this come from", and far
			// better than reading it off the filename.
			AppSource struct {
				AndroidPackageName string `json:"androidPackageName"`
			} `json:"appSource"`
		}
		err = json.NewDecoder(file).Decode(&meta)
		file.Close()
		if err != nil {
			continue
		}
		// photoTakenTime is when the shutter fired; creationTime is when it
		// reached Google, which for a scanned photograph is decades later.
		// Only the first is a capture date.
		if seconds, err := strconv.ParseInt(meta.PhotoTakenTime.Timestamp, 10, 64); err == nil && seconds > 0 {
			return time.Unix(seconds, 0).UTC(), meta.AppSource.AndroidPackageName, true
		}
	}
	return time.Time{}, "", false
}

// findSidecar returns the path of a media file's sidecar, or "" if it has none.
func findSidecar(mediaPath string) string {
	for _, candidate := range sidecarCandidates(mediaPath) {
		if exists(candidate) {
			return candidate
		}
	}
	return ""
}

func sidecarCandidates(mediaPath string) []string {
	dir := filepath.Dir(mediaPath)
	base := filepath.Base(mediaPath)
	stem := strings.TrimSuffix(base, filepath.Ext(base))

	names := []string{
		base + ".supplemental-metadata.json",
		base + ".json",
		stem + ".json",
	}
	// Truncated variants: Takeout cuts the suffix to fit a length limit, so
	// "long-name.jpg.supplemental-met.json" is a real sidecar in the wild.
	for cut := len("supplemental-metadata"); cut > 3; cut-- {
		names = append(names, base+".supplemental-metadata"[:cut+1]+".json")
	}

	paths := make([]string, 0, len(names))
	for _, name := range names {
		paths = append(paths, filepath.Join(dir, name))
	}
	return paths
}

// exifDate reads DateTimeOriginal out of a JPEG or TIFF, and nothing else.
//
// Deliberately minimal: one tag, read-only, no dependency. We never write a
// byte of metadata — that is a product decision as much as an engineering one —
// so a full EXIF library would be a large surface for a single number.
//
// Formats it cannot read (HEIC, MP4, MOV) fall through to the year folder.
// That is acceptable because sidecars cover nearly everything; EXIF is the
// fallback for when Takeout left one out.
func exifDate(path string) (time.Time, bool) {
	file, err := os.Open(path)
	if err != nil {
		return time.Time{}, false
	}
	defer file.Close()

	// The APP1 segment lives near the front; a megabyte is far more than enough
	// and keeps this cheap on a library of tens of thousands of files.
	head := make([]byte, 1<<20)
	n, err := io.ReadFull(file, head)
	if err != nil && n == 0 {
		return time.Time{}, false
	}
	head = head[:n]

	offset, order, ok := exifHeader(head)
	if !ok {
		return time.Time{}, false
	}
	raw, ok := findDateTimeOriginal(head, offset, order)
	if !ok {
		return time.Time{}, false
	}
	// EXIF writes "2019:07:14 18:22:05", with no timezone. Taking it as UTC is
	// a choice, not a truth: without a zone the wall-clock time is all there is,
	// and shifting it by a guess would move photos across midnight.
	when, err := time.Parse("2006:01:02 15:04:05", strings.TrimRight(raw, "\x00 "))
	if err != nil || when.Year() < 1900 {
		return time.Time{}, false
	}
	return when, true
}

// exifHeader finds the TIFF header inside a JPEG's APP1 segment, or at the very
// start for a bare TIFF, and returns where it begins and its byte order.
func exifHeader(data []byte) (int, binary.ByteOrder, bool) {
	if len(data) > 4 {
		switch {
		case data[0] == 'I' && data[1] == 'I':
			return 0, binary.LittleEndian, true
		case data[0] == 'M' && data[1] == 'M':
			return 0, binary.BigEndian, true
		}
	}
	marker := []byte("Exif\x00\x00")
	at := indexOf(data, marker)
	if at < 0 || at+6+4 > len(data) {
		return 0, nil, false
	}
	tiff := at + 6
	switch {
	case data[tiff] == 'I' && data[tiff+1] == 'I':
		return tiff, binary.LittleEndian, true
	case data[tiff] == 'M' && data[tiff+1] == 'M':
		return tiff, binary.BigEndian, true
	}
	return 0, nil, false
}

const (
	tagDateTimeOriginal = 0x9003
	tagExifIFD          = 0x8769
)

func findDateTimeOriginal(data []byte, tiff int, order binary.ByteOrder) (string, bool) {
	if tiff+8 > len(data) {
		return "", false
	}
	first := int(order.Uint32(data[tiff+4 : tiff+8]))

	// The tag lives in the Exif sub-IFD, which the main one points at. Both are
	// walked because some writers put it in either.
	for _, ifd := range []int{first, subIFD(data, tiff, first, order)} {
		if ifd <= 0 {
			continue
		}
		if value, ok := readTag(data, tiff, ifd, tagDateTimeOriginal, order); ok {
			return value, true
		}
	}
	return "", false
}

func subIFD(data []byte, tiff, ifd int, order binary.ByteOrder) int {
	value, ok := readTagOffset(data, tiff, ifd, tagExifIFD, order)
	if !ok {
		return 0
	}
	return value
}

func eachEntry(data []byte, tiff, ifd int, order binary.ByteOrder, visit func(tag uint16, kind uint16, count uint32, at int) bool) {
	start := tiff + ifd
	if start+2 > len(data) {
		return
	}
	count := int(order.Uint16(data[start : start+2]))
	for i := 0; i < count; i++ {
		at := start + 2 + i*12
		if at+12 > len(data) {
			return
		}
		if visit(order.Uint16(data[at:at+2]), order.Uint16(data[at+2:at+4]),
			order.Uint32(data[at+4:at+8]), at) {
			return
		}
	}
}

func readTag(data []byte, tiff, ifd int, want uint16, order binary.ByteOrder) (string, bool) {
	var found string
	var ok bool
	eachEntry(data, tiff, ifd, order, func(tag, kind uint16, count uint32, at int) bool {
		if tag != want || kind != 2 || count == 0 { // 2 = ASCII
			return false
		}
		valueAt := tiff + int(order.Uint32(data[at+8:at+12]))
		end := valueAt + int(count)
		if valueAt < 0 || end > len(data) {
			return true
		}
		found, ok = string(data[valueAt:end]), true
		return true
	})
	return found, ok
}

func readTagOffset(data []byte, tiff, ifd int, want uint16, order binary.ByteOrder) (int, bool) {
	var found int
	var ok bool
	eachEntry(data, tiff, ifd, order, func(tag, kind uint16, count uint32, at int) bool {
		if tag != want {
			return false
		}
		found, ok = int(order.Uint32(data[at+8:at+12])), true
		return true
	})
	return found, ok
}

func indexOf(haystack, needle []byte) int {
	limit := len(haystack) - len(needle)
	for i := 0; i <= limit; i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
