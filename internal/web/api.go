// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package web

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"strconv"
	"sync"

	"github.com/ronalder100/homewend/internal/engine"
)

// apiRoutes are the engine calls the page makes. Each is a thin translation
// to JSON: what they do lives in the engine.
func apiRoutes() http.Handler {
	mux := http.NewServeMux()
	// One sign-in at a time: a second click must not open a second browser.
	var signingIn sync.Mutex
	mux.HandleFunc("GET /api/settings", func(w http.ResponseWriter, r *http.Request) {
		s, err := engine.LoadSettings()
		reply(w, s, err)
	})
	mux.HandleFunc("PUT /api/settings", func(w http.ResponseWriter, r *http.Request) {
		var s engine.Settings
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		reply(w, s, engine.SaveSettings(s))
	})
	mux.HandleFunc("GET /api/overview", func(w http.ResponseWriter, r *http.Request) {
		s, err := engine.LoadSettings()
		if err != nil {
			reply(w, nil, err)
			return
		}
		o, err := engine.LibraryOverview(s)
		reply(w, o, err)
	})
	// Sign-in happens in the system browser; the request waits for it. With
	// no account named it signs in a new one.
	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		if !signingIn.TryLock() {
			http.Error(w, "a sign-in is already open", http.StatusConflict)
			return
		}
		defer signingIn.Unlock()
		id := r.URL.Query().Get("account")
		if id == "" {
			var err error
			if id, err = engine.NextAccount(); err != nil {
				reply(w, nil, err)
				return
			}
		}
		sess, err := engine.AccountSession(id)
		if err != nil {
			reply(w, nil, err)
			return
		}
		if _, _, err := engine.Login(r.Context(), sess, nil); err != nil {
			reply(w, nil, err)
			return
		}
		reply(w, engine.AccountInfo{ID: id, Email: engine.Account(sess)}, nil)
	})
	mux.HandleFunc("GET /api/photos", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		offset, _ := strconv.Atoi(q.Get("offset"))
		limit, _ := strconv.Atoi(q.Get("limit"))
		list, err := engine.Photos(q.Get("year"), q.Get("album"), q.Get("nodate") != "", offset, limit)
		reply(w, list, err)
	})
	// A thumbnail never changes: it is named by the photo's content.
	mux.HandleFunc("GET /api/thumb/{hash}", func(w http.ResponseWriter, r *http.Request) {
		path, err := engine.Thumbnail(r.Context(), r.PathValue("hash"))
		if err != nil {
			reply(w, nil, err)
			return
		}
		w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
		http.ServeFile(w, r, path)
	})
	mux.HandleFunc("GET /api/original/{hash}", func(w http.ResponseWriter, r *http.Request) {
		path, err := engine.Original(r.PathValue("hash"))
		if err != nil {
			reply(w, nil, err)
			return
		}
		http.ServeFile(w, r, path)
	})
	mux.HandleFunc("GET /api/takeouts", func(w http.ResponseWriter, r *http.Request) {
		sess, err := engine.AccountSession(r.URL.Query().Get("account"))
		if err != nil {
			reply(w, nil, err)
			return
		}
		list, err := engine.Takeouts(sess)
		reply(w, list, err)
	})
	return mux
}

// reply writes v as JSON, or the error: a mistake in the request is the
// caller's, anything else is ours.
func reply(w http.ResponseWriter, v any, err error) {
	switch {
	case errors.Is(err, engine.ErrBadTheme), errors.Is(err, engine.ErrNoSuchAccount):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, fs.ErrNotExist), errors.Is(err, engine.ErrNoLibrary):
		http.Error(w, err.Error(), http.StatusNotFound)
	case err != nil:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	default:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(v)
	}
}
