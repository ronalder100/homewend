// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"crypto/rand"
	"errors"
	"io/fs"
	"log"
	"maps"
	"os"
	"path/filepath"
	"sync"

	"github.com/ronalder100/homewend/internal/library"
	"github.com/ronalder100/homewend/internal/progress"
)

// Importing runs the window's import in the background, one at a time: the
// look at what a folder holds, then the copy. The window asks where it
// stands every second, as it does for a download.
type Importing struct {
	mu    sync.Mutex
	state ImportState
	// cancel stops the run going on; done closes when it has returned.
	cancel context.CancelFunc
	done   chan struct{}
	// hashes are the files of hashesOf as the last look read them: a look
	// again, with dates given, reads nothing twice, and a file's thumbnail
	// is found by its content.
	hashes   library.Hashes
	hashesOf string
}

// ImportAsk is what the window asks of a look or a copy, as it sends it;
// kept in the state, for a copy taken up again.
type ImportAsk struct {
	Dir     string `json:"dir"`
	Profile string `json:"profile"` // whose photos they are
	library.Asked
	Link bool `json:"link,omitempty"` // hard links, not copies
}

// ImportState is where the window's import stands.
type ImportState struct {
	Running bool `json:"running"`
	ImportAsk
	// Card is whether Dir was a card when the run began: one taken out is
	// waited for, and carries on by itself when it is back.
	Card bool `json:"card,omitempty"`
	// "plan" while the folder is read to know what comes in, "import" while
	// it is copied.
	Kind string `json:"kind,omitempty"`
	// The file being read or copied, N of Of; Done of Total its bytes.
	// Stage says which: "read" (a copy reads first the dates that decide
	// its folders) or "copy".
	Stage string `json:"stage,omitempty"`
	N     int    `json:"n"`
	Of    int    `json:"of"`
	Name  string `json:"name,omitempty"`
	Done  int64  `json:"done"`
	Total int64  `json:"total"`
	// Filed are the files in the library and checked, by year and month;
	// year "" month 0, the undated. A copy carried on after its card came
	// out goes on counting.
	Filed map[string]map[int]int `json:"filed,omitempty"`

	Plan *library.ImportPlan `json:"plan,omitempty"`
	// CanLink is the folder on the library's disk: its photos can come in
	// as hard links, taking no room. Known once the look has ended.
	CanLink  bool              `json:"canLink,omitempty"`
	Imported *library.Imported `json:"imported,omitempty"`
	Stopped  bool              `json:"stopped,omitempty"` // by the person
	Error    string            `json:"error,omitempty"`
	// Problem names what stopped it when a person can act on it:
	// "taken-out" (the folder went away: a card taken out), "no-space" (Need
	// and Free say how much), "busy" (another run holds the library),
	// "library-gone", "unknown-name" or "ambiguous-name" (a date given to a
	// name the folder has not, or has twice), "no-link" (the disk refused a
	// hard link).
	Problem string `json:"problem,omitempty"`
	Need    int64  `json:"need,omitempty"`
	Free    int64  `json:"free,omitempty"`
}

// ErrImportRunning is a second import asked for while one runs.
var ErrImportRunning = errors.New("an import is already running")

// Plan reads a.Dir in the background, to say what importing it into
// a.Profile would bring.
func (im *Importing) Plan(a ImportAsk) error {
	// Said to the caller before the card is read, not after.
	if !ValidProfile(a.Profile) {
		return ErrBadProfile
	}
	return im.start(a, "plan", func(ctx context.Context, emit progress.Func, s *ImportState, hashes library.Hashes) error {
		plan, err := PlanImport(ctx, a.Dir, library.PlanOptions{Asked: a.Asked, Profile: a.Profile, Hashes: hashes}, emit)
		if err == nil {
			s.Plan = &plan
			s.CanLink = canLink(a.Dir)
		}
		return err
	})
}

