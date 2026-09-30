// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"os"
	"path/filepath"
	"testing"
)

// A copy cut short leaves a name that is not the part: it must be written
// again whole, not taken for a finished link.
func TestABareVideoReplacesWhatAnInterruptedCopyLeft(t *testing.T) {
	dir := t.TempDir()
	part := filepath.Join(dir, "IMG_0010-025.mov")
	if err := os.WriteFile(part, []byte("the whole video"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	target := filepath.Join(out, "Takeout", "Google Photos", "IMG_0010-025.mov")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("the who"), 0o644); err != nil {
		t.Fatal(err)
	}

	if n, err := UnpackPart(part, out); err != nil || n != 1 {
		t.Fatalf("got %d, %v", n, err)
	}
	if got, _ := os.ReadFile(target); string(got) != "the whole video" {
		t.Errorf("got %q", got)
	}
}
