// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

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
	// Of them and of the twins, those a date given would move.
	Redated int `json:"redated"`
	// Those given a date but filed in another profile, whose date it is
	// to give: counted when the profile is known.
	NotRedated int `json:"not_redated"`
	// The same photo twice in the folder: it is copied once.
	Twins int `json:"twins"`
	// Files and folders that could not be read, passed over.
	Unreadable int `json:"unreadable"`

	// The folder's photos in groups, as they come in or stay out: its
	// own folders, a project, an app's previews.
	Groups []ImportGroup `json:"groups"`

	// Years, newest first; "" is the photos with no date.
	Years []ImportYear `json:"years"`
	// Every photo and video read, by the date its camera wrote: where a
	// person finds the ones a reset clock dated wrong.
	Files []ImportFile `json:"files"`
	// What is not a photo or a video, by the folder it sits in.
	Others []ImportOther `json:"others"`
}

// ImportYear is what a year would receive, and each of its months.
type ImportYear struct {
	Year   string `json:"year"`
	Photos int    `json:"photos"`
	Videos int    `json:"videos"`
	// January first; none for the photos with no date.
	Months []ImportMonth `json:"months,omitempty"`
}

// ImportMonth is what a month would receive.
type ImportMonth struct {
	Month  int `json:"month"` // 1 to 12
	Photos int `json:"photos"`
	Videos int `json:"videos"`
}

// ImportFile is a photo or a video of the folder, as its camera dated it.
type ImportFile struct {
	Path  string    `json:"path"`            // below the folder, with /
	Taken time.Time `json:"taken,omitzero"`  // the camera's clock; none, no date
	Video bool      `json:"video,omitempty"` // else a photo
	Group string    `json:"group"`           // its group's folder
	// Small is an image of a few hundred pixels at most: an icon, a logo.
	Small bool `json:"small,omitempty"`
}

// ImportGroup is a group of the folder's photos and what is in it, whether
// it comes in or not.
type ImportGroup struct {
	Group
	In        bool      `json:"in"` // whole, by its kind or as chosen
	Photos    int       `json:"photos"`
	Videos    int       `json:"videos"`
	InLibrary int       `json:"in_library"`
	First     time.Time `json:"first,omitzero"` // the camera's dates, first and last
	Last      time.Time `json:"last,omitzero"`
}

// PlanOptions are what a look is for.
type PlanOptions struct {
	// Profile, when not "", is whose import it would be: a photo another
	// profile holds is not its to date.
	Profile string
	// Redates are dates given to photos whose camera had them wrong.
	Redates []Redate
	// Hashes, when not nil, keeps each file's content as read, for the
	// next look at the same folder: one with dates given reads nothing
	// again.
	Hashes Hashes
	// Choice is which groups come in; nil, each as its kind says.
	Choice *Choice
}

// Hashes are the contents of a folder's files, by path, as a look read
// them. A file changed since, in size or time, is read again.
type Hashes map[string]readHash

type readHash struct {
	size int64
	mod  time.Time
	hash string
}

// of is the content of path, read only when not known.
func (h Hashes) of(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if k, ok := h[path]; ok && k.size == info.Size() && k.mod.Equal(info.ModTime()) {
		return k.hash, nil
	}
	hash, err := hashOf(path)
	if err == nil && h != nil {
		h[path] = readHash{size: info.Size(), mod: info.ModTime(), hash: hash}
	}
	return hash, err
}

