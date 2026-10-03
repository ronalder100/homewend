// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	s, err := Listen()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.ln.Close() })
	return s
}

func get(s *Server, target string, cookie *http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, target, nil)
	r.Host = s.host
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	s.handler().ServeHTTP(w, r)
	return w
}

func TestURLIsLoopbackWithToken(t *testing.T) {
	s := newTestServer(t)
	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(u.Host, "127.0.0.1:") || u.Query().Get("t") != s.token {
		t.Fatalf("URL %q: want 127.0.0.1 and the token", s.URL)
	}
}

func TestNoTokenNoPage(t *testing.T) {
	s := newTestServer(t)
	if w := get(s, "/", nil); w.Code != http.StatusForbidden {
		t.Fatalf("without a token: %d, want 403", w.Code)
	}
	if w := get(s, "/?t=wrong", nil); w.Code != http.StatusForbidden {
		t.Fatalf("wrong token: %d, want 403", w.Code)
	}
	if w := get(s, "/", &http.Cookie{Name: cookieName, Value: "wrong"}); w.Code != http.StatusForbidden {
		t.Fatalf("wrong cookie: %d, want 403", w.Code)
	}
}

func TestTokenBecomesCookie(t *testing.T) {
	s := newTestServer(t)
	w := get(s, "/?t="+s.token, nil)
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/" {
		t.Fatalf("token: %d to %q, want 303 to /", w.Code, w.Header().Get("Location"))
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Value != s.token || !cookies[0].HttpOnly {
		t.Fatalf("cookie: %+v", cookies)
	}
	if w := get(s, "/", cookies[0]); w.Code == http.StatusForbidden {
		t.Fatal("with the cookie: still refused")
	}
}

func TestOtherHostRefused(t *testing.T) {
	s := newTestServer(t)
	r := httptest.NewRequest(http.MethodGet, "/?t="+s.token, nil)
	r.Host = "evil.example:80"
	w := httptest.NewRecorder()
	s.handler().ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("other host: %d, want 403", w.Code)
	}
}

func TestRoutesFallBackToThePage(t *testing.T) {
	files := fstest.MapFS{
		fallback:      {Data: []byte("page")},
		"_app/app.js": {Data: []byte("js")},
	}
	for target, want := range map[string]string{
		"/":             "page",
		"/library/2025": "page",
		"/_app/app.js":  "js",
	} {
		w := httptest.NewRecorder()
		servePage(w, httptest.NewRequest(http.MethodGet, target, nil), files)
		if body, _ := io.ReadAll(w.Body); string(body) != want {
			t.Errorf("%s: %q, want %q", target, body, want)
		}
	}
}

func TestNotBuilt(t *testing.T) {
	w := httptest.NewRecorder()
	servePage(w, httptest.NewRequest(http.MethodGet, "/", nil), fstest.MapFS{})
	if w.Code != http.StatusNotFound {
		t.Fatalf("no page: %d, want 404", w.Code)
	}
}

func TestSettingsThroughTheAPI(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AppData", t.TempDir())
	s := newTestServer(t)
	cookie := &http.Cookie{Name: cookieName, Value: s.token}

	put := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(`{"theme":"dark"}`))
	put.Host = s.host
	put.AddCookie(cookie)
	w := httptest.NewRecorder()
	s.handler().ServeHTTP(w, put)
	if w.Code != http.StatusOK {
		t.Fatalf("put: %d %s", w.Code, w.Body)
	}
	if got := get(s, "/api/settings", cookie); !strings.Contains(got.Body.String(), `"theme":"dark"`) {
		t.Fatalf("get: %s", got.Body)
	}

	bad := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(`{"theme":"blue"}`))
	bad.Host = s.host
	bad.AddCookie(cookie)
	w = httptest.NewRecorder()
	s.handler().ServeHTTP(w, bad)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad theme: %d, want 400", w.Code)
	}
}
