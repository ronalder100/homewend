// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"errors"
	"time"

	"github.com/ronalder100/homewend/internal/library"
)

// PlanImport is what importing dir would bring into the library, read and
// never written: dir, a card or the library are left as they are.
func PlanImport(dir string) (library.ImportPlan, error) {
	items, others, err := library.ReadFolder(dir)
	if err != nil {
		return library.ImportPlan{}, err
	}
	// No library chosen yet is an empty one: everything would come in.
	catalog, _, err := openLibrary()
	if err != nil && !errors.Is(err, ErrNoLibrary) {
		return library.ImportPlan{}, err
	}
	if catalog != nil {
		defer catalog.Close()
	}
	return library.PlanImport(dir, items, others, catalog, time.Local)
}
