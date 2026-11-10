// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"crypto/rand"
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
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

// ImportState is where the window's import stands.
type ImportState struct {
	Running bool   `json:"running"`
	Dir     string `json:"dir,omitempty"`
	// whose photos they are, the dates given and the groups chosen, for a
	// copy taken up again
	Profile string           `json:"profile,omitempty"`
	Redates []library.Redate `json:"redates,omitempty"`
	Choice  *library.Choice  `json:"choice,omitempty"`
	Link    bool             `json:"link,omitempty"` // hard links, not copies
	// Card is whether Dir was a card when the run began: one taken out is
	// waited for, and carries on by itself when it is back.
	Card bool `json:"card,omitempty"`
	// "plan" while the folder is read to know what comes in, "import" while
	// it is copied.
	Kind string `json:"kind,omitempty"`
	// The file being read or copied, N of Of; Done of Total its bytes.
	N     int    `json:"n"`
	Of    int    `json:"of"`
	Name  string `json:"name,omitempty"`
	Done  int64  `json:"done"`
	Total int64  `json:"total"`

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

// Plan reads dir in the background, to say what importing it into profile
// would bring.
func (im *Importing) Plan(dir, profile string, redates []library.Redate, choice *library.Choice) error {
	// Said to the caller before the card is read, not after.
	if !ValidProfile(profile) {
		return ErrBadProfile
	}
	// Kept by the folder as the look reads it, a link to it followed: the
	// path a thumbnail is asked by is made from it.
	real, err := filepath.Abs(dir)
	if err == nil {
		real, err = filepath.EvalSymlinks(real)
	}
	if err != nil {
		real = dir
	}
	im.mu.Lock()
	if im.hashesOf != real {
		im.hashes, im.hashesOf = library.Hashes{}, real
	}
	hashes := im.hashes
	im.mu.Unlock()
	return im.start(dir, profile, redates, choice, false, "plan", func(ctx context.Context, emit progress.Func, s *ImportState) error {
		plan, err := PlanImport(ctx, dir, library.PlanOptions{Profile: profile, Redates: redates, Hashes: hashes, Choice: choice}, emit)
		if err == nil {
			s.Plan = &plan
			s.CanLink = canLink(dir)
		}
		return err
	})
}

// Import copies dir into the library, into profile, in the background. The
// plan read before stays in the state, for the screens that show it.
func (im *Importing) Import(dir, profile string, link bool, redates []library.Redate, choice *library.Choice) error {
	// Said to the caller, not in the state: it is the page's mistake.
	if !ValidProfile(profile) {
		return ErrBadProfile
	}
	return im.start(dir, profile, redates, choice, link, "import", func(ctx context.Context, emit progress.Func, s *ImportState) error {
		done, err := Import(ctx, dir, profile, library.ImportOptions{Link: link, Redates: redates, Choice: choice}, emit)
		s.Imported = &done
		return err
	})
}

func (im *Importing) start(dir, profile string, redates []library.Redate, choice *library.Choice, link bool, kind string, run func(context.Context, progress.Func, *ImportState) error) error {
	// Looked at before the lock: a drive that does not answer must not
	// hold up the window asking where things stand.
	card := slices.ContainsFunc(Cards(), func(c Card) bool { return c.Path == dir })
	im.mu.Lock()
	defer im.mu.Unlock()
	if im.state.Running {
		return ErrImportRunning
	}
	ctx, cancel := context.WithCancel(context.Background())
	// The plan of the same folder is kept for the copy that follows it, and
	// what came in before a card went out for the screen that waits for it.
	// A new look makes a new plan: an old one is not shown for it.
	var plan *library.ImportPlan
	var before *library.Imported
	if im.state.Dir == dir {
		card = card || im.state.Card
		if kind == "import" {
			plan = im.state.Plan
		}
		if im.state.Problem == "taken-out" {
			before = im.state.Imported
		}
	}
	im.state = ImportState{Running: true, Dir: dir, Profile: profile, Redates: redates, Choice: choice, Link: link, Kind: kind, Plan: plan, Card: card}
	im.cancel, im.done = cancel, make(chan struct{})
	done := im.done
	go func() {
		defer close(done)
		var out ImportState
		err := run(ctx, func(e progress.Event) {
			im.mu.Lock()
			im.state.record(e)
			im.mu.Unlock()
		}, &out)
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
	case progress.Read, progress.Copy:
		// A new file starts at nothing; its bytes follow.
		if e.N != s.N {
			s.Done, s.Total = 0, 0
		}
		s.N, s.Of, s.Name = e.N, e.Of, e.Name
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
	return im.state
}
