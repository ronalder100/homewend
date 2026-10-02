// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ronalder100/homewend/internal/progress"
)

// Organized is what an organise pass produced.
type Organized struct {
	Placed  int   `json:"placed"`  // distinct photos now in the date layout
	Bytes   int64 `json:"bytes"`   // bytes those photos occupy, counted once each
	Linked  int   `json:"linked"`  // album entries pointing at them
	Copied  int   `json:"copied"`  // album entries that had to be copied, no hardlinks available
	Skipped int   `json:"skipped"` // duplicates that cost nothing because they were already there

	// Photos with no date anyone could establish. They are placed, not dropped,
	// under a folder that says so.
	Undated int `json:"undated"`
	// Where each date came from, so the app can say how much it is guessing.
	BySource map[DateSource]int `json:"by_source"`
	// What made each photo. A quarter of a real library arrives through chats,
	// and the user deserves to be told rather than to scroll past it.
	ByOrigin map[Origin]int `json:"by_origin"`
}

// Add counts another pass in with this one.
func (o *Organized) Add(more Organized) {
	o.Placed += more.Placed
	o.Bytes += more.Bytes
	o.Linked += more.Linked
	o.Copied += more.Copied
	o.Skipped += more.Skipped
	o.Undated += more.Undated
	if o.BySource == nil {
		o.BySource = map[DateSource]int{}
	}
	for source, n := range more.BySource {
		o.BySource[source] += n
	}
	if o.ByOrigin == nil {
		o.ByOrigin = map[Origin]int{}
	}
	for origin, n := range more.ByOrigin {
		o.ByOrigin[origin] += n
	}
}

// Options for an organise pass.
type Options struct {
	// The timezone the dates are read in. Google's sidecars are UTC, and the
	// user thinks in local time: a photo taken at half past midnight in Rome is
	// 22:30 the previous day in UTC, so filing by UTC puts it in the wrong day
	// and, on New Year's Eve, the wrong year.
	//
	// Nil means the machine's own timezone, which is right for the common case
	// of someone rescuing their own library on their own computer. It is a
	// setting rather than a guess because the honest answer — what zone was the
	// camera in — is not in the export: no sidecar carries one, and on the
	// library measured here not a single file had GPS coordinates to infer it
	// from.
	Location *time.Location

	// SeparateMessaging puts WhatsApp, Signal and Telegram pictures under
	// "messaging/" instead of among the photographs. Off by default: it is the
	// user's library and a quarter of it arrived through chats, so hiding that
	// by default would be deciding for them.
	SeparateMessaging bool
}

func (o Options) location() *time.Location {
	if o.Location != nil {
		return o.Location
	}
	return time.Local
}

// Item is one media file waiting to be placed, with what its source knows
// about it. A Takeout export is one source of items; a folder on disk will be
// another. The organiser does not know, or need to know, which it is.
type Item struct {
	Path    string
	Capture Capture
	Album   string // "" when the file belongs to no album
	// A metadata file that travels with the photo, "" if none. Takeout's
	// sidecar holds what the photo itself often does not: the description,
	// the place, the people. It is kept, never merged into the photo.
	Sidecar string
}

// Organize moves items into a date layout under root, and rebuilds the albums
// beside it without storing anything twice.
//
//	<root>/2019/07/IMG_1234.jpg
//	<root>/albums/Greece 2019/IMG_1234.jpg   -> a hardlink to the file above
//	<root>/undated/SCAN_0003.jpg
//
// **Deduplication is by content, never by name.** The same photo appears under
// its year and again under every album it belongs to, and Google sometimes
// hands the same video twice as two different parts — on the 344 GB export,
// 23 GB of it. Two files with the same SHA-256 are one photo, and one photo is
// stored once.
//
// Names, on the other hand, are not identity: two cameras both produce
// IMG_0001.jpg, and "photo.jpg" and "photo-edited.jpg" are different pictures
// that a name-based rule would either merge or miss. When two different photos
// want the same place, the second one gets a suffix and both are kept.
//
// **Files are moved, never modified.** Items must be on the same filesystem as
// root, so a move is a rename: instant, and no second copy on the disk. A
// duplicate is left where it was. Each photo is recorded in the catalog before
// it is moved, so a run that stops halfway finishes the move the next time
// instead of losing track of the file.
func Organize(items []Item, root string, catalog *Catalog, opt Options, emit progress.Func) (Organized, error) {
	result := Organized{BySource: map[DateSource]int{}, ByOrigin: map[Origin]int{}}

	for i, item := range items {
		emit.Emit(progress.Event{Stage: progress.Place, N: i + 1, Of: len(items), Name: filepath.Base(item.Path)})

		hash, err := hashOf(item.Path)
		if err != nil {
			return result, err
		}

		relative, known, err := catalog.PathOf(hash)
		if err != nil {
			return result, err
		}
		home := filepath.Join(root, filepath.FromSlash(relative))
		switch {
		case known && exists(home):
			result.Skipped++
		case known:
			// Recorded by a run that stopped before the move.
			if err := place(item, home, root); err != nil {
				return result, err
			}
		default:
			capture := item.Capture
			result.BySource[capture.Source]++
			result.ByOrigin[capture.Origin]++
			if capture.Source == NoDate {
				result.Undated++
			}

			home, err = destination(item.Path, root, capture, opt, catalog)
			if err != nil {
				return result, err
			}
			size := sizeOf(item.Path)
			record := Photo{
				Hash:   hash,
				Path:   relativeTo(root, home),
				Name:   filepath.Base(home),
				Bytes:  size,
				Taken:  capture.When,
				Source: capture.Source,
				Kind:   capture.Kind(),
				Origin: capture.Origin,
			}
			if capture.Source == NoDate {
				record.Taken = time.Time{}
			}
			if err := catalog.Put(record); err != nil {
				return result, err
			}
			if err := place(item, home, root); err != nil {
				return result, err
			}
			result.Placed++
			result.Bytes += size
		}

		// An album copy is a second name for a photo already stored, so it
		// costs a directory entry and no bytes.
		if item.Album != "" {
			linked, err := linkInto(home, filepath.Join(root, "albums", item.Album, filepath.Base(home)))
			if err != nil {
				return result, err
			}
			if linked {
				result.Linked++
			} else {
				result.Copied++
			}
			if err := catalog.addAlbum(item.Album, hash); err != nil {
				return result, err
			}
		}
	}
	return result, nil
}

