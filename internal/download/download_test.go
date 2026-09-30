// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package download

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ronalder100/homewend/internal/progress"
	"github.com/ronalder100/homewend/internal/takeout"
)

// dropping serves data with Range, and cuts each of the first drops
// transfers short after cut bytes, as a connection that goes down would.
type dropping struct {
	data  []byte
	cut   int
	drops int
}

func (d *dropping) Get(_ string, extra map[string]string) (*http.Response, error) {
	from, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(extra["Range"], "bytes="), "-"))
	if extra["Range"] == "bytes=0-0" {
		return &http.Response{
			StatusCode: http.StatusPartialContent,
			Header:     http.Header{"Content-Range": {fmt.Sprintf("bytes 0-0/%d", len(d.data))}},
			Body:       io.NopCloser(bytes.NewReader(d.data[:1])),
		}, nil
	}
	body := d.data[from:]
	if d.drops > 0 && len(body) > d.cut {
		d.drops--
		body = body[:d.cut]
	}
	return &http.Response{StatusCode: http.StatusPartialContent, Body: io.NopCloser(bytes.NewReader(body))}, nil
}

func TestADroppedTransferIsPickedUpWhereItStopped(t *testing.T) {
	data := bytes.Repeat([]byte("0123456789"), 100)
	dir := t.TempDir()
	part := takeout.Part{Filename: "a.zip"}
	err := Part(context.Background(), &dropping{data: data, cut: 300, drops: 2}, takeout.Target{}, part, 1, 1, dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "a.zip"))
	if !bytes.Equal(got, data) {
		t.Errorf("got %d bytes, want the %d sent", len(got), len(data))
	}
}

// A progress bar needs to hear how far a part has got, counted from the
// start of the part and not of the connection, across a drop.
func TestArrivingBytesAreReported(t *testing.T) {
	reportEvery = 0
	defer func() { reportEvery = time.Second }()
	data := bytes.Repeat([]byte("0123456789"), 100)
	var done []int64
	emit := func(e progress.Event) {
		if e.Stage == progress.Receiving {
			if e.Total != int64(len(data)) {
				t.Errorf("total %d, want %d", e.Total, len(data))
			}
			done = append(done, e.Done)
		}
	}
	err := Part(context.Background(), &dropping{data: data, cut: 300, drops: 2}, takeout.Target{}, takeout.Part{Filename: "a.zip"}, 1, 1, t.TempDir(), emit)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(done); i++ {
		if done[i] <= done[i-1] {
			t.Fatalf("done went from %d to %d", done[i-1], done[i])
		}
	}
	if len(done) < 3 || done[len(done)-1] != int64(len(data)) {
		t.Errorf("reported %v, want at least one per segment ending at %d", done, len(data))
	}
}

func TestATransferThatBringsNothingStops(t *testing.T) {
	data := bytes.Repeat([]byte("0123456789"), 100)
	err := Part(context.Background(), &dropping{data: data, cut: 0, drops: 1000}, takeout.Target{}, takeout.Part{Filename: "a.zip"}, 1, 1, t.TempDir(), nil)
	if err != ErrIncomplete {
		t.Errorf("got %v, want ErrIncomplete", err)
	}
}

// offline fails to connect a number of times, then answers.
type offline struct{ fails int }

func (o *offline) Get(string, map[string]string) (*http.Response, error) {
	if o.fails > 0 {
		o.fails--
		return nil, errors.New("dial tcp: connect: network is unreachable")
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
}

func TestADownNetworkIsWaitedForNotGivenUpOn(t *testing.T) {
	after = func(time.Duration) <-chan time.Time { return time.After(0) }
	defer func() { after = time.After }()
	if _, err := request(context.Background(), &offline{fails: 50}, "https://example.com", nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestAStopEndsTheWaitForTheNetwork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := request(ctx, &offline{fails: 1}, "https://example.com", nil, nil); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
}

// quiet sends the first cut bytes and then nothing, without closing, as a
// connection that has gone dead would; after that it serves the rest.
type quiet struct {
	data   []byte
	cut    int
	hanged bool
}

func (q *quiet) Get(_ string, extra map[string]string) (*http.Response, error) {
	if extra["Range"] == "bytes=0-0" {
		return &http.Response{
			StatusCode: http.StatusPartialContent,
			Header:     http.Header{"Content-Range": {fmt.Sprintf("bytes 0-0/%d", len(q.data))}},
			Body:       io.NopCloser(bytes.NewReader(q.data[:1])),
		}, nil
	}
	from, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(extra["Range"], "bytes="), "-"))
	if q.hanged {
		return &http.Response{StatusCode: http.StatusPartialContent, Body: io.NopCloser(bytes.NewReader(q.data[from:]))}, nil
	}
	q.hanged = true
	r, w := io.Pipe()
	go w.Write(q.data[from : from+q.cut])
	return &http.Response{StatusCode: http.StatusPartialContent, Body: r}, nil
}

func TestATransferThatGoesQuietIsPickedUpAgain(t *testing.T) {
	stallLimit = 100 * time.Millisecond
	defer func() { stallLimit = 2 * time.Minute }()
	data := bytes.Repeat([]byte("0123456789"), 100)
	dir := t.TempDir()
	if err := Part(context.Background(), &quiet{data: data, cut: 300}, takeout.Target{}, takeout.Part{Filename: "a.zip"}, 1, 1, dir, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "a.zip")); !bytes.Equal(got, data) {
		t.Errorf("got %d bytes, want %d", len(got), len(data))
	}
}

func TestAStopEndsATransferInFlight(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	data := bytes.Repeat([]byte("0123456789"), 100)
	err := Part(ctx, &quiet{data: data, cut: 300}, takeout.Target{}, takeout.Part{Filename: "a.zip"}, 1, 1, t.TempDir(), nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("got %v, want the stop", err)
	}
}
