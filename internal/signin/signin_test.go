// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package signin

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func listen(t *testing.T) *Listener {
	t.Helper()
	l, err := Listen()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	return l
}

// get asks the page what Google would, and returns the status and the body.
func get(t *testing.T, address string) (int, string) {
	t.Helper()
	res, err := http.Get(address)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(body)
}

func back(l *Listener, q url.Values) string { return l.redirectURI() + "?" + q.Encode() }

func TestStartOpensGoogle(t *testing.T) {
	l := listen(t)
	status, body := get(t, l.URL())
	if status != http.StatusOK || !strings.Contains(body, "accounts.google.com") || !strings.Contains(body, "window.open") {
		t.Errorf("status %d, body %s", status, body)
	}
}

func TestGoogleSendsTheUserBackHere(t *testing.T) {
	l := listen(t)
	u, err := url.Parse(l.googleURL())
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if u.Host != "accounts.google.com" || q.Get("client_id") != ClientID || q.Get("scope") != "openid email" {
		t.Errorf("unexpected sign-in address %s", u)
	}
	if !strings.HasPrefix(q.Get("redirect_uri"), "http://127.0.0.1:") || q.Get("state") == "" {
		t.Errorf("redirect %q, state %q", q.Get("redirect_uri"), q.Get("state"))
	}
}

func TestSignedIn(t *testing.T) {
	l := listen(t)
	status, body := get(t, back(l, url.Values{"state": {l.state}, "code": {"x"}}))
	if status != http.StatusOK || !strings.Contains(body, text["ok"]["title"]) {
		t.Errorf("status %d, body without %q", status, text["ok"]["title"])
	}
	if err := wait(t, l); err != nil {
		t.Errorf("got %v, want nil", err)
	}
}

func TestCancelled(t *testing.T) {
	l := listen(t)
	status, body := get(t, back(l, url.Values{"state": {l.state}, "error": {"access_denied"}}))
	if status != http.StatusOK || !strings.Contains(body, text["cancelled"]["title"]) {
		t.Errorf("status %d, body without %q", status, text["cancelled"]["title"])
	}
	if err := wait(t, l); !errors.Is(err, ErrCancelled) {
		t.Errorf("got %v, want ErrCancelled", err)
	}
}

// Anything on the machine can reach the port; only Google, carrying our
// state, ends the sign-in.
func TestStrangerIsTurnedAway(t *testing.T) {
	l := listen(t)
	if status, _ := get(t, back(l, url.Values{"state": {"other"}, "code": {"x"}})); status != http.StatusNotFound {
		t.Errorf("status %d, want 404", status)
	}
	select {
	case err := <-l.Done():
		t.Errorf("sign-in ended with %v", err)
	default:
	}
}

func TestFontsAreServed(t *testing.T) {
	l := listen(t)
	if status, _ := get(t, l.redirectURI()+"fonts/inter-700.woff2"); status != http.StatusOK {
		t.Errorf("status %d", status)
	}
}

func wait(t *testing.T, l *Listener) error {
	t.Helper()
	select {
	case err := <-l.Done():
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("no result")
		return nil
	}
}
