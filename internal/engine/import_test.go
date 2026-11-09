// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"errors"
	"testing"
)

// A profile is a folder of the library: one that would leave it is refused
// before anything is read or copied.
func TestAnImportRefusesAProfileOutsideTheLibrary(t *testing.T) {
	for _, profile := range []string{"../escaped", ".homewend", "a/b", ""} {
		if _, err := Import(context.Background(), t.TempDir(), profile, false, nil); !errors.Is(err, ErrBadProfile) {
			t.Errorf("profile %q: %v, want ErrBadProfile", profile, err)
		}
	}
}
