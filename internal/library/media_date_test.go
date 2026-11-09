// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"encoding/binary"
	"path/filepath"
	"testing"
	"time"
)

// movie is the smallest MP4 the date reader takes: a moov holding an mvhd
// whose creation time is created, in seconds since 1904 as the format counts.
func movie(created time.Time) []byte {
	mvhd := make([]byte, 108)
	binary.BigEndian.PutUint32(mvhd[0:], 108)
	copy(mvhd[4:], "mvhd")
	// version 0, no flags, then creation and modification time.
	if !created.IsZero() {
		seconds := uint32(created.Sub(time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)) / time.Second)
		binary.BigEndian.PutUint32(mvhd[12:], seconds)
		binary.BigEndian.PutUint32(mvhd[16:], seconds)
	}
	binary.BigEndian.PutUint32(mvhd[20:], 1000)       // timescale
	binary.BigEndian.PutUint32(mvhd[28:], 0x00010000) // rate 1.0
	binary.BigEndian.PutUint16(mvhd[32:], 0x0100)     // volume 1.0
	moov := make([]byte, 8)
	binary.BigEndian.PutUint32(moov[0:], uint32(8+len(mvhd)))
	copy(moov[4:], "moov")
	return append(moov, mvhd...)
}

func TestAVideoIsDatedByItsHeader(t *testing.T) {
	dir := t.TempDir()
	// Measured on a Samsung phone: 11:57:38 UTC, the end of a recording
	// started at 12:56:35 in Rome.
	ended := time.Date(2026, 2, 18, 11, 57, 38, 0, time.UTC)
	write(t, dir, "20260218_125635.mp4", movie(ended))
	when, source, ok := cameraDate(filepath.Join(dir, "20260218_125635.mp4"))
	// A camera's clock: the instant on this computer's clock.
	local := ended.In(time.Local)
	want := time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), local.Minute(), local.Second(), 0, time.UTC)
	if !ok || source != FromVideo || !when.Equal(want) {
		t.Fatalf("got %v %q %v, want %v from the video's header", when, source, ok, want)
	}
}

func TestAVideoWithNoDateHasNone(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "clip.mov", movie(time.Time{}))
	if when, _, ok := cameraDate(filepath.Join(dir, "clip.mov")); ok {
		t.Errorf("dated %v, want none: zero is 1904, not a day anyone filmed", when)
	}
	write(t, dir, "broken.mp4", []byte("not a movie"))
	if _, _, ok := cameraDate(filepath.Join(dir, "broken.mp4")); ok {
		t.Error("a file that is not a movie got a date")
	}
}

// The camera's clock has no zone and is the clock of where the photo was
// taken: it is not moved, or New Year's Eve lands in the next year.
func TestNewYearsEveStaysInItsYear(t *testing.T) {
	rome, err := time.LoadLocation("Europe/Rome")
	if err != nil {
		t.Skip("no timezone database")
	}
	src := t.TempDir()
	write(t, src, "DCIM/IMG_1.JPG", exifPhoto("2018:12:31 23:30:00", "eve"))
	folder, err := ReadFolder(src)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanImport(src, folder, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Years) != 1 || plan.Years[0].Year != "2018" {
		t.Errorf("years %+v, want 2018", plan.Years)
	}

	out := t.TempDir()
	catalog, err := OpenCatalog(filepath.Join(t.TempDir(), "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	items := []Item{{Path: filepath.Join(src, "DCIM", "IMG_1.JPG"), Capture: DateOf(filepath.Join(src, "DCIM", "IMG_1.JPG"), "DCIM")}}
	if _, err := Organize(items, out, catalog, Options{Location: rome}, nil); err != nil {
		t.Fatal(err)
	}
	mustExist(t, filepath.Join(out, "2018", "12", "IMG_1.JPG"))
}

// A damaged HEIC has no date; it does not bring the import down. Measured:
// a meta box claiming more than the file holds made the decoder panic.
func TestADamagedHEICHasNoDate(t *testing.T) {
	dir := t.TempDir()
	damaged := []byte("\x00\x00\x00\x18ftypheic\x00\x00\x00\x00mif1heic" +
		"\x00\x00\x00\x01meta\x7f\xff\xff\xff\xff\xff\xff\xff")
	write(t, dir, "IMG_0001.HEIC", damaged)
	if when, _, ok := cameraDate(filepath.Join(dir, "IMG_0001.HEIC")); ok {
		t.Errorf("dated %v, want none", when)
	}
}

// A JPEG saved under a .heic name keeps its EXIF where a JPEG keeps it.
func TestAJPEGNamedHEICIsStillDated(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "IMG_0002.heic", exifPhoto("2024:07:22 08:14:30", "a jpeg"))
	when, source, ok := cameraDate(filepath.Join(dir, "IMG_0002.heic"))
	if !ok || source != FromEXIF || when.Year() != 2024 {
		t.Errorf("got %v %q %v, want 2024 from its EXIF", when, source, ok)
	}
}

// A video that came through a chat was made again on the way: its date is
// when it was received.
func TestAChatVideoWasReceived(t *testing.T) {
	c := Capture{When: time.Now(), Source: FromVideo, Origin: OriginOf("", "VID-20190714-WA0001.mp4")}
	if got := c.Kind(); got != DateReceived {
		t.Errorf("kind %q, want %q", got, DateReceived)
	}
}
