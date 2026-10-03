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
	"strconv"
	"strings"
	"time"

	cdpbrowser "github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

// Page is a page in a browser the program drives: with no window, or in one
// the user sees (Watched).
type Page struct {
	sess *Session
	// ctx carries the DevTools connection. It outlives the caller's context,
	// so that a page the caller gave up on can still be closed through it.
	ctx context.Context
	// gaveUp is the caller's context ending: waits stop there.
	gaveUp  <-chan struct{}
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

// Watched is Headless in a window the user sees, for the one step only the
// user can take: typing the password Google asks for, again, of a session
// already signed in. It opens blank, and every page from host is kept off
// screen from its first instant, before any of its own scripts run: hidden,
// not stopped, so what the page starts still happens.
//
// Google takes the password in a browser with a debug port, here: on a
// session signed in the day before, the password page came up, was answered,
// and the download began, with the session still accepted afterwards
// (Chromium 144, 2026-10-03). The sign-in itself it refuses (see Open).
func (s *Session) Watched(ctx context.Context, host string) (*Page, error) {
	return s.drive(ctx, nil, hidden(host))
}

// hidden keeps every page from host off screen, from the next one the tab
// opens. A script added this way runs as each document is created, before
// the page's own and before anything is drawn (Page.addScriptToEvaluateOnNewDocument).
func hidden(host string) chromedp.Action {
	hide := `if (location.hostname === ` + strconv.Quote(host) + `) {
  const s = new CSSStyleSheet();
  s.replaceSync('html{visibility:hidden!important;background:#fff!important}');
  document.adoptedStyleSheets = [s];
}`
	return chromedp.ActionFunc(func(ctx context.Context) error {
		_, err := page.AddScriptToEvaluateOnNewDocument(hide).Do(ctx)
		return err
	})
}

// headless starts the browser with no window and opens the page with open.
func (s *Session) headless(ctx context.Context, open chromedp.Action) (*Page, error) {
	return s.drive(ctx, []string{"--headless=new", "--window-size=1400,1000"}, open)
}

// drive starts the browser with flags and a debug port, attaches to the tab
// it opens with, and runs open there.
//
// That tab, not a new one: in a window the user would see a second tab, and
// whatever the program did would happen in the one they are not looking at.
func (s *Session) drive(ctx context.Context, flags []string, open chromedp.Action) (*Page, error) {
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

	cmd := exec.Command(browser, append(flags,
		"--user-data-dir="+s.Profile,
		// The same fixed cookie key as the window (see Open): same profile.
		"--password-store=basic", "--use-mock-keychain",
		"--remote-debugging-port=0",
		"--no-first-run", "--no-default-browser-check", "--disable-sync",
		"--disable-session-crashed-bubble", "--hide-crash-restore-bubble",
		blank,
	)...)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting %s: %w", browser, err)
	}
	p := &Page{sess: s, browser: cmd, exited: make(chan struct{}), cancel: func() {}, gaveUp: ctx.Done()}
	go func() {
		cmd.Wait()
		close(p.exited)
	}()

	ws, err := browserSocket(portFile)
	if err != nil {
		p.Close()
		return nil, err
	}
	// Closing the browser after the caller gave up needs the connection: it
	// is not tied to the caller's context (see Close).
	allocCtx, cancelAlloc := chromedp.NewRemoteAllocator(context.WithoutCancel(ctx), ws, chromedp.NoModifyURL)
	rootCtx, cancelRoot := chromedp.NewContext(allocCtx)
	p.cancel = func() { cancelRoot(); cancelAlloc() }
	tab, err := firstTab(rootCtx)
	if err != nil {
		p.Close()
		return nil, err
	}
	tabCtx, cancelTab := chromedp.NewContext(rootCtx, chromedp.WithTargetID(tab))
	p.ctx, p.cancel = tabCtx, func() { cancelTab(); cancelRoot(); cancelAlloc() }
	if err := chromedp.Run(p.ctx, open); err != nil {
		p.Close()
		return nil, err
	}
	return p, nil
}

// blank is what the browser opens with: a page, so that there is a tab to
// attach to, and nothing in it.
const blank = "about:blank"

// firstTab waits for the tab the browser opens with. chromedp leaves the wait
// to the caller: a browser just started may list no tab yet
// (chromedp.Targets).
func firstTab(ctx context.Context) (target.ID, error) {
	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(20 * time.Millisecond) {
		targets, err := chromedp.Targets(ctx)
		if err != nil {
			return "", err
		}
		for _, t := range targets {
			if t.Type == "page" && t.URL == blank {
				return t.TargetID, nil
			}
		}
		if time.Now().After(deadline) {
			return "", errors.New("the browser opened no tab")
		}
	}
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
			return "", errors.New("the browser did not open its debug port")
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

// ErrExited means the browser went away before a download began: in a window,
// the user closed it.
var ErrExited = errors.New("the browser exited")

// Download follows address, which should end in a download, and returns the
// address the download finally came from; "" when none began before giveUp
// fired, because the page went somewhere else, to a password prompt for one.
// A nil giveUp waits as long as the browser is there. What is downloaded lands
// in dir.
//
// Followed a few minutes after a sign-in, the download address on an export's
// page went through Google's sign-in addresses without a question and ended at
// the download host: the event that announces the download carried that last
// address. Fourteen minutes after a sign-in the same address ended on the
// password page, and no download began. Both 2026-10-02.
func (p *Page) Download(address, dir string, giveUp <-chan time.Time) (string, error) {
	began := make(chan string, 1)
	finished := make(chan struct{}, 1)
	chromedp.ListenBrowser(p.ctx, func(event any) {
		switch e := event.(type) {
		case *cdpbrowser.EventDownloadWillBegin:
			select {
			case began <- e.URL:
			default:
			}
		case *cdpbrowser.EventDownloadProgress:
			if e.State != cdpbrowser.DownloadProgressStateInProgress {
				select {
				case finished <- struct{}{}:
				default:
				}
			}
		}
	})
	err := chromedp.Run(p.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		allow := cdpbrowser.SetDownloadBehavior(cdpbrowser.SetDownloadBehaviorBehaviorAllow).WithDownloadPath(dir).WithEventsEnabled(true)
		if err := allow.Do(cdp.WithExecutor(ctx, chromedp.FromContext(ctx).Browser)); err != nil {
			return err
		}
		_, _, _, _, err := page.Navigate(address).Do(ctx)
		return err
	}))
	if err != nil {
		return "", err
	}
	select {
	case from := <-began:
		// The file is small and ours to throw away, but a browser closed in the
		// middle of it leaves a half-written one behind.
		select {
		case <-finished:
		case <-giveUp:
		case <-p.exited:
		}
		return from, nil
	case <-giveUp:
		return "", nil
	case <-p.exited:
		return "", ErrExited
	case <-p.gaveUp:
		return "", context.Canceled
	}
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
