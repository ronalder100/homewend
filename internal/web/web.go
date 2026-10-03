// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package web serves the interface: the built page, on the loopback only, to
// whoever holds the token in the URL. The desktop app is a window on this
// address; the page commands the downloader, so nothing else may reach it.
package web

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"errors"
	"io/fs"
	"net"
	"net/http"
	"path"
	"strings"
	"time"
)

// dist is the page as built by web/ (npm run build). Only .gitkeep is
// committed, so a binary built without the page serves "notBuilt".
//
//go:embed all:dist
var dist embed.FS

// fallback is the page SvelteKit writes for every route it did not prerender.
const fallback = "200.html"

const cookieName = "homewend"

// Server is the interface on 127.0.0.1, on a port the system picks.
type Server struct {
	// URL opens the page; it carries the token once, the cookie after that.
	URL   string
	token string
	host  string
	ln    net.Listener
	srv   *http.Server
}

// Listen binds the loopback and makes a token for this run.
func Listen() (*Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	token := rand.Text()
	s := &Server{token: token, host: ln.Addr().String(), ln: ln}
	s.URL = "http://" + s.host + "/?t=" + token
	s.srv = &http.Server{Handler: s.handler(), ReadHeaderTimeout: 10 * time.Second}
	return s, nil
}

// Serve answers until Shutdown.
func (s *Server) Serve() error {
	if err := s.srv.Serve(s.ln); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown stops taking requests and waits for the ones in flight.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func (s *Server) handler() http.Handler {
	files, _ := fs.Sub(dist, "dist")
	api := apiRoutes()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A page on another site can resolve its own name to 127.0.0.1;
		// the Host header is what tells that request apart from ours.
		if r.Host != s.host {
			http.Error(w, "wrong host", http.StatusForbidden)
			return
		}
		if t := r.URL.Query().Get("t"); t != "" {
			if !s.valid(t) {
				http.Error(w, "wrong token", http.StatusForbidden)
				return
			}
			// The token leaves the address bar: from here on the cookie
			// carries it, and it is never sent to another site.
			http.SetCookie(w, &http.Cookie{Name: cookieName, Value: t, Path: "/",
				HttpOnly: true, SameSite: http.SameSiteStrictMode})
			q := r.URL.Query()
			q.Del("t")
			r.URL.RawQuery = q.Encode()
			http.Redirect(w, r, r.URL.String(), http.StatusSeeOther)
			return
		}
		if c, err := r.Cookie(cookieName); err != nil || !s.valid(c.Value) {
			http.Error(w, "open Homewend from the app", http.StatusForbidden)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			api.ServeHTTP(w, r)
			return
		}
		servePage(w, r, files)
	})
}

func (s *Server) valid(t string) bool {
	return subtle.ConstantTimeCompare([]byte(t), []byte(s.token)) == 1
}

// servePage serves a built file, or the app's fallback for a route.
func servePage(w http.ResponseWriter, r *http.Request, files fs.FS) {
	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if name == "" {
		name = fallback
	}
	if _, err := fs.Stat(files, name); err != nil {
		name = fallback
	}
	if _, err := fs.Stat(files, name); err != nil {
		http.Error(w, notBuilt, http.StatusNotFound)
		return
	}
	http.ServeFileFS(w, r, files, name)
}

const notBuilt = "the interface is not in this build: run npm run build in web/"
