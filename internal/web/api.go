// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package web

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ronalder100/homewend/internal/engine"
)

// apiRoutes are the engine calls the page makes. Each is a thin translation
// to JSON: what they do lives in the engine.
func apiRoutes() http.Handler {
	mux := http.NewServeMux()
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
	return mux
}

// reply writes v as JSON, or the error: a mistake in the request is the
// caller's, anything else is ours.
func reply(w http.ResponseWriter, v any, err error) {
	switch {
	case errors.Is(err, engine.ErrBadTheme):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case err != nil:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	default:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(v)
	}
}
