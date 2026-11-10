// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"context"
	"path/filepath"
	"testing"
)

// An old laptop's folder: photos come in by themselves; a project, an
// app's previews and images with no camera's date stay out until chosen.
func TestAMessyFolderIsToldApart(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "Pictures/Greece 2019/DSC_0001.JPG", exifPhoto("2019:07:14 10:00:00", "greece"))
	write(t, dir, "Pictures/Greece 2019/DSC_0002.JPG", exifPhoto("2019:07:15 10:00:00", "greece 2"))
	write(t, dir, "code/family-site/package.json", []byte("{}"))
	write(t, dir, "code/family-site/public/img/logo.png", []byte("not a photo"))
	write(t, dir, "code/family-site/public/img/us.jpg", exifPhoto("2020:05:01 12:00:00", "us"))
	write(t, dir, "Downloads/WhatsApp Images/IMG-1.jpg", []byte("no camera date"))
	write(t, dir, "Lightroom/Catalog Smart Previews.lrdata/0/A.dng", []byte("a preview"))
	write(t, dir, "Card copy/DCIM/100_FUJI/DSCF0001.JPG", []byte("a camera's, no date read"))
	folder, err := ReadFolder(dir)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanImport(context.Background(), dir, folder, nil, PlanOptions{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]struct {
		kind GroupKind
		mark string
		in   bool
	}{
		"Pictures/Greece 2019":                    {Photos, "", true},
		"code/family-site":                        {Project, "package.json", false},
		"Downloads/WhatsApp Images":               {Unsure, "", false},
		"Lightroom/Catalog Smart Previews.lrdata": {Previews, "Lightroom", false},
		"Card copy/DCIM/100_FUJI":                 {Photos, "", true},
	}
	if len(plan.Groups) != len(want) {
		t.Errorf("groups %+v, want %d", plan.Groups, len(want))
	}
	for _, g := range plan.Groups {
		w, ok := want[g.Folder]
		if !ok || g.Kind != w.kind || g.Mark != w.mark || g.In != w.in {
			t.Errorf("group %s: %s %q in %v, want %+v", g.Folder, g.Kind, g.Mark, g.In, w)
		}
	}
	if plan.Photos != 3 {
		t.Errorf("%d photos would come in, want Greece's 2 and the card's 1", plan.Photos)
	}

	// The WhatsApp folder ticked, Greece left out, one photo picked from
	// the project: that is what comes in.
	choice := &Choice{In: []string{"Downloads/WhatsApp Images"}, Out: []string{"Pictures/Greece 2019"}, Pick: []string{"code/family-site/public/img/us.jpg"}}
	plan, err = PlanImport(context.Background(), dir, folder, nil, PlanOptions{Choice: choice}, nil)
	if err != nil || plan.Photos != 3 {
		t.Errorf("chosen: %d photos, %v; want WhatsApp's, the card's and the one picked", plan.Photos, err)
	}

	// The copy brings in what the choice says, and nothing else.
	root := t.TempDir()
	catalog, err := OpenCatalog(filepath.Join(root, "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	done, err := Import(context.Background(), folder, root, catalog, Options{Profile: "ron"}, ImportOptions{Choice: choice, Fits: func(int64) error { return nil }}, nil)
	if err != nil || done.Placed != 3 {
		t.Errorf("copied %d, %v; want 3", done.Placed, err)
	}
}

// A folder that is a project itself is one: all of it stays out.
func TestAProjectImportedWholeStaysOut(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "go.mod", []byte("module x"))
	write(t, dir, "assets/photo.jpg", exifPhoto("2020:05:01 12:00:00", "x"))
	folder, err := ReadFolder(dir)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanImport(context.Background(), dir, folder, nil, PlanOptions{}, nil)
	if err != nil || len(plan.Groups) != 1 || plan.Groups[0].Folder != "." || plan.Groups[0].Kind != Project || plan.Photos != 0 {
		t.Errorf("%+v, %v; want the whole folder one project, nothing in", plan.Groups, err)
	}
}
