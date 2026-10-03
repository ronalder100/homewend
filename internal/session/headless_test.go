// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package session

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"
)

// A page of the hidden host is never on screen, and the download it starts
// still begins and lands where it was sent. Runs a real browser, with no
// window.
func TestHiddenPageStillStartsItsDownload(t *testing.T) {
	if _, err := findBrowser(); err != nil {
		t.Skip("no Chromium-based browser here")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/page":
			// Like Takeout's page: it starts the download itself, a moment
			// after it loads.
			fmt.Fprint(w, `<!doctype html><title>page</title><p>seen?</p><script>setTimeout(() => location.href = "/file", 300)</script>`)
		case "/file":
			w.Header().Set("Content-Disposition", `attachment; filename="part.zip"`)
			fmt.Fprint(w, "zip")
		}
	}))
	defer srv.Close()
	host, _ := url.Parse(srv.URL)

	sess, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	p, err := sess.headless(ctx, hidden(host.Hostname()))
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	landing := t.TempDir()
	from, err := p.Download(srv.URL+"/page", landing, time.After(20*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if from != srv.URL+"/file" {
		t.Errorf("download came from %q", from)
	}
	var visibility string
	if err := p.Eval(`getComputedStyle(document.documentElement).visibility`, &visibility); err != nil {
		t.Fatal(err)
	}
	if visibility != "hidden" {
		t.Errorf("the page was %q", visibility)
	}
	if files, _ := os.ReadDir(landing); len(files) != 1 {
		t.Errorf("%d files landed", len(files))
	}
}