// destination is where a photo's date says it belongs, with a suffix when
// another photo already holds that name.
func destination(src, root string, capture Capture, opt Options, catalog *Catalog) (string, error) {
	base := root
	if opt.SeparateMessaging && capture.Origin.FromMessaging() {
		base = filepath.Join(root, "messaging")
	}

	// Read in the user's own timezone, not UTC — see Options.Location.
	local := capture.When.In(opt.location())

	var dir string
	switch capture.Source {
	case FromSidecar, FromEXIF:
		dir = filepath.Join(base, local.Format("2006"), local.Format("01"))
	case FromFolder:
		// The year is Google's and the month is not known. Saying so in the
		// path is more honest than picking January and looking precise. No
		// timezone conversion here: a folder name has no clock to convert.
		dir = filepath.Join(base, capture.When.Format("2006"), "unknown-month")
	default:
		dir = filepath.Join(base, "undated")
	}

	// Same name, different photo: keep both. Silently overwriting here is how a
	// library quietly loses pictures. A name is taken when a file holds it, or
	// when the catalog has promised it to a photo whose move was interrupted.
	target := filepath.Join(dir, filepath.Base(src))
	for {
		promised, err := catalog.PathTaken(relativeTo(root, target))
		if err != nil {
			return "", err
		}
		if !promised && !exists(target) {
			return target, nil
		}
		target = withSuffix(target)
	}
}

// place moves a photo to home, and its sidecar to the same path under the
// library's metadata folder, out of sight but beside nothing else:
//
//	<root>/2019/07/IMG_1234.jpg
//	<root>/.homewend/metadata/2019/07/IMG_1234.jpg.json
func place(item Item, home, root string) error {
	if err := move(item.Path, home); err != nil {
		return err
	}
	if item.Sidecar == "" {
		return nil
	}
	return move(item.Sidecar, filepath.Join(root, WorkDir, "metadata", relativeTo(root, home)+".json"))
}

// move renames src to dst. Both are on the library's filesystem, so this never
// copies a byte.
func move(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("placing %s: %w", filepath.Base(src), err)
	}
	return nil
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func sizeOf(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func withSuffix(path string) string {
	ext := filepath.Ext(path)
	stem := strings.TrimSuffix(path, ext)

	// "name-2.jpg", then "name-3.jpg", rather than "name-2-2.jpg".
	if at := strings.LastIndex(stem, "-"); at > 0 {
		var n int
		if _, err := fmt.Sscanf(stem[at+1:], "%d", &n); err == nil {
			return fmt.Sprintf("%s-%d%s", stem[:at], n+1, ext)
		}
	}
	return stem + "-2" + ext
}

// linkInto makes path a second name for src. Reports whether a hardlink worked;
// a copy is the fallback on filesystems that have none, or across devices.
func linkInto(src, path string) (bool, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	if _, err := os.Lstat(path); err == nil {
		return true, nil // already there from an earlier run
	}
	if err := os.Link(src, path); err == nil {
		return true, nil
	}
	if _, err := copyFile(src, path); err != nil {
		return false, fmt.Errorf("adding %s to an album: %w", filepath.Base(src), err)
	}
	return false, nil
}

func hashOf(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", fmt.Errorf("hashing %s: %w", path, err)
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}
