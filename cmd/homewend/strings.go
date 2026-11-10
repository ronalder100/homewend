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
On Windows, a double-click on homewend.exe asks, then does the same.

Commands:
  login     sign in to Google in a window of its own
  logout    sign out and forget the Google sign-in
  status    show the account, the export and the library
  takeout   bring photos home from Google Takeout: all, or a year
  import    bring in the photos of a card or a folder
  verify    count the library again against its exports
  ui        serve the window of the desktop app
  update    install the latest homewend
  version   show the version
  help      show help for a command

For the flags of a command, run: homewend help <command>
https://homewend.app`,

	"help login": `homewend login — sign in to Google, once

Usage:
  homewend login [--new | --account EMAIL] [--fallback] [--json]

Opens a small Chrome, Chromium, Brave or Edge window of its own and waits while
you sign in to Google there. Google asks you to let homewend see your email
address: that is how the window knows you are done, and nothing is read with
it. homewend never sees your password. A new account is asked, once, whose
photos they are: the folder of the library they go in, its profile. Run it
again whenever a command says the sign-in expired.

Flags:
  --new           sign in to another account, beside the ones there are
  --account EMAIL sign in to this account again
  --fallback      sign in on Google Takeout's page, in a full window: for when
                  the small window shows a Google error instead of the sign-in
  --json          one JSON object per line, for scripts

Examples:
  homewend login
  homewend login --new
  homewend login --fallback
  BROWSER_BIN=/usr/bin/brave-browser homewend login`,

	"help logout": `homewend logout — sign out of Google

Usage:
  homewend logout [--account EMAIL]

Forgets an account's Google sign-in: the next command that needs it asks you
to sign in again. Your photos and your library are not touched.

Flags:
  --account EMAIL the account to sign out of, when there are several

Examples:
  homewend logout`,

	"help status": `homewend status — where things stand

Usage:
  homewend status [--json]

Shows each Google account and its profile, the export homewend asked for last
for it, the library folder and the version. Nothing is downloaded and nothing
changes.

Flags:
  --json          one JSON object, for scripts

Examples:
  homewend status`,

	"help takeout": `homewend takeout — your photos, from Google Takeout to your disk

Usage:
  homewend takeout [YEAR] [--new] [--yes] [--library DIR] [--export ID]
                   [--account EMAIL] [--profile NAME] [--json]
  homewend takeout --list

Brings your Google Photos home, one year or all of them, into the library
folder, under the account's profile: the photos by year and month, with
your albums beside them and the pictures that came through chats apart, each
one counted against the list Google puts in the export.

If Google Takeout still has the export homewend asked for last, it asks you
whether to download it or to ask for a new one. When there is none, it asks
before asking Google: Google takes minutes to prepare a year, a few hours
for everything.

Leave it running and it downloads the export as soon as Google has it
ready. If you have to stop it, with Ctrl-C or by closing the terminal, run
the same command again: it resumes where it was, and never asks Google twice.

Flags:
  --list          list the exports on Google Takeout
  --new           ask Google for a new export even if there is one
  --yes           answer yes to every question, for scripts
  --library DIR   where the photos go, for this run (the first run asks, and
                  remembers it)
  --export ID     this export, by the id --list shows
  --account EMAIL the account, when there are several (otherwise you are asked)
  --profile NAME  whose photos, for this run: the library's folder they go in
  --json          one JSON object per line, for scripts

Examples:
  homewend takeout 2025
  homewend takeout
  homewend takeout --list
  homewend takeout --library /mnt/nas/photos`,

	"help import": `homewend import — bring in the photos of a card or a folder

Usage:
  homewend import [DIR] [--profile NAME] [--link] [--include FOLDER]
                  [--bring-back [NAMES]] [--json]
  homewend import --dry-run [DIR] [--json]

Without DIR, the card that is in: a disk with a camera's DCIM folder on it,
or one chosen among several.

Copies the photos and videos under DIR into the library, reads each copy back
to check it, and files it by the date the camera wrote. A photo the library
already holds is found by its content, whatever its name, and kept once.
Files that are not photos or videos stay where they are; nothing on DIR
changes, and nothing is removed from a card.

Some folders stay out by themselves, and are said: a project (a folder with
.git, package.json and the like), an app's previews (Lightroom, Apple
Photos), and a folder whose images mostly have no camera's date (scans,
screenshots, photos sent or saved). A card's DCIM folder always comes in.

With --dry-run nothing is copied: it tells, by year, what would come in.

Flags:
  --dry-run       only tell what would happen
  --profile NAME  whose photos: the library's folder they go in; asked when
                  there are several
  --link          give each file a second name in the library instead of a
                  copy: no space used, but only for a folder on the library's
                  disk, and a change to one is a change to both
  --date NAMES=WHEN
                  the right date of photos whose camera had it wrong, a
                  clock reset to 1 January: the first of NAMES takes WHEN,
                  the others keep their distance from it. The photos are
                  not changed: the date goes in a NAME.xmp beside each copy,
                  which Immich reads, and digiKam or darktable when set to.
                  A photo already in the library moves to its new date.
                  A name in two folders is given by its path. Repeatable
  --include FOLDER
                  bring in a folder that stays out by itself, named as
                  --dry-run lists it ("." for DIR itself). Repeatable
  --bring-back [NAMES]
                  bring in again photos deleted from the library, by name
                  as the run lists them, or all when no name follows: by
                  default a card that still holds them leaves them out.
                  Repeatable
  --json          one JSON object per line, for scripts

Examples:
  homewend import
  homewend import --dry-run
  homewend import /media/card
  homewend import ~/Pictures/old-laptop --link
  homewend import ~/Backup --include Scans
  homewend import --bring-back DSCF0010.JPG
  homewend import --date DSCF0004.JPG,DSCF0005.JPG=2026-09-14T10:20`,

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
	"checking":        "Checking your Google sign-in",
	"sign in":         "Waiting for you to sign in to Google in the new window",
	"session ready":   "Setting up your Google account",
	"signed in":       "Signed in",
	"signed in as":    "Signed in as **%s**",
	"already":         "Already signed in",
	"already as":      "Already signed in as **%s**",
	"after login":     "to bring your %d photos home, run: **homewend takeout %d**",
	"signed out":      "Signed out of Google",
	"signed out of":   "Signed out of **%s**",
	"profile is":      "Your photos go in the profile **%s**",
	"which account":   "Which account?",
	"which profile":   "Whose photos?",
	"sign-in expired": " · sign-in expired",
	"was not signed":  "Not signed in: there was nothing to sign out of",
	"password":        "Waiting for you to enter your password in the Google window",
	"prepare":         "Setting up the download",
	"prepare hint":    "The first download of an account: Google may ask for your password, in a small window.",
	"password hint":   "Google asks once, before the first download. Homewend never sees it.",

	// Questions.
	"ask new":          "Ask Google to export your %s?",
	"yes":              "Yes",
	"no":               "No",
	"have one":         "Google already has an export of your %s. What next?",
	"have one meta":    "%s · made %s · %s · ",
	"download it":      "Download it",
	"ask for new":      "Ask for a new export",
	"ask library":      "Where should your photos go?",
	"ask library meta": "Asked once and remembered. --library DIR overrides it for one run.",
	"guided year":      "Which photos?",
	"guided year meta": "One year to try it, like 2025, or leave it empty for all of them.",
	"guided bad year":  "Not a year: type one like 2025, or leave it empty for all of them.",
	"guided done":      "Press Enter to close.",
	"key move":         "move",
	"key select":       "select",
	"key accept":       "accept",
	"key quit":         "quit",
	"not asked":        "Nothing asked: no export was requested",
	"photos of":        "**%d** photos",
	"all photos":       "photos",

	// The work.
	"looking":                "Looking for an export of your %s on Google Takeout",
	"asking":                 "Asking Google for an export",
	"asking hint":            "Takeout takes two or three minutes to take the request.",
	"asked":                  "Asked Google for an export",
	"asked at":               " at %s",
	"preparing":              "Google is preparing the export",
	"preparing hint":         "Minutes for a year, a few hours for everything. The download starts by itself as soon as it is ready.",
	"preparing lead":         "Have to go?",
	"preparing note":         "Close the terminal and run the same command later: it picks up where it left off.",
	"found":                  "Found export %s",
	"found details":          " · %d parts · %s",
	"in library":             " · already in the library",
	"in library now":         "Already in the library",
	"downloading":            "Downloading part %d of %d",
	"download sizes":         "%s of %s",
	"left":                   " · %s left",
	"pace":                   " · %s/s",
	"downloaded":             "Downloaded part **%d of %d**",
	"downloaded in":          " · %s in %s",
	"network":                "Waiting for the network to come back",
	"network try":            " · attempt %d",
	"damaged":                "Part %d of %d was damaged on the way",
	"damaged again":          " · downloading it again",
	"unpacking":              "Unpacking part %d of %d",
	"sorting":                "Sorting photos",
	"sorted":                 "Sorted **%d photos**",
	"sorted details":         " in %s · %d duplicates · %d undated · %d in albums",
	"checked":                "Checked **%d of %d**",
	"checked details":        " against the export's list",
	"result":                 "All %d photos are there",
	"export details":         " · %s · %d parts · %s",
	"result verify":          "All %d photos from your exports are in %s",
	"photos":                 "%d photos",
	"photos one":             "1 photo",
	"videos":                 "%d videos",
	"videos one":             "1 video",
	"files":                  "%d files",
	"files one":              "1 file",
	"import one folder":      "one folder at a time",
	"import cannot read":     "%s cannot be read: is it yours to open?",
	"import not found":       "%s is not there",
	"import not found photo": "a photo deleted from the library is brought back so: **homewend import --bring-back %s**",
	"import not a folder":    "%s is a file, not a folder",
	"import unreadable":      "%d files or folders could not be read: they are left out",
	"import unreadable one":  "1 file or folder could not be read: it is left out",
	"library gone":           "the library folder %s is not there: is its disk connected?",
	"library gone hint":      "connect the library's disk, then run: **%s**",
	"reading":                "Reading the photos",
	"copying":                "Bringing in",
	"imported undated":       "%d with no date the camera wrote: in undated/",
	"result imported":        "%s brought in, %s, each read back, in %s's photos",
	"result linked":          "%s linked into %s's photos: no space used",
	"brought":                "%d photos and videos",
	"brought one":            "1 photo or video",
	"import whose":           "whose photos are these? there is no profile yet",
	"import whose hint":      "name one: **%s --profile NAME**",
	"no link":                "the disk refused a hard link: is the folder on the library's disk?",
	"no link hint":           "to copy it instead, run the same command without --link",
	"bad date":               "%s is not NAMES=YYYY-MM-DD or NAMES=YYYY-MM-DDTHH:MM",
	"name twice":             "%s is in more than one folder: %s",
	"name twice hint":        "say which, by its path: **%s**",
	"not redated":            "%d are filed in another profile: their date is given from there",
	"redated dry":            "%d would move to the date you gave",
	"redated":                "%d with the date you gave, written in a .xmp beside each",
	"no such photo":          "%s is not among the photos and videos of the folder",
	"date hint":              "give it as: **--date DSCF0004.JPG,DSCF0005.JPG=2026-09-14T10:20**",
	"library waiting":        "Waiting for another import to finish placing photos",
	"library busy":           "another import or download is placing photos in this library",
	"library busy hint":      "when it is done, run: **%s**",
	"bad profile":            "a profile is a name: no / or \\, and not starting with a dot",
	"folder gone":            "%s was taken out before the end: what was copied is in the library",
	"folder gone hint":       "put it back, then run: **%s**",
	"card found":             "Found a card: %s",
	"no card":                "no card found: no disk with a camera's DCIM folder is in",
	"no card hint":           "put the card in, or name a folder: **homewend import DIR**",
	"cards found":            "more than one card is in: %s",
	"one folder hint":        "name one: **homewend import DIR**",
	"which card":             "Which card?",
	"import no date":         "no date",
	"group project":          "a project (%s): its images stay out",
	"group previews":         "%s's own previews of other photos: they stay out",
	"group unsure":           "most have no camera's date: scans, screenshots, photos sent or saved; they stay out",
	"group include":          "to bring one in: --include FOLDER",
	// The last line of a look or a copy that brings nothing new, by how the
	// engine says it ended; "dry" for a look. {n} and {dir} are filled in.
	"outcome redated":      "Nothing new to copy: {n} moved to the date you gave",
	"outcome redated dry":  "Nothing new to copy: {n} would move to the date you gave",
	"outcome left-out":     "Nothing new came in by itself: the folders above stay out",
	"outcome left-out dry": "Nothing would come in by itself: the folders above stay out",
	"outcome deleted":      "Nothing new: the photos deleted from the library stay out",
	"outcome known-noted":  "Nothing new among what could be read",
	"outcome known":        "Nothing new: every photo and video is already in the library",
	"outcome unread":       "No photos or videos could be read in {dir}",
	"outcome empty":        "No photos or videos in {dir}",
	"deleted back":         "%d deleted from the library come in again",
	"deleted back one":     "1 deleted from the library comes in again",
	"deleted back dry":     "%d deleted from the library would come in again",
	"deleted back dry one": "1 deleted from the library would come in again",
	"import deleted":       "%s · to bring one back: --bring-back NAME; all: --bring-back",
	"deleted":              "%d deleted from the library stay out",
	"deleted one":          "1 deleted from the library stays out",
	"import others card":   "Not photos, left on the card:",
	"import others folder": "Not photos, left in the folder:",
	"names more":           "and %d more",
	"bad include":          "%s is not a folder of photos in the folder imported",
	"bad include hint":     "the folders are named as **homewend import --dry-run DIR** lists them",
	"import already":       "%d already in the library, found by content",
	"import twins":         "%d twice in the folder, brought in once",
	"result import":        "%s, %s",
	"missing":              "%d photos are missing from the library",
	"fetch them":           "to fetch them, run: **%s**",
	"stopped":              "Stopped",
	"resume":               "to resume, run: **%s**",
	"elapsed":              " · %s",
	"please wait":          " · please wait…",

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
	"no such account":   "%v",
	"accounts hint":     "to see the accounts, run: **homewend status**",
	"which account err": "more than one account is signed in",
	"account hint":      "to say which, run: **%s --account EMAIL**",
	"no profile":        "there is no profile yet",
	"which profile err": "there is more than one profile",
	"profile hint":      "to say which, run: **%s --profile NAME**",
}
