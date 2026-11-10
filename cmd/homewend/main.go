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
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"
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
	"import":  importCommand,
	"login":   login,
	"logout":  logout,
	"status":  status,
	"takeout": takeoutCommand,
	"ui":      ui,
	"update":  updateProgram,
	"verify":  verify,
}

func main() {
	if len(os.Args) < 2 {
		// A double-click on the .exe: ask instead of showing help in a
		// window about to close. Typed at a shell, it gets the help.
		if doubleClicked() {
			os.Exit(guided())
		}
		cleanScreen(nil)
		showHelp(os.Stdout, text["usage"])
		os.Exit(exitOK)
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
		failed(fmt.Sprintf(text["unknown command"], name), nil, text["unknown hint"])
		os.Exit(exitError)
	}
	code := run(args)
	if name != "update" {
		sayNewer()
	}
	os.Exit(code)
}

// typed is the command as the user typed it, to show and to repeat in a hint.
func typed() string {
	words := []string{"homewend"}
	for _, w := range os.Args[1:] {
		words = append(words, shellWord(w))
	}
	return strings.Join(words, " ")
}

// shellWord is w as it must be typed: in single quotes when it holds
// anything a shell would read, so a pasted hint does what it says.
func shellWord(w string) string {
	if !strings.ContainsAny(w, " \t\n$`\"'\\!*?;&|<>(){}[]#~") {
		return w
	}
	return "'" + strings.ReplaceAll(w, "'", `'\''`) + "'"
}

// sayNewer ends a command with a hint when a newer release is out. Only for
// a person: a script is told nothing, and GitHub is not asked on its behalf.
func sayNewer() {
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		return
	}
	if newer := update.Newer(context.Background(), version); newer != "" {
		fmt.Fprintln(os.Stderr, "\n"+lookFor(os.Stderr).hint(fmt.Sprintf(text["newer"], newer)))
	}
}

// begin clears the screen and shows the command as typed, for a person at a
// terminal: what it says is then the one thing to look at. A script, or
// anyone asking for JSON, gets nothing of it.
func begin(args []string) *printer {
	asJSON := slices.Contains(args, "--json") || slices.Contains(args, "-json")
	cleanScreen(args)
	if !asJSON {
		// Before the live line starts: what is printed through it before it
		// runs is lost.
		fmt.Println(lookFor(os.Stdout).header(typed()))
		fmt.Println()
	}
	return newPrinter(asJSON)
}

// cleanScreen starts at the top of an empty screen, in a terminal.
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
		failed(fmt.Sprintf(text["unknown command"], args[0]), nil, text["unknown hint"])
		return exitError
	}
	showHelp(os.Stdout, text["help "+args[0]])
	return exitOK
}

// showHelp prints a help page: as it is written for a program, its parts
// told apart for a person at a terminal.
func showHelp(out *os.File, page string) {
	if coloured(out) {
		page = dressed(lookFor(out), page) + "\n"
	}
	fmt.Fprintln(out, page)
}

// dressed draws a help page by the shape of each line: the name bold, a
// section heading as a heading, what to type bold, the rest as it is. The
// pages are plain text in the strings table, so a page written like the
// others is drawn like the others.
func dressed(l look, page string) string {
	lines := strings.Split(page, "\n")
	section := ""
	for i, line := range lines {
		indented := strings.HasPrefix(line, "  ")
		switch {
		case i == 0:
			name, what, _ := strings.Cut(line, " — ")
			lines[i] = l.paint(name, nil, true) + "  " + l.paint(what, mutedColour, false)
		case line == "":
		case !indented && strings.HasSuffix(line, ":"):
			section = line
			lines[i] = l.heading(strings.TrimSuffix(line, ":"))
		case !indented && strings.HasPrefix(line, "https://"):
			lines[i] = l.paint(line, faintColour, false)
		case !indented:
			section = ""
			if before, cmd, ok := strings.Cut(line, ": "); ok && strings.HasPrefix(cmd, "homewend ") {
				lines[i] = before + ": " + l.paint(cmd, nil, true)
			}
		case section == "Commands:" || section == "Flags:":
			// A term, a gap, what it means; a line that carries on has no term.
			word, meaning, found := strings.Cut(line[2:], "  ")
			if found && !strings.HasPrefix(line, "   ") {
				lines[i] = "  " + l.paint(word, nil, true) + "  " + l.paint(meaning, mutedColour, false)
			} else {
				lines[i] = l.paint(line, mutedColour, false)
			}
		default:
			lines[i] = l.paint(line, nil, true)
		}
	}
	return strings.Join(lines, "\n")
}

func newFlags(name string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ExitOnError)
	flags.Usage = func() { showHelp(os.Stderr, text["help "+name]) }
	return flags
}

// interrupted is the context of a command, done on Ctrl-C.
func interrupted() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt)
}

func login(args []string) int {
	flags := newFlags("login")
	accountFlag := flags.String("account", "", "the account to sign in to again, by address")
	another := flags.Bool("new", false, "sign in to another account")
	fallback := flags.Bool("fallback", false, "sign in on Google Takeout's page, in a full window")
	asJSON := flags.Bool("json", false, "one JSON object per line")
	flags.Parse(args)
	out := begin(args)
	defer out.close()
	ctx, stop := interrupted()
	defer stop()

	// The account signed in to: the one named, the only one there is, or a
	// new one when there is none or another was asked for.
	id := ""
	if !*another {
		a, err := pickAccount(ctx, *accountFlag, *asJSON)
		switch {
		case err == nil:
			id = a.ID
		case !errors.Is(err, engine.ErrNotSignedIn):
			return out.fail(err)
		}
	}
	fresh := id == ""
	if fresh {
		var err error
		if id, err = engine.NextAccount(); err != nil {
			return out.fail(err)
		}
	}
	sess, err := engine.AccountSession(id)
	if err != nil {
		return out.fail(err)
	}
	signIn := engine.Login
	if *fallback {
		signIn = engine.LoginAtTakeout
	}
	_, already, err := signIn(ctx, sess, out.event)
	if err != nil {
		return out.fail(err)
	}
	a, err := engine.Adopt(id)
	if err != nil {
		return out.fail(err)
	}
	out.now(going{})
	said := map[bool]string{false: "signed in", true: "already"}[already]
	if a.Email != "" {
		said = fmt.Sprintf(text[said+" as"], a.Email)
	} else {
		said = text[said]
	}
	out.line("signed_in", a, out.out.done(said, ""))
	if already {
		return exitOK
	}
	// A new account's photos go in the profile named after its address: a
	// question here would stand between the user and the first takeout.
	if fresh {
		if err := engine.SetProfile(a.ID, a.Profile); err != nil {
			return out.fail(err)
		}
		out.say(out.out.done(fmt.Sprintf(text["profile is"], a.Profile), ""))
	}
	year := time.Now().Year() - 1
	out.say("")
	out.say(out.out.hint(fmt.Sprintf(text["after login"], year, year)))
	return exitOK
}

