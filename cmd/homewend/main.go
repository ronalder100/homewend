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
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/dustin/go-humanize"
	"golang.org/x/term"

	"github.com/ronalder100/homewend/internal/download"
	"github.com/ronalder100/homewend/internal/engine"
	"github.com/ronalder100/homewend/internal/library"
	"github.com/ronalder100/homewend/internal/progress"
	"github.com/ronalder100/homewend/internal/session"
	"github.com/ronalder100/homewend/internal/takeout"
	"github.com/ronalder100/homewend/internal/update"
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
	"update":   updateProgram,
	"verify":   verify,
}

func main() {
	if len(os.Args) < 2 {
		cleanScreen(nil)
		showHelp(os.Stderr, text["usage"])
		os.Exit(exitError)
	}
	name, args := os.Args[1], os.Args[2:]
	switch name {
	case "help", "-h", "--help":
		cleanScreen(args)
		os.Exit(help(args))
	case "version", "--version":
		fmt.Println(version)
		os.Exit(exitOK)
	}
	run, ok := commands[name]
	if !ok {
		fmt.Fprintf(os.Stderr, text["unknown command"], name)
		showHelp(os.Stderr, text["usage"])
		os.Exit(exitError)
	}
	code := run(args)
	if name != "update" {
		sayNewer()
	}
	os.Exit(code)
}

// sayNewer ends a command with one line when a newer release is out. Only
// for a person: a script reading the output is told nothing, and GitHub is
// not asked on its behalf.
func sayNewer() {
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		return
	}
	if newer := update.Newer(context.Background(), version); newer != "" {
		fmt.Fprintln(os.Stderr, paint(os.Stderr, mutedColour, fmt.Sprintf(text["newer"], newer)))
	}
}

// updateProgram replaces the running homewend with the latest release.
func updateProgram(args []string) int {
	flags := newFlags("update")
	asJSON := flags.Bool("json", false, "one JSON object per line")
	flags.Parse(args)
	out := newPrinter(*asJSON)
	defer out.close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	latest, err := update.Latest(ctx)
	if err != nil {
		return out.fail(err)
	}
	if latest == version {
		out.line("version", version, "%s", paint(os.Stdout, okColour, fmt.Sprintf(text["up to date"], version)))
		return exitOK
	}
	program, err := update.Program()
	if err != nil {
		return out.fail(err)
	}
	// The status line names the version that is arriving.
	arriving := func(e progress.Event) {
		e.Name = latest
		out.event(e)
	}
	if err := update.Install(ctx, program, arriving); err != nil {
		return out.fail(err)
	}
	out.line("version", latest, "%s", paint(os.Stdout, okColour, fmt.Sprintf(text["updated"], latest, version)))
	return exitOK
}

// cleanScreen starts a command at the top of an empty screen, for a person
// at a terminal: what it says is then the one thing to look at, not the last
// lines under everything that came before. A script, or anyone asking for
// JSON, gets nothing of it.
//
// Every command does it once it knows it has what it needs, but update: a
// bar and one line, added to what the user was doing. A command asked wrongly
// clears nothing either, and answers under what was typed.
func cleanScreen(args []string) {
	if !term.IsTerminal(int(os.Stdout.Fd())) || slices.Contains(args, "--json") || slices.Contains(args, "-json") {
		return
	}
	fmt.Print(ansi.CursorHomePosition + ansi.EraseEntireScreen)
}

// help prints the overview, or the page of one command.
func help(args []string) int {
	if len(args) == 0 {
		showHelp(os.Stdout, text["usage"])
		return exitOK
	}
	if _, ok := commands[args[0]]; !ok {
		fmt.Fprintf(os.Stderr, text["unknown command"], args[0])
		return exitError
	}
	showHelp(os.Stdout, text["help "+args[0]])
	return exitOK
}

// showHelp prints a help page: as it is written for a program, with room
// around it and its parts told apart for a person at a terminal.
func showHelp(out *os.File, page string) {
	if term.IsTerminal(int(out.Fd())) {
		page = "\n" + dressed(page) + "\n"
	}
	fmt.Fprintln(out, page)
}

