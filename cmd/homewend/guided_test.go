// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import "testing"

func TestParseYear(t *testing.T) {
	cases := []struct {
		answer string
		year   int
		ok     bool
	}{
		{"", 0, true},
		{"2025", 2025, true},
		{"2026", 2026, true},
		{"2027", 0, false},
		{"1989", 0, false},
		{"last year", 0, false},
		{"20 25", 0, false},
	}
	for _, c := range cases {
		if year, ok := parseYear(c.answer, 2026); year != c.year || ok != c.ok {
			t.Errorf("%q: got %d, %v; want %d, %v", c.answer, year, ok, c.year, c.ok)
		}
	}
}