// Again runs what ran last once more, as it was asked: a look or a copy
// whose card came back. The window does not ask it again: the engine keeps
// what was asked.
func (im *Importing) Again() error {
	im.mu.Lock()
	a, kind := im.state.ImportAsk, im.state.Kind
	im.mu.Unlock()
	if kind == "plan" {
		return im.Plan(a)
	}
	return im.Import(a)
}

// Import copies a.Dir into the library, into a.Profile, in the background.
// The plan read before stays in the state, for the screens that show it.
func (im *Importing) Import(a ImportAsk) error {
	// Said to the caller, not in the state: it is the page's mistake.
	if !ValidProfile(a.Profile) {
		return ErrBadProfile
	}
	// The dates the look read decide the groups again without a read.
	return im.start(a, "import", func(ctx context.Context, emit progress.Func, s *ImportState, hashes library.Hashes) error {
		how := library.ImportOptions{Asked: a.Asked, Link: a.Link, Hashes: hashes}
		done, err := Import(ctx, a.Dir, a.Profile, how, emit)
		s.Imported = &done
		return err
	})
}

func (im *Importing) start(a ImportAsk, kind string, run func(context.Context, progress.Func, *ImportState, library.Hashes) error) error {
	// Looked at before the lock: a drive that does not answer must not
	// hold up the window asking where things stand.
	_, card := cardAt(a.Dir)
	real, err := library.RealPath(a.Dir)
	if err != nil {
		real = a.Dir
	}
	im.mu.Lock()
	defer im.mu.Unlock()
	if im.state.Running {
		return ErrImportRunning
	}
	// What the last look read of the folder, for this look, this copy and
	// the thumbnails; another folder starts afresh. Kept by the folder as
	// the look reads it, a link to it followed: a thumbnail's path is made
	// from it.
	if im.hashesOf != real {
		im.hashes, im.hashesOf = library.Hashes{}, real
	}
	hashes := im.hashes
	ctx, cancel := context.WithCancel(context.Background())
	// The plan of the same folder is kept for the copy that follows it, and
	// what came in before a card went out for the screen that waits for it.
	// A new look makes a new plan: an old one is not shown for it.
	var plan *library.ImportPlan
	var before *library.Imported
	var filed map[string]map[int]int
	if im.state.Dir == a.Dir {
		card = card || im.state.Card
		if kind == "import" {
			plan = im.state.Plan
		}
		if im.state.Problem == "taken-out" {
			before, filed = im.state.Imported, im.state.Filed
		}
	}
	im.state = ImportState{Running: true, ImportAsk: a, Filed: filed, Kind: kind, Plan: plan, Card: card}
	im.cancel, im.done = cancel, make(chan struct{})
	done := im.done
	go func() {
		defer close(done)
		var out ImportState
		err := run(ctx, func(e progress.Event) {
			im.mu.Lock()
			im.state.record(e)
			im.mu.Unlock()
		}, &out, hashes)
		im.mu.Lock()
		defer im.mu.Unlock()
		s := &im.state
		s.Running = false
		if out.Plan != nil {
			s.Plan, s.CanLink = out.Plan, out.CanLink
		}
		s.Imported = out.Imported
		// Asked to carry on with the card still out: nothing new came in,
		// and what came in before is still what to show.
		if errors.As(err, new(FolderGoneError)) && before != nil && (out.Imported == nil || out.Imported.Placed == 0) {
			s.Imported = before
		}
		switch {
		case errors.Is(err, context.Canceled):
			s.Stopped = true
		case err != nil:
			log.Printf("import %s: %v", kind, err)
			s.Error = err.Error()
			s.Problem, s.Need, s.Free = importProblem(err)
		}
	}()
	return nil
}