// dressed colours a help page by what each line is: the name in bold, the
// headings in bold, what to type in the accent, the rest as it is. The pages
// are plain text in the strings table; this reads their shape, so a page
// written like the others is dressed like the others.
func dressed(page string) string {
	bold := lipgloss.NewStyle().Bold(true)
	accent := lipgloss.NewStyle().Foreground(accentColour)
	faint := lipgloss.NewStyle().Foreground(faintColour)
	lines := strings.Split(page, "\n")
	section := ""
	for i, line := range lines {
		indented := strings.HasPrefix(line, "  ")
		switch {
		case i == 0:
			name, what, _ := strings.Cut(line, " — ")
			lines[i] = bold.Render(name) + " — " + what
		case line == "":
		case !indented && strings.HasSuffix(line, ":"):
			section = line
			lines[i] = bold.Render(line)
		case !indented && strings.HasPrefix(line, "https://"):
			lines[i] = faint.Render(line)
		case !indented:
			section = ""
		case section == "Commands:" || section == "Flags:":
			// A term, a gap, what it means; a line that carries on has no term.
			word, meaning, found := strings.Cut(line[2:], "  ")
			if found && !strings.HasPrefix(line, "   ") {
				lines[i] = "  " + accent.Render(word) + "  " + meaning
			}
		default:
			lines[i] = accent.Render(line)
		}
	}
	return strings.Join(lines, "\n")
}

// newFlags is the flag set of one command, whose -h shows its help page.
func newFlags(name string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ExitOnError)
	flags.Usage = func() { showHelp(os.Stderr, text["help "+name]) }
	return flags
}