func logout(args []string) int {
	flags := newFlags("logout")
	accountFlag := flags.String("account", "", "the account to sign out of, by address")
	flags.Parse(args)
	out := begin(args)
	defer out.close()
	ctx, stop := interrupted()
	defer stop()
	a, err := pickAccount(ctx, *accountFlag, false)
	if errors.Is(err, engine.ErrNotSignedIn) {
		out.say(out.out.stopped(text["was not signed"], ""))
		return exitOK
	}
	if err != nil {
		return out.fail(err)
	}
	if _, err := engine.Logout(a.Dir); err != nil {
		return out.fail(err)
	}
	out.say(out.out.done(fmt.Sprintf(text["signed out of"], a.Email), ""))
	return exitOK
}

// pickAccount is the account a command works with: the one named, by address,
// the only one signed in, or, for a person, one chosen from a list. With none
// signed in it is engine.ErrNotSignedIn.
func pickAccount(ctx context.Context, given string, asJSON bool) (engine.AccountInfo, error) {
	all, err := engine.Accounts()
	if err != nil {
		return engine.AccountInfo{}, err
	}
	var signed []engine.AccountInfo
	for _, a := range all {
		if a.Email != "" {
			signed = append(signed, a)
		}
	}
	if given != "" {
		for _, a := range signed {
			if a.Email == given || a.ID == given {
				return a, nil
			}
		}
		return engine.AccountInfo{}, fmt.Errorf("%w: %s", errNoSuchAccount, given)
	}
	switch {
	case len(signed) == 0:
		return engine.AccountInfo{}, engine.ErrNotSignedIn
	case len(signed) == 1:
		return signed[0], nil
	case !interactive(asJSON):
		return engine.AccountInfo{}, errWhichAccount
	}
	var names []string
	for _, a := range signed {
		names = append(names, a.Email)
	}
	at, err := choose(ctx, text["which account"], "", names)
	return signed[at], err
}

var (
	errNoSuchAccount = errors.New("no account signed in with that address")
	errWhichAccount  = errors.New("more than one account is signed in")
)

// pickProfile is the profile of a command's photos: the one named, the only
// one there is, or, for a person, one chosen from a list.
func pickProfile(ctx context.Context, given string, asJSON bool) (string, error) {
	if given != "" {
		return given, nil
	}
	accounts, err := engine.Accounts()
	if err != nil {
		return "", err
	}
	s, err := engine.LoadSettings()
	if err != nil {
		return "", err
	}
	names := engine.ProfileNames(accounts, s)
	switch {
	case len(names) == 1:
		return names[0], nil
	case len(names) == 0:
		return "", errNoProfile
	case !interactive(asJSON):
		return "", errWhichProfile
	}
	at, err := choose(ctx, text["which profile"], "", names)
	return names[at], err
}

var (
	errNoProfile    = errors.New("there is no profile yet")
	errWhichProfile = errors.New("there is more than one profile")
)

func status(args []string) int {
	flags := newFlags("status")
	asJSON := flags.Bool("json", false, "one JSON object")
	flags.Parse(args)
	out := begin(args)
	defer out.close()
	accounts, err := engine.Accounts()
	if err != nil {
		return out.fail(err)
	}
	library, err := engine.DefaultLibrary()
	if err != nil {
		return out.fail(err)
	}
	out.now(going{said: text["checking"], bar: -1})
	type state struct {
		engine.AccountInfo
		engine.State
	}
	var states []state
	for _, a := range accounts {
		if a.Email == "" {
			continue
		}
		sess, err := engine.AccountSession(a.ID)
		if err != nil {
			return out.fail(err)
		}
		st, err := engine.Now(sess)
		if err != nil {
			return out.fail(err)
		}
		states = append(states, state{a, st})
	}
	out.now(going{})
	newer := update.Newer(context.Background(), version)
	if *asJSON {
		json.NewEncoder(os.Stdout).Encode(map[string]any{"accounts": states, "library": library, "version": version, "newer": newer})
		return exitOK
	}
	l := out.out
	none := func(name, said string) string { return l.field(name, signStopped, mutedColour, said, "") }
	if len(states) == 0 {
		out.say(none(text["field google"], text["not signed in"]))
	}
	for _, st := range states {
		if st.SignedIn {
			out.say(l.field(text["field google"], signDone, okColour, st.Email, " · "+st.Profile))
		} else {
			out.say(l.field(text["field google"], signStopped, mutedColour, st.Email, text["sign-in expired"]))
		}
		switch e := st.Export; {
		case e == nil:
			out.say(none(text["field takeout"], text["not requested"]))
		default:
			details := fmt.Sprintf(text["export of"], e.ID, e.Status)
			switch e.Status {
			case engine.Ready:
				out.say(l.field(text["field takeout"], signDone, okColour, what(e.Year), " · "+details))
			case engine.Expired:
				out.say(l.field(text["field takeout"], signStopped, mutedColour, what(e.Year), " · "+details))
			default:
				out.say(l.field(text["field takeout"], signWaiting, warnColour, what(e.Year), " · "+details))
			}
		}
	}
	if library != "" {
		out.say(l.field(text["field library"], "", nil, library, ""))
	} else {
		out.say(none(text["field library"], text["not chosen"]))
	}
	if newer != "" {
		out.say(l.field(text["field version"], signWarning, warnColour, version, fmt.Sprintf(text["available"], newer)))
	} else {
		out.say(l.field(text["field version"], "", nil, version, text["latest"]))
	}
	if len(states) == 0 {
		out.say("")
		out.say(l.hint(text["status start"]))
	}
	return exitOK
}

