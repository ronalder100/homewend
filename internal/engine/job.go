// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/ronalder100/homewend/internal/progress"
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
	state  JobState
	// bytes of the parts already complete, for the total arrived so far
	partsBytes int64
}

// JobState is where a job stands.
type JobState struct {
	Running  bool           `json:"running"`
	Year     int            `json:"year"` // 0: everything
	Stage    string         `json:"stage,omitempty"`
	Export   string         `json:"export,omitempty"` // Google's job, once known
	Parts    int            `json:"parts"`            // complete
	Of       int            `json:"of"`               // in the export, manifest included
	Done     int64          `json:"done"`             // bytes arrived
	Total    int64          `json:"total"`            // bytes in the export
	Years    []YearProgress `json:"years"`
	Retry    string         `json:"retry,omitempty"` // why the network is being tried again
	Finished bool           `json:"finished,omitempty"`
	Error    string         `json:"error,omitempty"`
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
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.jobs == nil {
		j.jobs = map[string]*job{}
	}
	if old := j.jobs[account]; old != nil && old.state.Running {
		return ErrJobRunning
	}
	ctx, cancel := context.WithCancel(context.Background())
	jb := &job{cancel: cancel, state: JobState{Running: true, Year: g.Year, Years: []YearProgress{}}}
	j.jobs[account] = jb
	go func() {
		_, err := g.Run(ctx, sess, func(e progress.Event) {
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
		default:
			jb.state.Finished = true
		}
		j.mu.Unlock()
	}()
	return nil
}

// Stop pauses the account's job; Start carries on from where it stopped.
func (j *Jobs) Stop(account string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if jb := j.jobs[account]; jb != nil {
		jb.cancel()
	}
}

// State is where the account's job stands; nothing ran is the zero state.
func (j *Jobs) State(account string) JobState {
	j.mu.Lock()
	defer j.mu.Unlock()
	if jb := j.jobs[account]; jb != nil {
		s := jb.state
		s.Years = append([]YearProgress(nil), s.Years...)
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
		s.Of, s.Total = e.Of, e.Total
	case progress.Waiting:
		s.Export = e.Name
	case progress.Receiving, progress.Download:
		s.Done = jb.partsBytes + e.Done
	case progress.Downloaded:
		s.Parts++
		jb.partsBytes = s.Done
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
