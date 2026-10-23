// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package session

import (
	"errors"
	"os"
)

// ErrNoBrowser means the machine has no Chromium-family browser we know of.
var ErrNoBrowser = errors.New("no Chromium-family browser found; set BROWSER_BIN")

// chromium is one Chromium-family browser, as it is found on each system.
type chromium struct {
	commands []string // Linux: names on PATH; the default one's .desktop id is the first plus ".desktop"
	bundle   string   // macOS: bundle id, as LaunchServices records it
	app      string   // macOS: the executable inside the app, in /Applications
	exe      string   // Windows: the executable's name, as the registry's open commands end
}

// In order of preference when the default browser is none of them. Safari and
// Firefox are not here: Safari cannot be opened on a profile of ours, and
// Firefox has never been tried. Arc is not here either: launched on a profile
// of ours while the user's Arc was open, it refused with "Only one instance of
// Arc can be opened at a time" and never wrote to the profile (Arc 1.156.0,
// 2026-09-27). Its users get the next browser in this list.
var chromiums = []chromium{
	{[]string{"google-chrome", "google-chrome-stable"}, "com.google.chrome",
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "chrome.exe"},
	// Chromium's Windows build is chrome.exe too (chrome/common/chrome_constants.cc).
	{[]string{"chromium", "chromium-browser"}, "org.chromium.chromium",
		"/Applications/Chromium.app/Contents/MacOS/Chromium", "chrome.exe"},
	{[]string{"brave-browser"}, "com.brave.browser",
		"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser", "brave.exe"},
	{[]string{"microsoft-edge", "microsoft-edge-stable"}, "com.microsoft.edgemac",
		"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge", "msedge.exe"},
}

// findBrowser picks the browser to sign in with: the user's default browser if
// it is Chromium-family, since that is the one they trust and are signed in to
// elsewhere; otherwise any Chromium-family browser installed. BROWSER_BIN
// overrides both.
func findBrowser() (string, error) {
	if set := os.Getenv("BROWSER_BIN"); set != "" {
		return set, nil
	}
	return pickBrowser(defaultBrowser(), installed)
}

func pickBrowser(def string, installed func(chromium) string) (string, error) {
	for _, c := range chromiums {
		if def != "" && matches(c, def) {
			if path := installed(c); path != "" {
				return path, nil
			}
		}
	}
	for _, c := range chromiums {
		if path := installed(c); path != "" {
			return path, nil
		}
	}
	return "", ErrNoBrowser
}
