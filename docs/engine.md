# The engine, and why it is built this way

Two halves. The download half is ours and exists nowhere else; the repair half
has been written before, and what we need of it is smaller than it looks.

Everything here was proven against a real 344 GB export and a real 4 MB one
before it was written. The measurements that shaped the code are in
[`architecture.md`](architecture.md), and three of them decide it:

1. **The host that serves the bytes wants cookies, not the `rapt` token**, and
   its URL is built rather than scraped:
   `<file>?j=<job>&i=<index>&user=<id>&authuser=0`.
2. **The five-downloads-per-archive counter lives on the redirect**, not on that
   host. Going through the redirect spends an attempt even when nothing
   arrives; building the URL spends none.
3. **A copied cookie header dies in under twenty minutes**; a profile re-read
   before every request does not. That is why the session is a directory we own
   and not a string we keep.

## Why Go

The plan was Go and Wails, and the detour through Rust did not survive
contact with the code. The case for Rust was ten thousand lines of
`shoon/takeout-helper-gphotos` for free — but 1,700 of those write EXIF, which
we have decided never to do; 1,100 are format tables; and its organiser has four
layouts where we need one. What we actually need is far smaller, and Go
cross-compiles to the three desktops from one machine, which for something
shipped as an app is worth more than borrowed code.

## Licence

**AGPL-3.0.** Free to use, to read, to change and to pass on. The one thing it
forbids is the one thing worth forbidding here: taking this code, running it as
a paid service in the cloud, and keeping the improvements. Anyone may do that —
but they publish their changes, which is what makes it a fair trade rather than
a free lunch.

The ordinary user is not affected. Running the app, or the server build on your
own NAS, is using it, not distributing it, and asks nothing of you.

**The mobile app is a separate, closed program**, and it is what pays for this
one. Keeping that true is an engineering constraint, not a wish: the phone app
talks to the desktop over the network and **shares no code with this
repository**. The moment a file crosses over, the AGPL crosses with it.

*Not legal advice — the file that governs is `LICENSE`.*

## Docs

- [`architecture.md`](architecture.md) — how the download works, and why
- [`principles.md`](principles.md) — the rules the code follows

## Layout

    cmd/homewend       the CLI: flags in, engine call, text or JSON out
    internal/engine    what a user asks for, start to finish; every shell calls this
    internal/session   the browser profile we own: sign-in window, cookies, requests
    internal/takeout   Google's pages: the export list, and the form that asks for one
    internal/download  one part at a time, Range, resume
    internal/library   unpack, place by date and album, catalogue, count against the manifest
    internal/progress  the one event type every stage reports with