// what names the photos of a year, or all of them.
func what(year int) string {
	if year != 0 {
		return fmt.Sprintf(text["photos of"], year)
	}
	return text["all photos"]
}

// interactive is whether a person can be asked: not with --json, and not when
// the input is not a terminal.
func interactive(asJSON bool) bool { return !asJSON && term.IsTerminal(int(os.Stdin.Fd())) }

func takeoutCommand(args []string) int {
	flags := newFlags("takeout")
	accountFlag := flags.String("account", "", "the account, by address, when there are several")
	profileFlag := flags.String("profile", "", "whose photos: the library's folder they go in")
	list := flags.Bool("list", false, "list the exports on Google Takeout")
	libraryDir := flags.String("library", "", "where the photos go, for this run")
	exportID := flags.String("export", "", "this export, by its id")
	fresh := flags.Bool("new", false, "ask Google for a new export even if there is one")
	yes := flags.Bool("yes", false, "answer yes to every question")
	asJSON := flags.Bool("json", false, "one JSON object per line")
	// The year may come before the flags or after them.
	flags.Parse(args)
	year := 0
	if rest := flags.Args(); len(rest) > 0 {
		y, err := strconv.Atoi(rest[0])
		if err != nil || len(rest[0]) != 4 {
			return failed(fmt.Sprintf(text["not a year"], rest[0]), nil, fmt.Sprintf(text["year hint"], time.Now().Year()-1))
		}
		year = y
		flags.Parse(rest[1:])
	}
	if *list {
		return takeoutList(args, *accountFlag, *asJSON)
	}

	ctx, stop := interrupted()
	defer stop()
	a, err := pickAccount(ctx, *accountFlag, *asJSON)
	if err != nil {
		return newPrinter(*asJSON).fail(err)
	}
	profile := a.Profile
	if *profileFlag != "" {
		profile = *profileFlag
		// Said before Google is asked anything, not after its questions.
		if !engine.ValidProfile(profile) {
			return newPrinter(*asJSON).fail(engine.ErrBadProfile)
		}
	}
	dir, code := libraryFor(ctx, *libraryDir, *asJSON)
	if dir == "" {
		return code
	}
	out := begin(args)
	defer func() { out.close() }()
	out.what = what(year)
	sess, err := engine.AccountSession(a.ID)
	if err != nil {
		return out.fail(err)
	}
	if _, _, err := engine.Login(ctx, sess, out.event); err != nil {
		return out.fail(err)
	}
	out.signedIn = true
	// Reading Takeout's list of exports takes seconds: say so, until there
	// is something to show.
	looking := going{said: fmt.Sprintf(text["looking"], out.what), bar: -1}
	out.now(looking)

	g := engine.Get{Year: year, Library: dir, Takeout: *exportID, New: *fresh, Profile: profile}
	var found *engine.Found
	if !g.New {
		if found, err = g.Existing(sess); err != nil {
			return out.fail(err)
		}
	}
	// A person is asked first: before Google is asked for a new export, which
	// takes it hours, and when one is already there, which may not be the one
	// they want. A script meant it; so did whoever named the export, and a
	// library that has begun one carries on.
	if interactive(*asJSON) && !*yes && !*fresh && (found == nil || g.Takeout == "" && !found.Started) {
		out.close()
		if found == nil {
			at, err := choose(ctx, fmt.Sprintf(text["ask new"], out.what), "", []string{text["yes"], text["no"]})
			if err != nil {
				return newPrinter(*asJSON).fail(err)
			}
			if at != 0 {
				fmt.Println(lookFor(os.Stdout).stopped(text["not asked"], ""))
				return exitOK
			}
		} else {
			detail := fmt.Sprintf(text["have one meta"], found.ID, found.Created.Local().Format("2 Jan 15:04"), size(found.Bytes)) + string(found.Status)
			at, err := choose(ctx, fmt.Sprintf(text["have one"], out.what), detail, []string{text["download it"], text["ask for new"]})
			if err != nil {
				return newPrinter(*asJSON).fail(err)
			}
			if g.New = at == 1; g.New {
				found = nil
			}
		}
		what := out.what
		out = newPrinter(*asJSON)
		out.what, out.signedIn = what, true
	}
	// The run, as a table: whose photos, which, and where they go. What
	// follows says what happens, not again what it happens to.
	out.now(going{})
	details := ""
	if found != nil {
		details = fmt.Sprintf(text["export details"], found.ID, len(found.Parts), size(found.Bytes))
		out.shown = found.Job
	}
	home, _ := os.UserHomeDir()
	out.say(out.out.param(text["field google"], a.Email, ""))
	out.say(out.out.param(text["field takeout"], unmarked(out.what), details))
	// Where the photos go: a takeout begun stays in the profile it began in.
	if found != nil {
		profile = found.Profile
	}
	out.say(out.out.param(text["field library"], tilde(filepath.Join(dir, profile), home), ""))
	out.say("")
	// The run reads Takeout's list again before it asks or downloads, for
	// seconds: the spinner starts now, not when that is over.
	if found == nil {
		out.asked = time.Now()
		out.now(going{said: text["asking"], bar: -1, since: out.asked, below: text["asking hint"]})
	} else {
		out.now(looking)
	}
	result, err := g.Run(ctx, sess, out.event)
	if err != nil {
		return out.fail(err)
	}
	out.now(going{})
	out.sorted(result.Organized)
	return out.checked(result.Verification, dir, year)
}

