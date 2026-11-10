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
	"strconv"
	"strings"
	"time"

	"github.com/ronalder100/homewend/internal/progress"
)

// ErrNoLink is a hard link the disk refused: the folder is on another disk
// than the library, or on a disk that has none.
var ErrNoLink = errors.New("the disk refused a hard link")

// Imported is what an import did.
type Imported struct {
	Placed  int   `json:"placed"`  // photos and videos now in the library
	Bytes   int64 `json:"bytes"`   // they take, counted once each
	Undated int   `json:"undated"` // of them, with no date the camera wrote
	Counts
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
	// What the person said, as the look before it heard it.
	Asked
	// Hashes are what a look before read of the folder, when it did: the
	// dates that decide a group are not read again.
	Hashes Hashes
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
	done := Imported{Counts: Counts{Unreadable: f.Unreadable}}
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
	// look said, unless chosen. A date is read only where it decides a
	// group, a card's DCIM never; once stopped, none: the copy below
	// stops before its first file.
	groups, err := classify(f, func(path string) bool { return ctx.Err() == nil && how.Hashes.dated(path) }, how.Choice, emit)
	if err != nil {
		return done, err
	}
	// Stopped while dates were read: the groups are not what the look said.
	if err := ctx.Err(); err != nil {
		return done, err
	}
	in := chosen(f, groups, how.Choice)
	done.LeftOut = leftOut(groups, how.Choice)
	items := slices.DeleteFunc(slices.Clone(f.Items), func(item Item) bool { return !in[item.Path] })
	seen := map[string]bool{}
	// The run's pace: its files' bytes gone through, each read once to be
	// known and, when new, once to be copied, half of its size for each.
	sizes := make([]int64, len(items))
	var all, handled int64
	for i, item := range items {
		sizes[i] = sizeOf(item.Path)
		all += sizes[i]
	}
	moved := func(n int64) { emit.Emit(progress.Event{Stage: progress.Moved, Done: n, Total: all}) }
	for i, item := range items {
		if i > 0 {
			handled += sizes[i-1]
		}
		moved(handled)
		if err := ctx.Err(); err != nil {
			return done, err
		}
		// A file of a few GB is minutes on a card, read once to know it and
		// once to copy it: its bytes are told, at most a few times a second.
		// A photo done within the first quarter of a second tells none: a
		// card of small files would flash its first bytes at each one.
		last, told, copying := time.Now(), false, false
		size := sizes[i]
		tell := func(n int64) {
			if now := time.Now(); now.Sub(last) > 250*time.Millisecond || (told && n == size) {
				last, told = now, true
				// The pace first: the line said with the Copy carries it.
				if copying {
					moved(handled + size/2 + n/2)
				} else {
					moved(handled + n/2)
				}
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
		relative, known, deleted, err := libraryHolds(catalog, root, hash)
		if err != nil {
			return done, err
		}
		if deleted {
			// The arrivals were emptied, so it is no move left halfway
			// either. It stays out unless asked back; brought back, it is
			// new, in this profile.
			done.Deleted++
			if !how.BringBack {
				// Its twin further on is deleted too, not a photo in.
				delete(seen, hash)
				continue
			}
			if err := catalog.forget(hash); err != nil {
				return done, err
			}
			known = false
		}
		if known {
			// Already in the library. Held by another profile, it gets a
			// name in this one too, and the card as a source: Organize is
			// handed the library's own file for that, never the folder's,
			// so it cannot move it. In this profile, the card is a source
			// too: hiding the account it came from first does not hide it.
			if !filedIn(relative, opt.Profile) {
				home := filepath.Join(root, filepath.FromSlash(relative))
				if _, err := Organize([]Item{{Path: home}}, root, catalog, opt, nil); err != nil {
					return done, err
				}
			} else if err := catalog.addOwner(hash, opt.Account); err != nil {
				return done, err
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
		copying = true
		placed, err := bringIn(item.Path, i, root, catalog, opt, how.Link, how.Fits, capture, tell)
		if err != nil {
			return done, err
		}
		done.Placed += placed.Placed
		done.Bytes += placed.Bytes
		done.Undated += placed.Undated
		filed, err := catalog.photo(hash)
		if err != nil {
			return done, err
		}
		// A date given goes beside the copy too, for the photo apps that
		// read it there.
		if dated {
			if err := writeXMP(filepath.Join(root, filepath.FromSlash(filed.Path)), when); err != nil {
				return done, err
			}
			done.Redated++
		}
		// Its month, for the window that counts what came in by month: as
		// it was filed, the catalog's date. bringIn dates it as the look
		// does (the date given, else the camera's), so the two agree.
		e := progress.Event{Stage: progress.Filed}
		if !filed.Taken.IsZero() {
			e.Name, e.N = strconv.Itoa(filed.Taken.Year()), int(filed.Taken.Month())
		}
		emit.Emit(e)
	}
	moved(all)
	done.end(done.Placed)
	return done, nil
}

// libraryHolds is where the library holds the photo of hash, below root,
// and whether someone deleted it: listed in the catalog, gone from the disk.
// A disk that answers late is not a deletion. The look and the copy judge
// a photo here, alike.
func libraryHolds(catalog *Catalog, root, hash string) (path string, known, deleted bool, err error) {
	path, known, err = catalog.PathOf(hash)
	if err != nil || !known {
		return path, known, false, err
	}
	_, gone := os.Lstat(filepath.Join(root, filepath.FromSlash(path)))
	return path, true, errors.Is(gone, fs.ErrNotExist), nil
}

// filedIn is whether a path of the library is in profile's folder, where
// that profile gives it its date; "" is the library's top, which holds all.
func filedIn(path, profile string) bool {
	first, _, _ := strings.Cut(path, "/")
	return profile == "" || first == profile
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
