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
	jobs := &engine.Jobs{}
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
		s, err := engine.LoadSettings()
		if err != nil {
			reply(w, nil, err)
			return
		}
		list, err := engine.Photos(s, q.Get("year"), q.Get("album"), q.Get("nodate") != "", offset, limit)
		reply(w, list, err)
	})
	mux.HandleFunc("GET /api/latest", func(w http.ResponseWriter, r *http.Request) {
		list, err := engine.Latest(6)
		reply(w, list, err)
	})
	mux.HandleFunc("GET /api/about/{hash}", func(w http.ResponseWriter, r *http.Request) {
		a, err := engine.About(r.PathValue("hash"))
		reply(w, a, err)
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
	// Where a photo is on disk, for the shell to show it in the file manager.
	mux.HandleFunc("GET /api/path/{hash}", func(w http.ResponseWriter, r *http.Request) {
		path, err := engine.Original(r.PathValue("hash"))
		reply(w, map[string]string{"path": path}, err)
	})
	mux.HandleFunc("GET /api/folder", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		path, err := engine.Folder(q.Get("year"), q.Get("album"), q.Get("nodate") != "")
		reply(w, map[string]string{"path": path}, err)
	})
	mux.HandleFunc("GET /api/library", func(w http.ResponseWriter, r *http.Request) {
		dir, err := engine.DefaultLibrary()
		var free int64
		if err == nil && dir != "" {
			free, _ = engine.Free(dir)
		}
		reply(w, map[string]any{"dir": dir, "free": free}, err)
	})
	mux.HandleFunc("PUT /api/library", func(w http.ResponseWriter, r *http.Request) {
		var body struct{ Dir string }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Dir == "" {
			http.Error(w, "a folder is needed", http.StatusBadRequest)
			return
		}
		reply(w, map[string]string{"dir": body.Dir}, engine.SetDefaultLibrary(body.Dir))
	})
	// Bringing photos home runs in the engine; the page asks where it stands.
	mux.HandleFunc("POST /api/get", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		dir, err := engine.DefaultLibrary()
		if err == nil && dir == "" {
			err = engine.ErrNoLibrary
		}
		if err != nil {
			reply(w, nil, err)
			return
		}
		year, _ := strconv.Atoi(q.Get("year"))
		g := engine.Get{Year: year, Library: dir, Takeout: q.Get("export"), New: q.Get("new") != ""}
		err = jobs.Start(q.Get("account"), g)
		reply(w, jobs.State(q.Get("account")), err)
	})
	mux.HandleFunc("GET /api/job", func(w http.ResponseWriter, r *http.Request) {
		reply(w, jobs.State(r.URL.Query().Get("account")), nil)
	})
	mux.HandleFunc("POST /api/job/stop", func(w http.ResponseWriter, r *http.Request) {
		account := r.URL.Query().Get("account")
		jobs.Stop(account)
		reply(w, jobs.State(account), nil)
	})
	// Signing out deletes the account's browser profile: its Google cookies
	// go, its photos stay.
	mux.HandleFunc("POST /api/logout", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("account")
		dir, err := engine.ProfileOf(id)
		if err == nil {
			jobs.Stop(id)
			_, err = engine.Logout(dir)
		}
		reply(w, map[string]string{}, err)
	})
	mux.HandleFunc("GET /api/about", func(w http.ResponseWriter, r *http.Request) {
		reply(w, map[string]string{"version": Version}, nil)
	})
	// What a takeout made on Google holds: its manifest, fetched alone.
	mux.HandleFunc("POST /api/takeouts/contents", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		sess, err := engine.AccountSession(q.Get("account"))
		if err != nil {
			reply(w, nil, err)
			return
		}
		root, err := engine.DefaultLibrary()
		if err == nil && root == "" {
			err = engine.ErrNoLibrary
		}
		if err != nil {
			reply(w, nil, err)
			return
		}
		years, err := engine.ReadContents(r.Context(), sess, root, q.Get("export"))
		reply(w, map[string][]string{"years": years}, err)
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
	case errors.Is(err, engine.ErrJobRunning), errors.Is(err, engine.ErrNoUserYet):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, fs.ErrNotExist), errors.Is(err, engine.ErrNoLibrary):
		http.Error(w, err.Error(), http.StatusNotFound)
	case err != nil:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	default:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(v)
	}
}
