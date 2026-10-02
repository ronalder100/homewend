// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package download brings the archives down. This is the product: 159 files
// arriving without anyone watching them.
//
// Three rules here were paid for in measurements, not taste:
//
//  1. **One connection per archive, one archive at a time.** Parallel streams
//     buy nothing from one host, and one stream is what a person downloading
//     their own export produces.
//  2. **A dropped connection is not an expired session.** They want opposite
//     answers — one waits and retries, the other stops and asks the user to sign
//     in — and conflating them is how a downloader loses a night's work.
//  3. **The size is asked for before the bytes.** A one-byte request returns
//     `Content-Range: bytes 0-0/<total>`, which is the only way to know what a
//     file should weigh before having it, and doubles as the session check.
package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ronalder100/homewend/internal/progress"
	"github.com/ronalder100/homewend/internal/takeout"
)

// ErrSessionExpired means Google no longer accepts the profile's cookies: it
// redirected instead of serving. The only cure is the user signing in again.
var ErrSessionExpired = errors.New("the Google session expired")

// Getter is the part of a session this package needs.
type Getter interface {
	Get(url string, extra map[string]string) (*http.Response, error)
}

// ErrIncomplete means a transfer stopped before the end. The bytes on disk are
// good; running again continues from them.
var ErrIncomplete = errors.New("the transfer stopped before the end")

// Part downloads one part of an export into dir, resuming whatever is already
// there. n and of only label the progress events.
func Part(ctx context.Context, g Getter, target takeout.Target, part takeout.Part, n, of int, dir string, emit progress.Func) error {
	path := filepath.Join(dir, part.Filename)

	total, err := declaredSize(ctx, g, part.URL(target), emit)
	if err != nil {
		return err
	}

	if sizeOnDisk(path) > total {
		// Never trust a file longer than the source says it is.
		os.Remove(path)
	}

	// One segment at a time, each from exactly where the last one stopped, and
	// no blind retry loop inside the HTTP client: the point is to know how much
	// arrived. A connection that drops after bringing bytes is picked up again
	// at once — on the file host, which spends none of Google's five downloads.
	// One that brings nothing ends the attempt.
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		have := sizeOnDisk(path)
		if have == total {
			emit.Emit(progress.Event{Stage: progress.Downloaded, N: n, Of: of, Name: part.Filename, Done: total, Total: total})
			return nil
		}
		emit.Emit(progress.Event{Stage: progress.Download, N: n, Of: of, Name: part.Filename, Done: have, Total: total})
		received, reported := have, time.Now()
		arrived := func(bytes int) {
			received += int64(bytes)
			if time.Since(reported) >= reportEvery {
				reported = time.Now()
				emit.Emit(progress.Event{Stage: progress.Receiving, N: n, Of: of, Name: part.Filename, Done: received, Total: total})
			}
		}
		if err := segment(ctx, g, part.URL(target), path, have, emit, arrived); err != nil {
			return err
		}
		got := sizeOnDisk(path)
		if got == total {
			continue
		}
		emit.Emit(progress.Event{Stage: progress.Short, N: n, Of: of, Name: part.Filename, Done: got, Total: total})
		if got == have {
			return ErrIncomplete
		}
	}
}

// segment appends to path whatever the server sends from byte have onwards,
// until it finishes, the connection drops, ctx is cancelled, or nothing
// arrives for stallLimit. arrived is told of every read that brings bytes.
func segment(ctx context.Context, g Getter, url, path string, have int64, emit progress.Func, arrived func(int)) error {
	res, err := request(ctx, g, url, map[string]string{"Range": fmt.Sprintf("bytes=%d-", have)}, emit)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusPartialContent && res.StatusCode != http.StatusOK {
		return ErrSessionExpired
	}
	// Closing the body is what ends a transfer from outside: a stop asked
	// for, or a connection that has gone quiet without dropping. No limit on
	// the whole transfer: a 2 GB part can take hours on a slow line.
	stall := time.AfterFunc(stallLimit, func() { res.Body.Close() })
	defer stall.Stop()
	defer context.AfterFunc(ctx, func() { res.Body.Close() })()

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}
	_, copyErr := io.Copy(file, watched{res.Body, stall, arrived})
	if closeErr := file.Close(); copyErr == nil && closeErr != nil {
		return fmt.Errorf("writing %s: %w", path, closeErr)
	}
	return ctx.Err()
}

// stallLimit is how long a transfer may bring nothing before it is taken for
// dead and picked up again from where it stopped.
var stallLimit = 2 * time.Minute

// reportEvery is how often a transfer says how far it has got: often enough
// for a progress bar, rarely enough for a log of JSON lines.
var reportEvery = time.Second

// watched is a body whose every read that brings bytes pushes the stall back
// and is reported.
type watched struct {
	io.Reader
	stall   *time.Timer
	arrived func(int)
}

func (w watched) Read(p []byte) (int, error) {
	n, err := w.Reader.Read(p)
	if n > 0 {
		w.stall.Reset(stallLimit)
		w.arrived(n)
	}
	return n, err
}

// declaredSize asks for a single byte and reads the total out of Content-Range.
func declaredSize(ctx context.Context, g Getter, url string, emit progress.Func) (int64, error) {
	res, err := request(ctx, g, url, map[string]string{"Range": "bytes=0-0"}, emit)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	io.Copy(io.Discard, res.Body)

	if res.StatusCode != http.StatusPartialContent {
		return 0, ErrSessionExpired
	}
	// "bytes 0-0/4332171"
	_, after, found := strings.Cut(res.Header.Get("Content-Range"), "/")
	if !found {
		return 0, fmt.Errorf("no size in Content-Range")
	}
	total, err := strconv.ParseInt(strings.TrimSpace(after), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("unreadable size in Content-Range: %w", err)
	}
	return total, nil
}

// request retries a connection that never happened, and only that, for as
// long as it takes: a network that is down comes back, and the export waits on
// Google's side for a week. The pause grows to a minute and stays there. An
// answer from Google — even an unwelcome one — is returned as it is.
func request(ctx context.Context, g Getter, url string, extra map[string]string, emit progress.Func) (*http.Response, error) {
	for attempt := 1; ; attempt++ {
		res, err := g.Get(url, extra)
		if err == nil {
			return res, nil
		}
		emit.Emit(progress.Event{Stage: progress.Retry, N: attempt, Note: err.Error()})
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-after(min(time.Duration(attempt)*5*time.Second, time.Minute)):
		}
	}
}

// after is time.After, replaced in tests.
var after = time.After

func sizeOnDisk(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
