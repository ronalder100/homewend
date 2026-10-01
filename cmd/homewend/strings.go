// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

// Every sentence the CLI shows a person. A second language is a second table.
var text = map[string]string{
	"usage": `homewend — Google Photos, on your own disk.
The open-source Google Photos downloader.

Usage:
  homewend <command> [flags]

Commands:
  login     sign in to Google, once, in your own browser
  get       ask Google for your photos, download them, check them
  fetch     download the newest export that is already ready
  verify    count a library again against the export's manifest
  version   print the version
  help      show help for a command

Start here:
  homewend login
  homewend get --year 2025 --library ~/Pictures/Homewend

Run "homewend help <command>" for its flags and examples.
Exit codes: 0 done, 1 error, 2 something is missing, 3 not signed in.
https://homewend.app`,

	"help login": `homewend login — sign in to Google, once

Usage:
  homewend login [--profile DIR] [--json]

Opens Chrome, Chromium, Brave or Edge on a profile of its own and waits while
you sign in to Google there. Homewend never sees your password: it keeps the
browser profile, and every later command reads the session from it. Run it
again whenever a command says the session expired.

Flags:
  --profile DIR   browser profile to use (default: homewend/profile in your
                  config directory)
  --json          one JSON object per line, for scripts

Examples:
  homewend login
  BROWSER_BIN=/usr/bin/brave-browser homewend login`,

	"help get": `homewend get — your photos, from Google to your disk

Usage:
  homewend get --library DIR [--year YYYY] [--profile DIR] [--json]

Asks Google Takeout for an export of your Google Photos — one year, or all of
them — waits while Google prepares it, downloads every part, files the photos
by date with your albums beside them, and counts them against the manifest
Google puts in the export.

Stop it at any time and run the same command again: it carries on where it
was, and never asks Google for a second export.

Flags:
  --library DIR   where your photos go (required)
  --year YYYY     only this year (default: all of your photos)
  --profile DIR   browser profile to use (default: the one login made)
  --json          one JSON object per line, for scripts

Examples:
  homewend get --year 2025 --library ~/Pictures/Homewend
  homewend get --library /mnt/nas/photos`,

	"help fetch": `homewend fetch — download an export that is already ready

Usage:
  homewend fetch --library DIR [--profile DIR] [--json]

For an export you asked Google Takeout for yourself. Finds the newest export
that is ready, downloads every part, puts the photos into the library, and
counts them against the manifest. Checks there is room on the disk first.
Stop it and run it again: it resumes from the byte where it stopped.

Flags:
  --library DIR   where your photos go (required)
  --profile DIR   browser profile to use (default: the one login made)
  --json          one JSON object per line, for scripts

Example:
  homewend fetch --library ~/Pictures/Homewend`,

	"help verify": `homewend verify — count a library again

Usage:
  homewend verify --library DIR --job ID [--json]

Counts the photos in the library against the manifest of an export already
downloaded into it, year by year, and names every file that is missing.
Nothing is downloaded and nothing is changed. The export id is the name of
its folder under DIR/.homewend/.

Flags:
  --library DIR   the library to check (required)
  --job ID        the export to check it against (required)
  --json          one JSON object per line, for scripts

Example:
  homewend verify --library ~/Pictures/Homewend --job <export id>`,

	"unknown command": "unknown command %q\n",

	"sign in":           "a browser window is open: sign in to Google there",
	"signed in":         "signed in",
	"already signed in": "already signed in",
	"not signed in":     "the browser was closed before sign-in finished: run homewend login again",
	"no browser":        "no Chrome, Chromium, Brave or Edge found on this machine: install one, or set BROWSER_BIN to its path",
	"export":            "export %s of %s: %d parts, %s",
	"request":           "asking Google Takeout for an export",
	"not offered":       "Google Takeout offers no export of %d. It offers these years: %s\nand these albums:\n  %s",
	"waiting":           "Google is preparing the export: this can take hours.\nLeave this open, the download starts when it is ready.\nIf the computer restarts, run the same command again.",
	"ready":             "\nthe export is ready: downloading %d parts, %s",
	"stopped":           "stopped: run the same command again to carry on where it was",
	"not requested":     "the export was not requested: %v",
	"first download":    "a browser window is open on your export: click Download on the first part, and enter your password if Google asks. This is needed once for this account; the window closes by itself once the download starts",
	"no download":       "the browser was closed before a download started: run the same command again",
	"no export":         "no export is ready on Google Takeout yet",
	"no space":          "not enough space for this export: %s needed, %s free",

	"asking status":   "asking Google for the export",
	"waiting status":  "waiting for Google",
	"download status": "[%d/%d] %s · %s of %s",
	"download rate":   " · %s/s · %s left",
	"place status":    "%s  placing %d of %d",

	"missing flags":   "missing: %s",
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
	"complete":        "\ndownload complete, congratulations 🎉",
	"missing file":    "  missing: %s",
	"session expired": "the Google session expired: run homewend login, then the same command",
	"error":           "error: %v",
}
