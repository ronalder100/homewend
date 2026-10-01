// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// Command homewend is the command-line shell over the engine. It parses flags,
// calls one engine function, and prints what happened: text for a person, or
// with --json one JSON object per line for a program.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/dustin/go-humanize"

	"github.com/ronalder100/homewend/internal/download"
	"github.com/ronalder100/homewend/internal/engine"
	"github.com/ronalder100/homewend/internal/library"
	"github.com/ronalder100/homewend/internal/progress"
	"github.com/ronalder100/homewend/internal/session"
	"github.com/ronalder100/homewend/internal/takeout"
)

const (
	exitOK         = 0
	exitError      = 1
	exitIncomplete = 2
	exitSignIn     = 3
)

// version is set by the release build; a build from source says so.
var version = "dev"

// commands maps each subcommand to its shell. Every one has a help page,
// text["help <name>"].
var commands = map[string]func(args []string) int{
	"login":  login,
	"get":    get,
	"fetch":  fetch,
	"verify": verify,
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, text["usage"])
		os.Exit(exitError)
	}
	name, args := os.Args[1], os.Args[2:]
	switch name {
	case "help", "-h", "--help":
		os.Exit(help(args))
	case "version", "--version":
		fmt.Println(version)
		os.Exit(exitOK)
	}
	run, ok := commands[name]
	if !ok {
		fmt.Fprintf(os.Stderr, text["unknown command"], name)
		fmt.Fprintln(os.Stderr, text["usage"])
		os.Exit(exitError)
	}
	os.Exit(run(args))
}

// help prints the overview, or the page of one command.
func help(args []string) int {
	if len(args) == 0 {
		fmt.Println(text["usage"])
		return exitOK
	}
	if _, ok := commands[args[0]]; !ok {
		fmt.Fprintf(os.Stderr, text["unknown command"], args[0])
		return exitError
	}
	fmt.Println(text["help "+args[0]])
	return exitOK
}

// newFlags is the flag set of one command, whose -h shows its help page.
func newFlags(name string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ExitOnError)
	flags.Usage = func() { fmt.Fprintln(flags.Output(), text["help "+name]) }
	return flags
}

