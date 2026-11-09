// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"os"
	"path/filepath"
	"time"
)

// Redate is a date given by the person to photos whose camera had it wrong:
// a camera whose clock was reset dates them 1 January of its first year
// (measured 09/11 on a Fujifilm card). Nobody can tell that from the photo,
// so nothing guesses it: it is said, before the import, by whoever knows.
type Redate struct {
	// Names are the files it is for, by their name on the card.
	Names []string `json:"names"`
	// When is the date and time of the first of them, by the camera's
	// clock: a clock, like EXIF, kept unmoved.
	When time.Time `json:"when"`
}

// given is the date each file of the redates gets: the first of them by the
// camera's own clock takes When, the others keep their distance from it —
// seventy seconds apart on the card are seventy seconds apart here. A file
// the camera gave no date takes When.
func given(redates []Redate, items []Item) map[string]time.Time {
	byName := map[string][]string{}
	for _, item := range items {
		byName[filepath.Base(item.Path)] = append(byName[filepath.Base(item.Path)], item.Path)
	}
	dates := map[string]time.Time{}
	for _, r := range redates {
		var paths []string
		for _, name := range r.Names {
			paths = append(paths, byName[name]...)
		}
		var first time.Time
		clocks := map[string]time.Time{}
		for _, p := range paths {
			if when, _, ok := cameraDate(p); ok {
				clocks[p] = when
				if first.IsZero() || when.Before(first) {
					first = when
				}
			}
		}
		for _, p := range paths {
			if clock, ok := clocks[p]; ok {
				dates[p] = r.When.Add(clock.Sub(first))
			} else {
				dates[p] = r.When
			}
		}
	}
	return dates
}

// writeXMP puts a given date in an XMP file beside the photo, named as
// Immich, digiKam and darktable look for it first: the photo's whole name
// and .xmp. The photo itself is never written. The date carries this
// computer's offset at that date: without one, Immich reads it as UTC.
func writeXMP(photo string, when time.Time) error {
	local := time.Date(when.Year(), when.Month(), when.Day(), when.Hour(), when.Minute(), when.Second(), 0, time.Local)
	stamp := local.Format("2006-01-02T15:04:05-07:00")
	body := `<?xpacket begin="\ufeff" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
 <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
  <rdf:Description rdf:about=""
    xmlns:exif="http://ns.adobe.com/exif/1.0/"
    xmlns:xmp="http://ns.adobe.com/xap/1.0/"
    exif:DateTimeOriginal="` + stamp + `"
    xmp:CreateDate="` + stamp + `"/>
 </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>
`
	tmp := photo + ".xmp.tmp"
	if err := os.WriteFile(tmp, []byte(body), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, photo+".xmp")
}
