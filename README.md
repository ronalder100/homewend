<p align="center">
  <a href="https://homewend.app">
    <img src="docs/assets/logo.svg" alt="Homewend" width="96" />
  </a>
</p>

<h1 align="center">Homewend</h1>

<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-AGPL--3.0-blue.svg" alt="License: AGPL-3.0" /></a>
  <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux-lightgrey.svg" alt="Platform: macOS | Linux" />
</p>

<div align="center">
  <strong>
    <a href="https://homewend.app">Homewend</a> downloads your whole Google Photos library to your own disk,
    and checks that every file arrived.
  </strong>
  <br /><br />
  Command line today. Desktop app on its way.
</div>

<p align="center">
  <br />
  <a href="https://homewend.app/guides/"><strong>Guides »</strong></a>
  ·
  <a href="https://homewend.app/faq/">FAQ</a>
  ·
  <a href="https://homewend.app/#join">Desktop app waitlist</a>
</p>

<p align="center">
  <img src="docs/demo.gif" alt="homewend login, then homewend get, in a terminal" width="800" />
</p>

## Features

- **One command, start to finish.** Asks Google Takeout for the export, waits, downloads, unpacks, checks.
- **Resumes.** Stop it, lose the network, close the laptop: run it again and it carries on from the byte where it stopped.
- **Plain folders.** By date, albums beside them, readable without Homewend.
- **Verified.** Every file counted against the list Google puts in the export; anything missing is named.
- **Untouched.** Photos are never modified; Google's `.json` is kept beside them, never written into them.
- **Private.** Google → your disk, no server between.

## Quick start

```bash
curl -fsSL https://homewend.app/install.sh | sh

homewend login                                              # once
homewend get --year 2025 --library ~/Pictures/Homewend      # start with one year
homewend get --library ~/Pictures/Homewend                  # then everything
```

- The first time, a browser window opens on your export: **click Download on the first part**. Once per account.
- Google needs hours to prepare a whole library: leave the terminal open.
- macOS or Linux, with Chrome, Chromium, Brave or Edge (Homewend drives it). Windows not yet.
- Room for your library plus one archive; checked before starting.
- The installer checks the release's checksums and puts `homewend` in `~/.local/bin` (or `$HOMEWEND_INSTALL_DIR`). No root.
- From source: `go build ./cmd/homewend` (Go 1.25+).

## Homewend vs. Takeout by hand

| | Takeout by hand | Homewend |
|---|---|---|
| **A large library** | a zip to click for every few GB: a 344 GB export came in 159 | one command |
| **Five downloads per archive, in seven days** | each retry spends one | none, apart from one click on the first part, once per account |
| **A dropped connection** | retry, and spend another | picks up at the byte where it stopped |
| **What you get** | zips, photos and `.json` files mixed | photos by date and album; `.json` kept aside |
| **Did everything arrive?** | compare `archive_browser.html` (100,000+ lines) by hand | counted against Google's manifest, missing files named |

## Commands

| Command | What it does |
|---|---|
| `login` | Sign in to Google, in a browser profile that belongs to Homewend |
| `get` | Ask Google for an export, download it, check it |
| `fetch` | Download an export you made yourself on Takeout |
| `verify` | Count a library again against its export |
| `version` | Print the version |
| `help <command>` | Flags and examples |

`--json` on any command prints one JSON object per line, for scripts.
Exit codes: `0` done · `1` error · `2` files missing, named · `3` not signed in or session expired: run `homewend login`.

## On your disk

```
Homewend/
├── 2019/07/IMG_1234.jpg
├── 2019/unknown-month/
├── albums/Greece 2019/IMG_1234.jpg    hard link: no extra space
├── undated/
└── .homewend/                         Google's .json, catalogue, progress
```

## How Homewend works with Google

- It uses **Google Takeout**, Google's own export of your data.
- You sign in **in your own browser**. Homewend never sees your password.
- It proves everything Google put in the export reached your disk. It cannot prove Google exported your whole library; nobody outside Google can.

In detail: [`architecture.md`](docs/architecture.md) · [`principles.md`](docs/principles.md) · [`engine.md`](docs/engine.md)

## License

[AGPL-3.0](LICENSE): free to use on your computer or NAS; run a changed version as a service and you publish your changes.
Not affiliated with Google. Google Photos is a trademark of Google LLC.
