// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"context"
	"os"
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

	// What is in already, twice, moved, deleted, unreadable; how it ends.
	// NotRedated is counted when the profile is known.
	Counts

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
	// OthersMore are those left unnamed past MaxOthers; OtherFiles, the
	// files of all of them.
	OthersMore int `json:"others_more,omitempty"`
	OtherFiles int `json:"other_files"`
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
	// What the person said, as the copy will hear it.
	Asked
	// Profile, when not "", is whose import it would be: a photo another
	// profile holds is not its to date.
	Profile string
	// Hashes, when not nil, keeps each file's content as read, for the
	// next look at the same folder: one with dates given reads nothing
	// again.
	Hashes Hashes
	// Root is the library's folder, when there is a catalog: a photo it
	// lists and its disk no longer holds was deleted.
	Root string
}

// Asked is what the person says of an import, the same for its look and
// its copy, and as the window sends it.
type Asked struct {
	// Dates given to photos whose camera had them wrong.
	Redates []Redate `json:"redates,omitempty"`
	// Which groups come in; nil, each as its kind says.
	Choice *Choice `json:"choice,omitempty"`
	// The photos the library had and someone deleted from it to bring in
	// again, by name as --date names them, or "all". The others stay out,
	// counted: a card still holds what was deleted, and would bring it back
	// at every import.
	BringBack []string `json:"bring_back,omitempty"`
}

// bringsBack says whether a deleted photo of f, by its path, comes in
// again: one named, or all. A name the folder does not have is said, as a
// date given to it would be.
func (a Asked) bringsBack(f Folder) (func(path string) bool, error) {
	for _, name := range a.BringBack {
		if strings.EqualFold(name, "all") {
			return func(string) bool { return true }, nil
		}
	}
	back := map[string]bool{}
	for _, name := range a.BringBack {
		path, err := named(name, f)
		if err != nil {
			return nil, err
		}
		back[path] = true
	}
	return func(path string) bool { return back[path] }, nil
}

// Hashes are the contents of a folder's files, by path, as a look read
// them. A file changed since, in size or time, is read again.
type Hashes map[string]readHash

type readHash struct {
	size int64
	mod  time.Time
	fileFacts
	// small, once asked: an icon's size; smallKnown, whether asked.
	small, smallKnown bool
}

// fileFacts are what a look reads of a file: its content, and the date its
// camera wrote, when it wrote one.
type fileFacts struct {
	hash  string
	taken time.Time
	dated bool
}

// facts are path's, read only when not known.
func (h Hashes) facts(path string) (fileFacts, error) {
	info, err := os.Stat(path)
	if err != nil {
		return fileFacts{}, err
	}
	if k, ok := h[path]; ok && k.size == info.Size() && k.mod.Equal(info.ModTime()) {
		return k.fileFacts, nil
	}
	hash, err := hashOf(path)
	if err != nil {
		return fileFacts{}, err
	}
	facts := fileFacts{hash: hash}
	facts.taken, _, facts.dated = cameraDate(path)
	if h != nil {
		h[path] = readHash{size: info.Size(), mod: info.ModTime(), fileFacts: facts}
	}
	return facts, nil
}

// small is small(path), kept with its facts when they are.
func (h Hashes) small(path string) bool {
	k, ok := h[path]
	if ok && k.smallKnown {
		return k.small
	}
	s := small(path)
	if ok {
		k.small, k.smallKnown = s, true
		h[path] = k
	}
	return s
}

// dated is whether the camera wrote a date in path, as a look read it; one
// not read is read now.
func (h Hashes) dated(path string) bool {
	if facts, ok := h.known(path); ok {
		return facts.dated
	}
	_, _, ok := cameraDate(path)
	return ok
}

