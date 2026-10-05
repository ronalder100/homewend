<!-- homewend — Copyright (C) 2026 Ron Alder -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

# The desktop app

The same Homewend as the command line, in a window: sign in, choose what to
bring home, and watch your library fill.

## Install it (Linux)

1. Download `Homewend-<version>.AppImage` from the
   [latest release](https://github.com/ronalder100/homewend/releases/latest).
2. Make it executable and open it:

   ```sh
   chmod +x Homewend-*.AppImage
   ./Homewend-*.AppImage
   ```

The first time it opens, Homewend adds itself to your desktop's application
menu. If you move the file, open it once from its new place and the menu
follows.

Sign-in needs Chrome, Chromium, Brave or Edge installed: Google's sign-in
opens in a window of its own, never inside the app.

## The first time

1. **Sign in to Google.** A browser window opens on Google's sign-in. Homewend
   never sees your password.
2. **Say whose photos they are.** The name is the folder of your library the
   photos go in, proposed from your address. It is asked once per account.
3. **Choose what to bring home:** one year, everything, or a takeout you
   already made on Google.
4. **Leave it.** Google prepares the takeout, often for hours; Homewend
   downloads it by itself when it is ready, part by part, and picks up where it
   was if the computer sleeps.

## Where your photos go

Into the library folder you chose, one folder per person, then by year and
month:

```text
<library>/<name>/2025/07/IMG_4471.HEIC
<library>/<name>/albums/Greece 2019/
```

Right-click a year, an album or a photo to open its folder or copy where it
is. Drag a photo out of the window to drop the original file anywhere.

## Two Google accounts

Use **Add a Google account**, beside the account on the screen where you
choose what to bring home (**New takeout**). Each account signs in on its own and
is asked whose photos it holds: give both the same name to keep them in one
folder, or different names to keep them apart. The library shows every
account; the boxes beside the accounts in the sidebar choose which ones you
see.

## Takeouts

Takeouts lists what Google holds for the account, and what of each is already
on your disk:

- **All here**: every file the takeout lists was found in your library.
- **Paused**: part of it is here; Resume carries on where it stopped.
- **Missing**: some files are not here; Retry downloads it again, and what you
  already have stays as it is.
- **Ready**: on Google, not downloaded yet.

## Updates

When a new version is out, Homewend says so in a strip under its title bar and
with a notification. Nothing is downloaded until you press **Update**; then it
installs the new version and opens again.
