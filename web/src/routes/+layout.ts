// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// One page in the browser: nothing is rendered on a server, nothing ahead of
// time. The Go binary serves the shell and the page fetches what it shows.
export const ssr = false;
export const prerender = false;
