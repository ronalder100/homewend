// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/ronalder100/homewend/internal/progress"
)

// ImportPlan is what importing a folder would do, worked out without
// writing anything: the screen "Check what comes in" and `import --dry-run`.
type ImportPlan struct {
	Photos int   `json:"photos"` // to copy
	Videos int   `json:"videos"` // to copy
	Bytes  int64 `json:"bytes"`  // they would add to the library

	// Already in the library, found by content: copying them again would
	// store nothing new.
	InLibrary int `json:"in_library"`
	// The same photo twice in the folder: it is copied once.
	Twins int `json:"twins"`
	// Files and folders that could not be read, passed over.
	Unreadable int `json:"unreadable"`

	// Years, newest first; "" is the photos with no date.
	Years []ImportYear `json:"years"`
	// What is not a photo or a video, by the folder it sits in.
	Others []ImportOther `json:"others"`
}

// ImportYear is what a year would receive.
type ImportYear struct {
	Year   string `json:"year"`
	Photos int    `json:"photos"`
	Videos int    `json:"videos"`
}

// ImportOther is a folder's files that stay out.
type ImportOther struct {
	Folder string `json:"folder"` // relative to the folder imported, "." for its top
	Files  int    `json:"files"`
	Bytes  int64  `json:"bytes"`
}

// PlanImport reads every photo and video of f through once, to know its
// content, and tells what would come in. The catalog, when there is one,
// says what the library already holds; nil is an empty library.
//
// A date is the camera's clock, the year it is filed under. Reading is the
// long part: emit hears each file, N of Of.
//
// Dates given are counted as the import would file the photos: a name the
// folder does not have is said here too, before the import.
func PlanImport(dir string, f Folder, catalog *Catalog, redates []Redate, emit progress.Func) (ImportPlan, error) {
	dates, err := given(redates, f)
	if err != nil {
		return ImportPlan{}, err
	}
	plan := ImportPlan{Unreadable: f.Unreadable, Years: []ImportYear{}, Others: []ImportOther{}}
	seen := map[string]bool{}
	years := map[string]*ImportYear{}
	for i, item := range f.Items {
		emit.Emit(progress.Event{Stage: progress.Read, N: i + 1, Of: len(f.Items)})
		hash, err := hashOf(item.Path)
		if err != nil {
			plan.Unreadable++
			continue
		}
		// The library first: a photo it holds is in it however many times
		// the folder has it.
		if catalog != nil {
			if _, known, err := catalog.PathOf(hash); err != nil {
				return plan, err
			} else if known {
				plan.InLibrary++
				continue
			}
		}
		if seen[hash] {
			plan.Twins++
			continue
		}
		seen[hash] = true
		if info, err := os.Stat(item.Path); err == nil {
			plan.Bytes += info.Size()
		}
		year := ""
		if when, ok := dates[item.Path]; ok {
			year = strconv.Itoa(when.Year())
		} else if when, _, ok := cameraDate(item.Path); ok {
			year = strconv.Itoa(when.Year())
		}
		y := years[year]
		if y == nil {
			y = &ImportYear{Year: year}
			years[year] = y
		}
		if IsVideo(item.Path) {
			plan.Videos++
			y.Videos++
		} else {
			plan.Photos++
			y.Photos++
		}
	}
	for _, y := range years {
		plan.Years = append(plan.Years, *y)
	}
	// Newest first, and the undated last.
	sort.Slice(plan.Years, func(i, j int) bool {
		a, b := plan.Years[i].Year, plan.Years[j].Year
		if a == "" || b == "" {
			return b == ""
		}
		return a > b
	})

	byFolder := map[string]*ImportOther{}
	for _, path := range f.Others {
		folder, err := filepath.Rel(dir, filepath.Dir(path))
		if err != nil {
			return plan, err
		}
		folder = filepath.ToSlash(folder)
		o := byFolder[folder]
		if o == nil {
			o = &ImportOther{Folder: folder}
			byFolder[folder] = o
		}
		o.Files++
		// Lstat: a link is counted as itself, not as what it points at.
		if info, err := os.Lstat(path); err == nil {
			o.Bytes += info.Size()
		}
	}
	for _, o := range byFolder {
		plan.Others = append(plan.Others, *o)
	}
	sort.Slice(plan.Others, func(i, j int) bool { return plan.Others[i].Folder < plan.Others[j].Folder })
	return plan, nil
}
