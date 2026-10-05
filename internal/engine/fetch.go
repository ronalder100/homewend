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
	"strings"
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
	// Account is the source of the export, "google/<address>": the catalog
	// keeps it as each photo's badge.
	Account string
	// Profile is whose photos they are: the library's folder they go in.
	Profile string
	// The timezone dates are filed in; nil means the machine's own.
	Location *time.Location
	// Again downloads every part once more, though all were unpacked: what
	// is already in the library stays, what is missing comes back.
	Again bool
}

// Result is what a fetch achieved.
type Result struct {
	Organized    library.Organized    `json:"organized"`
	Verification library.Verification `json:"verification"`
}

// Run brings the export into the library and checks it against the manifest.
//
// The manifest comes first. It is small, and it says what the export holds,
// year by year: whoever watches is told how much of each year is here from
// the first minute, and a year is complete the moment its last photo has
// arrived, whatever order Google's parts come in.
//
// Parts are taken one at a time: downloaded, unpacked, recorded, and only
// then deleted. After each one, the photos that came with Google's own date
// for them are placed in the library: it fills up while the download goes
// on, and whoever watches sees photos, not a promise of them. Nothing proves
// that a photo's sidecar travels in the same part as the photo, so one that
// has not got its sidecar yet waits for it, in the library's unassigned
// folder, where it can be seen, and is filed when the sidecar arrives; a
// photo that never gets one is filed at the end, by what else is known.
// Placing is a rename on the same disk, so the peak stays about the size of
// the export plus one part.
//
// Every step can be interrupted and run again: a part resumes from its last
// byte, an unpacked part is not unpacked twice, and the organiser finishes any
// move it had recorded.
func (f Fetch) Run(ctx context.Context, g download.Getter, emit progress.Func) (Result, error) {
	work := filepath.Join(f.Library, library.WorkDir, f.Target.Job)
	parts := filepath.Join(work, "parts")
	unpacked := filepath.Join(work, "unpacked")

	st, err := loadState(work)
	if err != nil {
		return Result{}, err
	}
	if f.Again {
		st.Unpacked, st.Checked = map[string]bool{}, nil
	}
	// The room is measured on the library's disk, and what is here counted in
	// the profile's folder, so both are made first, even for a first download
	// into a library or a profile not made yet; the takeout's own folder and
	// state wait for the check, and a download refused for room leaves
	// nothing of it.
	if err := os.MkdirAll(filepath.Join(f.Library, f.Profile), 0o755); err != nil {
		return Result{}, err
	}
	if err := f.checkSpace(st); err != nil {
		return Result{}, err
	}
	for _, dir := range []string{parts, unpacked} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return Result{}, err
		}
	}
	// Before anything comes, the profile is named, even when the manifest is
	// already here from reading what the takeout holds: a begun download is
	// told by it (begun), and the check counts in its folder.
	if st.Profile == "" || f.Again {
		st.Profile = f.Profile
		if err := st.save(work); err != nil {
			return Result{}, err
		}
	}

	// The manifest is one more archive, and counted as one.
	of := len(f.Export.Parts) + 1
	if st.complete(f.Export) {
		emit.Emit(progress.Event{Stage: progress.InLibrary, Name: f.Export.Job, Of: of})
	} else {
		emit.Emit(progress.Event{Stage: progress.Ready, Name: f.Export.Job, Of: of, Total: f.Export.Bytes})
	}
	// The manifest is the first archive of the count, the parts the rest.
	if st.Manifest == "" {
		if err := download.Part(ctx, g, f.Target, f.Export.Manifest, 1, of, work, emit); err != nil {
			return Result{}, err
		}
		st.Manifest = f.Export.Manifest.Filename
		if err := st.save(work); err != nil {
			return Result{}, err
		}
	}
	manifest, err := library.ParseManifestZip(filepath.Join(work, st.Manifest))
	if err != nil {
		// Nothing can be counted against a manifest that does not open: it
		// is fetched once more, and a second failure stops the run.
		if err := os.Remove(filepath.Join(work, st.Manifest)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return Result{}, err
		}
		if err := download.Part(ctx, g, f.Target, f.Export.Manifest, 1, of, work, emit); err != nil {
			return Result{}, err
		}
		if manifest, err = library.ParseManifestZip(filepath.Join(work, st.Manifest)); err != nil {
			return Result{}, err
		}
	}
	home := filepath.Join(f.Library, f.Profile)
	years, err := newYears(home, manifest)
	if err != nil {
		return Result{}, err
	}
	years.report(emit)

	catalog, err := library.OpenCatalog(filepath.Join(f.Library, library.WorkDir, "catalog.db"))
	if err != nil {
		return Result{}, err
	}
	defer catalog.Close()

	// place moves what has been unpacked into the library: what came with its
	// sidecar, or, once every part is here, all that is left. What still
	// waits is shown in the library's unassigned folder, so that the library
	// is never empty while the download is not.
	var organized library.Organized
	place := func(all bool) error {
		items, err := library.ReadTakeout(unpacked)
		if err != nil {
			return err
		}
		// Arrived is arrived, filed or not: the year a photo belongs to is the
		// manifest's, and does not wait for anything.
		years.arrived(items, emit)
		var waiting []library.Item
		if !all {
			ready := items[:0:0]
			for _, item := range items {
				if item.Capture.Source == library.FromSidecar {
					ready = append(ready, item)
				} else {
					waiting = append(waiting, item)
				}
			}
			items = ready
		}
		if len(items) > 0 {
			placed, err := library.Organize(items, f.Library, catalog, library.Options{Location: f.Location, Account: f.Account, Profile: f.Profile, SeparateMessaging: true}, emit)
			organized.Add(placed)
			if err != nil {
				return err
			}
		}
		shown, err := library.ShowUnassigned(home, waiting)
		emit.Emit(progress.Event{Stage: progress.Unassigned, N: shown})
		return err
	}
	// What a run that was stopped left unpacked.
	if err := place(false); err != nil {
		return Result{}, err
	}

	for i, part := range f.Export.Parts {
		path := filepath.Join(parts, part.Filename)
		if !st.Unpacked[part.Filename] {
			get := func() error { return download.Part(ctx, g, f.Target, part, i+2, of, parts, emit) }
			unpack := func() error {
				emit.Emit(progress.Event{Stage: progress.Unpack, N: i + 2, Of: of, Name: part.Filename})
				_, err := library.UnpackPart(path, unpacked)
				return err
			}
			if err := get(); err != nil {
				return Result{}, err
			}
			if err := unpackOrAgain(path, unpack, get, func() {
				emit.Emit(progress.Event{Stage: progress.Damaged, N: i + 2, Of: of, Name: part.Filename})
			}); err != nil {
				return Result{}, err
			}
			st.Unpacked[part.Filename] = true
			if err := st.save(work); err != nil {
				return Result{}, err
			}
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return Result{}, err
			}
			if err := place(false); err != nil {
				return Result{}, err
			}
		}
		// Deleted only once it is recorded as unpacked, and again on every
		// run, in case the last one stopped in between.
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return Result{}, err
		}
	}

	if err := place(true); err != nil {
		return Result{}, err
	}

	verification, err := library.Verify(home, manifest)
	if err != nil {
		return Result{}, err
	}
	// Kept, so the takeouts list says what of it is here without counting again.
	st.Checked = &checked{Declared: verification.Declared, Present: verification.Present}
	if err := st.save(work); err != nil {
		return Result{}, err
	}
	return Result{Organized: organized, Verification: verification}, nil
}

