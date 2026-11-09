// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Redate is a date given by the person to photos whose camera had it wrong:
// a camera whose clock was reset dates them 1 January of its first year
// (measured 09/11 on a Fujifilm card). Nobody can tell that from the photo,
// so nothing guesses it: it is said, before the import, by whoever knows.
type Redate struct {
	// Names are the files it is for: a name, in any case, or the path
	// below the folder when two of its folders hold the same name.
	Names []string `json:"names"`
	// When is the date and time of the first of them, by the camera's
	// clock: a clock, like EXIF, kept unmoved.
	When time.Time `json:"when"`
}

// ErrOthersPhoto is a date given to a photo another profile brought first.
var ErrOthersPhoto = errors.New("the photo is filed in another profile")

// UnknownNameError is a name given a date that the folder does not have.
type UnknownNameError struct{ Name string }

func (e UnknownNameError) Error() string { return e.Name + " is not in the folder" }

// AmbiguousNameError is a name two files of the folder have: two cameras
// both name a photo DSCF0004.JPG. Which one is meant is said by its path.
type AmbiguousNameError struct {
	Name  string
	Paths []string // below the folder
}

func (e AmbiguousNameError) Error() string { return e.Name + " is in more than one folder" }

// given is the date each file of the redates gets, by its path: the first
// of them by the camera's own clock takes When, the others keep their
// distance from it — seventy seconds apart on the card are seventy seconds
// apart here. A file the camera gave no date takes When. A name the folder
// does not have, or has twice, is said before anything is copied.
func given(redates []Redate, f Folder) (map[string]time.Time, error) {
	dates := map[string]time.Time{}
	for _, r := range redates {
		var paths []string
		for _, name := range r.Names {
			found, err := named(name, f)
			if err != nil {
				return nil, err
			}
			paths = append(paths, found)
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
	return dates, nil
}

// named is the one file of f that name means: its name, or its path below
// the folder, in any case — a card's FAT keeps names as the camera wrote
// them, and nobody types them so.
func named(name string, f Folder) (string, error) {
	// A path typed as the shell completes it — whole, or from where the
	// command runs — names a file below the folder when it leads there.
	if strings.ContainsAny(name, `/\`) {
		if rel, ok := below(name, f.Root); ok {
			name = rel
		}
	}
	want := filepath.ToSlash(filepath.Clean(name))
	var found []string
	for _, item := range f.Items {
		rel, err := filepath.Rel(f.Root, item.Path)
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(rel)
		if strings.EqualFold(rel, want) || (!strings.Contains(want, "/") && strings.EqualFold(filepath.Base(item.Path), want)) {
			found = append(found, item.Path)
		}
	}
	switch len(found) {
	case 0:
		return "", UnknownNameError{Name: name}
	case 1:
		return found[0], nil
	}
	e := AmbiguousNameError{Name: name}
	for _, p := range found {
		rel, _ := filepath.Rel(f.Root, p)
		e.Paths = append(e.Paths, filepath.ToSlash(rel))
	}
	return "", e
}

// below is path, as typed, relative to root, when it leads into root. root
// is whole and its links followed (see readFolder), and so is path made.
func below(path, root string) (string, bool) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	if !within(path, root) {
		return "", false
	}
	rel, err := filepath.Rel(root, path)
	return rel, err == nil
}

// redateInLibrary gives a date to a photo the library already holds: it
// moves to the folder of that date, on the same disk, and the catalog and
// a .xmp beside it say the date. The file is not written.
//
// A photo is filed once, in the profile that brought it first; others hold
// a second name of it. Its date is given from that profile: given from
// another, it would take the photo away from the first (ErrOthersPhoto).
func redateInLibrary(root string, catalog *Catalog, hash string, when time.Time, opt Options) error {
	record, err := catalog.photo(hash)
	if err != nil {
		return err
	}
	from := filepath.Join(root, filepath.FromSlash(record.Path))
	if !within(from, profileRoot(root, opt)) {
		return ErrOthersPhoto
	}
	capture := Capture{When: when, Source: FromGiven, Origin: record.Origin}
	to, err := destination(from, root, capture, opt, catalog)
	if err != nil {
		return err
	}
	if to != from {
		// The catalog first, as Organize does: the name is the photo's
		// before anything moves. A move that fails gives the old one back.
		if err := catalog.setPath(hash, relativeTo(root, to), filepath.Base(to)); err != nil {
			return err
		}
		if err := move(from, to); err != nil {
			catalog.setPath(hash, record.Path, filepath.Base(from))
			return err
		}
		// Google's sidecar, kept under the library's metadata folder by the
		// photo's path, goes with it.
		meta := func(p string) string { return filepath.Join(root, WorkDir, "metadata", relativeTo(root, p)+".json") }
		if exists(meta(from)) {
			if err := move(meta(from), meta(to)); err != nil {
				return err
			}
		}
		// A .xmp of an earlier date goes too, and is written again below.
		os.Remove(from + ".xmp")
		// The month and the year it leaves, if nothing else is in them: an
		// empty 2018 would say a year the camera never saw. Remove fails on
		// a folder that is not empty, which is the check.
		// Never the profile's own folder: undated/ is right under it.
		month := filepath.Dir(from)
		if os.Remove(month) == nil && filepath.Dir(month) != profileRoot(root, opt) {
			os.Remove(filepath.Dir(month))
		}
	}
	if err := catalog.setDate(hash, capture); err != nil {
		return err
	}
	return writeXMP(to, when)
}

// writeXMP puts a given date in an XMP file beside the photo, named as
// Immich, digiKam and darktable look for it first: the photo's whole name
// and .xmp. The photo itself is never written. The date is the clock given,
// with this computer's offset at that date: without one, Immich reads it as
// UTC. A time the clock skipped (the hour lost to summer time) keeps the
// time given, with the offset just after it.
func writeXMP(photo string, when time.Time) error {
	offset := time.Date(when.Year(), when.Month(), when.Day(), when.Hour(), when.Minute(), when.Second(), 0, time.Local).Format("-07:00")
	stamp := when.Format("2006-01-02T15:04:05") + offset
	body := "<?xpacket begin=\"\ufeff\" id=\"W5M0MpCehiHzreSzNTczkc9d\"?>\n" + `<x:xmpmeta xmlns:x="adobe:ns:meta/">
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
