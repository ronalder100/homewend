// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package engine is what every shell calls: the CLI, the MCP server, and later
// the interface. Each function here is one thing a user asks for, done from
// start to finish; the shells only parse input and show output.
package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ronalder100/homewend/internal/download"
	"github.com/ronalder100/homewend/internal/library"
	"github.com/ronalder100/homewend/internal/progress"
	"github.com/ronalder100/homewend/internal/takeout"
)

// Fetch is one export to bring into one library.
type Fetch struct {
	Target  takeout.Target
	Export  takeout.Export
	Library string
	// The timezone dates are filed in; nil means the machine's own.
	Location *time.Location
}

// Result is what a fetch achieved.
type Result struct {
	Organized    library.Organized    `json:"organized"`
	Verification library.Verification `json:"verification"`
}

// Run brings the export into the library and checks it against the manifest.
//
// Parts are taken one at a time: downloaded, unpacked, recorded, and only
// then deleted, so the disk holds the growing unpacked tree plus one part.
// Placing waits until every part is unpacked, because nothing proves that a
// photo's sidecar travels in the same part as the photo. Placing is a rename
// on the same disk, so the peak stays about the size of the export plus one
// part.
//
// Every step can be interrupted and run again: a part resumes from its last
// byte, an unpacked part is not unpacked twice, and the organiser finishes any
// move it had recorded.
func (f Fetch) Run(ctx context.Context, g download.Getter, emit progress.Func) (Result, error) {
	work := filepath.Join(f.Library, library.WorkDir, f.Target.Job)
	parts := filepath.Join(work, "parts")
	unpacked := filepath.Join(work, "unpacked")
	for _, dir := range []string{parts, unpacked} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return Result{}, err
		}
	}

	st, err := loadState(work)
	if err != nil {
		return Result{}, err
	}
	if err := f.checkSpace(st); err != nil {
		return Result{}, err
	}

	// The manifest is one more archive, and counted as one.
	of := len(f.Export.Parts) + 1
	for i, part := range f.Export.Parts {
		path := filepath.Join(parts, part.Filename)
		if !st.Unpacked[part.Filename] {
			if err := download.Part(ctx, g, f.Target, part, i+1, of, parts, emit); err != nil {
				return Result{}, err
			}
			emit.Emit(progress.Event{Stage: progress.Unpack, N: i + 1, Of: of, Name: part.Filename})
			if _, err := library.UnpackPart(path, unpacked); err != nil {
				return Result{}, err
			}
			st.Unpacked[part.Filename] = true
			if err := st.save(work); err != nil {
				return Result{}, err
			}
		}
		// Deleted only once it is recorded as unpacked, and again on every
		// run, in case the last one stopped in between.
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return Result{}, err
		}
	}

	if st.Manifest == "" {
		if err := download.Part(ctx, g, f.Target, f.Export.Manifest, of, of, work, emit); err != nil {
			return Result{}, err
		}
		st.Manifest = f.Export.Manifest.Filename
		if err := st.save(work); err != nil {
			return Result{}, err
		}
	}

	catalog, err := library.OpenCatalog(filepath.Join(f.Library, library.WorkDir, "catalog.db"))
	if err != nil {
		return Result{}, err
	}
	defer catalog.Close()

	items, err := library.ReadTakeout(unpacked)
	if err != nil {
		return Result{}, err
	}
	organized, err := library.Organize(items, f.Library, catalog, library.Options{Location: f.Location}, emit)
	if err != nil {
		return Result{}, err
	}

	verification, err := verify(f.Library, filepath.Join(work, st.Manifest))
	if err != nil {
		return Result{}, err
	}
	return Result{Organized: organized, Verification: verification}, nil
}

// NoSpaceError means the library's disk cannot hold what is left to fetch.
type NoSpaceError struct{ Need, Free int64 }

func (e NoSpaceError) Error() string {
	return fmt.Sprintf("not enough space: %d bytes needed, %d free", e.Need, e.Free)
}

// checkSpace compares what is left to fetch with the free space. The peak is
// every part not yet unpacked, once unpacked, plus the largest of them a
// second time — its archive still on disk while it is being unpacked — plus
// the manifest. Photos barely compress: the 4.1 MiB archive of the small
// export placed 4.1 MiB of photos.
func (f Fetch) checkSpace(st state) error {
	need := f.spaceNeeded(st)
	free, err := freeBytes(f.Library)
	if err != nil {
		return fmt.Errorf("reading the free space: %w", err)
	}
	if need > free {
		return NoSpaceError{Need: need, Free: free}
	}
	return nil
}

func (f Fetch) spaceNeeded(st state) int64 {
	var need, largest int64
	for _, p := range f.Export.Parts {
		if !st.Unpacked[p.Filename] {
			need += p.Size
			largest = max(largest, p.Size)
		}
	}
	need += largest
	if st.Manifest == "" {
		need += f.Export.Manifest.Size
	}
	return need
}

// Verify checks a library against the manifest of an export already fetched
// into it.
func Verify(libraryRoot, job string) (library.Verification, error) {
	work := filepath.Join(libraryRoot, library.WorkDir, job)
	st, err := loadState(work)
	if err != nil {
		return library.Verification{}, err
	}
	if st.Manifest == "" {
		return library.Verification{}, fmt.Errorf("export %s has not been fetched into %s", job, libraryRoot)
	}
	return verify(libraryRoot, filepath.Join(work, st.Manifest))
}

func verify(libraryRoot, manifestPath string) (library.Verification, error) {
	manifest, err := library.ParseManifestZip(manifestPath)
	if err != nil {
		return library.Verification{}, err
	}
	return library.Verify(libraryRoot, manifest)
}

// state is what survives between runs of one export.
type state struct {
	Unpacked map[string]bool `json:"unpacked"` // part filename -> unpacked
	Manifest string          `json:"manifest"` // manifest filename, once fetched
}

func loadState(work string) (state, error) {
	st := state{Unpacked: map[string]bool{}}
	data, err := os.ReadFile(filepath.Join(work, "state.json"))
	if errors.Is(err, os.ErrNotExist) {
		return st, nil
	}
	if err != nil {
		return st, err
	}
	if err := json.Unmarshal(data, &st); err != nil {
		return st, fmt.Errorf("reading %s: %w", filepath.Join(work, "state.json"), err)
	}
	if st.Unpacked == nil {
		st.Unpacked = map[string]bool{}
	}
	return st, nil
}

func (st state) save(work string) error {
	return writeJSON(filepath.Join(work, "state.json"), st)
}
