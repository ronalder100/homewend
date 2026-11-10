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
	"time"
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

// classify puts each photo and video of f in its group. dates are the
// camera's, by path: a folder whose photos mostly have none is unsure.
func classify(f Folder, dates map[string]time.Time) []Group {
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
	groups := make([]Group, 0, len(order))
	for _, folder := range order {
		g := byFolder[folder]
		// Under DCIM a camera wrote it, as the DCF standard has every
		// camera do: a date it lacks is a camera's, not a doubt.
		underDCIM := folder == "DCIM" || strings.HasPrefix(folder, "DCIM/") || strings.Contains(folder, "/DCIM/") || strings.HasSuffix(folder, "/DCIM")
		if g.Kind == Photos && !underDCIM {
			// Photos with no camera's date, more than half of them: a
			// camera writes one in every photo. Videos say nothing here,
			// many cameras' do not carry one.
			var photos, undated int
			for _, p := range g.Paths {
				if !IsVideo(p) {
					photos++
					if _, ok := dates[p]; !ok {
						undated++
					}
				}
			}
			if photos > 0 && undated*2 > photos {
				g.Kind = Unsure
			}
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
	return groups
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
		case strings.HasSuffix(name, ".photoslibrary"):
			// Apple Photos keeps the photos in originals (Masters before
			// 2015); the rest are its previews.
			if i+1 < len(dirs) && (dirs[i+1] == "originals" || dirs[i+1] == "Masters") {
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

// cameraDates are the dates the cameras wrote in f's photos and videos,
// by path; one with none is not in it.
func cameraDates(f Folder) map[string]time.Time {
	dates := map[string]time.Time{}
	for _, item := range f.Items {
		if when, _, ok := cameraDate(item.Path); ok {
			dates[item.Path] = when
		}
	}
	return dates
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
