// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/spinner"
	"github.com/charmbracelet/x/ansi"
	"github.com/dustin/go-humanize"
	"golang.org/x/term"

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
	live *liveLine

	// What the events carry from one to the next.
	partStart time.Time // the part downloading began
	sortStart time.Time // sorting began
	what      string    // the photos asked for: "2025 photos"
	signedIn  bool      // the sign-in is said: checking it again is not news
	shown     string    // the export in the table: finding it is not news
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

// close takes the line going on down. It can be called more than once.
func (p *printer) close() {
	if p.live != nil {
		p.live.stop()
		p.live = nil
	}
}

// say prints one line that stays.
func (p *printer) say(line string) {
	if p.json {
		return
	}
	if p.live != nil {
		p.live.println(line)
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
		p.live.set(g)
		return
	}
	if g.bar < 0 && g.said != "" {
		fmt.Println(signGoing + " " + p.out.marked(g.said, nil))
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
		if !p.signedIn {
			p.now(now(text["checking"], ""))
		}
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
		g := now(text["asking"], "")
		g.since = t
		p.now(g)
	case progress.Waiting:
		if !p.asked.IsZero() {
			p.say(p.out.done(text["asked"], fmt.Sprintf(text["asked at"], p.asked.Format("15:04"))))
			p.asked = time.Time{}
		}
		g := going{sign: signWaiting, said: text["preparing"], since: t, bar: -1, below: text["preparing hint"]}
		p.now(g)
	case progress.Ready, progress.InLibrary:
		if e.Name == p.shown {
			if e.Stage == progress.InLibrary {
				p.say(p.out.done(text["in library now"], ""))
			}
			break
		}
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

// liveLine draws the step going on at the bottom of a terminal and redraws
// it in place: carriage return, erase, draw again, and the spinner turned ten
// times a second. Lines printed meanwhile go above it and stay.
type liveLine struct {
	mu    sync.Mutex
	out   look
	now   going
	frame int
	drawn int // lines on screen now
	quit  chan struct{}
	done  chan struct{}
}

// startLive draws the line going on, in a terminal; nil anywhere else, where
// a redrawn line is only noise.
func startLive() *liveLine {
	if !coloured(os.Stdout) {
		return nil
	}
	l := &liveLine{out: lookFor(os.Stdout), quit: make(chan struct{}), done: make(chan struct{})}
	fmt.Print(ansi.HideCursor)
	go l.spin()
	return l
}

// frames are the spinner's: braille dots, as gh, uv and flyctl turn theirs.
var frames = spinner.MiniDot.Frames

func (l *liveLine) spin() {
	defer close(l.done)
	tick := time.NewTicker(spinner.MiniDot.FPS)
	defer tick.Stop()
	for {
		select {
		case <-l.quit:
			return
		case <-tick.C:
			l.mu.Lock()
			l.frame++
			l.draw()
			l.mu.Unlock()
		}
	}
}

func (l *liveLine) set(g going) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.now = g
	l.draw()
}

func (l *liveLine) println(line string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.erase()
	fmt.Println(line)
	l.draw()
}

func (l *liveLine) stop() {
	close(l.quit)
	<-l.done
	l.mu.Lock()
	defer l.mu.Unlock()
	l.erase()
	fmt.Print(ansi.ShowCursor)
}

// erase takes the line going on off the screen, the cursor left where it
// began.
func (l *liveLine) erase() {
	if l.drawn == 0 {
		return
	}
	s := "\r"
	if l.drawn > 1 {
		s += ansi.CursorUp(l.drawn - 1)
	}
	fmt.Print(s + ansi.EraseScreenBelow)
	l.drawn = 0
}

// draw puts the line going on on the screen, each row cut to the window so
// that none wraps and the count of rows stays true.
func (l *liveLine) draw() {
	l.erase()
	text := l.line(time.Now())
	if text == "" {
		return
	}
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		width = 80
	}
	rows := strings.Split(text, "\n")
	for i, r := range rows {
		rows[i] = ansi.Truncate(r, width-1, "")
	}
	fmt.Print(strings.Join(rows, "\n"))
	l.drawn = len(rows)
}

// line draws the step going on.
func (s *liveLine) line(t time.Time) string {
	g := s.now
	if g.said == "" {
		return ""
	}
	l := s.out
	spin := frames[s.frame%len(frames)]
	details := g.details
	if !g.since.IsZero() {
		details += fmt.Sprintf(text["elapsed"], t.Sub(g.since).Round(time.Second))
	}
	var line string
	switch {
	case g.sign != "":
		line = l.step(g.sign, warnColour, g.said, details)
	case g.bar >= 0:
		line = l.paint(spin, accentColour, false) + " " + l.marked(g.said, nil) + "  " + l.bar(g.bar) + "  " + l.paint(details, mutedColour, false)
	default:
		line = l.paint(spin, accentColour, false) + " " + l.marked(g.said, nil) + l.paint(details, mutedColour, false)
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
