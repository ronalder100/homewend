// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/dustin/go-humanize"

	"github.com/ronalder100/homewend/internal/engine"
	"github.com/ronalder100/homewend/internal/progress"
)

// printer shows the engine's events: as lines for a person, or with --json
// one object per line for a program. For a person the lines done stay and
// scroll like a log; the line of the step going on is redrawn in place under
// them, in a terminal.
type printer struct {
	json bool
	out  look
	live *tea.Program

	// What the events carry from one to the next.
	partStart time.Time // the part downloading began
	sortStart time.Time // sorting began
	what      string    // the photos asked for: "2025 photos"
	asked     time.Time // when Google was asked
	rate      float64   // bytes a second, smoothed over the current part
	lastAt    time.Time
	lastDone  int64
}

func newPrinter(json bool) *printer {
	if json {
		return &printer{json: true}
	}
	return &printer{out: lookFor(os.Stdout), live: startLive()}
}

// startLive draws the line going on, under the log, in a terminal; nil
// anywhere else, where a redrawn line is only noise.
func startLive() *tea.Program {
	if !coloured(os.Stdout) {
		return nil
	}
	// No input and no signal handler: Ctrl-C still reaches the command.
	p := tea.NewProgram(newStatus(), tea.WithInput(nil), tea.WithoutSignalHandler())
	go p.Run()
	return p
}

// close takes the line going on down. It can be called more than once.
func (p *printer) close() {
	if p.live != nil {
		p.live.Send(going{})
		p.live.Quit()
		p.live.Wait()
		p.live = nil
	}
}

// say prints one line that stays.
func (p *printer) say(line string) {
	if p.json {
		return
	}
	if p.live != nil {
		// An empty line printed above the live one is dropped: a space keeps it.
		if line == "" {
			line = " "
		}
		p.live.Println(line)
		return
	}
	fmt.Println(line)
}

// line prints a line for a person, or value as kind for a program.
func (p *printer) line(kind string, value any, line string) {
	if p.json {
		json.NewEncoder(os.Stdout).Encode(map[string]any{kind: value})
		return
	}
	p.say(line)
}

// now shows the step going on, in place of the one before; without a
// terminal it is said once, as a line.
func (p *printer) now(g going) {
	if p.json {
		return
	}
	if p.live != nil {
		p.live.Send(g)
		return
	}
	if g.bar < 0 && g.said != "" {
		fmt.Println(signGoing + " " + g.said)
	}
}

// signGoing stands for the spinner where nothing is redrawn.
const signGoing = "-"

func (p *printer) event(e progress.Event) {
	if p.json {
		json.NewEncoder(os.Stdout).Encode(map[string]any{"event": e})
		return
	}
	t, now := time.Now(), func(said, details string) going { return going{said: said, details: details, bar: -1} }
	switch e.Stage {
	case progress.Checking:
		p.now(now(text["checking"], ""))
	case progress.SignIn:
		p.now(now(text["sign in"], ""))
	case progress.SessionReady:
		p.now(now(text["session ready"], ""))
	case progress.Prepare:
		p.now(now(text["prepare"], ""))
	case progress.FirstDownload:
		p.now(now(text["password"], ""))
	case progress.Request:
		p.asked = t
		g := now(fmt.Sprintf(text["asking"], p.what), "")
		g.since = t
		p.now(g)
	case progress.Waiting:
		if !p.asked.IsZero() {
			p.say(p.out.done(fmt.Sprintf(text["asked"], p.what), fmt.Sprintf(text["asked at"], p.asked.Format("15:04"))))
			p.asked = time.Time{}
		}
		g := going{sign: signWaiting, said: text["preparing"], since: t, bar: -1, below: text["preparing hint"]}
		p.now(g)
	case progress.Ready, progress.InLibrary:
		details := fmt.Sprintf(text["found details"], e.Of, size(e.Total))
		if e.Stage == progress.InLibrary {
			details = text["in library"]
		}
		p.say(p.out.done(fmt.Sprintf(text["found"], engine.ShortID(e.Name)), details))
	case progress.Download:
		p.partStart, p.rate, p.lastAt, p.lastDone = t, 0, t, e.Done
		p.now(p.downloading(e))
	case progress.Receiving:
		if dt := t.Sub(p.lastAt).Seconds(); dt > 0 {
			seen := float64(e.Done-p.lastDone) / dt
			if p.rate == 0 {
				p.rate = seen
			} else {
				p.rate = 0.7*p.rate + 0.3*seen
			}
		}
		p.lastAt, p.lastDone = t, e.Done
		p.now(p.downloading(e))
	case progress.Downloaded:
		took := t.Sub(p.partStart).Round(time.Second)
		p.say(p.out.done(fmt.Sprintf(text["downloaded"], e.N, e.Of), fmt.Sprintf(text["downloaded in"], size(e.Total), took)))
	case progress.Retry:
		p.now(going{sign: signWaiting, said: text["network"], details: fmt.Sprintf(text["network try"], e.N), bar: -1})
	case progress.Damaged:
		p.say(p.out.failed(fmt.Sprintf(text["damaged"], e.N, e.Of), text["damaged again"]))
	case progress.Unpack:
		p.now(now(fmt.Sprintf(text["unpacking"], e.N, e.Of), ""))
	case progress.Place:
		if p.sortStart.IsZero() {
			p.sortStart = t
		}
		p.now(going{said: text["sorting"], bar: fraction(int64(e.N), int64(e.Of)), details: fmt.Sprintf("%d of %d", e.N, e.Of)})
	case progress.Update:
		p.now(going{said: fmt.Sprintf(text["updating"], e.Name), bar: fraction(e.Done, e.Total),
			details: fmt.Sprintf(text["download sizes"], size(e.Done), size(e.Total))})
	}
}

