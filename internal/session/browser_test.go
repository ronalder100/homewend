// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package session

import (
	"errors"
	"runtime"
	"testing"
)

func TestPickBrowser(t *testing.T) {
	// What the test machine has installed, by the first command name.
	on := func(names ...string) func(chromium) string {
		return func(c chromium) string {
			for _, n := range names {
				if c.commands[0] == n {
					return "/bin/" + n
				}
			}
			return ""
		}
	}
	// The default browser's id, as the system in use would give it.
	id := func(c chromium) string {
		if runtime.GOOS == "darwin" {
			return c.bundle
		}
		return c.commands[len(c.commands)-1] + ".desktop"
	}
	brave, chrome := chromiums[2], chromiums[0]

	cases := []struct {
		name, def string
		installed func(chromium) string
		want      string
	}{
		{"the default wins over the order", id(brave), on("google-chrome", "brave-browser"), "/bin/brave-browser"},
		{"a default that is not Chromium falls back", "firefox.desktop", on("brave-browser"), "/bin/brave-browser"},
		{"no default known falls back in order", "", on("brave-browser", "google-chrome"), "/bin/google-chrome"},
		{"a default that is not installed falls back", id(chrome), on("chromium"), "/bin/chromium"},
	}
	for _, c := range cases {
		got, err := pickBrowser(c.def, c.installed)
		if err != nil || got != c.want {
			t.Errorf("%s: got %q, %v; want %q", c.name, got, err, c.want)
		}
	}

	if _, err := pickBrowser("", on()); !errors.Is(err, ErrNoBrowser) {
		t.Errorf("nothing installed: got %v, want ErrNoBrowser", err)
	}
}
