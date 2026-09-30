# Principles

Rules the code follows. Each one closes a door on purpose; changing one is a
decision, not a refactor.

## 1. The user performs every interaction with Google's interface

The app never fills a field, clicks a control or submits anything on a Google
page. It opens a browser at a URL, reads pages the user can see, reads a status
page at the pace of a person refreshing a tab, and downloads files from links
Google issued.

Google's terms (in force since 22 May 2024) forbid "using automated means to
access content from any of our services in violation of the machine-readable
instructions on our web pages (for example, robots.txt files …)". The pages we
read are allowed by `takeout.google.com/robots.txt`; it disallows `*/_/*` and
`/u/*`. Whether the app may also press controls on those pages is an open
decision; until it is taken, this rule stands.

**Making a control usable is allowed; using it is not.** Google keeps some
controls hidden or disabled (the per-year selector in the Photos options). The
app may remove `display: none` and `disabled` so the control exists; the user
makes the selection with their own clicks.

## 2. No disguise

Sign-in happens in a real browser. The app does not rewrite client hints,
spoof user agents, or hide automation to get past a Google policy. If Google
refuses something, we find another way or we do not do it.

## 3. No reimplementation of Google's internal protocols

Data is read from pages and links Google serves to the user. Internal RPCs
(`batchexecute` and similar) are not called or imitated.

## 4. Rate, not stealth

One part at a time, one connection per part, status polling at human cadence.
Never more requests than a person would produce.

## 5. No file is ever modified

What Google sent is what lands. No EXIF or XMP writing, no conversion, no
"repair". Sidecars are read to place and index files, never merged into them.

## 6. Plain files first

The library is ordinary folders, readable without homewend. Any database is an
index that can be deleted and rebuilt from the files.

## 7. We host nothing

No user content ever passes through a server of ours. The app talks to Google
and to the user's own disk.

## 8. Say what cannot be checked

The manifest proves that everything Google put in the export reached the disk.
Nobody outside Google can prove the export contains the whole library, and the
app never claims it does.

## 9. The mobile app shares no code with this repository

This repository is AGPL-3.0; the phone app is a separate, closed program that
talks to the desktop over the network. A shared file would carry the AGPL with
it.
