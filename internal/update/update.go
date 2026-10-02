// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package update brings the installed homewend up to the latest release, and
// says when there is one.
//
// A release is what install.sh installs: on GitHub, a bare binary named
// homewend_<os>_<arch> and homewend_checksums.txt beside it
// (.goreleaser.yaml). Nothing here goes through a server of ours.
//
// It has an HTTP client of its own, and must: the session's Get carries the
// user's Google cookies, which are for Google and nobody else.
package update

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ronalder100/homewend/internal/progress"
)

// latest is the newest release. GitHub answers it with a redirect to
// releases/tag/v<version>, and serves the release's files under
// latest/download/. Read 2026-10-02.
var latest = "https://github.com/ronalder100/homewend/releases/latest"

// ErrDamaged means the download does not match the release's checksum.
// Nothing was installed.
var ErrDamaged = errors.New("the download does not match the release's checksum")

// Latest asks GitHub for the version of the newest release, without the v.
func Latest(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, latest, nil)
	if err != nil {
		return "", err
	}
	// The answer wanted is the redirect itself.
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("asking for the latest release: %w", err)
	}
	res.Body.Close()
	tag := path.Base(res.Header.Get("Location"))
	if res.StatusCode != http.StatusFound || !strings.HasPrefix(tag, "v") {
		return "", fmt.Errorf("asking for the latest release: HTTP %d", res.StatusCode)
	}
	return strings.TrimPrefix(tag, "v"), nil
}

// Install downloads the latest release for this machine and puts it in place
// of target, the homewend that is running. The download is checked against the
// release's checksums first, and lands beside target so that replacing it is
// one rename: target is either the old program or the new one, never half of
// each.
func Install(ctx context.Context, target string, emit progress.Func) error {
	name := "homewend_" + runtime.GOOS + "_" + runtime.GOARCH
	want, err := checksum(ctx, name)
	if err != nil {
		return err
	}
	res, err := get(ctx, latest+"/download/"+name)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	file, err := os.CreateTemp(filepath.Dir(target), ".homewend-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()

	sum := sha256.New()
	arriving := &counted{total: res.ContentLength, emit: emit}
	if _, err := io.Copy(io.MultiWriter(file, sum, arriving), res.Body); err != nil {
		return fmt.Errorf("downloading %s: %w", name, err)
	}
	if hex.EncodeToString(sum.Sum(nil)) != want {
		return ErrDamaged
	}
	if err := file.Chmod(0o755); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), target)
}

// Program is the homewend that is running, as a file: the one Install
// replaces. A link to it is followed, so that the link stays a link.
func Program() (string, error) {
	program, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(program)
}

// checksum reads the SHA-256 the release declares for name: one line per
// file, "<hex>  <name>".
func checksum(ctx context.Context, name string) (string, error) {
	res, err := get(ctx, latest+"/download/homewend_checksums.txt")
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	lines := bufio.NewScanner(res.Body)
	for lines.Scan() {
		if fields := strings.Fields(lines.Text()); len(fields) == 2 && fields[1] == name {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("the release has no %s", name)
}

func get(ctx context.Context, address string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", path.Base(address), err)
	}
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		return nil, fmt.Errorf("downloading %s: HTTP %d", path.Base(address), res.StatusCode)
	}
	return res, nil
}

// counted reports the bytes as they arrive.
type counted struct {
	done, total int64
	emit        progress.Func
}

func (c *counted) Write(p []byte) (int, error) {
	c.done += int64(len(p))
	c.emit.Emit(progress.Event{Stage: progress.Update, Done: c.done, Total: c.total})
	return len(p), nil
}

// Newer reports the latest release when it is not the one running, asking
// GitHub at most once a day: the answer is kept in the user's cache
// directory. It is for a line at the end of a command, so it never fails and
// never waits long: no answer is "".
//
// A build that is not a release, whose version is "dev", is never told.
func Newer(ctx context.Context, running string) string {
	cache, err := os.UserCacheDir()
	if err != nil || running == "dev" {
		return ""
	}
	dir := filepath.Join(cache, "homewend")
	kept := filepath.Join(dir, "latest")
	version := ""
	if info, err := os.Stat(kept); err == nil && time.Since(info.ModTime()) < 24*time.Hour {
		raw, _ := os.ReadFile(kept)
		version = strings.TrimSpace(string(raw))
	} else {
		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		if version, err = Latest(ctx); err != nil {
			return ""
		}
		if os.MkdirAll(dir, 0o700) == nil {
			os.WriteFile(kept, []byte(version+"\n"), 0o600)
		}
	}
	if version == running {
		return ""
	}
	return version
}