// libraryFor is the library folder of this run: the one given, or the one
// chosen once, or one asked now and kept. "" with the exit code when there is
// none and no one to ask.
func libraryFor(ctx context.Context, given string, asJSON bool) (string, int) {
	if given != "" {
		return given, exitOK
	}
	// The library kept from before: there, or said away.
	if kept, err := engine.LibraryHere(); err == nil {
		return kept, exitOK
	} else if !errors.Is(err, engine.ErrNoLibrary) {
		return "", newPrinter(asJSON).fail(err)
	}
	if !interactive(asJSON) {
		return "", failed(text["no library"], nil, fmt.Sprintf(text["no library hint"], typed()))
	}
	home, _ := os.UserHomeDir()
	proposed := filepath.Join(home, "Pictures", "Homewend")
	cleanScreen(nil)
	fmt.Println(lookFor(os.Stdout).header(typed()))
	fmt.Println()
	answer, err := field(ctx, text["ask library"], text["ask library meta"], tilde(proposed, home))
	if err != nil {
		return "", newPrinter(asJSON).fail(err)
	}
	if strings.HasPrefix(answer, "~/") {
		answer = filepath.Join(home, answer[2:])
	}
	if answer == "" {
		answer = proposed
	}
	if err := engine.ChooseLibrary(answer); err != nil {
		return "", newPrinter(asJSON).fail(err)
	}
	return answer, exitOK
}

// tilde writes path under home with ~, as a person types it.
func tilde(path, home string) string {
	if rel, err := filepath.Rel(home, path); err == nil && !strings.HasPrefix(rel, "..") {
		return "~/" + rel
	}
	return path
}

// takeoutList lists the exports on Google Takeout.
func takeoutList(args []string, account string, asJSON bool) int {
	ctx, stop := interrupted()
	defer stop()
	a, err := pickAccount(ctx, account, asJSON)
	if err != nil {
		return newPrinter(asJSON).fail(err)
	}
	out := begin(args)
	defer out.close()
	sess, err := engine.AccountSession(a.ID)
	if err != nil {
		return out.fail(err)
	}
	if _, _, err := engine.Login(ctx, sess, out.event); err != nil {
		return out.fail(err)
	}
	list, err := engine.Takeouts(sess)
	out.now(going{})
	if err != nil {
		return out.fail(err)
	}
	if asJSON {
		for _, t := range list {
			json.NewEncoder(os.Stdout).Encode(map[string]any{"takeout": t})
		}
		return exitOK
	}
	l, account := out.out, engine.Account(sess)
	if len(list) == 0 {
		out.say(l.summary(fmt.Sprintf(text["list none"], account)))
		out.say("")
		out.say(l.hint(fmt.Sprintf(text["list none hint"], time.Now().Year()-1)))
		return exitOK
	}
	out.say(l.marked(fmt.Sprintf(text["list title"], account), nil))
	out.say("")
	out.say(l.paint(text["list head"], faintColour, false))
	ready := ""
	for _, t := range list {
		holds := text["list unknown"]
		switch {
		case t.Known && t.Year != 0:
			holds = strconv.Itoa(t.Year)
		case t.Known:
			holds = text["list all"]
		}
		until := ""
		if !t.Expires.IsZero() {
			until = t.Expires.Local().Format("2 Jan")
		}
		row := fmt.Sprintf(text["list row"], t.ID, t.Created.Local().Format("2 Jan 15:04"), size(t.Bytes), len(t.Parts), t.Status, until, holds)
		switch t.Status {
		case engine.Ready:
			// Only the state is coloured, and only because it tells this
			// export apart from the others.
			s := string(t.Status)
			i := strings.Index(row, s)
			row = row[:i] + l.paint(s, okColour, false) + row[i+len(s):]
			if ready == "" && t.Known {
				ready = map[bool]string{true: " " + holds, false: ""}[t.Year != 0]
			}
		case engine.Expired:
			row = l.paint(row, faintColour, false)
		}
		out.say(row)
	}
	if ready != "" {
		out.say("")
		out.say(l.hint(fmt.Sprintf(text["list ready"], ready)))
	}
	return exitOK
}

func verify(args []string) int {
	flags := newFlags("verify")
	exportID := flags.String("export", "", "the export to count against, by its id")
	flags.Bool("json", false, "one JSON object per line")
	flags.Parse(args)
	dir := flags.Arg(0)
	if dir == "" {
		kept, err := engine.DefaultLibrary()
		if err != nil || kept == "" {
			return failed(text["no library"], nil, fmt.Sprintf(text["no library hint"], typed()))
		}
		dir = kept
	}
	out := begin(args)
	defer out.close()
	v, err := engine.Verify(dir, *exportID)
	if err != nil {
		return out.fail(err)
	}
	if out.json {
		json.NewEncoder(os.Stdout).Encode(map[string]any{"verification": v})
	} else {
		for _, y := range v.Years {
			out.say("  " + out.out.paint(y.Year, mutedColour, false) + "   " + fmt.Sprintf("%d of %d", y.Present, y.Declared))
		}
		out.say("")
		if v.Complete() {
			out.say(out.out.result(fmt.Sprintf(text["result verify"], v.Present, dir)))
		} else {
			out.missing(v, "homewend takeout")
		}
	}
	if !v.Complete() {
		return exitIncomplete
	}
	return exitOK
}

