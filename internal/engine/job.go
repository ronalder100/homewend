// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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
	// counts are the accounts' photos being counted on Google (see Count).
	counts map[string]*counting
	// File, when set, keeps the downloads under way, for Resume to start
	// again when the program next runs: quitting it stops them, and a person
	// told the download carries on expects it to. A pause, an end or a
	// failure takes one out.
	File  string
	saved map[string]Get
}

// JobsFile is where the app keeps the downloads under way.
func JobsFile() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "homewend", "jobs.json"), nil
}

// Resume starts again the downloads under way when the program last quit.
func (j *Jobs) Resume() {
	if j.File == "" {
		return
	}
	var saved map[string]Get
	data, err := os.ReadFile(j.File)
	if err != nil || json.Unmarshal(data, &saved) != nil {
		return
	}
	for account, g := range saved {
		// An account gone, or signed out, says so as any download does;
		// one that cannot even start is forgotten.
		if j.Start(account, g) != nil {
			j.mu.Lock()
			j.forget(account)
			j.mu.Unlock()
		}
	}
}

// keep and forget change the downloads under way; the caller holds mu.
func (j *Jobs) keep(account string, g Get) {
	if j.File == "" {
		return
	}
	if j.saved == nil {
		j.saved = map[string]Get{}
	}
	j.saved[account] = g
	writeJSON(j.File, j.saved)
}

func (j *Jobs) forget(account string) {
	if _, ok := j.saved[account]; !ok {
		return
	}
	delete(j.saved, account)
	writeJSON(j.File, j.saved)
}

// counting is one account's count, running or ended.
type counting struct {
	cancel context.CancelFunc
	done   chan struct{} // closed once it has let go of the browser profile
	err    error
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
	// "signed-out", "no-space" (Need and Free say how much), "expired",
	// "not-offered" (Google has no photos of the year; Offered, the years it has).
	Problem string `json:"problem,omitempty"`
	Offered []int  `json:"offered,omitempty"`
	Need    int64  `json:"need,omitempty"`
	Free    int64  `json:"free,omitempty"`
}

// problem names an error a person can act on.
func problem(err error, s *JobState) {
	var space NoSpaceError
	var notOffered *takeout.NotOfferedError
	switch {
	case errors.As(err, &notOffered):
		s.Problem = "not-offered"
		for _, e := range notOffered.Offered {
			if y, ok := takeout.YearOf(e.ID); ok {
				s.Offered = append(s.Offered, y)
			}
		}
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
	j.keep(account, g)
	// A count holds the account's browser profile, which one browser at a
	// time may open: the job begins once it has let go.
	var counted <-chan struct{} = ended
	if c := j.counts[account]; c != nil {
		counted = c.done
		select {
		case <-counted:
		default:
			jb.state.Stage = progress.Counting
		}
	}
	go func() {
		defer close(jb.done)
		select {
		case <-counted:
		case <-ctx.Done():
		}
		// Paused before the count let go: the job never ran.
		err := context.Canceled
		select {
		case <-counted:
			err = run(ctx, func(e progress.Event) {
				j.mu.Lock()
				jb.record(e)
				j.mu.Unlock()
			})
		default:
		}
		j.mu.Lock()
		jb.state.Running = false
		// Stopped, it is kept or forgotten by whoever stopped it.
		if !errors.Is(err, context.Canceled) {
			j.forget(account)
		}
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
	// A count of the account is stopped too: it holds the same profile.
	var waits []chan struct{}
	// Paused by the person: not started again next time.
	j.forget(account)
	if jb := j.jobs[account]; jb != nil {
		jb.cancel()
		waits = append(waits, jb.done)
	}
	if c := j.counts[account]; c != nil {
		c.cancel()
		waits = append(waits, c.done)
	}
	switch len(waits) {
	case 0:
		return ended
	case 1:
		return waits[0]
	}
	all := make(chan struct{})
	go func() {
		for _, w := range waits {
			<-w
		}
		close(all)
	}()
	return all
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
	for _, c := range j.counts {
		c.cancel()
		done = append(done, c.done)
	}
	j.mu.Unlock()
	for _, d := range done {
		<-d
	}
}

// Count counts the account's photos and videos on Google, by year, in the
// background, and keeps the count (LoadYearCounts). A count already running
// goes on; an account bringing photos home is not counted, its browser
// profile being in use.
func (j *Jobs) Count(account string) error {
	sess, err := AccountSession(account)
	if err != nil {
		return err
	}
	return j.count(account, func(ctx context.Context) error { return countYears(ctx, sess) })
}

func (j *Jobs) count(account string, run func(context.Context) error) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if jb := j.jobs[account]; jb != nil && jb.state.Running {
		return ErrJobRunning
	}
	if c := j.counts[account]; c != nil {
		select {
		case <-c.done:
		default:
			return nil
		}
	}
	if j.counts == nil {
		j.counts = map[string]*counting{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	c := &counting{cancel: cancel, done: make(chan struct{})}
	j.counts[account] = c
	go func() {
		err := run(ctx)
		j.mu.Lock()
		c.err = err
		j.mu.Unlock()
		cancel()
		close(c.done)
	}()
	return nil
}

// Counting says whether the account is being counted, and why the last count
// failed, if it did.
func (j *Jobs) Counting(account string) (running bool, err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	c := j.counts[account]
	if c == nil {
		return false, nil
	}
	select {
	case <-c.done:
		return false, c.err
	default:
		return true, nil
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