// Hash is the content of path as a look read it, if it did and the file
// has not changed since: another card in the same place, with the same
// names, is other photos.
func (h Hashes) Hash(path string) (string, bool) {
	k, ok := h[path]
	if !ok {
		return "", false
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() != k.size || !info.ModTime().Equal(k.mod) {
		return "", false
	}
	return k.hash, true
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
func PlanImport(ctx context.Context, dir string, f Folder, catalog *Catalog, opt PlanOptions, emit progress.Func) (ImportPlan, error) {
	profile := opt.Profile
	dates, err := given(opt.Redates, f)
	if err != nil {
		return ImportPlan{}, err
	}
	plan := ImportPlan{Unreadable: f.Unreadable, Years: []ImportYear{}, Files: []ImportFile{}, Others: []ImportOther{}}
	cameras := cameraDates(f)
	groups := classify(f, cameras)
	in := chosen(f, groups, opt.Choice)
	byPath := map[string]*ImportGroup{}
	plan.Groups = make([]ImportGroup, len(groups))
	for i, g := range groups {
		plan.Groups[i] = ImportGroup{Group: g, In: opt.Choice.GroupIn(g)}
		for _, p := range g.Paths {
			byPath[p] = &plan.Groups[i]
		}
	}
	seen := map[string]bool{}
	years := map[string]*ImportYear{}
	for i, item := range f.Items {
		// A card is minutes to read: a stop is heard between two files.
		if err := ctx.Err(); err != nil {
			return plan, err
		}
		emit.Emit(progress.Event{Stage: progress.Read, N: i + 1, Of: len(f.Items)})
		hash, err := opt.Hashes.of(item.Path)
		if err != nil {
			plan.Unreadable++
			continue
		}
		camera, hasDate := cameras[item.Path]
		group := byPath[item.Path]
		file := ImportFile{Path: relativeTo(dir, item.Path), Video: IsVideo(item.Path), Group: group.Folder}
		if hasDate {
			file.Taken = camera
		}
		// The size of an image only where the person picks among them.
		if group.Kind == Project || group.Kind == Unsure {
			file.Small = small(item.Path)
		}
		plan.Files = append(plan.Files, file)
		if file.Video {
			group.Videos++
		} else {
			group.Photos++
		}
		if hasDate {
			if group.First.IsZero() || camera.Before(group.First) {
				group.First = camera
			}
			if camera.After(group.Last) {
				group.Last = camera
			}
		}
		var held string // where the library holds it, "" not
		if catalog != nil {
			path, known, err := catalog.PathOf(hash)
			if err != nil {
				return plan, err
			}
			if known {
				held = path
				group.InLibrary++
			}
		}
		if !in[item.Path] {
			continue
		}
		_, dated := dates[item.Path]
		// The library first: a photo it holds is in it however many times
		// the folder has it.
		if held != "" {
			plan.InLibrary++
			// As redateInLibrary decides: by the profile it is filed in.
			first, _, _ := strings.Cut(held, "/")
			switch {
			case !dated:
			case profile != "" && first != profile:
				plan.NotRedated++
			default:
				plan.Redated++
			}
			continue
		}
		if seen[hash] {
			plan.Twins++
			if dated {
				plan.Redated++
			}
			continue
		}
		seen[hash] = true
		if info, err := os.Stat(item.Path); err == nil {
			plan.Bytes += info.Size()
		}
		year, month := "", 0
		when, ok := dates[item.Path]
		if !ok {
			when, ok = camera, hasDate
		}
		if ok {
			year, month = strconv.Itoa(when.Year()), int(when.Month())
		}
		y := years[year]
		if y == nil {
			y = &ImportYear{Year: year}
			years[year] = y
		}
		var m *ImportMonth
		if month > 0 {
			if len(y.Months) == 0 {
				y.Months = make([]ImportMonth, 12)
				for i := range y.Months {
					y.Months[i].Month = i + 1
				}
			}
			m = &y.Months[month-1]
		}
		video := IsVideo(item.Path)
		count := func(photos, videos *int) {
			if video {
				*videos++
			} else {
				*photos++
			}
		}
		count(&plan.Photos, &plan.Videos)
		count(&y.Photos, &y.Videos)
		if m != nil {
			count(&m.Photos, &m.Videos)
		}
	}
	for i := range plan.Groups {
		plan.Groups[i].Paths = nil
	}
	for _, y := range years {
		// The months nothing would go in are not said.
		y.Months = slices.DeleteFunc(y.Months, func(m ImportMonth) bool { return m.Photos+m.Videos == 0 })
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
