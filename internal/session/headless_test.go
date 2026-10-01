// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package session

import (
	"testing"

	"github.com/chromedp/cdproto/network"
)

func TestAllOnDisk(t *testing.T) {
	osid := &network.Cookie{Domain: "takeout.google.com", Name: "OSID", Value: "a"}
	session := &network.Cookie{Domain: ".google.com", Name: "S", Value: "x", Session: true}
	other := &network.Cookie{Domain: "example.com", Name: "K", Value: "y"}
	for _, c := range []struct {
		name   string
		held   []*network.Cookie
		onDisk map[string]bool
		want   bool
	}{
		{"written", []*network.Cookie{osid}, map[string]bool{"OSID=a": true}, true},
		{"not yet written", []*network.Cookie{osid}, map[string]bool{}, false},
		{"rotated, old value on disk", []*network.Cookie{osid}, map[string]bool{"OSID=old": true}, false},
		{"session cookies are never written", []*network.Cookie{session}, map[string]bool{}, true},
		{"not Google's", []*network.Cookie{other}, map[string]bool{}, true},
	} {
		if got := allOnDisk(c.held, c.onDisk); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
