// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ronalder100/homewend/internal/progress"
)

// ErrNoLink is a hard link the disk refused: the folder is on another disk
// than the library, or a disk that has no hard links.
var ErrNoLink = errors.New("a hard link needs the folder on the library's disk")

// arrivals is where a file from a folder waits, copied and read back, for
// Organize to place it: inside the library, so placing it is a rename.
func arrivals(root string) string { return filepath.Join(root, WorkDir, "import") }

// Import brings f's photos and videos into the library at root, one at a
// time: each is copied into the library and read back — or, with link, given
// a second name there, which costs no space and needs the same disk — then
// placed by Organize, which files it by its date and stores a photo the
// library already holds only once. Nothing in the folder is changed.
//
// One at a time, so that a card taken out halfway leaves in the library
// every photo that was already in, and the next run brings the rest.
func Import(f Folder, root string, catalog *Catalog, opt Options, link bool, emit progress.Func) (Organized, error) {
	result := Organized{BySource: map[DateSource]int{}, ByOrigin: map[Origin]int{}}
	// What a stopped run left here was never placed: it starts again.
	if err := os.RemoveAll(arrivals(root)); err != nil {
		return result, err
	}
	defer os.RemoveAll(arrivals(root))
	for i, item := range f.Items {
		emit.Emit(progress.Event{Stage: progress.Copy, N: i + 1, Of: len(f.Items), Name: filepath.Base(item.Path)})
		// A folder of its own for each file: two cameras both name a photo
		// DSCF0001.JPG.
		arrived := filepath.Join(arrivals(root), fmt.Sprint(i), filepath.Base(item.Path))
		if err := os.MkdirAll(filepath.Dir(arrived), 0o755); err != nil {
			return result, err
		}
		if link {
			if err := os.Link(item.Path, arrived); err != nil {
				return result, fmt.Errorf("%w: %v", ErrNoLink, err)
			}
		} else if _, err := copyFile(item.Path, arrived); err != nil {
			return result, fmt.Errorf("copying %s: %w", item.Path, err)
		}
		capture := Capture{Source: NoDate, Origin: OriginOf("", filepath.Base(item.Path))}
		if when, source, ok := cameraDate(arrived); ok {
			capture.When, capture.Source = when, source
		}
		placed, err := Organize([]Item{{Path: arrived, Capture: capture}}, root, catalog, opt, nil)
		if err != nil {
			return result, err
		}
		result.Add(placed)
	}
	return result, nil
}
