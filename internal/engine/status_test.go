// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import "testing"

// The library folder chosen once is there the next time, and none is "".
func TestDefaultLibraryIsKept(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	if dir, err := DefaultLibrary(); err != nil || dir != "" {
		t.Fatalf("before: %q %v", dir, err)
	}
	if err := SetDefaultLibrary("/mnt/nas/photos"); err != nil {
		t.Fatal(err)
	}
	if dir, err := DefaultLibrary(); err != nil || dir != "/mnt/nas/photos" {
		t.Errorf("after: %q %v", dir, err)
	}
}
