// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

// Every sentence the CLI shows a person. A second language is a second table.
var text = map[string]string{
	"usage": `homewend — Google Photos, on your own disk.

Usage:
  homewend <command> [flags]

Commands:
  login     sign in to Google, once, in your own browser
  logout    sign out: delete the browser profile login made
  get       bring your photos from Google into a folder
  takeouts  list the exports on your Google Takeout
  verify    count a library again against the export's manifest
  update    get the latest homewend
  version   print the version
  help      show help for a command

Start here:
  homewend login
  homewend get --year 2025 --library ~/Pictures/Homewend

Run "homewend help <command>" for its flags and examples.

https://homewend.app`,

	"help login": `homewend login — sign in to Google, once

Usage:
  homewend login [--profile DIR] [--fallback] [--json]

Opens a small Chrome, Chromium, Brave or Edge window on a profile of its own
and waits while you sign in to Google there. Google asks you to let Homewend
see your email address: that is how the window knows you are done, and nothing
is read with it. Homewend never sees your password: it keeps the browser
profile, and every later command reads the session from it. Run it again
whenever a command says the session expired.

Flags:
  --profile DIR   browser profile to use (default: homewend/profile in your
                  config directory)
  --fallback      sign in on Google Takeout's page, in a full browser window:
                  for when the small window shows a Google error instead of
                  the sign-in
  --json          one JSON object per line, for scripts

Examples:
  homewend login
  homewend login --fallback
  BROWSER_BIN=/usr/bin/brave-browser homewend login`,

	"help logout": `homewend logout — sign out of Google

Usage:
  homewend logout [--profile DIR]

Deletes the browser profile login made, and with it the Google session: the
next command that needs Google asks you to sign in again. Your photos and
libraries are not touched, and homewend still knows which year each export
it asked for holds.

Flags:
  --profile DIR   browser profile to delete (default: homewend/profile in your
                  config directory)

Examples:
  homewend logout
  homewend logout && homewend login`,

	"help get": `homewend get — your photos, from Google to your disk

Usage:
  homewend get --library DIR [--year YYYY | --takeout ID] [--new]
               [--profile DIR] [--json]

Brings your Google Photos — one year, or all of them — into a folder: the
photos by date, with your albums beside them, each one counted against the
list Google puts in the export.

If Google Takeout still has the export of that year Homewend asked for last,
it asks you whether to download that one or to ask Google for a new one. When
there is none, or it has expired, it asks you before asking Google: Google
takes hours to prepare an export.

Stop it at any time and run the same command again: it carries on where it
was, without a question, and never asks Google twice.

Flags:
  --library DIR   where your photos go (required)
  --year YYYY     only this year (default: all of your photos)
  --takeout ID    this export, by the id homewend takeouts shows
  --new           ask Google for a new export even if there is one, for the
                  photos taken since
  --profile DIR   browser profile to use (default: the one login made)
  --json          one JSON object per line, for scripts

Examples:
  homewend get --year 2025 --library ~/Pictures/Homewend
  homewend get --library /mnt/nas/photos
  homewend get --takeout 8f6c3233 --library ~/Pictures/Homewend`,

	"help takeouts": `homewend takeouts — the exports on your Google Takeout

Usage:
  homewend takeouts [--profile DIR] [--json]

Lists every export Google Takeout keeps for your account, newest first: its
id, when it was made, its size, and whether it is ready, still being prepared,
or expired. Google keeps an export for 7 days after it is ready. For the ones
Homewend asked for, it also says which year they hold; Google does not say it
for the others.

Download one of them with get --takeout ID.

Flags:
  --profile DIR   browser profile to use (default: the one login made)
  --json          one JSON object per line, for scripts

Examples:
  homewend takeouts
  homewend get --takeout 8f6c3233 --library ~/Pictures/Homewend`,

	"help verify": `homewend verify — count a library again

Usage:
  homewend verify --library DIR [--takeout ID] [--json]

Counts the photos in the library against the list of an export already
downloaded into it, year by year, and names every file that is missing.
Nothing is downloaded and nothing is changed, and Google is not contacted.

Flags:
  --library DIR   the library to check (required)
  --takeout ID    the export to check it against, by the id homewend takeouts
                  shows; needed only when more than one was downloaded there
  --json          one JSON object per line, for scripts

Examples:
  homewend verify --library ~/Pictures/Homewend
  homewend verify --library ~/Pictures/Homewend --takeout 8f6c3233`,

	"help update": `homewend update — get the latest homewend

Usage:
  homewend update [--json]

Downloads the latest release from GitHub, checks it against the release's
checksums, and puts it in place of the homewend you are running. Your Google
session, your photos and your libraries are not touched.

Once a day, at the end of a command, homewend asks GitHub whether a newer
release is out, and says so in one line. Nothing is installed until you run
this command.

Flags:
  --json          one JSON object per line, for scripts

Examples:
  homewend update`,

	"unknown command": "unknown command %q\n",

	"sign in":           "sign in to Google in a separate window",
	"signed in":         "signed in",
	"signed in as":      "signed in as %s",
	"signed out":        "signed out: the browser profile is deleted",
	"was not signed in": "not signed in: there was no browser profile to delete",
	"all photos":        "all your photos",
	"photos of":         "your photos of %d",
	"confirm new":       "you are about to ask Google for a new takeout of %s. Continue? [Y/n] ",
	"have one":          "Google already has a takeout of %s\n%s · made %s · %s · %s\n\n",
	"option":            "  %s  %s\n",
	"download that":     "download that one",
	"ask for new":       "ask Google for a new one",
	"choose":            "choose [1]: ",
	"not asked":         "nothing asked: no new export was requested",
	"already signed in": "already signed in",
	"already as":        "already signed in as %s",
	"not signed in":     "sign-in did not finish: run homewend login again",
	"sign-in closed":    "the sign-in window was closed before Google finished: run homewend login again, or homewend login --fallback if the window showed a Google error",
	"sign-in declined":  "Google did not sign you in: run homewend login again",
	"session lost":      "you signed in, but the session was not kept (%v): run homewend login again",
	"no browser":        "no Chrome, Chromium, Brave or Edge found on this machine: install one, or set BROWSER_BIN to its path",
	"in library":        "\nthis export is already all in this folder: nothing to download, checking it again",
	"takeouts header":   "id        made              size       parts  status     until             holds",
	"takeout row":       "%-8s  %-16s  %9s  %5d  %-9s  %-16s  %s",
	"holds year":        "%d",
	"holds all":         "all photos",
	"holds unknown":     "unknown",
	"no takeouts":       "Google Takeout has no exports for this account",
	"request":           "asking Google Takeout for an export",
	"not offered":       "Google Takeout offers no export of %d. It offers these years: %s\nand these albums:\n  %s",
	"waiting":           "Google is preparing the export: this can take hours.\nLeave this open and the download starts when it is ready, or close it and run the same command later.",
	"restart":           "If the computer restarts, run the same command again.",
	"ready":             "\nthe export is ready: downloading %d parts, %s",
	"stopped":           "stopped: run the same command again to carry on where it was",
	"not requested":     "the export was not requested: %v",
	"first download":    "Google asks for your password once more, in the window that opened: type it there. It is needed once for this account, and the window closes by itself",
	"first download of": "Google asks for the password of %s once more, in the window that opened: type it there. It is needed once for this account, and the window closes by itself",
	"no download":       "the browser was closed before a download started: run the same command again",
	"no such takeout":   "%v: run homewend takeouts to see them",
	"expired":           "%v: Google keeps an export for 7 days after it is ready. Run get without --takeout to ask for a new one",
	"no space":          "not enough space for this export: %s needed, %s free",

	"checking status": "checking your Google session",
	"session status":  "setting up your Google account",
	"asking status":   "sending the request to Google",
	"waiting status":  "waiting for Google, it can take a few hours",
	"download status": "[%d/%d] %s · %s of %s",
	"download rate":   " · %s/s · %s left",
	"place status":    "%s  placing %d of %d",
	"update status":   "getting homewend %s",
	"update sizes":    " · %s of %s",

	"newer":          "\nhomewend %s is out: run homewend update",
	"up to date":     "homewend %s is the latest",
	"updated":        "homewend %s is installed (it was %s)",
	"update damaged": "the download was damaged on the way, and nothing was installed: run homewend update again",

	"get needs library": `library folder missing:

  homewend get %s--library ~/Pictures/Homewend`,
	"verify needs library": `library folder missing:

  homewend verify %s--library ~/Pictures/Homewend`,
	"download":        "[%d/%d] %s  from %s of %s",
	"downloaded":      "[%d/%d] downloaded, %s",
	"short":           "[%d/%d] %s  stopped at %s of %s; run again to continue",
	"retry":           "network: %s (attempt %d; waiting for it to come back)",
	"damaged":         "[%d/%d] damaged on the way: downloading it again",
	"unpack":          "[%d/%d] unpacking",
	"place":           "placing %d of %d",
	"placed":          "placed %d photos, %s: %d duplicates, %d undated, %d in albums",
	"verified":        "declared %d, on disk %d, missing %d",
	"year":            "  %s  %d of %d",
	"complete":        "download complete, congratulations 🎉",
	"missing file":    "  missing: %s",
	"session expired": "the Google session expired: run homewend login, then the same command",
	"error":           "error: %v",
}