func login(args []string) int {
	flags := newFlags("login")
	profile := flags.String("profile", "", "browser profile directory (default: in the user's config directory)")
	asJSON := flags.Bool("json", false, "one JSON object per line")
	flags.Parse(args)
	out := newPrinter(*asJSON)
	defer out.close()

	sess, err := openSession(*profile)
	if err != nil {
		return out.fail(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	already, err := engine.Login(ctx, sess, out.event)
	if err != nil {
		return out.fail(err)
	}
	if already {
		out.line("signed_in", true, "%s", text["already signed in"])
	} else {
		out.line("signed_in", true, "%s", text["signed in"])
	}
	return exitOK
}

func get(args []string) int {
	flags := newFlags("get")
	profile := flags.String("profile", "", "browser profile directory (default: in the user's config directory)")
	year := flags.Int("year", 0, "the year whose photos to get (default: all of them)")
	libraryDir := flags.String("library", "", "library directory")
	asJSON := flags.Bool("json", false, "one JSON object per line")
	flags.Parse(args)
	if missing := unset(map[string]string{"--library": *libraryDir}); missing != "" {
		fmt.Fprintf(os.Stderr, text["missing flags"]+"\n", missing)
		return exitError
	}
	out := newPrinter(*asJSON)
	defer out.close()

	sess, err := openSession(*profile)
	if err != nil {
		return out.fail(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	result, err := engine.Get{Year: *year, Library: *libraryDir}.Run(ctx, sess, out.event)
	if err != nil {
		return out.fail(err)
	}
	out.organized(result.Organized)
	return out.finished(result.Verification)
}

// openSession opens the session over profile, or over the default profile when
// none is given.
func openSession(profile string) (*session.Session, error) {
	if profile == "" {
		var err error
		if profile, err = engine.DefaultProfile(); err != nil {
			return nil, err
		}
	}
	return session.New(profile)
}

func fetch(args []string) int {
	flags := newFlags("fetch")
	profile := flags.String("profile", "", "browser profile directory (default: in the user's config directory)")
	libraryDir := flags.String("library", "", "library directory")
	asJSON := flags.Bool("json", false, "one JSON object per line")
	flags.Parse(args)
	if missing := unset(map[string]string{"--library": *libraryDir}); missing != "" {
		fmt.Fprintf(os.Stderr, text["missing flags"]+"\n", missing)
		return exitError
	}
	out := newPrinter(*asJSON)
	defer out.close()

	sess, err := openSession(*profile)
	if err != nil {
		return out.fail(err)
	}
	export, err := engine.Latest(sess)
	if err != nil {
		return out.fail(err)
	}
	out.line("export", export, text["export"],
		export.Job, export.Created.Format(time.DateOnly), len(export.Parts), humanize.IBytes(uint64(export.Bytes)))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	user, err := engine.User(ctx, sess, export.Job, out.event)
	if err != nil {
		return out.fail(err)
	}

	var result engine.Result
	err = engine.Patiently(ctx, out.event, func() error {
		result, err = engine.Fetch{
			Target:  takeout.Target{Job: export.Job, User: user},
			Export:  export,
			Library: *libraryDir,
		}.Run(ctx, sess, out.event)
		return err
	})
	if err != nil {
		return out.fail(err)
	}
	out.organized(result.Organized)
	return out.finished(result.Verification)
}

func verify(args []string) int {
	flags := newFlags("verify")
	libraryDir := flags.String("library", "", "library directory")
	job := flags.String("job", "", "export id")
	asJSON := flags.Bool("json", false, "one JSON object per line")
	flags.Parse(args)
	if missing := unset(map[string]string{"--library": *libraryDir, "--job": *job}); missing != "" {
		fmt.Fprintf(os.Stderr, text["missing flags"]+"\n", missing)
		return exitError
	}
	out := printer{json: *asJSON}

	v, err := engine.Verify(*libraryDir, *job)
	if err != nil {
		return out.fail(err)
	}
	return out.verification(v)
}

func unset(flags map[string]string) string {
	var missing []string
	for name, value := range flags {
		if value == "" {
			missing = append(missing, name)
		}
	}
	return strings.Join(missing, ", ")
}

// printer shows engine output either as text or as JSON lines. In a
// terminal, text goes above a live status line.
type printer struct {
	json bool
	live *tea.Program
}

func newPrinter(json bool) printer {
	if json {
		return printer{json: true}
	}
	return printer{live: startLive()}
}

// close takes the status line down. It can be called more than once.
func (p printer) close() {
	if p.live != nil {
		p.live.Send(idle{})
		p.live.Quit()
		p.live.Wait()
	}
}

// notice colours s in a terminal, and leaves it plain anywhere else.
func (p printer) notice(s string) string {
	if p.live == nil {
		return s
	}
	return noticed.Render(s)
}

// say prints one line of text for a person.
func (p printer) say(format string, args ...any) {
	if p.live != nil {
		p.live.Printf(format, args...)
		return
	}
	fmt.Printf(format+"\n", args...)
}

func (p printer) line(kind string, value any, format string, args ...any) {
	if p.json {
		json.NewEncoder(os.Stdout).Encode(map[string]any{kind: value})
		return
	}
	p.say(format, args...)
}

func (p printer) event(e progress.Event) {
	if p.json {
		json.NewEncoder(os.Stdout).Encode(map[string]any{"event": e})
		return
	}
	if p.live != nil {
		p.live.Send(e)
	}
	switch e.Stage {
	case progress.SignIn:
		p.say("%s", text["sign in"])
	case progress.Request:
		p.say("%s", text["request"])
	case progress.Waiting:
		p.say("%s", text["waiting"])
		p.say("%s", p.notice(text["restart"]))
	case progress.Ready:
		p.say(text["ready"], e.Of, size(e.Total))
	case progress.FirstDownload:
		p.say("%s", text["first download"])
	case progress.Download:
		// In a terminal the bar says it.
		if p.live == nil {
			p.say(text["download"], e.N, e.Of, e.Name, size(e.Done), size(e.Total))
		}
	case progress.Downloaded:
		p.say(text["downloaded"], e.N, e.Of, size(e.Total))
	case progress.Short:
		p.say(text["short"], e.N, e.Of, e.Name, size(e.Done), size(e.Total))
	case progress.Retry:
		p.say(text["retry"], e.Note, e.N)
	case progress.Damaged:
		p.say(text["damaged"], e.N, e.Of)
	case progress.Unpack:
		p.say(text["unpack"], e.N, e.Of)
	case progress.Place:
		// One line per photo would bury everything else: every thousandth,
		// unless the bar is there to count them.
		if p.live == nil && (e.N == e.Of || e.N%1000 == 0) {
			p.say(text["place"], e.N, e.Of)
		}
	}
}

func (p printer) organized(o library.Organized) {
	p.line("organized", o, text["placed"],
		o.Placed, humanize.IBytes(uint64(o.Bytes)), o.Skipped, o.Undated, o.Linked+o.Copied)
}

func (p printer) verification(v library.Verification) int {
	p.line("verification", v, text["verified"], v.Declared, v.Present, len(v.Missing))
	if !p.json {
		for _, y := range v.Years {
			p.say(text["year"], y.Year, y.Present, y.Declared)
		}
		for _, name := range v.Missing {
			p.say(text["missing file"], name)
		}
	}
	if !v.Complete() {
		return exitIncomplete
	}
	return exitOK
}

// finished reports a download: the count, and when nothing is missing, that
// it is over.
func (p printer) finished(v library.Verification) int {
	code := p.verification(v)
	if code == exitOK && !p.json {
		p.say("%s", text["complete"])
	}
	return code
}

func (p printer) fail(err error) int {
	p.close()
	code, message := exitError, fmt.Sprintf(text["error"], err)
	var space engine.NoSpaceError
	var notOffered *takeout.NotOfferedError
	switch {
	case errors.As(err, &notOffered):
		if p.json {
			json.NewEncoder(os.Stdout).Encode(map[string]any{"error": err.Error(), "offered": notOffered.Offered, "exit": code})
			return code
		}
		var years, albums []string
		for _, e := range notOffered.Offered {
			if y, ok := takeout.YearOf(e.ID); ok {
				years = append(years, fmt.Sprint(y))
			} else {
				albums = append(albums, e.Name)
			}
		}
		message = fmt.Sprintf(text["not offered"], notOffered.Year, strings.Join(years, ", "), strings.Join(albums, "\n  "))
	case errors.Is(err, download.ErrSessionExpired):
		code, message = exitSignIn, text["session expired"]
	case errors.Is(err, engine.ErrNotSignedIn):
		code, message = exitSignIn, text["not signed in"]
	case errors.Is(err, context.Canceled):
		message = text["stopped"]
	case errors.Is(err, takeout.ErrFormChanged), errors.Is(err, engine.ErrNotRequested):
		message = fmt.Sprintf(text["not requested"], err)
	case errors.Is(err, engine.ErrNoExport):
		message = text["no export"]
	case errors.Is(err, engine.ErrNoDownload):
		message = text["no download"]
	case errors.As(err, &space):
		message = fmt.Sprintf(text["no space"], humanize.IBytes(uint64(space.Need)), humanize.IBytes(uint64(space.Free)))
	case errors.Is(err, session.ErrNoBrowser):
		message = text["no browser"]
	}
	if p.json {
		json.NewEncoder(os.Stdout).Encode(map[string]any{"error": message, "exit": code})
	} else {
		fmt.Fprintln(os.Stderr, message)
	}
	return code
}
