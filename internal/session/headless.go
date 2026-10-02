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

	cdpbrowser "github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// Page is a page in a browser the program drives, with no window.
type Page struct {
	sess    *Session
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
	return s.headless(ctx, chromedp.Navigate(url))
}

// Glimpse is Headless for a page that only has to be reached, not used: it
// returns as soon as the page has answered, without waiting for it to load.
// By then the cookies the visit was for are set, and the start of the page is
// there to read. Takeout's /manage answered in 0.8 seconds and took 3.9 to
// load, 1.4 MB of it (2026-10-02).
func (s *Session) Glimpse(ctx context.Context, url string) (*Page, error) {
	return s.headless(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		_, _, failed, _, err := page.Navigate(url).Do(ctx)
		if err == nil && failed != "" {
			err = errors.New(failed)
		}
		return err
	}))
}

// headless starts the browser and opens the page with open.
func (s *Session) headless(ctx context.Context, open chromedp.Action) (*Page, error) {
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
	p := &Page{sess: s, browser: cmd, exited: make(chan struct{}), cancel: func() {}}
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
	if err := chromedp.Run(p.ctx, open); err != nil {
		p.Close()
		return nil, err
	}
	return p, nil
}

// browserSocket reads the address the browser writes once it listens: the
// port on the first line, the browser's DevTools path on the second.
func browserSocket(file string) (string, error) {
	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(20 * time.Millisecond) {
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

// Close closes the browser through DevTools, which writes the profile out at
// once: closed this way right after a page set 60 cookies, the browser was
// gone within a quarter of a second with all 60 on disk, four runs of four
// (Chromium 144, 2026-10-02). Stopped by SIGTERM instead it loses them, and
// left to its 30-second timer it kept the user waiting half a minute.
//
// The command goes through the page's own session. chromedp refuses it on
// the browser's, and its Cancel only closes the tab of a browser it did not
// start itself.
//
// A cookie Google rotated and the profile never kept is a session going stale.
func (p *Page) Close() {
	if p.ctx != nil {
		if target := chromedp.FromContext(p.ctx).Target; target != nil {
			closing, cancel := context.WithTimeout(p.ctx, 5*time.Second)
			cdpbrowser.Close().Do(cdp.WithExecutor(closing, target))
			cancel()
			select {
			case <-p.exited:
			case <-time.After(10 * time.Second):
			}
		}
	}
	shutDown(p.browser, p.exited)
	// chromedp takes a second to give up on a browser that is gone (measured
	// 2026-10-02): nobody has to wait for that.
	go p.cancel()
}
