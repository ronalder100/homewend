// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/ronalder100/homewend/internal/engine"
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

func TestProfileSurvivesTheWindowsSettings(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AppData", cfg)
	// The account is signed in: its folder is there.
	dir, err := engine.AccountDir("google/ann@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	s := newTestServer(t)
	cookie := &http.Cookie{Name: cookieName, Value: s.token}
	put := func(path, body string) int {
		r := httptest.NewRequest(http.MethodPut, path, strings.NewReader(body))
		r.Host = s.host
		r.AddCookie(cookie)
		w := httptest.NewRecorder()
		s.handler().ServeHTTP(w, r)
		return w.Code
	}

	if code := put("/api/profile?account=google/ann@example.com", `{"profile":"Ann"}`); code != http.StatusOK {
		t.Fatalf("profile: %d", code)
	}
	// A page that read the settings before the profile was set.
	if code := put("/api/settings", `{"theme":"dark"}`); code != http.StatusOK {
		t.Fatalf("settings: %d", code)
	}
	if got := get(s, "/api/settings", cookie).Body.String(); !strings.Contains(got, `"google/ann@example.com":"Ann"`) {
		t.Fatalf("the profile was lost: %s", got)
	}
	if code := put("/api/profile?account=google/ann@example.com", `{"profile":"../x"}`); code != http.StatusBadRequest {
		t.Fatalf("bad profile: %d, want 400", code)
	}
	// An account that is not here gets no profile.
	if code := put("/api/profile?account=google/bob@example.com", `{"profile":"Bob"}`); code != http.StatusNotFound {
		t.Fatalf("unknown account: %d, want 404", code)
	}
}

// A takeout's folder is asked by the profile it went in, which may no longer
// be its account's.
func TestFolderOfAProfile(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AppData", cfg)
	s := newTestServer(t)
	cookie := &http.Cookie{Name: cookieName, Value: s.token}

	if got := get(s, "/api/folder?profile=ann", cookie); got.Code != http.StatusNotFound {
		t.Errorf("no library yet: %d, want 404", got.Code)
	}
	lib := t.TempDir()
	if err := engine.SetDefaultLibrary(lib); err != nil {
		t.Fatal(err)
	}
	got := get(s, "/api/folder?profile=ann", cookie)
	if want := strings.ReplaceAll(filepath.Join(lib, "ann"), `\`, `\\`); got.Code != http.StatusOK || !strings.Contains(got.Body.String(), want) {
		t.Errorf("got %d %s, want %s", got.Code, got.Body, want)
	}
	if got := get(s, "/api/folder?profile=..%2Fx", cookie); got.Code != http.StatusBadRequest {
		t.Errorf("a path as a profile: %d, want 400", got.Code)
	}
}
