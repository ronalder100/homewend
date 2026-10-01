// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// Command homewend is the command-line shell over the engine. It parses flags,
// calls one engine function, and prints what happened: text for a person, or
// with --json one JSON object per line for a program.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/dustin/go-humanize"
	"golang.org/x/term"

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
	"login":    login,
	"logout":   logout,
	"get":      get,
	"takeouts": takeouts,
	"verify":   verify,
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
		out.line("signed_in", true, "%s", paint(os.Stdout, okColour, text["already signed in"]))
	} else {
		out.line("signed_in", true, "%s", paint(os.Stdout, okColour, text["signed in"]))
	}
	return exitOK
}

func get(args []string) int {
	flags := newFlags("get")
	profile := flags.String("profile", "", "browser profile directory (default: in the user's config directory)")
	year := flags.Int("year", 0, "the year whose photos to get (default: all of them)")
	libraryDir := flags.String("library", "", "library directory")
	takeoutID := flags.String("takeout", "", "the export to download, by its id")
	fresh := flags.Bool("new", false, "ask Google for a new export even if there is one")
	asJSON := flags.Bool("json", false, "one JSON object per line")
	flags.Parse(args)
	if missing := unset(map[string]string{"--library": *libraryDir}); missing != "" {
		fmt.Fprintf(os.Stderr, text["missing flags"]+"\n", missing)
		return exitError
	}
	g := engine.Get{Year: *year, Library: *libraryDir, Takeout: *takeoutID, New: *fresh}
	out := newPrinter(*asJSON)
	defer func() { out.close() }()

	sess, err := openSession(*profile)
	if err != nil {
		return out.fail(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	// A person at a terminal is asked before Google is asked for an export:
	// it takes hours. A script that runs get meant it.
	if !*asJSON && term.IsTerminal(int(os.Stdin.Fd())) {
		if _, err := engine.Login(ctx, sess, out.event); err != nil {
			return out.fail(err)
		}
		ask, err := g.WillAsk(sess)
		if err != nil {
			return out.fail(err)
		}
		if ask {
			out.close()
			if !confirmGet(g) {
				return exitOK
			}
			out = newPrinter(*asJSON)
		}
	}
	result, err := g.Run(ctx, sess, out.event)
	if err != nil {
		return out.fail(err)
	}
	out.organized(result.Organized)
	return out.finished(result.Verification)
}

// confirmGet says that get is about to ask Google for an export, and what
// that means, and asks to go on.
func confirmGet(g engine.Get) bool {
	what := text["all photos"]
	if g.Year != 0 {
		what = fmt.Sprintf(text["photos of"], g.Year)
	}
	fmt.Printf(text["get intro"], what)
	fmt.Print(paint(os.Stdout, accentColour, text["continue"]))
	answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true
	}
	fmt.Println(text["not asked"])
	return false
}

func logout(args []string) int {
	flags := newFlags("logout")
	profile := flags.String("profile", "", "browser profile directory (default: in the user's config directory)")
	flags.Parse(args)
	dir, err := profileDir(*profile)
	if err == nil {
		var was bool
		if was, err = engine.Logout(dir); err == nil {
			if was {
				fmt.Println(paint(os.Stdout, okColour, text["signed out"]))
			} else {
				fmt.Println(text["was not signed in"])
			}
			return exitOK
		}
	}
	fmt.Fprintln(os.Stderr, paint(os.Stderr, errColour, err.Error()))
	return exitError
}

// profileDir is profile, or the default profile when none is given.
func profileDir(profile string) (string, error) {
	if profile != "" {
		return profile, nil
	}
	return engine.DefaultProfile()
}

// openSession opens the session over profile, or over the default profile when
// none is given.
func openSession(profile string) (*session.Session, error) {
	dir, err := profileDir(profile)
	if err != nil {
		return nil, err
	}
	return session.New(dir)
}

func takeouts(args []string) int {
	flags := newFlags("takeouts")
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
	if _, err := engine.Login(ctx, sess, out.event); err != nil {
		return out.fail(err)
	}
	list, err := engine.Takeouts(sess)
	if err != nil {
		return out.fail(err)
	}
	if *asJSON {
		for _, t := range list {
			json.NewEncoder(os.Stdout).Encode(map[string]any{"takeout": t})
		}
		return exitOK
	}
	if len(list) == 0 {
		out.say("%s", text["no takeouts"])
		return exitOK
	}
	out.say("%s", text["takeouts header"])
	for _, t := range list {
		holds := text["holds unknown"]
		switch {
		case t.Known && t.Year != 0:
			holds = fmt.Sprintf(text["holds year"], t.Year)
		case t.Known:
			holds = text["holds all"]
		}
		until := ""
		if !t.Expires.IsZero() {
			until = t.Expires.Local().Format("2006-01-02 15:04")
		}
		row := fmt.Sprintf(text["takeout row"], t.ID, t.Created.Local().Format("2006-01-02 15:04"),
			size(t.Bytes), len(t.Parts), t.Status, until, holds)
		switch t.Status {
		case engine.Ready:
			row = paint(os.Stdout, okColour, row)
		case engine.Expired:
			row = paint(os.Stdout, faintColour, row)
		}
		out.say("%s", row)
	}
	return exitOK
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
	return blended(s)
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
	case progress.SessionReady:
		p.say("%s", text["session ready"])
	case progress.Request:
		p.say("%s", text["request"])
	case progress.Waiting:
		p.say("%s", text["waiting"])
		p.say("%s", p.notice(text["restart"]))
	case progress.InLibrary:
		p.say("%s", text["in library"])
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

// organized reports what was placed; when the export was already all in the
// library there is nothing to report, and the count that follows says it.
func (p printer) organized(o library.Organized) {
	if o.Placed == 0 && o.Skipped == 0 && !p.json {
		return
	}
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
			p.say("%s", paint(os.Stdout, errColour, fmt.Sprintf(text["missing file"], name)))
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
		p.say("")
		p.say("%s", paint(os.Stdout, okColour, text["complete"]))
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
	case errors.Is(err, engine.ErrNoSuchTakeout):
		message = fmt.Sprintf(text["no such takeout"], err)
	case errors.Is(err, engine.ErrExpired):
		message = fmt.Sprintf(text["expired"], err)
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
		fmt.Fprintln(os.Stderr, paint(os.Stderr, errColour, message))
	}
	return code
}