func login(args []string) int {
	flags := newFlags("login")
	profile := flags.String("profile", "", "browser profile directory (default: in the user's config directory)")
	fallback := flags.Bool("fallback", false, "sign in on Google Takeout's page, in a full browser window")
	asJSON := flags.Bool("json", false, "one JSON object per line")
	flags.Parse(args)
	cleanScreen(args)
	out := newPrinter(*asJSON)
	defer out.close()
	// What to do in the browser is the one thing to look at while it is
	// open, and login has no log above it to push away.
	out.centre()

	sess, err := openSession(*profile)
	if err != nil {
		return out.fail(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	signIn := engine.Login
	if *fallback {
		signIn = engine.LoginAtTakeout
	}
	account, already, err := signIn(ctx, sess, out.event)
	if err != nil {
		return out.fail(err)
	}
	// With the account, when Google's page names it: a person with two
	// accounts has to know whose photos these are. A sign-in brings it; for a
	// session already there it is the one kept at sign-in.
	if already {
		account = engine.Account(sess)
	}
	said := text["signed in"]
	switch {
	case already && account != "":
		said = fmt.Sprintf(text["already as"], account)
	case already:
		said = text["already signed in"]
	case account != "":
		said = fmt.Sprintf(text["signed in as"], account)
	}
	// The sign-in leaves the middle of the window before its result is said,
	// which then is the first line of an empty screen.
	out.clear()
	out.line("signed_in", true, "%s", paint(os.Stdout, okColour, said))
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
	if *libraryDir == "" {
		return askedWrongly(text["get needs library"], args)
	}
	cleanScreen(args)
	g := engine.Get{Year: *year, Library: *libraryDir, Takeout: *takeoutID, New: *fresh}
	out := newPrinter(*asJSON)
	defer func() { out.close() }()

	sess, err := openSession(*profile)
	if err != nil {
		return out.fail(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	// A person at a terminal is asked first: before Google is asked for a new
	// export, which takes it hours, and when one is already there, which may
	// not be the one they want. A script that runs get meant it; so did
	// whoever named the export, and a library that has begun one carries on.
	if !*asJSON && term.IsTerminal(int(os.Stdin.Fd())) {
		if _, _, err := engine.Login(ctx, sess, out.event); err != nil {
			return out.fail(err)
		}
		found, err := g.Existing(sess)
		if err != nil {
			return out.fail(err)
		}
		if found == nil || g.Takeout == "" && !found.Started {
			out.close()
			if found == nil && !confirmNew(g) {
				return exitOK
			}
			if found != nil && wantsNew(g, *found) {
				g.New = true
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

// what names the photos a get is for.
func what(g engine.Get) string {
	if g.Year != 0 {
		return fmt.Sprintf(text["photos of"], g.Year)
	}
	return text["all photos"]
}

// answer asks a question and reads the reply, in lower case.
func answer(question string) string {
	fmt.Print(paint(os.Stdout, accentColour, question))
	reply, _ := replies.ReadString('\n')
	return strings.ToLower(strings.TrimSpace(reply))
}

// replies is what the user types. One reader for every question: a second
// one would not see what the first had already read ahead.
var replies = bufio.NewReader(os.Stdin)

// confirmNew says that get is about to ask Google for a new export, and asks
// to go on. Enter is yes.
func confirmNew(g engine.Get) bool {
	switch answer(fmt.Sprintf(text["confirm new"], what(g))) {
	case "", "y", "yes":
		return true
	}
	fmt.Println(text["not asked"])
	return false
}

// wantsNew says that Google already has an export of these photos, which one,
// and asks which of two things to do, by number: download it, or ask for a
// new one. Enter downloads it; anything that is neither number is asked again.
func wantsNew(g engine.Get, found engine.Found) bool {
	number := func(n string) string { return paint(os.Stdout, accentColour, n) }
	fmt.Printf(text["have one"], what(g), found.ID, found.Created.Local().Format("2006-01-02 15:04"), size(found.Bytes), found.Status)
	fmt.Printf(text["option"], number("1"), text["download that"])
	fmt.Printf(text["option"], number("2"), text["ask for new"])
	fmt.Println()
	for {
		switch answer(text["choose"]) {
		case "", "1":
			return false
		case "2":
			return true
		}
	}
}

func logout(args []string) int {
	flags := newFlags("logout")
	profile := flags.String("profile", "", "browser profile directory (default: in the user's config directory)")
	flags.Parse(args)
	cleanScreen(args)
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
	cleanScreen(args)
	out := newPrinter(*asJSON)
	defer out.close()

	sess, err := openSession(*profile)
	if err != nil {
		return out.fail(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if _, _, err := engine.Login(ctx, sess, out.event); err != nil {
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
	takeoutID := flags.String("takeout", "", "the export to check against, by its id")
	asJSON := flags.Bool("json", false, "one JSON object per line")
	flags.Parse(args)
	if *libraryDir == "" {
		return askedWrongly(text["verify needs library"], args)
	}
	cleanScreen(args)
	out := printer{json: *asJSON}

	v, err := engine.Verify(*libraryDir, *takeoutID)
	if err != nil {
		return out.fail(err)
	}
	return out.verification(v)
}

// askedWrongly says what a command was missing and shows what to type
// instead: what the user typed, with what was missing added, in the accent at
// a terminal like everything there is to type.
func askedWrongly(said string, typed []string) int {
	given := ""
	for _, word := range typed {
		if strings.ContainsAny(word, " \t") {
			word = strconv.Quote(word)
		}
		given += word + " "
	}
	lines := strings.Split(fmt.Sprintf(said, given), "\n")
	if term.IsTerminal(int(os.Stderr.Fd())) {
		for i, line := range lines {
			if strings.HasPrefix(line, "  ") {
				lines[i] = lipgloss.NewStyle().Foreground(accentColour).Render(line)
			}
		}
	}
	fmt.Fprintln(os.Stderr, strings.Join(lines, "\n"))
	return exitError
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
	p.clear()
	if p.live != nil {
		p.live.Quit()
		p.live.Wait()
	}
}

// clear empties the status, so that what is said next is not said around it.
func (p printer) clear() {
	if p.live != nil {
		p.live.Send(idle{})
	}
}

// centre puts what the user is asked to do in the middle of the window, when
// there is a window.
func (p printer) centre() {
	if p.live != nil {
		p.live.Send(centre{})
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
		// In a terminal the status line says both, and takes them away.
		if p.live == nil {
			p.say("%s", text["sign in"])
		}
	case progress.SessionReady:
		if p.live == nil {
			p.say("%s", text["session status"])
		}
	case progress.Request:
		// In a terminal the status line says it, and counts.
		if p.live == nil {
			p.say("%s", text["request"])
		}
	case progress.Waiting:
		p.say("%s", text["waiting"])
		p.say("%s", p.notice(text["restart"]))
	case progress.InLibrary:
		p.say("%s", text["in library"])
	case progress.Ready:
		p.say(text["ready"], e.Of, size(e.Total))
	case progress.FirstDownload:
		if e.Name != "" {
			p.say(text["first download of"], e.Name)
		} else {
			p.say("%s", text["first download"])
		}
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
	case errors.Is(err, engine.ErrSignInClosed):
		code, message = exitSignIn, text["sign-in closed"]
	case errors.Is(err, engine.ErrSignInDeclined):
		code, message = exitSignIn, text["sign-in declined"]
	case errors.Is(err, engine.ErrSessionNotWritten), errors.Is(err, engine.ErrSessionNotAccepted):
		code, message = exitSignIn, fmt.Sprintf(text["session lost"], err)
	case errors.Is(err, engine.ErrNotSignedIn):
		code, message = exitSignIn, text["not signed in"]
	case errors.Is(err, update.ErrDamaged):
		message = text["update damaged"]
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
