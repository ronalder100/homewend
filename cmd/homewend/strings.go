// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

// Every sentence the CLI shows a person. A second language is a second table.
var text = map[string]string{
	"usage": `homewend — your Google Photos export, onto your own disk

usage:
  homewend login  [--profile DIR] [--json]
  homewend get    [--profile DIR] [--year YYYY] --library DIR [--json]
  homewend fetch  [--profile DIR] --library DIR [--json]
  homewend verify --library DIR --job ID [--json]

exit codes: 0 done (for fetch: everything declared is on disk), 2 something is
missing, 3 not signed in to Google (run homewend login), 1 any other error`,

	"sign in":           "a browser window is open: sign in to Google there",
	"signed in":         "signed in",
	"already signed in": "already signed in",
	"not signed in":     "the browser was closed before sign-in finished: run homewend login again",
	"no browser":        "no Chrome, Chromium, Brave or Edge found on this machine",
	"export":            "export %s of %s: %d parts, %s",
	"request":           "asking Google Takeout for an export: Google takes a minute or two to list what can be exported",
	"not offered":       "Google Takeout offers no export of %d. It offers these years: %s\nand these albums:\n  %s",
	"waiting":           "Google is preparing export %s: minutes for a year, hours for everything, and Google also sends an email when it is ready. The download starts by itself. Keep this running, or stop it and run the same command again: it carries on where it was",
	"stopped":           "stopped: run the same command again to carry on where it was",
	"not requested":     "the export was not requested: %v",
	"first download":    "a browser window is open on your export: click Download on the first part, and enter your password if Google asks. This is needed once for this account; the window closes by itself once the download starts",
	"no download":       "the browser was closed before a download started: run the same command again",
	"no export":         "no export is ready on Google Takeout yet",
	"no space":          "not enough space for this export: %s needed, %s free",

	"missing flags":   "missing: %s",
	"download":        "[%d/%d] %s  from %s of %s",
	"downloaded":      "[%d/%d] %s  downloaded (%s)",
	"short":           "[%d/%d] %s  stopped at %s of %s; run again to continue",
	"retry":           "network: %s (attempt %d; waiting for it to come back)",
	"unpack":          "[%d/%d] %s  unpacking",
	"place":           "placing %d of %d",
	"placed":          "placed %d photos (%s), %d duplicates skipped, %d undated, %d album entries",
	"verified":        "declared %d, on disk %d, missing %d",
	"year":            "  %s  %d of %d",
	"missing file":    "  missing: %s",
	"session expired": "the Google session expired: run homewend login, then the same command",
	"error":           "error: %v",
}
