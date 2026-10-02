// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"bytes"
	"strings"
	"testing"
)

// The date is read out of whatever bytes a photo holds, and a library has
// tens of thousands of files nobody has checked: no input may bring the
// program down, and a date it finds is a date it can parse.
func FuzzExifDate(f *testing.F) {
	tiff := "II*\x00\x08\x00\x00\x00" + // little-endian TIFF, first IFD at 8
		"\x01\x00" + // one entry
		"\x03\x90\x02\x00\x14\x00\x00\x00\x1a\x00\x00\x00" + // DateTimeOriginal, ASCII, 20 bytes, at 26
		"\x00\x00\x00\x00" + // no next IFD
		"2019:07:14 18:22:05\x00"
	f.Add([]byte(tiff))
	f.Add([]byte("\xff\xd8\xff\xe1\x00\x40Exif\x00\x00" + tiff))
	f.Add([]byte("MM\x00*\x00\x00\x00\x08"))
	f.Add([]byte("Exif\x00\x00II"))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		offset, order, ok := exifHeader(data)
		if !ok {
			return
		}
		findDateTimeOriginal(data, offset, order)
	})
}

// The manifest is a page Google writes and may change: whatever it holds,
// reading it ends in a list of named files or in an error, never in neither.
func FuzzParseManifest(f *testing.F) {
	f.Add([]byte(`<div class="extracted-folder-name">Photos from 2019</div><div class="extracted-file-name">IMG_1234.jpg</div>`))
	f.Add([]byte(`<div class="extracted-file-name">a.json</div>`))
	f.Add([]byte(`<div class="extracted-file-name"> &amp; </div>`))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, page []byte) {
		manifest, err := ParseManifest(bytes.NewReader(page))
		if err == nil && len(manifest.Media) == 0 {
			t.Fatal("no error and no files")
		}
		for _, entry := range manifest.Media {
			if entry.Name == "" || strings.HasSuffix(entry.Name, ".json") {
				t.Fatalf("an entry that is not a photo: %+v", entry)
			}
		}
	})
}
