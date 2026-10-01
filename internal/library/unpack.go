// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"archive/zip"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// UnpackPart opens one downloaded part into dstDir, keeping the paths Takeout
// put inside it, and returns how many files came out.
//
// Some parts are not zips at all. In a real export 14 parts out of 159 are
// bare videos, handed over as they are because they were larger than the
// chunk size. They are media, not containers, so they are linked into the tree
// where Takeout would have filed them — a second name, not a copy, where the
// filesystem allows it — and the caller deletes the part once it is recorded
// as unpacked.
//
// Running it again over the same part is harmless: files are rewritten whole,
// and a link that already exists is left alone.
func UnpackPart(path, dstDir string) (int, error) {
	if strings.HasSuffix(path, ".zip") {
		return unpackOne(path, dstDir)
	}
	target := filepath.Join(dstDir, "Takeout", "Google Photos", filepath.Base(path))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return 0, err
	}
	if err := os.Link(path, target); err != nil && !linked(path, target) {
		// No hard links here — an SMB share refuses them, measured on a NAS
		// on 2026-09-28 — or a name left by an interrupted copy: the file is
		// copied, whole.
		if _, err := copyFile(path, target); err != nil {
			return 0, fmt.Errorf("unpacking %s: %w", filepath.Base(path), err)
		}
	}
	return 1, nil
}

// linked reports whether target is already a second name for path.
func linked(path, target string) bool {
	a, errA := os.Stat(path)
	b, errB := os.Stat(target)
	return errA == nil && errB == nil && os.SameFile(a, b)
}

// ErrDamaged means a part's bytes are not what Google zipped: a file in it
// fails its CRC-32, or the archive cannot be read as one. Downloading the part
// again is the cure.
var ErrDamaged = errors.New("the part is damaged")

// damaged marks err as ErrDamaged when it says the archive's bytes are wrong,
// and leaves it alone otherwise: a full disk is not a reason to download again.
func damaged(err error) error {
	if errors.Is(err, zip.ErrChecksum) || errors.Is(err, zip.ErrFormat) || errors.Is(err, io.ErrUnexpectedEOF) {
		return fmt.Errorf("%w: %w", ErrDamaged, err)
	}
	return err
}

func unpackOne(archivePath, dstDir string) (files int, err error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return 0, damaged(fmt.Errorf("opening %s: %w", archivePath, err))
	}
	defer reader.Close()

	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		// An archive that names a file outside the destination is either
		// corrupt or hostile. Neither deserves to be written.
		target, err := safeJoin(dstDir, entry.Name)
		if err != nil {
			return files, err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return files, err
		}
		// Go's zip reader checks each entry against its CRC-32 as it reads, so
		// a part that unpacks without error arrived intact.
		if err := extract(entry, target); err != nil {
			return files, damaged(err)
		}
		files++
	}
	return files, nil
}

func extract(entry *zip.File, target string) error {
	source, err := entry.Open()
	if err != nil {
		return fmt.Errorf("reading %s from the archive: %w", entry.Name, err)
	}
	defer source.Close()

	file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("writing %s: %w", target, err)
	}
	_, copyErr := io.Copy(file, source)
	if syncErr := file.Sync(); copyErr == nil {
		copyErr = syncErr
	}
	if closeErr := file.Close(); copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		return fmt.Errorf("writing %s: %w", target, copyErr)
	}
	return readsBack(target, entry.CRC32)
}

// ErrNotWritten means a file read back from the disk is not the file that was
// written to it.
var ErrNotWritten = errors.New("the file on disk is not the file written")

// readsBack reads path back through the filesystem and checks it against the
// CRC-32 its bytes were written with. The archive proves what came out of the
// zip; this proves what is on the disk: a write that went wrong on the way — a
// network share that dropped a block, a disk that filled without saying so —
// is caught now, not years later when the photo will not open. It cannot see
// past a cache that the operating system answers from, and does not pretend to.
func readsBack(path string, want uint32) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	sum := crc32.NewIEEE()
	if _, err := io.Copy(sum, file); err != nil {
		return fmt.Errorf("reading back %s: %w", path, err)
	}
	if sum.Sum32() != want {
		return fmt.Errorf("%w: %s", ErrNotWritten, path)
	}
	return nil
}

// safeJoin refuses any path that would land outside root — the zip-slip check.
func safeJoin(root, name string) (string, error) {
	target := filepath.Join(root, filepath.Clean("/"+name))
	if !strings.HasPrefix(target, filepath.Clean(root)+string(os.PathSeparator)) {
		return "", fmt.Errorf("archive entry %q would write outside the output directory", name)
	}
	return target, nil
}

func copyFile(src, dst string) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return 0, err
	}
	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	file, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	sum := crc32.NewIEEE()
	written, copyErr := io.Copy(file, io.TeeReader(source, sum))
	if syncErr := file.Sync(); copyErr == nil {
		copyErr = syncErr
	}
	if closeErr := file.Close(); copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		return written, copyErr
	}
	return written, readsBack(dst, sum.Sum32())
}
