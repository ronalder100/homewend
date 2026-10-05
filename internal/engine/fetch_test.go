// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ronalder100/homewend/internal/library"
	"github.com/ronalder100/homewend/internal/progress"
	"github.com/ronalder100/homewend/internal/takeout"
)

// archive is a zip with the files given, each with its content.
func archive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		io.WriteString(f, content)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

// host serves an export's files by name, with Range, as the download host
// does, and writes down the order they were asked for in.
type host struct {
	files map[string][]byte
	asked []string
}

func (h *host) Get(address string, extra map[string]string) (*http.Response, error) {
	u, _ := url.Parse(address)
	name := path.Base(u.Path)
	data := h.files[name]
	if extra["Range"] == "bytes=0-0" {
		return &http.Response{
			StatusCode: http.StatusPartialContent,
			Header:     http.Header{"Content-Range": {fmt.Sprintf("bytes 0-0/%d", len(data))}},
			Body:       io.NopCloser(bytes.NewReader(data[:1])),
		}, nil
	}
	h.asked = append(h.asked, name)
	from, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(extra["Range"], "bytes="), "-"))
	return &http.Response{StatusCode: http.StatusPartialContent, Body: io.NopCloser(bytes.NewReader(data[from:]))}, nil
}

// The manifest is fetched before the parts, so that whoever watches is told,
// year by year, how much there is and how much of it is here: nothing at
// first, then each year filling up as its photos arrive.
func TestTheManifestComesFirstAndTheYearsAreFollowed(t *testing.T) {
	manifest := archive(t, map[string]string{"Takeout/archive_browser.html": `
		<div class="extracted-folder-name">Photos from 2024</div>
		<div class="extracted-file-name">a.jpg</div>
		<div class="extracted-folder-name">Photos from 2025</div>
		<div class="extracted-file-name">b.jpg</div>
		<div class="extracted-file-name">c.jpg</div>`})
	part := archive(t, map[string]string{
		"Takeout/Google Photos/Photos from 2024/a.jpg": "photo a",
		"Takeout/Google Photos/Photos from 2025/b.jpg": "photo b",
		"Takeout/Google Photos/Photos from 2025/c.jpg": "photo c",
	})
	served := &host{files: map[string][]byte{"part-001.zip": part, "manifest.zip": manifest}}
	f := Fetch{
		Target: takeout.Target{Job: "job", User: "1"},
		Export: takeout.Export{
			Job:      "job",
			Parts:    []takeout.Part{{Index: 0, Filename: "part-001.zip", Size: int64(len(part))}},
			Manifest: takeout.Part{Index: 1, Filename: "manifest.zip", Size: int64(len(manifest))},
		},
		Library: t.TempDir(),
	}
	var years []string
	downloads := map[string]int{}
	result, err := f.Run(context.Background(), served, func(e progress.Event) {
		switch e.Stage {
		case progress.Year:
			years = append(years, fmt.Sprintf("%s %d/%d", e.Name, e.N, e.Of))
		case progress.Download:
			downloads[e.Name] = e.N
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(served.asked, " ") != "manifest.zip part-001.zip" {
		t.Errorf("asked for %v, want the manifest first", served.asked)
	}
	if downloads["manifest.zip"] != 1 || downloads["part-001.zip"] != 2 {
		t.Errorf("numbered %v, want the manifest 1 and the part 2", downloads)
	}
	got := strings.Join(years, ", ")
	for _, want := range []string{"2024 0/1, 2025 0/2", "2024 1/1", "2025 1/2", "2025 2/2"} {
		if !strings.Contains(got, want) {
			t.Errorf("the years went %q, without %q", got, want)
		}
	}
	if !result.Verification.Complete() || result.Verification.Declared != 3 {
		t.Errorf("verification %+v", result.Verification)
	}

	// A second run finds it all there: no download, and every year full from
	// its first word.
	served.asked, years = nil, nil
	if _, err := f.Run(context.Background(), served, func(e progress.Event) {
		if e.Stage == progress.Year {
			years = append(years, fmt.Sprintf("%s %d/%d", e.Name, e.N, e.Of))
		}
	}); err != nil {
		t.Fatal(err)
	}
	if len(served.asked) != 0 || strings.Join(years, ", ") != "2024 1/1, 2025 2/2" {
		t.Errorf("a second run asked for %v and said %q", served.asked, strings.Join(years, ", "))
	}

	// The list of takeouts says it is all here, as counted at the end.
	if l := localOf(f.Library, f.Export); l == nil || l.Parts != 1 || l.Of != 1 || l.Declared != 3 || l.Present != 3 {
		t.Errorf("local %+v, want 1 of 1 parts and 3 of 3 files", l)
	}

	// Again, the part is downloaded once more, though it was all here.
	served.asked = nil
	f.Again = true
	if _, err := f.Run(context.Background(), served, func(progress.Event) {}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(served.asked, " ") != "part-001.zip" {
		t.Errorf("again asked for %v, want the part", served.asked)
	}
}

// The library fills while the download goes on: a photo is placed as soon as
// the part with its sidecar is unpacked, before the next part is asked for.
// One whose sidecar travels in a later part waits for it, and is filed by it.
func TestPhotosArePlacedAsTheirPartsArrive(t *testing.T) {
	const taken = `{"photoTakenTime":{"timestamp":"1721000000"}}` // July 2024
	manifest := archive(t, map[string]string{"Takeout/archive_browser.html": `
		<div class="extracted-folder-name">Photos from 2024</div>
		<div class="extracted-file-name">a.jpg</div>
		<div class="extracted-file-name">b.jpg</div>`})
	first := archive(t, map[string]string{
		"Takeout/Google Photos/Photos from 2024/a.jpg":                            "photo a",
		"Takeout/Google Photos/Photos from 2024/a.jpg.supplemental-metadata.json": taken,
		"Takeout/Google Photos/Photos from 2024/b.jpg":                            "photo b",
	})
	second := archive(t, map[string]string{
		"Takeout/Google Photos/Photos from 2024/b.jpg.supplemental-metadata.json": taken,
	})
	served := &host{files: map[string][]byte{"part-001.zip": first, "part-002.zip": second, "manifest.zip": manifest}}
	f := Fetch{
		Target: takeout.Target{Job: "job", User: "1"},
		Export: takeout.Export{
			Job: "job",
			Parts: []takeout.Part{
				{Index: 0, Filename: "part-001.zip", Size: int64(len(first))},
				{Index: 1, Filename: "part-002.zip", Size: int64(len(second))},
			},
			Manifest: takeout.Part{Index: 2, Filename: "manifest.zip", Size: int64(len(manifest))},
		},
		Library:  t.TempDir(),
		Location: time.UTC,
	}
	waiting := filepath.Join(f.Library, "unassigned", "Photos from 2024", "b.jpg")
	var story []string
	result, err := f.Run(context.Background(), served, func(e progress.Event) {
		switch e.Stage {
		case progress.Year:
			story = append(story, fmt.Sprintf("%s %d/%d", e.Name, e.N, e.Of))
		case progress.Unassigned:
			story = append(story, fmt.Sprintf("unassigned %d", e.N))
		case progress.Download:
			story = append(story, "download "+e.Name)
			// While the second part comes down, the photo that waits for it
			// is there to be seen.
			if _, err := os.Stat(waiting); e.Name == "part-002.zip" && err != nil {
				t.Errorf("the waiting photo is not shown: %v", err)
			}
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(f.Library, "unassigned")); !os.IsNotExist(err) {
		t.Error("the unassigned folder is still there at the end")
	}
	// The year is full when both photos have arrived, the one still waiting
	// for its date included: where it is filed is another matter.
	want := "download manifest.zip, 2024 0/2, unassigned 0, download part-001.zip, 2024 1/2, 2024 2/2, unassigned 1, download part-002.zip, unassigned 0, unassigned 0"
	if got := strings.Join(story, ", "); got != want {
		t.Errorf("it went\n  %s\nwant\n  %s", got, want)
	}
	// Both by the sidecar's date, the late one too: not by the year folder.
	for _, name := range []string{"a.jpg", "b.jpg"} {
		if _, err := os.Stat(filepath.Join(f.Library, "2024", "07", name)); err != nil {
			t.Errorf("%s is not filed under July 2024: %v", name, err)
		}
	}
	if result.Organized.Placed != 2 || !result.Verification.Complete() {
		t.Errorf("placed %d, verification %+v", result.Organized.Placed, result.Verification)
	}
}

// The other way round works the same: a sidecar that arrives before its
// photo waits beside nothing, and the photo is filed by it the moment it
// comes.
func TestASidecarThatArrivesFirstIsUsedWhenThePhotoComes(t *testing.T) {
	manifest := archive(t, map[string]string{"Takeout/archive_browser.html": `
		<div class="extracted-folder-name">Photos from 2024</div>
		<div class="extracted-file-name">b.jpg</div>`})
	first := archive(t, map[string]string{
		"Takeout/Google Photos/Photos from 2024/b.jpg.supplemental-metadata.json": `{"photoTakenTime":{"timestamp":"1721000000"}}`,
	})
	second := archive(t, map[string]string{"Takeout/Google Photos/Photos from 2024/b.jpg": "photo b"})
	served := &host{files: map[string][]byte{"part-001.zip": first, "part-002.zip": second, "manifest.zip": manifest}}
	f := Fetch{
		Target: takeout.Target{Job: "job", User: "1"},
		Export: takeout.Export{
			Job: "job",
			Parts: []takeout.Part{
				{Index: 0, Filename: "part-001.zip", Size: int64(len(first))},
				{Index: 1, Filename: "part-002.zip", Size: int64(len(second))},
			},
			Manifest: takeout.Part{Index: 2, Filename: "manifest.zip", Size: int64(len(manifest))},
		},
		Library:  t.TempDir(),
		Location: time.UTC,
	}
	result, err := f.Run(context.Background(), served, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(f.Library, "2024", "07", "b.jpg")); err != nil {
		t.Errorf("b.jpg is not filed under July 2024: %v", err)
	}
	if result.Organized.Placed != 1 || !result.Verification.Complete() {
		t.Errorf("placed %d, verification %+v", result.Organized.Placed, result.Verification)
	}
}

// Reading what a takeout holds leaves its manifest in the library and no
// profile. A download that follows names its profile first, so that, stopped
// before any part is unpacked, it is still begun.
func TestAFetchAfterReadingTheContentsIsBegun(t *testing.T) {
	manifest := archive(t, map[string]string{"Takeout/archive_browser.html": `
		<div class="extracted-folder-name">Photos from 2025</div>
		<div class="extracted-file-name">a.jpg</div>`})
	lib := t.TempDir()
	work := filepath.Join(lib, library.WorkDir, "job")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "manifest.zip"), manifest, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := (state{Manifest: "manifest.zip"}).save(work); err != nil {
		t.Fatal(err)
	}
	f := Fetch{
		Target: takeout.Target{Job: "job", User: "1"},
		Export: takeout.Export{
			Job:      "job",
			Parts:    []takeout.Part{{Index: 0, Filename: "part-001.zip", Size: 9}},
			Manifest: takeout.Part{Index: 1, Filename: "manifest.zip", Size: int64(len(manifest))},
		},
		Library: lib,
		Profile: "ann",
	}
	// The part never unpacks: the run stops before any photo comes.
	damaged := &host{files: map[string][]byte{"part-001.zip": []byte("not a zip")}}
	if _, err := f.Run(context.Background(), damaged, func(progress.Event) {}); err == nil {
		t.Fatal("a damaged part went through")
	}
	if l := localOf(lib, f.Export); l == nil || l.Parts != 0 || l.Of != 1 {
		t.Errorf("local %+v, want begun, 0 of 1 parts", l)
	}
}

// A download with no room for it is refused before anything is kept: the
// takeout is not begun, no folder is made for it, and one asked again keeps
// the parts it had.
func TestADownloadWithNoRoomLeavesNoTrace(t *testing.T) {
	f := Fetch{
		Target:  takeout.Target{Job: "job", User: "1"},
		Export:  takeout.Export{Job: "job", Parts: []takeout.Part{{Index: 0, Filename: "part-001.zip", Size: 1 << 60}}},
		Library: t.TempDir(),
		Profile: "ann",
	}
	var space NoSpaceError
	if _, err := f.Run(context.Background(), &host{files: map[string][]byte{}}, func(progress.Event) {}); !errors.As(err, &space) {
		t.Fatalf("got %v, want NoSpaceError", err)
	}
	for _, dir := range []string{filepath.Join(library.WorkDir, "job"), "ann"} {
		if _, err := os.Stat(filepath.Join(f.Library, dir)); !os.IsNotExist(err) {
			t.Errorf("%s was made for it: %v", dir, err)
		}
	}

	// Again, with a part already here: refused, it still has the part.
	f.Export.Parts = append(f.Export.Parts, takeout.Part{Index: 1, Filename: "part-002.zip", Size: 1})
	work := filepath.Join(f.Library, library.WorkDir, "job")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := (state{Unpacked: map[string]bool{"part-002.zip": true}, Profile: "ann"}).save(work); err != nil {
		t.Fatal(err)
	}
	f.Again = true
	if _, err := f.Run(context.Background(), &host{files: map[string][]byte{}}, func(progress.Event) {}); !errors.As(err, &space) {
		t.Fatalf("again: got %v, want NoSpaceError", err)
	}
	if l := localOf(f.Library, f.Export); l == nil || l.Parts != 1 {
		t.Errorf("local %+v, want the part still unpacked", l)
	}
}

// The first download into a library and a profile not made yet makes them,
// rather than fail to measure the room or to count what is there.
func TestAFirstDownloadMakesTheLibrary(t *testing.T) {
	manifest := archive(t, map[string]string{"Takeout/archive_browser.html": `
		<div class="extracted-folder-name">Photos from 2025</div>
		<div class="extracted-file-name">a.jpg</div>`})
	part := archive(t, map[string]string{"Takeout/Google Photos/Photos from 2025/a.jpg": "photo a"})
	f := Fetch{
		Target: takeout.Target{Job: "job", User: "1"},
		Export: takeout.Export{
			Job:      "job",
			Parts:    []takeout.Part{{Index: 0, Filename: "part-001.zip", Size: int64(len(part))}},
			Manifest: takeout.Part{Index: 1, Filename: "manifest.zip", Size: int64(len(manifest))},
		},
		Library: filepath.Join(t.TempDir(), "Pictures", "Homewend"),
		Profile: "ann",
	}
	served := &host{files: map[string][]byte{"part-001.zip": part, "manifest.zip": manifest}}
	if _, err := f.Run(context.Background(), served, func(progress.Event) {}); err != nil {
		t.Fatal(err)
	}
}