func (s *ImportState) record(e progress.Event) {
	switch e.Stage {
	case progress.Filed:
		if s.Filed == nil {
			s.Filed = map[string]map[int]int{}
		}
		if s.Filed[e.Name] == nil {
			s.Filed[e.Name] = map[int]int{}
		}
		s.Filed[e.Name][e.N]++
	case progress.Read, progress.Copy:
		// A new file starts at nothing; its bytes follow.
		if e.N != s.N {
			s.Done, s.Total = 0, 0
		}
		s.Stage, s.N, s.Of, s.Name = e.Stage, e.N, e.Of, e.Name
		if e.Total > 0 {
			s.Done, s.Total = e.Done, e.Total
		}
	}
}

// importProblem names an error a person can act on.
func importProblem(err error) (problem string, need, free int64) {
	var gone FolderGoneError
	var noLibrary LibraryGoneError
	var space NoSpaceError
	var unknown library.UnknownNameError
	var ambiguous library.AmbiguousNameError
	switch {
	case errors.As(err, &gone):
		return "taken-out", 0, 0
	case errors.As(err, &space):
		return "no-space", space.Need, space.Free
	case errors.Is(err, ErrLibraryBusy):
		return "busy", 0, 0
	case errors.As(err, &noLibrary):
		return "library-gone", 0, 0
	case errors.As(err, &unknown):
		return "unknown-name", 0, 0
	case errors.As(err, &ambiguous):
		return "ambiguous-name", 0, 0
	case errors.Is(err, library.ErrNoLink):
		return "no-link", 0, 0
	}
	return "", 0, 0
}

// Thumbnail is the thumbnail of a file of the folder last looked at, by
// its path below it: made from the card's file, kept as the library's are,
// by content, so a photo that comes in has it already. Only a file the
// look read is shown: no other path of the disk is reached from here.
func (im *Importing) Thumbnail(ctx context.Context, path string) (string, error) {
	im.mu.Lock()
	// The hashes are written by a look while it runs.
	if im.state.Running && im.state.Kind == "plan" {
		im.mu.Unlock()
		return "", fs.ErrNotExist
	}
	src := filepath.Join(im.hashesOf, filepath.FromSlash(path))
	hash, ok := im.hashes.Hash(src)
	im.mu.Unlock()
	if !ok {
		return "", fs.ErrNotExist
	}
	return thumbnailOf(ctx, hash, func() (string, error) { return src, nil })
}

// canLink is whether the disk lets a file of dir have a second name in the
// library: tried on the first file found, the name taken away at once. The
// disks' ids would say it on Linux and the Mac, not on Windows; the link
// itself says it everywhere, a share included.
func canLink(dir string) bool {
	root, err := ChosenLibrary()
	if err != nil {
		return false
	}
	work := filepath.Join(root, library.WorkDir)
	if err := os.MkdirAll(work, 0o755); err != nil {
		return false
	}
	var first string
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			first = path
			return fs.SkipAll
		}
		return nil
	})
	if first == "" {
		return false
	}
	probe := filepath.Join(work, "link-probe-"+rand.Text())
	if err := os.Link(first, probe); err != nil {
		return false
	}
	os.Remove(probe)
	return true
}

// Stop ends the run going on between two files; what is in stays in. A
// copy waiting for its card is ended too: it is not carried on when the
// card, or another with its name, comes back.
func (im *Importing) Stop() <-chan struct{} {
	im.mu.Lock()
	defer im.mu.Unlock()
	if !im.state.Running && im.state.Problem == "taken-out" {
		im.state.Problem, im.state.Error, im.state.Stopped = "", "", true
	}
	if im.cancel == nil {
		return ended
	}
	im.cancel()
	return im.done
}

// State is where the import stands; none run is the zero state.
func (im *Importing) State() ImportState {
	im.mu.Lock()
	defer im.mu.Unlock()
	s := im.state
	// The run goes on counting in its own map while this one is written out.
	filed := map[string]map[int]int{}
	for year, months := range s.Filed {
		filed[year] = maps.Clone(months)
	}
	s.Filed = filed
	return s
}
