// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/ronalder100/homewend/internal/download"
	"github.com/ronalder100/homewend/internal/library"
	"github.com/ronalder100/homewend/internal/progress"
)

// The test binary stands in for homewend when asked to, so the tests can run
// the command line as a person does, exit codes included.
func TestMain(m *testing.M) {
	if os.Getenv("HOMEWEND_TEST_MAIN") == "1" {
		main()
	}
	os.Exit(m.Run())
}

func homewend(t *testing.T, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = append(os.Environ(), "HOMEWEND_TEST_MAIN=1", "XDG_CONFIG_HOME="+t.TempDir(), "HOME="+t.TempDir())
	out, err := cmd.CombinedOutput()
	if exit, ok := err.(*exec.ExitError); ok {
		return string(out), exit.ExitCode()
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(out), 0
}

// A command without a help page, or missing from the overview, is a command
// nobody finds.
func TestEveryCommandIsDocumented(t *testing.T) {
	for name := range commands {
		page := text["help "+name]
		if !strings.HasPrefix(page, "homewend "+name+" — ") {
			t.Errorf("help page of %s is missing or does not start with its name", name)
		}
		if !strings.Contains(page, "Example") {
			t.Errorf("help page of %s has no example", name)
		}
		if !strings.Contains(text["usage"], "\n  "+name+" ") {
			t.Errorf("the overview does not list %s", name)
		}
	}
}

func TestHelp(t *testing.T) {
	for name := range commands {
		for _, args := range [][]string{{"help", name}, {name, "-h"}, {name, "--help"}} {
			out, code := homewend(t, args...)
			if code != exitOK || strings.TrimSpace(out) != text["help "+name] {
				t.Errorf("homewend %s: exit %d, output\n%s", strings.Join(args, " "), code, out)
			}
		}
	}
	for _, args := range [][]string{{"help"}, {"--help"}, {}} {
		if out, code := homewend(t, args...); code != exitOK || strings.TrimSpace(out) != text["usage"] {
			t.Errorf("homewend %s: exit %d, output\n%s", strings.Join(args, " "), code, out)
		}
	}
}

// Drawing a page for a terminal changes how it looks and none of its words,
// but the colons of its headings.
func TestDressedKeepsEveryWord(t *testing.T) {
	on := look{on: true}
	for key, page := range text {
		if key != "usage" && !strings.HasPrefix(key, "help ") {
			continue
		}
		want := strings.Replace(page, " — ", "  ", 1)
		for _, h := range []string{"Usage:", "Flags:", "Examples:", "Commands:", "Start here:"} {
			want = strings.ReplaceAll(want, "\n"+h+"\n", "\n"+strings.TrimSuffix(h, ":")+"\n")
		}
		if got := ansi.Strip(dressed(on, page)); got != want {
			t.Errorf("%s: dressed changed the words:\n%s", key, got)
		}
	}
	got := dressed(on, text["help login"])
	if !strings.Contains(got, on.paint("--profile DIR", nil, true)) || !strings.Contains(got, on.paint("  homewend login", nil, true)) {
		t.Error("what to type is not bold")
	}
	if strings.Contains(got, on.paint("config directory)", nil, true)) {
		t.Error("the second line of a flag is drawn as a flag")
	}
}

// Off a terminal every line is its words: the signs stay, colour and weight
// go, and the ** that mark what stands out are not printed.
func TestPlainLines(t *testing.T) {
	l := look{}
	for got, want := range map[string]string{
		l.done("Downloaded **all parts**", " in 12s"):   "✓ Downloaded all parts in 12s",
		l.failed("Checked **8660 of 8662**", ""):        "× Checked 8660 of 8662",
		l.waiting("Google is preparing the export", ""): "* Google is preparing the export",
		l.failure("your Google sign-in expired"):        "error: your Google sign-in expired",
		l.hint("to resume, run: **homewend takeout**"):  "hint: to resume, run: homewend takeout",
		l.result("All your photos are in /photos"):      "✓ All your photos are in /photos",
		l.cause("IMG_1.jpg"):                            "  IMG_1.jpg",
		l.field("Google", signDone, nil, "a@b.c", ""):   "  Google     ✓ a@b.c",
		l.field("Library", "", nil, "/photos", ""):      "  Library      /photos",
	} {
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

// A bar is 20 cells, whatever it counts.
func TestBarIsTwentyCells(t *testing.T) {
	for _, f := range []float64{0, 0.33, 1, 2} {
		if got := len([]rune(look{}.bar(f))); got != barWidth {
			t.Errorf("bar(%v) is %d cells", f, got)
		}
	}
}

// The line going on carries its time when it is worth counting, and a line
// under it when there is one.
func TestTheLineGoingOn(t *testing.T) {
	s := &liveLine{out: look{}}
	if s.line(timeZero) != "" {
		t.Error("nothing going on draws something")
	}
	start := timeZero.Add(1)
	s.now = going{said: "Asking Google to export your 2025 photos", bar: -1, since: start}
	if got := s.line(start.Add(63e9)); !strings.HasSuffix(got, "Asking Google to export your 2025 photos · 1m3s") {
		t.Errorf("got %q", got)
	}
	s.now = going{sign: signWaiting, said: text["preparing"], bar: -1, below: text["preparing hint"]}
	if got := s.line(start); got != "* "+text["preparing"]+"\n  "+text["preparing hint"] {
		t.Errorf("got %q", got)
	}
}

// The questions answer to the arrows and Enter, and Ctrl-C stops them.
func TestChoosing(t *testing.T) {
	c := &chooser{choices: []string{"Yes", "No"}}
	for _, k := range []string{"down", "down", "up", "down"} {
		c.Update(keyPress(k))
	}
	c.Update(keyPress("enter"))
	if c.at != 1 || !c.done || c.stopped() {
		t.Errorf("at %d, done %v, stopped %v", c.at, c.done, c.stopped())
	}
	c = &chooser{choices: []string{"Yes", "No"}}
	c.Update(keyPress("ctrl+c"))
	if !c.stopped() {
		t.Error("Ctrl-C did not stop the question")
	}
}

// What is not a year is said so, with what to type instead.
func TestTakeoutWantsAYear(t *testing.T) {
	out, code := homewend(t, "takeout", "Greece")
	if code != exitError || !strings.Contains(out, `error: "Greece" is not a year`) || !strings.Contains(out, "hint: to bring one year home, run: homewend takeout ") {
		t.Errorf("exit %d, output\n%s", code, out)
	}
}

// With no library chosen and no one to ask, the command says how to give one.
func TestTakeoutWithoutALibrary(t *testing.T) {
	out, code := homewend(t, "takeout", "2025")
	if code != exitError || !strings.Contains(out, "error: no library folder chosen") || !strings.Contains(out, "homewend takeout 2025 --library DIR") {
		t.Errorf("exit %d, output\n%s", code, out)
	}
}

func TestUnknownCommand(t *testing.T) {
	out, code := homewend(t, "download")
	if code != exitError || !strings.Contains(out, `error: unknown command "download"`) || !strings.Contains(out, "hint: to see the commands, run: homewend help") {
		t.Errorf("exit %d, output\n%s", code, out)
	}
}

func TestVersion(t *testing.T) {
	if out, code := homewend(t, "version"); code != exitOK || strings.TrimSpace(out) != "dev" {
		t.Errorf("exit %d, output %q", code, out)
	}
}

// Each failure says what went wrong, and the hint names the command to type.
func TestFailuresSayWhatToDo(t *testing.T) {
	_, said, _, hints := explain(download.ErrSessionExpired)
	if said != text["session expired"] || len(hints) != 1 || !strings.Contains(hints[0], "**homewend login**") {
		t.Errorf("session expired: %q %q", said, hints)
	}
	if code, said, _, _ := explain(errors.New("boom")); code != exitError || said != "boom" {
		t.Errorf("anything else: %d %q", code, said)
	}
	if _, said, _, _ := explain(context.Canceled); said != text["stopped"] {
		t.Errorf("Ctrl-C: %q", said)
	}
}

// A download that ends with nothing missing says where the photos are; one
// with files missing names them, and the command that brings them.
func TestTheLastLine(t *testing.T) {
	complete := captured(t, func() {
		(&printer{out: look{}}).checked(library.Verification{Declared: 2, Present: 2}, "/photos", 2025)
	})
	if !strings.HasSuffix(complete, "✓ All 2 photos are there\n") {
		t.Errorf("complete:\n%s", complete)
	}
	missing := captured(t, func() {
		(&printer{out: look{}}).checked(library.Verification{Declared: 2, Present: 1, Missing: []string{"a.jpg"}}, "/photos", 2025)
	})
	for _, want := range []string{"× Checked 1 of 2", "error: 1 photos are missing", "  a.jpg", "hint: to fetch them, run: "} {
		if !strings.Contains(missing, want) {
			t.Errorf("missing lacks %q:\n%s", want, missing)
		}
	}
}

// captured is what f prints to stdout.
func captured(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = w
	f()
	os.Stdout = stdout
	w.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}

// Once the sign-in is said, the engine checking it again shows nothing.
func TestTheSignInIsSaidOnce(t *testing.T) {
	out := captured(t, func() {
		p := &printer{out: look{}, signedIn: true}
		p.event(progress.Event{Stage: progress.Checking})
	})
	if out != "" {
		t.Errorf("printed %q", out)
	}
}

// The parameters of the run are a table: the name bold, the value plain.
func TestParamLine(t *testing.T) {
	if got := (look{}).param("Takeout", "2025 photos", " · 4184685c"); got != "  Takeout   2025 photos · 4184685c" {
		t.Errorf("got %q", got)
	}
}

// An export already in the table is not found again on the screen.
func TestTheExportInTheTableIsNotFoundAgain(t *testing.T) {
	out := captured(t, func() {
		p := &printer{out: look{}, shown: "job"}
		p.event(progress.Event{Stage: progress.Ready, Name: "job", Of: 2, Total: 10})
	})
	if out != "" {
		t.Errorf("printed %q", out)
	}
}
