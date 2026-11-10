// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/ronalder100/homewend/internal/progress"
)

// ErrNoLink is a hard link the disk refused: the folder is on another disk
// than the library, or on a disk that has none.
var ErrNoLink = errors.New("the disk refused a hard link")

// Imported is what an import did.
type Imported struct {
	Placed    int   `json:"placed"`     // photos and videos now in the library
	Bytes     int64 `json:"bytes"`      // they take, counted once each
	Undated   int   `json:"undated"`    // of them, with no date the camera wrote
	InLibrary int   `json:"in_library"` // already there before: nothing copied
	Twins     int   `json:"twins"`      // the same photo twice in the folder: brought in once
	Redated   int   `json:"redated"`    // given the date the person gave: new or already in
	// Given a date but filed in another profile, whose date it is to give.
	NotRedated int `json:"not_redated"`
	// Files and folders that could not be read, passed over.
	Unreadable int `json:"unreadable"`
}

// redate gives a photo of the library its date, and counts it given or
// left to the profile it is filed in.
func (done *Imported) redate(root string, catalog *Catalog, hash string, when time.Time, opt Options) error {
	switch err := redateInLibrary(root, catalog, hash, when, opt); {
	case errors.Is(err, ErrOthersPhoto):
		done.NotRedated++
	case err != nil:
		return err
	default:
		done.Redated++
	}
	return nil
}

// ImportOptions are how an import brings files in.
type ImportOptions struct {
	// Link gives each file a second name in the library instead of a copy:
	// no space, the same disk needed.
	Link bool
	// Fits is asked before each copy whether the library's disk has room.
	Fits func(size int64) error
	// Redates are dates given to photos whose camera had them wrong.
	Redates []Redate
	// Choice is which of the folder's groups come in; nil, each as its
	// kind says, as the look before it said.
	Choice *Choice
}

// arrivals is where a file from a folder waits, copied and read back, for
// Organize to place it: inside the library, so placing it is a rename.
func arrivals(root string) string { return filepath.Join(root, WorkDir, "import") }

