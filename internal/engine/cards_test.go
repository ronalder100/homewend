// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"os"
	"path/filepath"
	"testing"
)

// A card is a disk with a camera's DCIM folder: found one level under where
// the system mounts disks, whatever its name and whatever case the camera
// wrote; other disks, and a mount root that is not there, are passed over.
func TestACardIsADiskWithADCIMFolder(t *testing.T) {
	media := t.TempDir()
	for _, dir := range []string{"NO NAME/DCIM/100_FUJI", "EOS_DIGITAL/dcim", "BACKUP/Photos", "NO NAME/MISC"} {
		if err := os.MkdirAll(filepath.Join(media, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	cards := cardsUnder([]string{media, filepath.Join(media, "not-there")})
	if len(cards) != 2 || cards[0].Name != "EOS_DIGITAL" || cards[1].Name != "NO NAME" {
		t.Errorf("cards %+v, want EOS_DIGITAL and NO NAME", cards)
	}
}
