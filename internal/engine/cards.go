// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Card is a disk with a camera's photos on it: a memory card, a camera
// plugged in as a disk, a phone that shows itself as one.
type Card struct {
	Path string `json:"path"`
	Name string `json:"name"` // the disk's name, as the system shows it
}

// Cards are the disks mounted now that hold a DCIM folder at their top,
// where every camera keeps its photos (the DCF standard). The folder, not
// the disk's "removable" flag, is what says a card: the reader of a laptop
// measured 09/11 calls its cards fixed disks.
func Cards() []Card {
	return cardsUnder(mountRoots())
}

// mountRoots are the folders a system mounts a disk it is given in.
func mountRoots() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{"/Volumes"}
	case "windows":
		// Every drive is its own root: D: to Z:, the system's own C: left out.
		var drives []string
		for letter := 'D'; letter <= 'Z'; letter++ {
			drives = append(drives, string(letter)+`:\`)
		}
		return drives
	default:
		// udisks mounts under /run/media/<user> (measured 09/11), older
		// systems under /media/<user>, some under /media itself.
		roots := []string{"/media"}
		if u, err := user.Current(); err == nil {
			roots = append(roots, filepath.Join("/run/media", u.Username), filepath.Join("/media", u.Username))
		}
		return roots
	}
}

// cardsUnder looks one level down each root, and at the root itself, for a
// DCIM folder.
func cardsUnder(roots []string) []Card {
	seen := map[string]bool{}
	var cards []Card
	add := func(disk string) {
		if seen[disk] || !hasDCIM(disk) {
			return
		}
		seen[disk] = true
		name := filepath.Base(disk)
		if runtime.GOOS == "windows" {
			name = strings.TrimSuffix(disk, `\`)
		}
		cards = append(cards, Card{Path: disk, Name: name})
	}
	for _, root := range roots {
		add(root)
		entries, err := os.ReadDir(root)
		if err != nil {
			continue // a root this system does not have
		}
		for _, e := range entries {
			if e.IsDir() {
				add(filepath.Join(root, e.Name()))
			}
		}
	}
	sort.Slice(cards, func(i, j int) bool { return cards[i].Path < cards[j].Path })
	return cards
}

// hasDCIM is whether disk has a DCIM folder at its top, in any case: FAT
// keeps the name as the camera wrote it.
func hasDCIM(disk string) bool {
	entries, err := os.ReadDir(disk)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() && strings.EqualFold(e.Name(), "DCIM") {
			return true
		}
	}
	return false
}