// importCommand brings a card or a folder into the library, or with
// --dry-run tells what it would bring.
func importCommand(args []string) int {
	w := importArgs(args)
	if w.dir == "" {
		card, code := findCard(w)
		if code != exitOK {
			return code
		}
		w.dir, w.card = card.Path, true
	}
	dir := w.dir
	if w.more {
		return w.failed(text["import one folder"], text["one folder hint"])
	}
	if info, err := os.Stat(dir); err != nil {
		// A photo's name typed where the folder goes: the dry run named it
		// among those deleted, to bring back.
		if filepath.Ext(dir) != "" {
			return w.failed(fmt.Sprintf(text["import not found"], dir), fmt.Sprintf(text["import not found photo"], dir))
		}
		return w.failed(fmt.Sprintf(text["import not found"], dir))
	} else if !info.IsDir() {
		return w.failed(fmt.Sprintf(text["import not a folder"], dir))
	} else if _, err := os.ReadDir(dir); err != nil {
		return w.failed(fmt.Sprintf(text["import cannot read"], dir))
	}
	if !w.dryRun {
		return importCopy(w, args)
	}
	out := begin(args)
	defer out.close()
	w.sayCard(out)
	redates, err := w.dates.redates()
	if err != nil {
		return out.fail(err)
	}
	// Whose import it would be, when that is known without asking: a date
	// another profile holds is said as the import would say it.
	profile := w.profile
	if profile == "" && len(redates) > 0 {
		profile, _ = pickProfile(context.Background(), "", true)
	}
	ctx, stop := interrupted()
	defer stop()
	plan, err := engine.PlanImport(ctx, dir, library.PlanOptions{Asked: w.asked(redates), Profile: profile}, out.event)
	if err != nil {
		return out.fail(err)
	}
	if out.json {
		json.NewEncoder(os.Stdout).Encode(map[string]any{"import": plan, "dir": dir})
		return exitOK
	}
	for _, y := range plan.Years {
		year := y.Year
		if year == "" {
			year = text["import no date"]
		}
		out.say("  " + out.out.paint(fmt.Sprintf("%-7s", year), mutedColour, false) + "  " + mediaCount(y.Photos, y.Videos))
	}
	// What is not a photo or a video, by its name at the top of the card
	// or the folder: a person knows FFDB/ when they see it. The engine names
	// a few (library.MaxOthers) and counts the rest.
	if len(plan.Others) > 0 {
		where := "import others folder"
		if w.card {
			where = "import others card"
		}
		out.say("  " + out.out.paint(text[where], faintColour, false))
		var names []string
		for _, o := range plan.Others {
			names = append(names, o.Name)
		}
		sayNames(out, names, plan.OthersMore)
	}
	sayImport(out, w, plan.Counts, 0, fmt.Sprintf(text["result import"], mediaCount(plan.Photos, plan.Videos), size(plan.Bytes)))
	return exitOK
}

// importArgs reads import's words: the folder, whether --dry-run was given,
// and whether more than one folder was. The flags may come before the folder
// or after it: Go's parser stops at the first word that is not a flag, so
// what follows the folder is parsed again.
func importArgs(args []string) importWords {
	flags := newFlags("import")
	dry := flags.Bool("dry-run", false, "only tell what would happen")
	link := flags.Bool("link", false, "a second name on the library's disk instead of a copy")
	var back listFlags
	flags.Var(&back, "bring-back", "NAMES: photos deleted from the library brought in again; alone, all")
	profile := flags.String("profile", "", "whose photos: the library's folder they go in")
	asJSON := flags.Bool("json", false, "one JSON object per line")
	var dates, include listFlags
	flags.Var(&dates, "date", "NAMES=WHEN: the right date of photos whose camera had it wrong")
	flags.Var(&include, "include", "FOLDER: a folder that stays out by itself (a project, no camera data) brought in")
	flags.Parse(bareBringBack(args))
	dir := flags.Arg(0)
	if flags.NArg() > 0 {
		flags.Parse(flags.Args()[1:])
	}
	return importWords{dir: dir, dryRun: *dry, link: *link, profile: *profile, json: *asJSON, dates: dates, include: include, bringBack: back, more: dir != "" && flags.NArg() > 0}
}

// bareBringBack gives an empty value, all as asked reads it, to a
// --bring-back with no names after it: nothing, another option, or the
// folder to import.
func bareBringBack(args []string) []string {
	var out []string
	for i, a := range args {
		out = append(out, a)
		if a != "--bring-back" && a != "-bring-back" {
			continue
		}
		if i+1 == len(args) || strings.HasPrefix(args[i+1], "-") || isFolder(args[i+1]) {
			out = append(out, "")
		}
	}
	return out
}

// isFolder is whether p is a folder there.
func isFolder(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

// listFlags are a flag given again and again: each --date, "NAMES=WHEN" as
// typed; each --include.
type listFlags []string

func (d *listFlags) String() string     { return strings.Join(*d, " ") }
func (d *listFlags) Set(v string) error { *d = append(*d, v); return nil }

// redates reads each --date: names separated by commas, an equals sign,
// and a date with or without its time, as the camera's clock reads it.
func (d listFlags) redates() ([]library.Redate, error) {
	var out []library.Redate
	for _, v := range d {
		names, when, ok := strings.Cut(v, "=")
		if !ok {
			return nil, fmt.Errorf("%w: %s", errBadDate, v)
		}
		var t time.Time
		var err error
		for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02 15:04", "2006-01-02"} {
			if t, err = time.Parse(layout, strings.TrimSpace(when)); err == nil {
				break
			}
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %s", errBadDate, v)
		}
		r := library.Redate{When: t, Names: commaList(names)}
		if len(r.Names) == 0 {
			return nil, fmt.Errorf("%w: %s", errBadDate, v)
		}
		out = append(out, r)
	}
	return out, nil
}

// commaList is the names typed in v, split by commas.
func commaList(v string) []string {
	var out []string
	for _, name := range strings.Split(v, ",") {
		if name = strings.TrimSpace(name); name != "" {
			out = append(out, name)
		}
	}
	return out
}

// errBadDate is a --date that does not read as NAMES=WHEN.
var errBadDate = errors.New("bad date")

// importWords is what import was told.
type importWords struct {
	dir     string
	dryRun  bool
	link    bool
	profile string
	dates   listFlags
	include listFlags // group folders that stay out by themselves, brought in
	// bringBack are the photos deleted from the library to bring in again,
	// by name, or "all"; each --bring-back holds names split by commas
	bringBack listFlags
	card      bool // a card was found, no folder named
	json      bool
	more      bool // more than one folder
}

