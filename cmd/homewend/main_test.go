// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
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
