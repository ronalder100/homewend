// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

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
