// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ronalder100/homewend/internal/engine"
	"github.com/ronalder100/homewend/internal/library"
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
	cmd.Env = append(os.Environ(), "HOMEWEND_TEST_MAIN=1")
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
	for _, args := range [][]string{{"help"}, {"--help"}} {
		if out, code := homewend(t, args...); code != exitOK || strings.TrimSpace(out) != text["usage"] {
			t.Errorf("homewend %s: exit %d, output\n%s", strings.Join(args, " "), code, out)
		}
	}
}

// Dressing a page for a terminal changes its colours and none of its words:
// what to type stands out, and a flag's second line is not taken for a flag.
func TestDressedKeepsEveryWord(t *testing.T) {
	for key, page := range text {
		if key != "usage" && !strings.HasPrefix(key, "help ") {
			continue
		}
		if got := ansi.Strip(dressed(page)); got != page {
			t.Errorf("%s: dressed changed the words:\n%s", key, got)
		}
	}
	accent := lipgloss.NewStyle().Foreground(accentColour)
	got := dressed(text["help login"])
	for _, want := range []string{accent.Render("--profile DIR"), accent.Render("  homewend login")} {
		if !strings.Contains(got, want) {
			t.Errorf("dressed lacks %q", want)
		}
	}
	if strings.Contains(got, accent.Render("config directory)")) || strings.Contains(got, accent.Render("                config directory)")) {
		t.Error("the second line of a flag is dressed as a flag")
	}
}

// A command asked for without its folder says so and shows what to type:
// what was typed, with the folder added.
func TestACommandAskedWronglySaysWhatToType(t *testing.T) {
	for typed, want := range map[string]string{
		"get":             "\n  homewend get --library ~/Pictures/Homewend",
		"get --year 2025": "\n  homewend get --year 2025 --library ~/Pictures/Homewend",
		"verify":          "\n  homewend verify --library ~/Pictures/Homewend",
	} {
		out, code := homewend(t, strings.Fields(typed)...)
		if code != exitError || !strings.HasPrefix(out, "library folder missing:") || !strings.Contains(out, want) {
			t.Errorf("homewend %s: exit %d, output\n%s", typed, code, out)
		}
	}
}

// typing stands in for what the user types at the questions that follow.
func typing(t *testing.T, reply string) {
	t.Helper()
	keyboard := replies
	replies = bufio.NewReader(strings.NewReader(reply))
	t.Cleanup(func() { replies = keyboard })
}

// Before a new export, Enter is yes and only a no stops it. With an export
// already there, the choice is by number: Enter or 1 downloads it, 2 asks
// Google for a new one.
func TestGetAsksBeforeGoogleIsAsked(t *testing.T) {
	g := engine.Get{Year: 2025}
	for reply, want := range map[string]bool{"\n": true, "y\n": true, "Y\n": true, "n\n": false, "no\n": false} {
		typing(t, reply)
		if got := confirmNew(g); got != want {
			t.Errorf("a new export, reply %q: got %v, want %v", reply, got, want)
		}
	}
	// A reply that is neither number is asked again: here the n of a habit,
	// then the answer.
	for reply, want := range map[string]bool{"\n": false, "1\n": false, "2\n": true, "n\n2\n": true, "n\n\n": false} {
		typing(t, reply)
		if got := wantsNew(g, engine.Found{}); got != want {
			t.Errorf("an export already there, reply %q: asks for a new one %v, want %v", reply, got, want)
		}
	}
}

// What comes after a sign-in is said with the command to type, and stays
// plain text where there is no terminal to colour it for.
func TestAfterLoginSaysWhatToType(t *testing.T) {
	said := text["after login"]
	if !strings.Contains(said, "\n  homewend get --year 2025 --library ~/Pictures/Homewend") {
		t.Errorf("no command to type in %q", said)
	}
	file, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if got := toType(file, said); got != said {
		t.Errorf("coloured outside a terminal: %q", got)
	}
}

func TestUnknownCommand(t *testing.T) {
	out, code := homewend(t, "download")
	if code != exitError || !strings.Contains(out, `unknown command "download"`) {
		t.Errorf("exit %d, output\n%s", code, out)
	}
}

func TestVersion(t *testing.T) {
	if out, code := homewend(t, "version"); code != exitOK || strings.TrimSpace(out) != "dev" {
		t.Errorf("exit %d, output %q", code, out)
	}
}

// A download that ends with nothing missing says so; one with files missing
// does not.
func TestFinishedCongratulatesOnlyAComplete(t *testing.T) {
	for _, tc := range []struct {
		v    library.Verification
		want bool
	}{
		{library.Verification{Declared: 2, Present: 2}, true},
		{library.Verification{Declared: 2, Present: 1, Missing: []string{"a.jpg"}}, false},
	} {
		out := captured(t, func() { printer{}.finished(tc.v) })
		if got := strings.Contains(out, text["complete"]); got != tc.want {
			t.Errorf("%+v: congratulated %v, want %v:\n%s", tc.v, got, tc.want, out)
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
