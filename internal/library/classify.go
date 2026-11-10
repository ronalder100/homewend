// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ronalder100/homewend/internal/progress"
)

// GroupKind is what a group of a folder's photos is, and so whether it comes
// in unless the person says otherwise.
type GroupKind string

const (
	// Camera files: they come in.
	Photos GroupKind = "photos"
	// Images with no camera's date: sent, received, saved, screenshots,
	// maybe copies. They stay out until ticked.
	Unsure GroupKind = "unsure"
	// A project's folder — code, a website — whose images are mostly its
	// own icons and pictures. It stays out; photos in it can be picked.
	Project GroupKind = "project"
	// An app's own previews of photos kept elsewhere. They stay out.
	Previews GroupKind = "previews"
)

// projectMarks are the files and folders that make a folder a project:
// what a version control or a build tool keeps at a project's top.
var projectMarks = []string{
	".git", ".hg", ".svn", "node_modules", "package.json", "go.mod", "Cargo.toml",
	"pyproject.toml", "setup.py", "requirements.txt", "pom.xml", "build.gradle",
	"Gemfile", "composer.json", "CMakeLists.txt",
}

// projectSuffixes are folders an IDE makes a project's top with.
var projectSuffixes = []string{".xcodeproj", ".xcworkspace", ".sln"}

// Group is photos of a folder that go together: its own photos, a project
// and everything in it, an app's previews.
type Group struct {
	Folder string    `json:"folder"` // below the folder imported, with /; "." its top
	Kind   GroupKind `json:"kind"`
	// What made it what it is: "package.json", "Lightroom", "Apple Photos".
	Mark  string   `json:"mark,omitempty"`
	Paths []string `json:"-"`
}

// Choice is which groups come in: those the person left out, and photos
// picked one by one from a group left out. Nil is each group as its kind
// says.
type Choice struct {
	Out  []string `json:"out"`  // group folders left out
	In   []string `json:"in"`   // group folders brought in though their kind keeps them out
	Pick []string `json:"pick"` // paths below the folder, from groups left out
}

// In is whether a group comes in by its kind alone.
func (k GroupKind) In() bool { return k == Photos }

// UnknownGroupError is a folder chosen in or out that is no group of the
// folder: a name mistyped, or a folder named from elsewhere.
type UnknownGroupError struct{ Folder string }

func (e UnknownGroupError) Error() string { return e.Folder + " is not a folder of photos here" }

// rawExtensions are a camera's raw files: a camera's, whatever date can be
// read of them (a Canon CR3 keeps it where no reader here looks).
var rawExtensions = map[string]bool{
	".dng": true, ".cr2": true, ".cr3": true, ".nef": true, ".arw": true, ".raf": true,
	".orf": true, ".rw2": true, ".pef": true, ".srw": true,
}

// classify puts each photo and video of f in its group. dated says whether
// a file's camera wrote a date, asked only where it decides: a folder whose
// photos mostly have none is unsure. A folder the choice names that is no
// group is said.
//
// emit hears each date asked, N of Of: a folder of ten thousand photos is a
// minute of reading before anything is copied.
func classify(f Folder, dated func(path string) bool, choice *Choice, emit progress.Func) ([]Group, error) {
	marks := map[string]string{} // folder → its project mark, "" none
	markOf := func(dir string) string {
		if m, ok := marks[dir]; ok {
			return m
		}
		m := ""
		if entries, err := os.ReadDir(dir); err == nil {
		look:
			for _, e := range entries {
				for _, p := range projectMarks {
					if e.Name() == p {
						m = p
						break look
					}
				}
				for _, s := range projectSuffixes {
					if strings.HasSuffix(e.Name(), s) {
						m = e.Name()
						break look
					}
				}
			}
		}
		marks[dir] = m
		return m
	}
	byFolder := map[string]*Group{}
	var order []string
	for _, item := range f.Items {
		folder, kind, mark := groupOf(f.Root, item.Path, markOf)
		g := byFolder[folder]
		if g == nil {
			g = &Group{Folder: folder, Kind: kind, Mark: mark}
			byFolder[folder] = g
			order = append(order, folder)
		}
		g.Paths = append(g.Paths, item.Path)
	}
	// The photos whose date decides their group: in a folder of photos
	// outside DCIM — under it a camera wrote them, as the DCF standard has
	// every camera do, and a date one lacks is not a doubt. Videos and raw
	// files say nothing: many cameras' videos carry no date read here, and
	// a raw file is a camera's.
	asked := map[string][]string{}
	var of int
	for _, folder := range order {
		underDCIM := folder == "DCIM" || strings.HasPrefix(folder, "DCIM/") || strings.Contains(folder, "/DCIM/") || strings.HasSuffix(folder, "/DCIM")
		if byFolder[folder].Kind != Photos || underDCIM {
			continue
		}
		for _, p := range byFolder[folder].Paths {
			if !IsVideo(p) && !rawExtensions[strings.ToLower(filepath.Ext(p))] {
				asked[folder] = append(asked[folder], p)
				of++
			}
		}
	}
	groups := make([]Group, 0, len(order))
	var n int
	for _, folder := range order {
		g := byFolder[folder]
		// More than half of them with no camera's date: a camera writes
		// one in every photo.
		var undated int
		for _, p := range asked[folder] {
			n++
			emit.Emit(progress.Event{Stage: progress.Read, N: n, Of: of})
			if !dated(p) {
				undated++
			}
		}
		if len(asked[folder]) > 0 && undated*2 > len(asked[folder]) {
			g.Kind = Unsure
		}
		groups = append(groups, *g)
	}
	// What comes in first, then what is to decide, then what stays out.
	rank := map[GroupKind]int{Photos: 0, Unsure: 1, Project: 2, Previews: 3}
	slices.SortStableFunc(groups, func(a, b Group) int {
		if d := rank[a.Kind] - rank[b.Kind]; d != 0 {
			return d
		}
		return strings.Compare(a.Folder, b.Folder)
	})
	if choice != nil {
		for _, name := range slices.Concat(choice.In, choice.Out) {
			if !slices.ContainsFunc(groups, func(g Group) bool { return g.Folder == name }) {
				return nil, UnknownGroupError{Folder: name}
			}
		}
	}
	return groups, nil
}