// asked is what the person said on the command line, as the engine hears
// it: the dates given, the groups included, the deleted brought back.
func (w importWords) asked(redates []library.Redate) library.Asked {
	var back []string
	for _, v := range w.bringBack {
		names := commaList(v)
		if len(names) == 0 { // --bring-back with no names
			names = []string{library.BringAllBack}
		}
		back = append(back, names...)
	}
	return library.Asked{Redates: redates, Choice: w.choice(), BringBack: back}
}

// choice is the groups --include brings in besides those that come in by
// themselves; nil when none was named. A folder is named as the dry run
// says it, with / and from the folder imported.
func (w importWords) choice() *library.Choice {
	if len(w.include) == 0 {
		return nil
	}
	c := &library.Choice{}
	top := engine.TopName(w.dir)
	for _, f := range w.include {
		c.In = append(c.In, includeFolder(w.dir, top, f))
	}
	return c
}

// includeFolder is name as a group folder of dir: as the dry run writes it,
// or a path the shell completed, whole or from here, that leads into dir.
func includeFolder(dir, top, name string) string {
	clean := filepath.Clean(name)
	// The folder itself, as "." or by its own name, as the dry run writes
	// it — unless a folder inside has that name too.
	if _, inside := os.Stat(filepath.Join(dir, clean)); clean == "." || (clean == top && inside != nil) {
		return "."
	}
	// A path the shell completed is one that is there from here.
	if _, err := os.Stat(clean); err == nil || filepath.IsAbs(clean) {
		if rel, ok := library.Below(clean, dir); ok {
			return rel
		}
	}
	return strings.Trim(filepath.ToSlash(clean), "/")
}

// findCard is the card to import when no folder was named: the one in, or
// the one chosen among several.
func findCard(w importWords) (engine.Card, int) {
	cards := engine.Cards()
	switch {
	case len(cards) == 1:
		return cards[0], exitOK
	case len(cards) == 0:
		return engine.Card{}, w.failed(text["no card"], text["no card hint"])
	case !interactive(w.json):
		var names []string
		for _, c := range cards {
			names = append(names, shellWord(c.Path))
		}
		return engine.Card{}, w.failed(fmt.Sprintf(text["cards found"], strings.Join(names, ", ")), text["one folder hint"])
	}
	ctx, stop := interrupted()
	defer stop()
	var names []string
	for _, c := range cards {
		names = append(names, c.Path)
	}
	at, err := choose(ctx, text["which card"], "", names)
	if err != nil {
		return engine.Card{}, newPrinter(w.json).fail(err)
	}
	return cards[at], exitOK
}

// sayCard says which card was found, when no folder was named.
func (w importWords) sayCard(out *printer) {
	if w.card && !out.json {
		out.say("  " + fmt.Sprintf(text["card found"], shellWord(w.dir)))
		out.say("")
	}
}

// failed says what is wrong before anything starts: as a line of JSON when
// that was asked for, as printer.fail does.
func (w importWords) failed(said string, hints ...string) int {
	if w.json {
		json.NewEncoder(os.Stdout).Encode(map[string]any{"error": said, "exit": exitError})
		return exitError
	}
	return failed(said, nil, hints...)
}

// importCopy brings the folder into the library.
func importCopy(w importWords, args []string) int {
	ctx, stop := interrupted()
	defer stop()
	if _, code := libraryFor(ctx, "", w.json); code != exitOK {
		return code
	}
	profile, err := pickProfile(ctx, w.profile, w.json)
	if errors.Is(err, errNoProfile) {
		// No Google account, no profile yet: a card needs no sign-in, only
		// a name for whose photos these are.
		return w.failed(text["import whose"], fmt.Sprintf(text["import whose hint"], typed()))
	}
	if err != nil {
		return newPrinter(w.json).fail(err)
	}
	out := begin(args)
	defer out.close()
	w.sayCard(out)
	redates, err := w.dates.redates()
	if err != nil {
		return out.fail(err)
	}
	done, err := engine.Import(ctx, w.dir, profile, library.ImportOptions{Asked: w.asked(redates), Link: w.link}, out.event)
	if err != nil {
		return out.fail(err)
	}
	if out.json {
		json.NewEncoder(os.Stdout).Encode(map[string]any{"imported": done, "dir": w.dir})
		return exitOK
	}
	fresh := fmt.Sprintf(text["result imported"], counted(done.Placed, "brought"), size(done.Bytes), profile)
	if w.link {
		fresh = fmt.Sprintf(text["result linked"], counted(done.Placed, "brought"), profile)
	}
	sayImport(out, w, done.Counts, done.Undated, fresh)
	return exitOK
}

// sayNames lists names under the line before them, faint, and how many
// more there are past them.
func sayNames(out *printer, names []string, more int) {
	for _, n := range names {
		out.say("    " + out.out.paint(n, faintColour, false))
	}
	if more > 0 {
		out.say("    " + out.out.paint(fmt.Sprintf(text["names more"], more), faintColour, false))
	}
}

