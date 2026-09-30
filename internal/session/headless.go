// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package session

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// Page is a page in a browser the program drives, with no window.
type Page struct {
	ctx     context.Context
	cancel  func()
	browser *exec.Cmd
	exited  chan struct{}
}

// Headless opens url in a browser with no window, on the session's profile.
//
// The profile itself, not a copy of its cookies. Google rotates session
// cookies as they are used; a copy that rotates them leaves the profile with
// stale ones. On 2026-09-26, after about ten runs on copies, the profile's
// session was revoked; driven directly, more times than that, it held. So the
// profile must not be open in another browser: the sign-in window is closed
// before this runs (see Login).
//
// The debug port is not refused here: Google refuses sign-in in a browser
// that has one, but Takeout serves a session already signed in, measured
// 2026-09-26. It says what it is — its user-agent reads HeadlessChrome and
// navigator.webdriver is true — and Takeout serves it all the same.
func (s *Session) Headless(ctx context.Context, url string) (*Page, error) {
	browser, err := findBrowser()
	if err != nil {
		return nil, err
	}
	// A port file left by a browser that did not shut down would point at a
	// port nobody listens on.
	portFile := filepath.Join(s.Profile, "DevToolsActivePort")
	if err := os.Remove(portFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	cmd := exec.Command(browser,
		"--headless=new", "--window-size=1400,1000",
		"--user-data-dir="+s.Profile,
		// The same fixed cookie key as the window (see Open): same profile.
		"--password-store=basic", "--use-mock-keychain",
		"--remote-debugging-port=0",
		"--no-first-run", "--no-default-browser-check", "--disable-sync",
	)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting %s: %w", browser, err)
	}
	p := &Page{browser: cmd, exited: make(chan struct{}), cancel: func() {}}
	go func() {
		cmd.Wait()
		close(p.exited)
	}()

	ws, err := browserSocket(portFile)
	if err != nil {
		p.Close()
		return nil, err
	}
	allocCtx, cancelAlloc := chromedp.NewRemoteAllocator(ctx, ws, chromedp.NoModifyURL)
	tabCtx, cancelTab := chromedp.NewContext(allocCtx)
	p.ctx, p.cancel = tabCtx, func() { cancelTab(); cancelAlloc() }
	if err := chromedp.Run(p.ctx, chromedp.Navigate(url)); err != nil {
		p.Close()
		return nil, err
	}
	return p, nil
}

// browserSocket reads the address the browser writes once it listens: the
// port on the first line, the browser's DevTools path on the second.
func browserSocket(file string) (string, error) {
	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(200 * time.Millisecond) {
		if data, err := os.ReadFile(file); err == nil {
			lines := strings.Split(strings.TrimSpace(string(data)), "\n")
			if len(lines) == 2 {
				return "ws://127.0.0.1:" + lines[0] + lines[1], nil
			}
		}
		if time.Now().After(deadline) {
			return "", errors.New("the headless browser did not open its debug port")
		}
	}
}

// Eval runs expression in the page, waits for it if it is a promise, and
// stores its value in out.
func (p *Page) Eval(expression string, out any) error {
	return chromedp.Run(p.ctx, chromedp.Evaluate(expression, out,
		func(e *runtime.EvaluateParams) *runtime.EvaluateParams { return e.WithAwaitPromise(true) }))
}

// Click presses and releases the mouse at x, y. A click from a script never
// reaches Google's handlers — they are bound through jsaction, which answers
// only to input that came through the browser — so it is real input, like a
// person's.
func (p *Page) Click(x, y float64) error {
	return chromedp.Run(p.ctx, chromedp.MouseClickXY(x, y))
}

// Close stops the browser, cleanly, so the profile is written out.
func (p *Page) Close() {
	p.cancel()
	shutDown(p.browser, p.exited)
}