// years follows how much of each year of an export has arrived, for whoever
// watches the download. A photo counts for its year, the one the manifest
// lists it under, as soon as it is on this disk: filed in the library, or
// shown in unassigned while it waits for its date. The date decides where a
// photo is filed, not whether it is here.
type years struct {
	order    []string
	declared map[string]int
	of       map[string]string          // a photo's name -> its year
	here     map[string]map[string]bool // year -> the names that have arrived
}

// newYears reads the years off the manifest and what is already in the
// library off the disk: a run that carries on starts from where the last one
// stopped, not from nothing.
func newYears(libraryRoot string, manifest library.Manifest) (*years, error) {
	found, err := library.Verify(libraryRoot, manifest)
	if err != nil {
		return nil, err
	}
	missing := make(map[string]bool, len(found.Missing))
	for _, name := range found.Missing {
		missing[name] = true
	}
	y := &years{of: map[string]string{}, here: map[string]map[string]bool{}}
	y.declared, y.order = manifest.ByYear()
	for _, year := range y.order {
		y.here[year] = map[string]bool{}
	}
	for _, entry := range manifest.Media {
		year, ok := library.YearOf(entry)
		if !ok {
			continue
		}
		y.of[entry.Name] = year
		if !missing[entry.Name] {
			y.here[year][entry.Name] = true
		}
	}
	return y, nil
}

// report says where every year stands.
func (y *years) report(emit progress.Func) {
	for _, year := range y.order {
		emit.Emit(progress.Event{Stage: progress.Year, Name: year, N: len(y.here[year]), Of: y.declared[year]})
	}
}