// sayImport says how a look (a dry run) or its copy went, one way for
// both: the groups that stay out with why and how to bring one in, what is
// in, moved, twice, undated (a copy's), left to another profile, deleted,
// unreadable; then its last line.
func sayImport(out *printer, w importWords, c library.Counts, undated int, fresh string) {
	top := engine.TopName(w.dir)
	for _, g := range c.LeftOut {
		folder := g.Folder
		if folder == "." {
			folder = top
		}
		why := text["group "+string(g.Kind)]
		if g.Mark != "" {
			why = fmt.Sprintf(why, g.Mark)
		}
		out.say("  " + out.out.paint(folder, faintColour, false) + "  " + out.out.paint(counted(g.Files, "files")+" · "+why, faintColour, false))
	}
	if len(c.LeftOut) > 0 {
		out.say("  " + out.out.paint(text["group include"], faintColour, false))
	}
	for _, line := range []struct {
		n    int
		said string
	}{
		{c.InLibrary, fmt.Sprintf(text["import already"], c.InLibrary)},
		{c.Redated, fmt.Sprintf(w.words("redated"), c.Redated)},
		{c.Twins, fmt.Sprintf(text["import twins"], c.Twins)},
		{undated, fmt.Sprintf(text["imported undated"], undated)},
	} {
		if line.n > 0 {
			out.say("  " + line.said)
		}
	}
	if c.NotRedated > 0 {
		out.say("  " + out.out.paint(fmt.Sprintf(text["not redated"], c.NotRedated), warnColour, false))
	}
	if c.BroughtBack > 0 {
		out.say("  " + counted(c.BroughtBack, w.key("deleted back")))
	}
	// Those left out by name: one deleted by mistake is brought back by it.
	if c.Deleted > 0 {
		out.say("  " + out.out.paint(fmt.Sprintf(text["import deleted"], counted(c.Deleted, "deleted")), faintColour, false))
		sayNames(out, c.DeletedNames, c.DeletedMore)
	}
	if c.Unreadable > 0 {
		out.say("  " + out.out.paint(counted(c.Unreadable, "import unreadable"), warnColour, false))
	}
	out.say("")
	// The last line: fresh when something comes in, else the outcome's own.
	if c.Outcome == library.OutcomeNew {
		out.say(out.out.result(fresh))
		return
	}
	said := strings.NewReplacer("{n}", strconv.Itoa(c.Redated), "{dir}", w.dir).Replace(w.words("outcome " + string(c.Outcome)))
	// What happened as asked is said as done; what is to look at, as a
	// summary.
	switch c.Outcome {
	case library.OutcomeRedated, library.OutcomeDeleted, library.OutcomeKnown:
		out.say(out.out.result(said))
	default:
		out.say(out.out.summary(said))
	}
}

// key is the table's key for this run: a dry run's words are in the
// conditional, key+" dry", where the table has them.
func (w importWords) key(key string) string {
	if _, ok := text[key+" dry"]; ok && w.dryRun {
		return key + " dry"
	}
	return key
}

// words are key's words for this run.
func (w importWords) words(key string) string { return text[w.key(key)] }

// counted is n of a thing, "1 photo" or "3 photos": key names the plural in
// the table, key+" one" the singular.
func counted(n int, key string) string {
	if n == 1 {
		return text[key+" one"]
	}
	return fmt.Sprintf(text[key], n)
}

// mediaCount is "3 photos · 1 video", leaving out what there is none of.
func mediaCount(photos, videos int) string {
	var parts []string
	if photos > 0 {
		parts = append(parts, counted(photos, "photos"))
	}
	if videos > 0 {
		parts = append(parts, counted(videos, "videos"))
	}
	return strings.Join(parts, " · ")
}

// updateProgram replaces the running homewend with the latest release.
func updateProgram(args []string) int {
	flags := newFlags("update")
	flags.Bool("json", false, "one JSON object per line")
	flags.Parse(args)
	out := begin(args)
	defer out.close()

	ctx, stop := interrupted()
	defer stop()
	latest, err := update.Latest(ctx)
	if err != nil {
		return out.fail(err)
	}
	if latest == version {
		out.line("version", version, out.out.result(fmt.Sprintf(text["up to date"], version)))
		return exitOK
	}
	program, err := update.Program()
	if err != nil {
		return out.fail(err)
	}
	start := time.Now()
	var got int64
	arriving := func(e progress.Event) {
		e.Name, got = latest, e.Total
		out.event(e)
	}
	if err := update.Install(ctx, program, arriving); err != nil {
		return out.fail(err)
	}
	out.now(going{})
	out.say(out.out.done(fmt.Sprintf(text["update got"], latest), fmt.Sprintf(text["downloaded in"], size(got), time.Since(start).Round(time.Second))))
	out.say(out.out.done(text["update sums"], ""))
	out.say("")
	out.line("version", latest, out.out.result(fmt.Sprintf(text["updated"], version, latest)))
	return exitOK
}

// sorted reports the photos placed; nothing when the export was already all
// in the library.
func (p *printer) sorted(o library.Organized) {
	if o.Placed == 0 && o.Skipped == 0 {
		return
	}
	took := time.Duration(0)
	if !p.sortStart.IsZero() {
		took = time.Since(p.sortStart).Round(time.Second)
	}
	p.line("organized", o, p.out.done(fmt.Sprintf(text["sorted"], o.Placed),
		fmt.Sprintf(text["sorted details"], took, o.Skipped, o.Undated, o.Linked+o.Copied)))
}

// checked reports the library against the export's list, and ends: where the
// photos are, or which are missing.
func (p *printer) checked(v library.Verification, dir string, year int) int {
	if p.json {
		json.NewEncoder(os.Stdout).Encode(map[string]any{"verification": v})
	} else {
		said := fmt.Sprintf(text["checked"], v.Present, v.Declared)
		if v.Complete() {
			p.say(p.out.done(said, text["checked details"]))
			p.say("")
			p.say(p.out.result(fmt.Sprintf(text["result"], v.Present)))
		} else {
			p.say(p.out.failed(said, text["checked details"]))
			p.missing(v, typed())
		}
	}
	if !v.Complete() {
		return exitIncomplete
	}
	return exitOK
}

// missing names the files that are not there, and what brings them.
func (p *printer) missing(v library.Verification, again string) {
	p.say("")
	p.say(p.out.failure(fmt.Sprintf(text["missing"], len(v.Missing))))
	for _, name := range v.Missing {
		p.say(p.out.cause(name))
	}
	p.say("")
	p.say(p.out.hint(fmt.Sprintf(text["fetch them"], again)))
}

// failed says what went wrong, under the screen, and what to do about it.
func failed(said string, causes []string, hints ...string) int {
	l := lookFor(os.Stderr)
	fmt.Fprintln(os.Stderr, "\n"+l.failure(said))
	for _, c := range causes {
		fmt.Fprintln(os.Stderr, l.cause(c))
	}
	if len(hints) > 0 {
		fmt.Fprintln(os.Stderr)
	}
	for _, h := range hints {
		fmt.Fprintln(os.Stderr, l.hint(h))
	}
	return exitError
}

