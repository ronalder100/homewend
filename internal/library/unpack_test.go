// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"archive/zip"
	"bytes"
	"errors"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"
)

// A copy cut short leaves a name that is not the part: it must be written
// again whole, not taken for a finished link.
func TestABareVideoReplacesWhatAnInterruptedCopyLeft(t *testing.T) {
	dir := t.TempDir()
	part := filepath.Join(dir, "IMG_0010-025.mov")
	if err := os.WriteFile(part, []byte("the whole video"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	target := filepath.Join(out, "Takeout", "Google Photos", "IMG_0010-025.mov")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("the who"), 0o644); err != nil {
		t.Fatal(err)
	}

	if n, err := UnpackPart(part, out); err != nil || n != 1 {
		t.Fatalf("got %d, %v", n, err)
	}
	if got, _ := os.ReadFile(target); string(got) != "the whole video" {
		t.Errorf("got %q", got)
	}
}

// A part whose bytes changed on the way fails its CRC-32 on unpacking, and
// says it is damaged, so it can be downloaded again rather than trusted.
func TestADamagedPartSaysSo(t *testing.T) {
	dir := t.TempDir()
	part := filepath.Join(dir, "takeout-001.zip")
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	// Stored, not deflated, so the photo's bytes sit in the file as they are.
	f, err := w.CreateHeader(&zip.FileHeader{Name: "Takeout/Google Photos/IMG_1.jpg", Method: zip.Store})
	if err != nil {
		t.Fatal(err)
	}
	f.Write([]byte("the photo, every byte of it"))
	w.Close()
	if err := os.WriteFile(part, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if n, err := UnpackPart(part, filepath.Join(dir, "good")); err != nil || n != 1 {
		t.Fatalf("intact part: %d files, %v", n, err)
	}

	data := bytes.Replace(buf.Bytes(), []byte("every"), []byte("EVERY"), 1)
	if err := os.WriteFile(part, data, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := UnpackPart(part, filepath.Join(dir, "out")); !errors.Is(err, ErrDamaged) {
		t.Errorf("got %v, want ErrDamaged", err)
	}
	if err := os.WriteFile(part, []byte("not a zip at all"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := UnpackPart(part, filepath.Join(dir, "out")); !errors.Is(err, ErrDamaged) {
		t.Errorf("not a zip: got %v, want ErrDamaged", err)
	}
}

// What is on the disk is checked against what was written to it.
func TestAFileIsReadBackAgainstItsChecksum(t *testing.T) {
	path := filepath.Join(t.TempDir(), "IMG_1.jpg")
	data := []byte("the photo, every byte of it")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := readsBack(path, crc32.ChecksumIEEE(data)); err != nil {
		t.Errorf("intact file: %v", err)
	}
	if err := readsBack(path, crc32.ChecksumIEEE([]byte("another photo"))); !errors.Is(err, ErrNotWritten) {
		t.Errorf("changed file: got %v, want ErrNotWritten", err)
	}
}