// groupOf is the group of path, below root: the shallowest project or
// app's library around it, or else the folder it is in.
func groupOf(root, path string, markOf func(string) string) (folder string, kind GroupKind, mark string) {
	rel := relativeTo(root, path)
	parts := strings.Split(rel, "/")
	dirs := parts[:len(parts)-1]
	// The folder imported may be a project itself: all of it is.
	if m := markOf(root); m != "" {
		return ".", Project, m
	}
	for i, name := range dirs {
		at := strings.Join(dirs[:i+1], "/")
		switch {
		case strings.HasSuffix(name, ".lrdata"):
			return at, Previews, "Lightroom"
		case strings.HasSuffix(name, ".photoslibrary"), strings.HasSuffix(name, ".photolibrary"), strings.HasSuffix(name, ".aplibrary"):
			// Apple Photos keeps the photos in originals (Masters before
			// 2015, and in iPhoto and Aperture); the rest are its previews.
			// iPhoto before 2009 wrote Originals.
			if i+1 < len(dirs) && (strings.EqualFold(dirs[i+1], "originals") || dirs[i+1] == "Masters") {
				return at + "/" + dirs[i+1], Photos, ""
			}
			return at, Previews, "Apple Photos"
		}
		if m := markOf(filepath.Join(root, filepath.FromSlash(at))); m != "" {
			return at, Project, m
		}
	}
	if len(dirs) == 0 {
		return ".", Photos, ""
	}
	return strings.Join(dirs, "/"), Photos, ""
}

// GroupIn is whether g comes in whole, by its kind and the choice.
func (c *Choice) GroupIn(g Group) bool {
	if c == nil {
		return g.Kind.In()
	}
	return (g.Kind.In() && !slices.Contains(c.Out, g.Folder)) || slices.Contains(c.In, g.Folder)
}

// leftOut are the groups that do not come in whole, by their kind or as
// chosen.
func leftOut(groups []Group, choice *Choice) []OutGroup {
	var out []OutGroup
	for _, g := range groups {
		if !choice.GroupIn(g) {
			// Its files counted, not kept: the look's state carries it.
			files := len(g.Paths)
			g.Paths = nil
			out = append(out, OutGroup{Group: g, Files: files})
		}
	}
	return out
}

// chosen is which of f's items come in, by path.
func chosen(f Folder, groups []Group, choice *Choice) map[string]bool {
	in := map[string]bool{}
	picked := map[string]bool{}
	if choice != nil {
		for _, p := range choice.Pick {
			picked[p] = true
		}
	}
	for _, g := range groups {
		whole := choice.GroupIn(g)
		for _, p := range g.Paths {
			if whole || picked[relativeTo(f.Root, p)] {
				in[p] = true
			}
		}
	}
	return in
}

// small is whether the image at path is a few hundred pixels at most, read
// from its header: the icons and logos of a project, not its photos.
func small(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	c, _, err := image.DecodeConfig(file)
	return err == nil && max(c.Width, c.Height) <= 400
}
