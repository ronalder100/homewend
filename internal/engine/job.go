// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/ronalder100/homewend/internal/progress"
	"github.com/ronalder100/homewend/internal/takeout"
)

// Jobs runs Get in the background, one per account, and keeps where each
// stands for a window that looks every second. Pausing cancels a job; running
// it again carries on where it stopped, as Get always does.
type Jobs struct {
	mu   sync.Mutex
	jobs map[string]*job
}

type job struct {
	cancel context.CancelFunc
	// closed once the job has returned, and let go of the account's browser
	// profile and of the library
	done  chan struct{}
	state JobState
	// bytes of the parts already complete, for the total arrived so far
	partsBytes int64
}

// JobState is where a job stands.
type JobState struct {
	Running bool           `json:"running"`
	Year    int            `json:"year"` // 0: everything
	Stage   string         `json:"stage,omitempty"`
	Export  string         `json:"export,omitempty"` // Google's job, once known
	Parts   int            `json:"parts"`            // complete
	Of      int            `json:"of"`               // in the export, manifest included
	Done    int64          `json:"done"`             // bytes arrived
	Total   int64          `json:"total"`            // bytes in the export
	Years   []YearProgress `json:"years"`
	Retry   string         `json:"retry,omitempty"` // why the network is being tried again
	// LastLooked is when /manage was last read while Google prepared it.
	LastLooked time.Time `json:"lastLooked,omitzero"`
	// Unassigned photos have arrived and wait for their date.
	Unassigned int    `json:"unassigned"`
	Finished   bool   `json:"finished,omitempty"`
	Error      string `json:"error,omitempty"`
	// Problem names what stopped the job when a person can act on it:
	// "signed-out", "no-space" (Need and Free say how much), "expired".
	Problem string `json:"problem,omitempty"`
	Need    int64  `json:"need,omitempty"`
	Free    int64  `json:"free,omitempty"`
}

// problem names an error a person can act on.
func problem(err error, s *JobState) {
	var space NoSpaceError
	switch {
	case errors.As(err, &space):
		s.Problem, s.Need, s.Free = "no-space", space.Need, space.Free
	case errors.Is(err, ErrNotSignedIn), errors.Is(err, takeout.ErrSignedOut):
		s.Problem = "signed-out"
	case errors.Is(err, ErrExpired), errors.Is(err, ErrExportGone):
		s.Problem = "expired"
	}
}

// YearProgress is how much of a year has arrived.
type YearProgress struct {
	Year    string `json:"year"`
	Arrived int    `json:"arrived"`
	Of      int    `json:"of"`
}

// ErrJobRunning is starting a job for an account that has one running.
var ErrJobRunning = errors.New("this account is already bringing photos home")

// Start runs g for the account in the background.
func (j *Jobs) Start(account string, g Get) error {
	sess, err := AccountSession(account)
	if err != nil {
		return err
	}
	return j.start(account, g, func(ctx context.Context, emit progress.Func) error {
		_, err := g.Run(ctx, sess, emit)
		return err
	})
}

func (j *Jobs) start(account string, g Get, run func(context.Context, progress.Func) error) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.jobs == nil {
		j.jobs = map[string]*job{}
	}
	if old := j.jobs[account]; old != nil && old.state.Running {
		return ErrJobRunning
	}
	ctx, cancel := context.WithCancel(context.Background())
	// The takeout asked for is named from the start: one already ready is
	// never waited for, and the page must still tell which one runs.
	jb := &job{cancel: cancel, done: make(chan struct{}), state: JobState{Running: true, Year: g.Year, Export: g.Takeout, Years: []YearProgress{}}}
	j.jobs[account] = jb
	go func() {
		defer close(jb.done)
		err := run(ctx, func(e progress.Event) {
			j.mu.Lock()
			jb.record(e)
			j.mu.Unlock()
		})
		j.mu.Lock()
		jb.state.Running = false
		switch {
		case errors.Is(err, context.Canceled):
		case err != nil:
			jb.state.Error = err.Error()
			problem(err, &jb.state)
		default:
			jb.state.Finished = true
		}
		j.mu.Unlock()
	}()
	return nil
}

// Stop pauses the account's job; Start carries on from where it stopped. The
// channel closes when the job has ended, for whoever must wait for it before
// deleting what it uses.
func (j *Jobs) Stop(account string) <-chan struct{} {
	j.mu.Lock()
	defer j.mu.Unlock()
	if jb := j.jobs[account]; jb != nil {
		jb.cancel()
		return jb.done
	}
	return ended
}

// ended is the end of a job that never ran.
var ended = func() chan struct{} {
	c := make(chan struct{})
	close(c)
	return c
}()

// StopAll stops every job and waits for each to end. A program that exits
// under a running job leaves its browser running, and the account's profile
// locked to the next run.
func (j *Jobs) StopAll() {
	j.mu.Lock()
	var done []chan struct{}
	for _, jb := range j.jobs {
		jb.cancel()
		done = append(done, jb.done)
	}
	j.mu.Unlock()
	for _, d := range done {
		<-d
	}
}

// State is where the account's job stands; nothing ran is the zero state.
func (j *Jobs) State(account string) JobState {
	j.mu.Lock()
	defer j.mu.Unlock()
	if jb := j.jobs[account]; jb != nil {
		s := jb.state
		s.Years = append([]YearProgress{}, s.Years...)
		return s
	}
	return JobState{Years: []YearProgress{}}
}

func (jb *job) record(e progress.Event) {
	s := &jb.state
	switch e.Stage {
	case progress.Retry:
		s.Retry = e.Note
		return
	case progress.Ready:
		s.Export, s.Of, s.Total = e.Name, e.Of, e.Total
	case progress.Waiting:
		s.Export = e.Name
		s.LastLooked = time.Now()
	case progress.Unassigned:
		s.Unassigned = e.N
	case progress.Receiving, progress.Download:
		s.Done = jb.partsBytes + e.Done
	// A part already here from a run before counts as one just downloaded,
	// so a job that carries on starts where the last one stopped.
	case progress.Downloaded, progress.Already:
		s.Parts++
		jb.partsBytes += e.Total
		s.Done = jb.partsBytes
	// The damaged part was counted when it arrived; it comes again.
	case progress.Damaged:
		s.Parts--
		jb.partsBytes -= e.Total
		s.Done = jb.partsBytes
	case progress.Year:
		jb.year(e)
	}
	s.Stage = e.Stage
	s.Retry = ""
}

func (jb *job) year(e progress.Event) {
	s := &jb.state
	for i := range s.Years {
		if s.Years[i].Year == e.Name {
			s.Years[i].Arrived, s.Years[i].Of = e.N, e.Of
			return
		}
	}
	s.Years = append(s.Years, YearProgress{Year: e.Name, Arrived: e.N, Of: e.Of})
	sort.Slice(s.Years, func(a, b int) bool { return s.Years[a].Year > s.Years[b].Year })
}