// fail ends a command on err: what went wrong, in words a person reads, and
// the command that puts it right.
func (p *printer) fail(err error) int {
	p.close()
	code, said, causes, hints := explain(err)
	if p.json {
		json.NewEncoder(os.Stdout).Encode(map[string]any{"error": said, "exit": code})
		return code
	}
	if errors.Is(err, context.Canceled) {
		l := lookFor(os.Stdout)
		fmt.Println(l.stopped(text["stopped"], ""))
		fmt.Println()
		fmt.Println(l.hint(fmt.Sprintf(text["resume"], typed())))
		return code
	}
	failed(said, causes, hints...)
	return code
}

// explain turns err into what a person is told: what went wrong, the details
// under it, and what to do.
func explain(err error) (code int, said string, causes, hints []string) {
	again := typed()
	var space engine.NoSpaceError
	var notOffered *takeout.NotOfferedError
	var gone engine.LibraryGoneError
	var unknown library.UnknownNameError
	var ambiguous library.AmbiguousNameError
	var folderGone engine.FolderGoneError
	var unknownGroup library.UnknownGroupError
	switch {
	case errors.As(err, &unknownGroup):
		return exitError, fmt.Sprintf(text["bad include"], unknownGroup.Folder), nil, []string{text["bad include hint"]}
	case errors.As(err, &notOffered):
		var years []string
		for _, e := range notOffered.Offered {
			if y, ok := takeout.YearOf(e.ID); ok {
				years = append(years, strconv.Itoa(y))
			}
		}
		return exitError, fmt.Sprintf(text["not offered"], notOffered.Year),
			[]string{fmt.Sprintf(text["offered years"], strings.Join(years, ", "))}, []string{text["not offered hint"]}
	case errors.Is(err, download.ErrSessionExpired), errors.Is(err, takeout.ErrSignedOut):
		return exitSignIn, text["session expired"], nil, []string{fmt.Sprintf(text["login then"], again)}
	case errors.Is(err, engine.ErrSignInClosed):
		return exitSignIn, text["sign-in closed"], nil, []string{text["try login"], text["try fallback"]}
	case errors.Is(err, engine.ErrSignInDeclined):
		return exitSignIn, text["sign-in declined"], nil, []string{text["try login"]}
	case errors.Is(err, engine.ErrSessionNotWritten), errors.Is(err, engine.ErrSessionNotAccepted):
		return exitSignIn, fmt.Sprintf(text["session lost"], err), nil, []string{text["try login"]}
	case errors.Is(err, engine.ErrNotSignedIn):
		return exitSignIn, text["not signed in yet"], nil, []string{text["sign in hint"]}
	case errors.Is(err, update.ErrDamaged):
		return exitError, text["update damaged"], nil, []string{text["update again"]}
	case errors.Is(err, context.Canceled):
		return exitError, text["stopped"], nil, nil
	case errors.Is(err, takeout.ErrFormChanged), errors.Is(err, engine.ErrNotRequested):
		return exitError, fmt.Sprintf(text["not requested err"], err), nil, []string{fmt.Sprintf(text["try again"], again)}
	case errors.Is(err, engine.ErrNoSuchTakeout):
		return exitError, fmt.Sprintf(text["no such export"], err), nil, []string{text["list hint"]}
	case errors.Is(err, engine.ErrExpired):
		return exitError, fmt.Sprintf(text["expired"], err), nil, []string{fmt.Sprintf(text["expired hint"], again)}
	case errors.Is(err, engine.ErrNoDownload):
		return exitError, text["no download"], nil, []string{fmt.Sprintf(text["try again"], again)}
	case errors.As(err, &space):
		return exitError, text["no space"], []string{fmt.Sprintf(text["no space sizes"], size(space.Need), size(space.Free))},
			[]string{fmt.Sprintf(text["no space hint"], again)}
	case errors.Is(err, errBadDate):
		return exitError, fmt.Sprintf(text["bad date"], strings.TrimPrefix(err.Error(), errBadDate.Error()+": ")), nil, []string{text["date hint"]}
	case errors.As(err, &ambiguous):
		return exitError, fmt.Sprintf(text["name twice"], ambiguous.Name, strings.Join(ambiguous.Paths, ", ")), nil, []string{fmt.Sprintf(text["name twice hint"], ambiguous.Paths[0])}
	case errors.As(err, &unknown):
		return exitError, fmt.Sprintf(text["no such photo"], unknown.Name), nil, nil
	case errors.Is(err, engine.ErrLibraryBusy):
		return exitError, text["library busy"], nil, []string{fmt.Sprintf(text["library busy hint"], again)}
	case errors.Is(err, engine.ErrBadProfile):
		return exitError, text["bad profile"], nil, nil
	case errors.Is(err, library.ErrNoLink):
		return exitError, text["no link"], nil, []string{text["no link hint"]}
	case errors.As(err, &folderGone):
		return exitIncomplete, fmt.Sprintf(text["folder gone"], folderGone.Dir), nil, []string{fmt.Sprintf(text["folder gone hint"], again)}
	case errors.As(err, &gone):
		return exitError, fmt.Sprintf(text["library gone"], gone.Root), nil, []string{fmt.Sprintf(text["library gone hint"], again)}
	case errors.Is(err, errNoSuchAccount):
		return exitError, fmt.Sprintf(text["no such account"], err), nil, []string{text["accounts hint"]}
	case errors.Is(err, errWhichAccount):
		return exitError, text["which account err"], nil, []string{fmt.Sprintf(text["account hint"], again)}
	case errors.Is(err, errNoProfile):
		return exitError, text["no profile"], nil, []string{text["sign in hint"]}
	case errors.Is(err, errWhichProfile):
		return exitError, text["which profile err"], nil, []string{fmt.Sprintf(text["profile hint"], again)}
	case errors.Is(err, session.ErrNoBrowser):
		return exitError, text["no browser"], nil, []string{text["no browser hint"]}
	}
	return exitError, fmt.Sprintf(text["error"], err), nil, nil
}