// arrived counts the photos just unpacked, and says where each year they
// belong to now stands.
func (y *years) arrived(items []library.Item, emit progress.Func) {
	for _, item := range items {
		name := filepath.Base(item.Path)
		year, known := y.of[name]
		if !known || y.here[year][name] {
			continue
		}
		y.here[year][name] = true
		emit.Emit(progress.Event{Stage: progress.Year, Name: year, N: len(y.here[year]), Of: y.declared[year]})
	}
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
//
// id is the export's id or its first characters; empty, it is the only export
// downloaded into the library.
func Verify(libraryRoot, id string) (library.Verification, error) {
	if err := settle(libraryRoot); err != nil {
		return library.Verification{}, err
	}
	job, err := downloaded(libraryRoot, id)
	if err != nil {
		return library.Verification{}, err
	}
	work := filepath.Join(libraryRoot, library.WorkDir, job)
	st, err := loadState(work)
	if err != nil {
		return library.Verification{}, err
	}
	if st.Manifest == "" {
		return library.Verification{}, fmt.Errorf("export %s has not been fetched into %s", job, libraryRoot)
	}
	return verify(filepath.Join(libraryRoot, st.Profile), filepath.Join(work, st.Manifest))
}

// downloaded finds the export in the library whose id starts with id: each
// one fetched there has its own folder, with its state, under the work dir.
func downloaded(libraryRoot, id string) (string, error) {
	entries, err := os.ReadDir(filepath.Join(libraryRoot, library.WorkDir))
	if err != nil {
		return "", fmt.Errorf("no export has been downloaded into %s", libraryRoot)
	}
	var found []string
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), id) {
			continue
		}
		if st, ok := stateIn(filepath.Join(libraryRoot, library.WorkDir, e.Name())); ok && st.begun() {
			found = append(found, e.Name())
		}
	}
	switch {
	case len(found) == 1:
		return found[0], nil
	case len(found) == 0 && id == "":
		return "", fmt.Errorf("no export has been downloaded into %s", libraryRoot)
	case len(found) == 0:
		return "", fmt.Errorf("%w: %s, in %s", ErrNoSuchTakeout, id, libraryRoot)
	case id == "":
		return "", fmt.Errorf("%d exports are in %s: say which with --takeout", len(found), libraryRoot)
	}
	return "", fmt.Errorf("%s matches %d exports in %s: give more of it", id, len(found), libraryRoot)
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
	Unpacked map[string]bool `json:"unpacked"`          // part filename -> unpacked
	Manifest string          `json:"manifest"`          // manifest filename, once fetched
	Profile  string          `json:"profile,omitempty"` // the library's folder it went in
	Checked  *checked        `json:"checked,omitempty"` // the last count against the manifest
}

// checked is how many of the files the manifest lists were found on disk.
type checked struct {
	Declared int `json:"declared"`
	Present  int `json:"present"`
}

// begun tells a download from a takeout whose list of files was only read:
// that fetches its manifest alone and names no profile; a download names its
// profile before anything comes. The one rule for the list of takeouts, the
// CLI's question and verify.
func (st state) begun() bool {
	for _, done := range st.Unpacked {
		if done {
			return true
		}
	}
	return st.Profile != "" || st.Checked != nil
}

// stateIn reads the state kept in work, if a fetch or a reading kept one.
func stateIn(work string) (state, bool) {
	if _, err := os.Stat(filepath.Join(work, "state.json")); err != nil {
		return state{}, false
	}
	st, err := loadState(work)
	return st, err == nil
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

// complete reports whether every part of e is unpacked here and its manifest
// fetched: nothing is left to download.
func (st state) complete(e takeout.Export) bool {
	for _, part := range e.Parts {
		if !st.Unpacked[part.Filename] {
			return false
		}
	}
	return st.Manifest != ""
}

func (st state) save(work string) error {
	return writeJSON(filepath.Join(work, "state.json"), st)
}

// unpackOrAgain unpacks a downloaded part and, when its bytes turn out damaged,
// deletes it and downloads it once more. Again costs none of Google's five
// downloads: the URL is built, not reached through Takeout's redirect. A part
// damaged twice is not a fluke, and stops the run.
func unpackOrAgain(path string, unpack, download func() error, damaged func()) error {
	err := unpack()
	if !errors.Is(err, library.ErrDamaged) {
		return err
	}
	damaged()
	if err := os.Remove(path); err != nil {
		return err
	}
	if err := download(); err != nil {
		return err
	}
	return unpack()
}
