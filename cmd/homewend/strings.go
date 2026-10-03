// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

// Every sentence the CLI shows a person. A second language is a second table.
// Between ** is what stands out: what to read, or what to type.
var text = map[string]string{
	"usage": `homewend — Google Photos to your disk, sorted by date and album

Start here:
  homewend login
  homewend takeout 2025

Commands:
  login     sign in to Google in a window of its own
  logout    sign out and forget the Google sign-in
  status    show the account, the export and the library
  takeout   bring photos home from Google Takeout: all, or a year
  verify    count the library again against its exports
  ui        serve the window of the desktop app
  update    install the latest homewend
  version   show the version
  help      show help for a command

For the flags of a command, run: homewend help <command>
https://homewend.app`,

	"help login": `homewend login — sign in to Google, once

Usage:
  homewend login [--profile DIR] [--fallback] [--json]

Opens a small Chrome, Chromium, Brave or Edge window of its own and waits while
you sign in to Google there. Google asks you to let homewend see your email
address: that is how the window knows you are done, and nothing is read with
it. homewend never sees your password. Run it again whenever a command says
the sign-in expired.

Flags:
  --profile DIR   where the sign-in is kept (default: homewend/profile in your
                  config directory)
  --fallback      sign in on Google Takeout's page, in a full window: for when
                  the small window shows a Google error instead of the sign-in
  --json          one JSON object per line, for scripts

Examples:
  homewend login
  homewend login --fallback
  BROWSER_BIN=/usr/bin/brave-browser homewend login`,

	"help logout": `homewend logout — sign out of Google

Usage:
  homewend logout [--profile DIR]

Forgets the Google sign-in: the next command that needs Google asks you to
sign in again. Your photos and your library are not touched.

Flags:
  --profile DIR   where the sign-in is kept (default: the one login made)

Examples:
  homewend logout`,

	"help status": `homewend status — where things stand

Usage:
  homewend status [--profile DIR] [--json]

Shows the Google account, the export homewend asked for last, the library
folder and the version. Nothing is downloaded and nothing changes.

Flags:
  --profile DIR   where the sign-in is kept (default: the one login made)
  --json          one JSON object, for scripts

Examples:
  homewend status`,

	"help takeout": `homewend takeout — your photos, from Google Takeout to your disk

Usage:
  homewend takeout [YEAR] [--new] [--yes] [--library DIR] [--export ID]
                   [--profile DIR] [--json]
  homewend takeout --list

Brings your Google Photos home, one year or all of them, into the library
folder: the photos by date, with your albums beside them, each one counted
against the list Google puts in the export.

If Google Takeout still has the export homewend asked for last, it asks you
whether to download it or to ask for a new one. When there is none, it asks
before asking Google: Google takes hours to prepare an export.

Stop it at any time with Ctrl-C and run the same command again: it resumes
where it was, and never asks Google twice.

Flags:
  --list          list the exports on Google Takeout
  --new           ask Google for a new export even if there is one
  --yes           answer yes to every question, for scripts
  --library DIR   where the photos go, for this run (the first run asks, and
                  remembers it)
  --export ID     this export, by the id --list shows
  --profile DIR   where the sign-in is kept (default: the one login made)
  --json          one JSON object per line, for scripts

Examples:
  homewend takeout 2025
  homewend takeout
  homewend takeout --list
  homewend takeout --library /mnt/nas/photos`,

	"help verify": `homewend verify — count a library again

Usage:
  homewend verify [DIR] [--export ID] [--json]

Counts the photos in the library against the list of the export downloaded
into it, year by year, and names every file that is missing. Nothing is
downloaded, nothing changes, and Google is not contacted.

Flags:
  --export ID     the export to count against, by the id takeout --list shows;
                  needed only when more than one was downloaded there
  --json          one JSON object per line, for scripts

Examples:
  homewend verify
  homewend verify /mnt/nas/photos`,

	"help ui": `homewend ui — serve the window of the desktop app

Usage:
  homewend ui

Serves the homewend window on 127.0.0.1 and prints its address once, on the
first line. The address carries a token: only who has it can open the page.
The desktop app starts this command and shows the address; it stops on
Ctrl-C, or when whoever started it closes its input.

Examples:
  homewend ui`,

	"help update": `homewend update — install the latest homewend

Usage:
  homewend update [--json]

Downloads the latest release from GitHub, checks it against the release's
checksums, and puts it in place of the homewend you are running. Your Google
sign-in, your photos and your library are not touched.

Once a day, at the end of a command, homewend asks GitHub whether a newer
release is out, and says so in one line. Nothing is installed until you run
this command.

Flags:
  --json          one JSON object per line, for scripts

Examples:
  homewend update`,

	// Sign-in.
	"checking":       "Checking your Google sign-in",
	"sign in":        "Waiting for you to sign in to Google in the new window",
	"session ready":  "Setting up your Google account",
	"signed in":      "Signed in",
	"signed in as":   "Signed in as **%s**",
	"already":        "Already signed in",
	"already as":     "Already signed in as **%s**",
	"after login":    "to bring your %d photos home, run: **homewend takeout %d**",
	"signed out":     "Signed out of Google",
	"was not signed": "Not signed in: there was nothing to sign out of",
	"password":       "Waiting for you to enter your password in the Google window",
	"prepare":        "Setting up the download",

	// Questions.
	"ask new":          "Ask Google to export your %s?",
	"ask new meta":     "Google takes hours to prepare the export. --yes skips this question.",
	"yes":              "Yes",
	"no":               "No",
	"have one":         "Google already has an export of your %s. What next?",
	"have one meta":    "%s · made %s · %s · ",
	"download it":      "Download it",
	"ask for new":      "Ask for a new export",
	"ask library":      "Where should your photos go?",
	"ask library meta": "Asked once and remembered. --library DIR overrides it for one run.",
	"key move":         "move",
	"key select":       "select",
	"key accept":       "accept",
	"key quit":         "quit",
	"not asked":        "Nothing asked: no export was requested",
	"photos of":        "**%d** photos",
	"all photos":       "photos",

	// The work.
	"looking":         "Looking for an export of your %s on Google Takeout",
	"asking":          "Asking Google for an export",
	"asked":           "Asked Google for an export",
	"asked at":        " at %s",
	"preparing":       "Google is preparing the export",
	"preparing hint":  "This takes hours. Ctrl-C is safe: run the same command to resume.",
	"found":           "Found export %s",
	"found details":   " · %d parts · %s",
	"in library":      " · already in the library",
	"in library now":  "Already in the library",
	"downloading":     "Downloading part %d of %d",
	"download sizes":  "%s of %s",
	"download left":   " · %s left",
	"downloaded":      "Downloaded part **%d of %d**",
	"downloaded in":   " · %s in %s",
	"network":         "Waiting for the network to come back",
	"network try":     " · attempt %d",
	"damaged":         "Part %d of %d was damaged on the way",
	"damaged again":   " · downloading it again",
	"unpacking":       "Unpacking part %d of %d",
	"sorting":         "Sorting photos",
	"sorted":          "Sorted **%d photos**",
	"sorted details":  " in %s · %d duplicates · %d undated · %d in albums",
	"checked":         "Checked **%d of %d**",
	"checked details": " against the export's list",
	"result":          "All %d photos are there",
	"export details":  " · %s · %d parts · %s",
	"result verify":   "All %d photos from your exports are in %s",
	"missing":         "%d photos are missing from the library",
	"fetch them":      "to fetch them, run: **%s**",
	"stopped":         "Stopped",
	"resume":          "to resume, run: **%s**",
	"elapsed":         " · %s",

	// takeout --list.
	"list title":     "Exports on Google Takeout for **%s**",
	"list head":      "id        made          size       parts  status     until     photos",
	"list row":       "%-8s  %-12s  %-9s  %-5d  %-9s  %-8s  %s",
	"list all":       "all",
	"list unknown":   "unknown",
	"list ready":     "to download the ready export, run: **homewend takeout%s**",
	"list none":      "No exports on Google Takeout for %s",
	"list none hint": "to ask Google for one, run: **homewend takeout %d**",

	// verify.
	"count": "%d   %d of %d",

	// status.
	"field google":  "Google",
	"field takeout": "Takeout",
	"field library": "Library",
	"field version": "Version",
	"not signed in": "not signed in",
	"not requested": "not requested",
	"not chosen":    "not chosen",
	"latest":        " · latest",
	"available":     " · %s available",
	"export of":     "%s · %s",
	"status start":  "to sign in to Google, run: **homewend login**",

	// update.
	"updating":    "Downloading homewend %s",
	"update got":  "Downloaded **homewend %s**",
	"update sums": "Checked the download against the release checksums",
	"updated":     "Updated homewend from %s to %s",
	"up to date":  "homewend %s is the latest version",
	"newer":       "to install homewend %s, run: **homewend update**",

	// Errors: what went wrong; the fix is the hint under it.
	"unknown command":   "unknown command %q",
	"unknown hint":      "to see the commands, run: **homewend help**",
	"not a year":        "%q is not a year",
	"year hint":         "to bring one year home, run: **homewend takeout %d**",
	"no library":        "no library folder chosen",
	"no library hint":   "to choose one, run: **%s --library DIR**",
	"session expired":   "your Google sign-in expired",
	"login then":        "to resume, run **homewend login**, then **%s**",
	"sign-in closed":    "the window closed before you finished signing in",
	"try login":         "to try again, run: **homewend login**",
	"try fallback":      "if Google showed an error, run: **homewend login --fallback**",
	"sign-in declined":  "Google did not sign you in",
	"session lost":      "you signed in, but the sign-in was not kept (%v)",
	"not signed in yet": "you are not signed in to Google",
	"sign in hint":      "to sign in, run: **homewend login**",
	"no browser":        "no Chrome, Chromium, Brave or Edge on this computer",
	"no browser hint":   "to use another, set **BROWSER_BIN** to its path",
	"no space":          "not enough disk space",
	"no space sizes":    "%s needed · %s free",
	"no space hint":     "to resume, free up space and run: **%s**",
	"not offered":       "Google Takeout has no photos of %d",
	"offered years":     "years it offers: %s",
	"not offered hint":  "to bring one of them home, run: **homewend takeout YEAR**",
	"expired":           "%v",
	"expired hint":      "to ask Google for a new export, run: **%s --new**",
	"no such export":    "%v",
	"list hint":         "to see them, run: **homewend takeout --list**",
	"not requested err": "Google did not take the request (%v)",
	"try again":         "to try again, run: **%s**",
	"no download":       "the window closed before the download began",
	"update damaged":    "the download was damaged on the way, and nothing was installed",
	"update again":      "to try again, run: **homewend update**",
	"error":             "%v",
}