// downloading is the line of a part coming down: its bar, how much, and how
// long is left once the pace is known.
func (p *printer) downloading(e progress.Event) going {
	details := fmt.Sprintf(text["download sizes"], size(e.Done), size(e.Total))
	if p.rate > 0 {
		left := time.Duration(float64(e.Total-e.Done) / p.rate * float64(time.Second))
		details += fmt.Sprintf(text["download left"], left.Round(time.Second))
	}
	return going{said: fmt.Sprintf(text["downloading"], e.N, e.Of), bar: fraction(e.Done, e.Total), details: details}
}

// going is the step going on: a spinner or a sign, what is said, a bar when
// there is something to count (bar ≥ 0), details, the time since it began
// when that is worth counting, and a line under it. The zero value clears it.
type going struct {
	sign    string
	said    string
	bar     float64
	details string
	since   time.Time
	below   string
}

type statusLine struct {
	out  look
	spin spinner.Model
	now  going
}

func newStatus() statusLine {
	return statusLine{out: lookFor(os.Stdout), spin: spinner.New(spinner.WithSpinner(spinner.MiniDot))}
}

func (s statusLine) Init() tea.Cmd { return s.spin.Tick }

func (s statusLine) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		s.spin, cmd = s.spin.Update(msg)
		return s, cmd
	case going:
		s.now = msg
	}
	return s, nil
}

func (s statusLine) View() tea.View { return tea.NewView(s.line(time.Now())) }

// line draws the step going on.
func (s statusLine) line(t time.Time) string {
	g := s.now
	if g.said == "" {
		return ""
	}
	l := s.out
	details := g.details
	if !g.since.IsZero() {
		details += fmt.Sprintf(text["elapsed"], t.Sub(g.since).Round(time.Second))
	}
	var line string
	switch {
	case g.sign != "":
		line = l.step(g.sign, warnColour, g.said, details)
	case g.bar >= 0:
		line = l.paint(s.spin.View(), accentColour, false) + " " + g.said + "  " + l.bar(g.bar) + "  " + l.paint(details, mutedColour, false)
	default:
		line = l.paint(s.spin.View(), accentColour, false) + " " + g.said + l.paint(details, mutedColour, false)
	}
	if g.below != "" {
		line += "\n" + l.cause(g.below)
	}
	return line
}

// barWidth is the cells of a bar: room for the line around it at 80 columns.
const barWidth = 20

// bar draws how much is done, in the accent, over what is left.
func (l look) bar(done float64) string {
	n := int(done*barWidth + 0.5)
	n = max(0, min(barWidth, n))
	return l.paint(strings.Repeat("━", n), accentColour, false) + l.paint(strings.Repeat("━", barWidth-n), lineColour, false)
}

func fraction(done, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(done) / float64(total)
}

func size(n int64) string { return humanize.IBytes(uint64(n)) }
