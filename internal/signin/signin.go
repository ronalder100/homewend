// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package signin gives the sign-in window an end. Without it the window has
// none of its own: Google lands the user on a page of its choosing, and the
// only sign that they are in is the cookie store, which Chrome writes every 30
// seconds (net/extras/sqlite/sqlite_persistent_cookie_store.cc).
//
// So sign-in goes through Google's OAuth page for a desktop app, whose
// redirect to 127.0.0.1 Google honours for registered clients and refuses for
// anyone else's address (measured 2026-10-01). Google sends the user back to
// a page served here the moment they finish, and the page says so.
//
// Google's page is shown in a popup: a small window whose only chrome is the
// address, read-only, so the user can see where they are typing their
// password. The browser opens on a page of ours that opens the popup and
// closes itself, which leaves the popup alone on the screen.
//
// The authorization code is never exchanged: no token is wanted, only the
// redirect. Homewend reads nothing through OAuth; the session it uses is the
// one the browser profile holds after signing in.
package signin

import (
	"crypto/rand"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/url"
)

// ClientID identifies Homewend to Google's OAuth page, as a desktop app. It is
// public by nature: it travels in the address bar of every sign-in.
const ClientID = "587168881326-9bcb23m220e0n8n636qke7ja3pvn017r.apps.googleusercontent.com"

// authURL is Google's authorization endpoint for installed apps
// (developers.google.com/identity/protocols/oauth2/native-app).
const authURL = "https://accounts.google.com/o/oauth2/v2/auth"

// ErrCancelled means Google sent the user back without signing them in: they
// declined, or Google refused.
var ErrCancelled = errors.New("sign-in was cancelled")

//go:embed page.html start.html fonts/inter-400.woff2 fonts/inter-700.woff2
var files embed.FS

var (
	page  = template.Must(template.ParseFS(files, "page.html"))
	start = template.Must(template.ParseFS(files, "start.html"))
)

// Every sentence the page shows a person. A second language is a second table.
var text = map[string]map[string]string{
	"ok": {
		"title": "Signed in.",
		"body":  "You're all set. This window closes by itself.",
	},
	"cancelled": {
		"title": "Sign-in cancelled.",
		"body":  "Nothing was changed. This window closes by itself.",
	},
}

// Listener is the page Google sends the user back to, on a free port of
// 127.0.0.1, for one sign-in.
type Listener struct {
	ln    net.Listener
	state string
	done  chan error
}

// Listen starts serving the page. Close stops it.
func Listen() (*Listener, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	l := &Listener{ln: ln, state: rand.Text(), done: make(chan error, 1)}
	mux := http.NewServeMux()
	mux.Handle("GET /fonts/", http.FileServerFS(files))
	mux.HandleFunc("GET /{$}", l.redirect)
	mux.HandleFunc("GET /start", func(w http.ResponseWriter, r *http.Request) {
		start.Execute(w, l.googleURL())
	})
	go http.Serve(ln, mux)
	return l, nil
}

// URL is where the browser opens: the page that opens Google's sign-in in a
// popup.
func (l *Listener) URL() string { return "http://" + l.ln.Addr().String() + "/start" }

// googleURL opens Google's sign-in, to end on this page. The scopes are the
// least Google lets a sign-in ask for.
func (l *Listener) googleURL() string {
	return authURL + "?" + url.Values{
		"client_id":     {ClientID},
		"redirect_uri":  {l.redirectURI()},
		"response_type": {"code"},
		"scope":         {"openid email"},
		"state":         {l.state},
	}.Encode()
}

func (l *Listener) redirectURI() string { return "http://" + l.ln.Addr().String() + "/" }

// Done receives once, when Google sends the user back: nil if they signed in,
// ErrCancelled if not.
func (l *Listener) Done() <-chan error { return l.done }

// Close stops serving the page.
func (l *Listener) Close() error { return l.ln.Close() }

// redirect is where Google sends the user. A request without our state did
// not come from this sign-in, and is turned away rather than ending it.
func (l *Listener) redirect(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("state") != l.state {
		http.NotFound(w, r)
		return
	}
	var result error
	shown := "ok"
	if q.Get("code") == "" {
		// With Google's own word for why, for whoever has to find out.
		result, shown = fmt.Errorf("%w: %s", ErrCancelled, q.Get("error")), "cancelled"
	}
	page.Execute(w, map[string]any{"OK": result == nil, "Text": text[shown]})
	select {
	case l.done <- result:
	default:
	}
}
