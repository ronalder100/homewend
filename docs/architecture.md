# Architecture

How homewend gets a Google Photos Takeout export onto a disk, and why each
piece is shaped the way it is. Every claim here was measured against real
exports in September 2026 — one of 344 GB in 159 parts, one of 4 MB in one
part — unless it says otherwise.

The rules that constrain all of it are in [`principles.md`](principles.md).

---

## The flow

| step | who |
|---|---|
| Open the system browser, on a profile the app owns, at `takeout.google.com/settings/takeout/custom/photos` (only Photos preselected) | app |
| Sign in, pick format and archive size, arm the schedule, create the export | **user** |
| Read `/manage` at the pace of a person refreshing a tab until the export exists | app |
| Read the part list and the manifest position, build every download URL | app |
| Download one part at a time, resumable, with the session re-read from the profile | app |
| Unpack, place each photo by date and album, count against the manifest | app |

Two human interactions in total: three fields once, and possibly a password
when the export is ready.

---

## Session: the browser profile is ours

### Why not an embedded browser

Google refuses sign-in inside Electron (*"Couldn't sign you in — this browser
or app may not be secure"*) even with a Chrome user-agent: the client hints
(`Sec-CH-UA`, `navigator.userAgentData.brands`) say `Electron`. Rewriting them
would work and is not done — see `principles.md`. OAuth for desktop apps
requires the system browser anyway.

### What works

The machine's own Chromium-family browser, launched with a **profile directory
created by the app**. Google accepts it because it is a real browser, two-step
verification included; from an empty profile the whole sign-in takes about
fifty seconds.

Owning the profile means owning its cookie store, and that is the point:

- **A copied `Cookie:` header dies in under twenty minutes.** Google rotates
  `SIDCC` continuously (`RotateCookiesPage` in the network log).
- **A profile re-read before every request follows the rotation.** Measured
  alive for 61 minutes of continuous downloading, every response `206`. The
  12–20 hours of a full export are not measured yet.

So the downloader is a plain process: the browser is needed for sign-in, then
the user closes it.

### Reading the cookies

- Launch with `--password-store=basic` (Linux) and `--use-mock-keychain`
  (macOS), otherwise cookies are encrypted with a keyring key that differs on
  every machine, and on a Mac reading it means a Keychain prompt. With the mock
  keychain the password is `mock_password`, derived with 1003 iterations
  (Chromium's `crypto/apple/fake_keychain_v2.mm` and
  `components/os_crypt/async/browser/keychain_key_provider.mm`). Read in the
  source, not yet tested on a Mac.
- On Linux and macOS, Chrome's cookie values are AES-128-CBC with a fixed key;
  since Chrome 130 the plaintext starts with a 32-byte SHA-256 of the domain,
  which must be stripped.
- **On Windows the key is protected with DPAPI.** Not implemented yet; it is the
  one real Windows risk, and it lives in the engine, not in the UI.

### Handing the window back

When sign-in lands, a page of ours comes up in the same window, in front of
Google's (*"You're signed in — the browser isn't needed any more"*). Closing the
window stays the user's gesture.

**No debug port.** Google refuses sign-in in a browser launched with
`--remote-debugging-port` (*"Couldn't sign you in — this browser or app may
not be secure"*) and accepts the same browser, same profile flags, without it:
Chromium 144, 2026-09-26, two empty profiles side by side. So the tab cannot be
replaced; the page is opened the browser's own way instead, by launching it
again on the same profile with the page as a `data:` URL. The running window
takes the address and shows it in a new tab, the second process exits, and
nothing of ours has to keep serving the page.

Sign-in is seen in the profile, not on the page: `SID` and `SSID` appear in the
cookie store. Chrome commits cookies every 30 seconds or 512 changes
(`net/extras/sqlite/sqlite_persistent_cookie_store.cc`), so the handover comes
up to half a minute after Google's page shows the user signed in.

---

## Takeout: Google's pages

### `/manage`

It is a JavaScript application, but the data is in the HTML, inside the
`AF_initDataCallback` blocks Google uses to seed the page. Each export is a
JSON array that starts with the marker `"ac.t.ta"`, repeated several times on
the page. The fields read, by position (two exports, one live and one expired,
2026-09-26):

| field | holds |
|---|---|
| 1 | job id |
| 6 | declared bytes, manifest excluded |
| 8 | the parts, each `[filename, bytes, downloads so far, …]`; `null` once expired |
| 22 | created, milliseconds since the epoch — the timestamp in the filenames |
| 24, 25 | expiry while live; the day it expired, once expired |
| 27 | the manifest's archive, shaped like a part; `null` once expired |
| 31 | the long user id |

So one page gives every export, whether it can be downloaded, and every part by
name and size: the report page is not needed. **Names are read, never
constructed**: in the 159-part export, 14 parts were not zips but bare video
files.

The marker matters: a bare UUID picked at random also matches things that are
not exports (the same page carries one that belongs to YouTube).

### The manifest

`archive_browser.html` lists every file in the export with its folder. Its
archive is field 27 of the record, and it sits **one index past the parts**
(parts `i=0…158`, manifest `i=159`; `i=N+1` answers 500). Its bytes are
**not** included in the declared size, which the disk-space check adds.

It counts files, not photos: sidecars are included, and the same photo appears
once per album plus once in `Photos from <year>`. 104,547 entries in the large
export are about 38,500 distinct photos.

### Asking for an export

The form at `/settings/takeout/custom/photos` is filled in a headless browser
on our own profile — the profile itself, never a copy of its cookies: after
about ten runs on copies, on 2026-09-26, Google revoked the profile's session,
while the profile driven directly kept it. A signed-in profile launched with a
debug port is served; only sign-in refuses one.

Choosing a year goes through Takeout's content picker, a dialog with one
checkbox per album and per year (measured 2026-09-26, one account, 133
entries):

- **It has no data until Google sends it.** The page polls the `OIek4b` RPC
  (`pollForContainerData`) every ~3.3 s; the album list arrives 1 m 34 s to
  2 m 3 s after the page loads, six loads, nothing cached between them. Only
  then does Google show and enable the picker's button (`jsname` `DNRMdf`).
  Pressed before, forced visible, Google's handler reads a list that is still
  null and fails (`Cannot read properties of null (reading 'Zc')`). So the
  button is pressed only once Google has enabled it.
- **Every checkbox's `value` is Google's id** for the entry. A year's id holds
  the year and nothing else: `EgUyAwjpDw` is protobuf `{2: {6: {1: 2025}}}`.
  The same ids with the page in English and in German, and the names did not
  change with the page's language either. A year is found by decoding the ids
  Google sends, never by its name. Years with no photos are simply absent
  (2008 and 2024, on this account).
- "Deselect all" is `jsname` `Si7An`, OK is `data-id` `EBS5u` — the value
  Google's own handler checks for. Deselect all clears the boxes
  asynchronously.
- The dialog slides in: a control's position is read until it holds still,
  then pressed. Each press that lands on the picker's button opens its own
  copy of the dialog.

A year asked for and requested this way, 2025, came back as one 4.2 MiB part:
32 photos declared, 32 on disk.

---

## Download

### The URL is built, and only cookies are needed

The link on Google's page goes through `takeout.google.com/takeout/download`,
which demands a fresh re-authentication proof (`rapt`, valid 12–16 minutes).
The host that actually serves the bytes does not:

```
https://takeout-download.usercontent.google.com/download/<filename>
  ?j=<job-id>&i=<index>&user=<short-user-id>&authuser=0
```

Cookies only, `accept-ranges: bytes`, resume from any offset. Verified by
downloading one part in two separate segments and comparing sha256 with the
browser's copy.

Requests carry the headers of a real navigation (`Sec-Fetch-Dest: document`
and friends): the same URL returns the file when navigated and HTML when
fetched from page JavaScript.

### The five-downloads limit is not ours to manage

Takeout allows five downloads per archive within seven days. **The counter
increments on the `takeout.google.com` redirect** — even when not one byte
arrives — and not on the file host. Four completed downloads from the file host
left it at zero. Building the URL spends no attempts.

### The short user id

`user=` wants a short numeric id. The long account id returns 400, and so does
omitting it. The short id appears on no page and in no cookie: Google's own
download links carry the long id, and the short one shows up only at the end of
their redirect — which, for a program with a valid session, is a redirect to
the password prompt (measured 2026-09-26). **We do not reimplement the internal
call that produces it.**

Chrome records every download's redirect chain in the profile's `History`
(`downloads_url_chains`), and the last link is the download host with the short
id. So it is read from there, once, and kept beside the profile: the user
downloads one part of an export in our window, entering the password if Google
asks, and can cancel it as soon as it starts. The same id served two exports a
week apart.

### Rules for the downloader

- **Disk space checked before starting**, including unpacking margin and the
  manifest.
- **One part at a time, one connection per part.** Parallelism buys nothing and
  would exceed what a person produces.
- **State on disk, written atomically** (`.tmp` + `rename`): for every part,
  arrived / partial / failed, so a crash or a closed lid resumes exactly.
- **Tell a dead network from an expired session**, and say which one it is
  instead of retrying blindly.
- **Nothing is estimated**: counters per phase, not invented percentages.

---

## Library

- **Detect gaps in the numbered part set** before unpacking.
- **Unpack with zip-slip protection.**
- **Date from the sidecar JSON, in local time.** Never the file's mtime:
  Takeout sets it to the day of the export.
- **Duplicates by content**, with one hashing implementation only, so two hashes
  computed differently are never compared. Google delivered the same video
  twice for 7% of the large export.
- **Albums as hardlinks** to the one file under its year.
- **Origin** read from the Android package when present.
- **Verification by name against the manifest**, with counts per year.
- **A SQLite catalogue** for the grid and filters. It is an index: delete it and
  it is rebuilt from the files.
- **Paths relative to the library root**, so the library can be moved.
- **No file is ever modified.**

What the manifest proves is export → disk. Whether Google exported the whole
library cannot be checked by anyone outside Google.

---

## Interface

### One page, two ways to show it

On a NAS or in Docker there is no screen, so the library must be a web page
served by the engine itself. That page exists regardless; the question was only
whether a desktop shell is needed on top.

- **Desktop: Wails.** A real app — `.dmg`, Windows installer, AppImage and
  `.deb` — at 10–25 MB instead of Electron's ~150.
- **Server: the same binary with a flag**, serving the same page. Not a port.
- **The shell has no UI of its own.** If a webview ever fails us, the same page
  goes into Electron without touching the engine.

On Windows Wails uses WebView2, which is Chromium. The engines that differ,
WebKitGTK and WKWebView, are on Linux and macOS.

### The grid decides

The first UI to build is a virtualised grid of tens of thousands of thumbnails:
it is the one place the three engines can really diverge. Conservative CSS, no
frontier APIs. Test on Linux and macOS first.

### The port

The app binds to `127.0.0.1` with a token in the URL: a page that commands the
downloader must not be reachable from outside. On a NAS it is exposed on
purpose, and there it needs a real password.

---

## Lessons taken from other projects' code

- `jpratt9/gphotos-export`, `vikas5914/google-photos-backup` (MIT): sign-in is a
  separate step done by hand once; progress is an atomically written file;
  an expired session is detected and reported.
- `shoon/takeout-helper-gphotos` (Apache-2.0): one hashing implementation on
  purpose; relative paths in the manifest; "nothing is estimated"; detect gaps
  in a numbered part series before unpacking.
- `TheLastGimbus/GooglePhotosTakeoutHelper` (Apache-2.0): Google changes and
  one-person projects die. The fragile parts must be easy to fix fast.
