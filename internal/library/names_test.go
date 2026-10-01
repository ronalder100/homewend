// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import "testing"

func TestWindowsFileName(t *testing.T) {
	cases := map[string]string{
		"Summer 2019":             "Summer 2019",
		"Trip: Rome?":             "Trip_ Rome_",
		`a<b>c"d|e*f`:             "a_b_c_d_e_f",
		"ends with a dot.":        "ends with a dot",
		"ends with spaces   ":     "ends with spaces",
		"CON":                     "_CON",
		"nul.txt":                 "_nul.txt",
		"COM1.tar.gz":             "_COM1.tar.gz",
		"COM¹":                    "_COM¹",
		"COMMON":                  "COMMON",
		"LPT10":                   "LPT10",
		"...":                     "_",
		"tab\there":               "tab_here",
		"IMG_20190101_120000.jpg": "IMG_20190101_120000.jpg",
	}
	for name, want := range cases {
		if got := windowsFileName(name); got != want {
			t.Errorf("%q: got %q, want %q", name, got, want)
		}
	}
}