// known are path's facts as a look read them, while the file is unchanged.
func (h Hashes) known(path string) (fileFacts, bool) {
	k, ok := h[path]
	if !ok {
		return fileFacts{}, false
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() != k.size || !info.ModTime().Equal(k.mod) {
		return fileFacts{}, false
	}
	return k.fileFacts, true
}

// Hash is the content of path as a look read it, if it did and the file
// has not changed since: another card in the same place, with the same
// names, is other photos.
func (h Hashes) Hash(path string) (string, bool) {
	facts, ok := h.known(path)
	return facts.hash, ok
}

// MaxOthers is how many names of what stays out a look gives.
const MaxOthers = 10

// ImportOther is what stays out at the top of the folder: a file that is
// not a photo or a video, or a folder holding such files, named as a person
// sees them there ("FWUP0013.DAT", "FFDB/").
type ImportOther struct {
	Name  string `json:"name"`
	Files int    `json:"files"`
	Bytes int64  `json:"bytes"`
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
	dates, err := given(opt.Redates, f)
	if err != nil {
		return ImportPlan{}, err
	}
	back, err := opt.bringsBack(f)
	if err != nil {
		return ImportPlan{}, err
	}
	plan := ImportPlan{Counts: Counts{Unreadable: f.Unreadable}, Years: []ImportYear{}, Files: []ImportFile{}, Others: []ImportOther{}}
	// Every file read once, its content and its camera's date, before
	// the groups are known: what the groups are depends on the dates.
	hashes := opt.Hashes
	if hashes == nil {
		hashes = Hashes{}
	}
	read := map[string]fileFacts{}
	for i, item := range f.Items {
		// A card is minutes to read: a stop is heard between two files.
		if err := ctx.Err(); err != nil {
			return plan, err
		}
		emit.Emit(progress.Event{Stage: progress.Read, N: i + 1, Of: len(f.Items)})
		facts, err := hashes.facts(item.Path)
		if err != nil {
			plan.Unreadable++
			continue
		}
		read[item.Path] = facts
	}
	groups, err := classify(f, func(path string) bool { return read[path].dated }, opt.Choice, nil)
	if err != nil {
		return plan, err
	}
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
	for _, item := range f.Items {
		facts, ok := read[item.Path]
		if !ok {
			continue // unreadable, counted
		}
		hash, camera, hasDate := facts.hash, facts.taken, facts.dated
		group := byPath[item.Path]
		file := ImportFile{Path: relativeTo(dir, item.Path), Video: IsVideo(item.Path), Group: group.Folder}
		if hasDate {
			file.Taken = camera
		}
		// The size of an image only where the person picks among them.
		if group.Kind == Project || group.Kind == Unsure {
			file.Small = hashes.small(item.Path)
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
		var deleted bool
		if catalog != nil {
			path, known, gone, err := libraryHolds(catalog, opt.Root, hash)
			if err != nil {
				return plan, err
			}
			deleted = gone
			if known && !gone {
				held = path
				group.InLibrary++
			}
		}
		if !in[item.Path] {
			continue
		}
		if deleted {
			plan.deleted(f.Root, item.Path, back(item.Path))
			if !back(item.Path) {
				continue
			}
		}
		when, dated := dates[item.Path]
		// The library first: a photo it holds is in it however many times
		// the folder has it.
		if held != "" {
			plan.InLibrary++
			// As redateInLibrary decides: by the profile it is filed in.
			switch {
			case !dated:
			case !filedIn(held, opt.Profile):
				plan.NotRedated++
			default:
				plan.Redated++
			}
			continue
		}
		// Not in the library: given a date, it moves to it, a twin too, as
		// the copy counts.
		if dated {
			plan.Redated++
		}
		if seen[hash] {
			plan.Twins++
			continue
		}
		seen[hash] = true
		if info, err := os.Stat(item.Path); err == nil {
			plan.Bytes += info.Size()
		}
		year, month := "", 0
		// Filed by the date given, else the camera's.
		if !dated {
			when = camera
		}
		if dated || hasDate {
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
		count := func(photos, videos *int) {
			if file.Video {
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
	plan.LeftOut = leftOut(groups, opt.Choice)
	plan.end(plan.Photos + plan.Videos)
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

	byName := map[string]*ImportOther{}
	for _, path := range f.Others {
		first, _, inside := strings.Cut(relativeTo(dir, path), "/")
		if inside {
			first += "/"
		}
		o := byName[first]
		if o == nil {
			o = &ImportOther{Name: first}
			byName[first] = o
		}
		o.Files++
		// Lstat: a link is counted as itself, not as what it points at.
		if info, err := os.Lstat(path); err == nil {
			o.Bytes += info.Size()
		}
	}
	for _, o := range byName {
		plan.Others = append(plan.Others, *o)
	}
	sort.Slice(plan.Others, func(i, j int) bool { return plan.Others[i].Name < plan.Others[j].Name })
	plan.OtherFiles = len(f.Others)
	// A few names say what they are; three thousand say nothing more.
	if len(plan.Others) > MaxOthers {
		plan.OthersMore = len(plan.Others) - MaxOthers
		plan.Others = plan.Others[:MaxOthers]
	}
	return plan, nil
}
