// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"errors"
	"log"
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
}

// ImportState is where the window's import stands.
type ImportState struct {
	Running bool   `json:"running"`
	Dir     string `json:"dir,omitempty"`
	// whose photos they are, for a copy taken up again
	Profile string `json:"profile,omitempty"`
	// "plan" while the folder is read to know what comes in, "import" while
	// it is copied.
	Kind string `json:"kind,omitempty"`
	// The file being read or copied, N of Of; Done of Total its bytes.
	N     int    `json:"n"`
	Of    int    `json:"of"`
	Name  string `json:"name,omitempty"`
	Done  int64  `json:"done"`
	Total int64  `json:"total"`

	Plan     *library.ImportPlan `json:"plan,omitempty"`
	Imported *library.Imported   `json:"imported,omitempty"`
	Stopped  bool                `json:"stopped,omitempty"` // by the person
	Error    string              `json:"error,omitempty"`
	// Problem names what stopped it when a person can act on it:
	// "taken-out" (the folder went away: a card taken out), "no-space" (Need
	// and Free say how much), "busy" (another run holds the library),
	// "library-gone", "unknown-name" or "ambiguous-name" (a date given to a
	// name the folder has not, or has twice).
	Problem string `json:"problem,omitempty"`
	Need    int64  `json:"need,omitempty"`
	Free    int64  `json:"free,omitempty"`
}

// ErrImportRunning is a second import asked for while one runs.
var ErrImportRunning = errors.New("an import is already running")

// Plan reads dir in the background, to say what importing it into profile
// would bring.
func (im *Importing) Plan(dir, profile string, redates []library.Redate) error {
	// Said to the caller before the card is read, not after.
	if !ValidProfile(profile) {
		return ErrBadProfile
	}
	return im.start(dir, profile, "plan", func(ctx context.Context, emit progress.Func, s *ImportState) error {
		plan, err := PlanImport(ctx, dir, profile, redates, emit)
		if err == nil {
			s.Plan = &plan
		}
		return err
	})
}

// Import copies dir into the library, into profile, in the background. The
// plan read before stays in the state, for the screens that show it.
func (im *Importing) Import(dir, profile string, link bool, redates []library.Redate) error {
	// Said to the caller, not in the state: it is the page's mistake.
	if !ValidProfile(profile) {
		return ErrBadProfile
	}
	return im.start(dir, profile, "import", func(ctx context.Context, emit progress.Func, s *ImportState) error {
		done, err := Import(ctx, dir, profile, link, redates, emit)
		s.Imported = &done
		return err
	})
}

func (im *Importing) start(dir, profile, kind string, run func(context.Context, progress.Func, *ImportState) error) error {
	im.mu.Lock()
	defer im.mu.Unlock()
	if im.state.Running {
		return ErrImportRunning
	}
	ctx, cancel := context.WithCancel(context.Background())
	// The plan of the same folder is kept for the copy that follows it.
	var plan *library.ImportPlan
	if im.state.Dir == dir {
		plan = im.state.Plan
	}
	im.state = ImportState{Running: true, Dir: dir, Profile: profile, Kind: kind, Plan: plan}
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
			s.Plan = out.Plan
		}
		s.Imported = out.Imported
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
	}
	return "", 0, 0
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