// Import brings f's photos and videos into the library at root, one at a
// time. Each is read once to know its content: one the library already
// holds is not copied again. A new one is copied into the library and read
// back — or, with link, given a second name there, which costs no space and
// needs the same disk — then placed by Organize, which files it by its date.
// Nothing in the folder is changed.
//
// One at a time, so that a card taken out halfway, or a stop asked through
// ctx, leaves in the library every photo already in, and the next run brings
// the rest. fits is asked before each copy whether the library's disk has
// room for it.
func Import(ctx context.Context, f Folder, root string, catalog *Catalog, opt Options, how ImportOptions, emit progress.Func) (Imported, error) {
	done := Imported{Unreadable: f.Unreadable}
	// A name the folder does not have, or has twice, is said before
	// anything is copied.
	dates, err := given(how.Redates, f)
	if err != nil {
		return done, err
	}
	// What a stopped run left here was never placed: it starts again.
	if err := os.RemoveAll(arrivals(root)); err != nil {
		return done, err
	}
	defer os.RemoveAll(arrivals(root))
	// A project's icons, an app's previews: left where they are, as the
	// look said, unless chosen.
	in := chosen(f, classify(f, cameraDates(f)), how.Choice)
	items := slices.DeleteFunc(slices.Clone(f.Items), func(item Item) bool { return !in[item.Path] })
	seen := map[string]bool{}
	for i, item := range items {
		if err := ctx.Err(); err != nil {
			return done, err
		}
		// A file of a few GB is minutes on a card, read once to know it and
		// once to copy it: its bytes are told, at most a few times a second.
		var last time.Time
		size := sizeOf(item.Path)
		tell := func(n int64) {
			if now := time.Now(); now.Sub(last) > 250*time.Millisecond || n == size {
				last = now
				emit.Emit(progress.Event{Stage: progress.Copy, N: i + 1, Of: len(items), Name: filepath.Base(item.Path), Done: n, Total: size})
			}
		}
		emit.Emit(progress.Event{Stage: progress.Copy, N: i + 1, Of: len(items), Name: filepath.Base(item.Path)})
		hash, err := hashReading(item.Path, tell)
		if err != nil {
			done.Unreadable++
			continue
		}
		when, dated := dates[item.Path]
		if seen[hash] {
			done.Twins++
			// The same photo, given a date under its other name.
			if dated {
				if err := done.redate(root, catalog, hash, when, opt); err != nil {
					return done, err
				}
			}
			continue
		}
		seen[hash] = true
		relative, known, err := catalog.PathOf(hash)
		if err != nil {
			return done, err
		}
		home := filepath.Join(root, filepath.FromSlash(relative))
		// Gone, not unreadable: a share that answers late is not a reason to
		// copy the photo again.
		if _, err := os.Lstat(home); known && errors.Is(err, fs.ErrNotExist) {
			// In the catalog, gone from the disk: someone deleted it. The
			// arrivals were emptied, so it is no move left halfway either.
			// Brought in again, it is new, in this profile.
			if err := catalog.forget(hash); err != nil {
				return done, err
			}
			known = false
		}
		if known {
			// Already in the library. Held by another profile, it gets a
			// name in this one too: Organize is handed the library's own
			// file for that, never the folder's, so it cannot move it.
			if !within(home, profileRoot(root, opt)) {
				if _, err := Organize([]Item{{Path: home}}, root, catalog, opt, nil); err != nil {
					return done, err
				}
			}
			// A date given now is for a photo already in: the likeliest
			// case, seen as 2018 in the library after a first import.
			if dated {
				if err := done.redate(root, catalog, hash, when, opt); err != nil {
					return done, err
				}
			}
			done.InLibrary++
			continue
		}
		capture := Capture{Source: NoDate, Origin: OriginOf("", filepath.Base(item.Path))}
		if dated {
			capture.When, capture.Source = when, FromGiven
		}
		placed, err := bringIn(item.Path, i, root, catalog, opt, how.Link, how.Fits, capture, tell)
		if err != nil {
			return done, err
		}
		// A date given goes beside the copy too, for the photo apps that
		// read it there.
		if dated {
			if relative, known, err := catalog.PathOf(hash); err == nil && known {
				if err := writeXMP(filepath.Join(root, filepath.FromSlash(relative)), when); err != nil {
					return done, err
				}
				done.Redated++
			}
		}
		done.Placed += placed.Placed
		done.Bytes += placed.Bytes
		done.Undated += placed.Undated
	}
	return done, nil
}

// bringIn copies or links one file into the arrivals and places it.
// A capture with no date is read from the copy, the camera's own.
func bringIn(src string, i int, root string, catalog *Catalog, opt Options, link bool, fits func(int64) error, capture Capture, copied func(int64)) (Organized, error) {
	// A folder of its own for each file: two cameras both name a photo
	// DSCF0001.JPG. Emptied once the file is placed.
	arrived := filepath.Join(arrivals(root), fmt.Sprint(i), filepath.Base(src))
	defer os.RemoveAll(filepath.Dir(arrived))
	if err := os.MkdirAll(filepath.Dir(arrived), 0o755); err != nil {
		return Organized{}, err
	}
	if link {
		// The file a link points at, not the link: a second name for a
		// link would be a link into the folder.
		target, err := filepath.EvalSymlinks(src)
		if err != nil {
			return Organized{}, err
		}
		if err := os.Link(target, arrived); err != nil {
			return Organized{}, fmt.Errorf("%w: %v", ErrNoLink, err)
		}
	} else {
		if err := fits(sizeOf(src)); err != nil {
			return Organized{}, err
		}
		if _, err := copyFile(src, arrived, copied); err != nil {
			return Organized{}, fmt.Errorf("copying %s: %w", src, err)
		}
	}
	if capture.Source == NoDate {
		if when, source, ok := cameraDate(arrived); ok {
			capture.When, capture.Source = when, source
		}
	}
	return Organize([]Item{{Path: arrived, Capture: capture}}, root, catalog, opt, nil)
}
