// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package session holds the Google session: a browser profile we create, and
// the cookies we read back out of it.
//
// Google refuses sign-in inside an embedded browser — measured with Electron on
// 2026-09-23, "this browser or app may not be secure" — so the browser is the
// machine's own. What is ours is the profile directory, and that is the whole
// trick: whoever owns the profile can re-read the cookies whenever, which means
// following Google's rotation instead of being outrun by it.
//
// A copied Cookie header was measured dead in under twenty minutes. A profile
// re-read before every request was still serving bytes after an hour.
package session

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/sha1"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	// The pure-Go SQLite driver: no cgo, so the app still cross-compiles to the
	// three desktops from one machine, which is half the reason for choosing Go.
	_ "modernc.org/sqlite"
)

// A real Chrome user-agent. The engine underneath really is Chromium, so this
// is not a disguise: it keeps the user-agent and the client hints telling the
// same story.
const UserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36"

// Session is a browser profile plus the means to use its cookies.
type Session struct {
	Profile string

	client *http.Client
	key    []byte
}

// New prepares a session over profileDir, creating it if needed. No browser is
// needed to use a session, only to sign in: see Open.
func New(profileDir string) (*Session, error) {
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		return nil, fmt.Errorf("creating the profile directory: %w", err)
	}
	return &Session{
		Profile: profileDir,
		key:     derivedKey(),
		client: &http.Client{
			// No limit on the whole request: it ended a slow 2 GB transfer
			// every 30 minutes (measured 2026-09-28). A server that never
			// answers is caught here, a transfer that goes quiet by the
			// downloader.
			Transport: transport(),
			// Redirects are never followed: on the download host a redirect
			// means the session is no longer good, and following it would turn
			// a clear answer into a sign-in page pretending to be an archive.
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func transport() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.ResponseHeaderTimeout = 2 * time.Minute
	// Go's default is 10 seconds. On a bad evening handshakes with Google
	// took 5 to 7 seconds and the slower ones timed out, over and over,
	// on a line that was otherwise working (measured 2026-09-29).
	t.TLSHandshakeTimeout = 30 * time.Second
	return t
}

// Chrome encrypts each cookie value with AES-128-CBC, under a key derived from
// a password it normally keeps in the system keyring. The browser is launched
// so that the password is a fixed one instead (see Open): otherwise the key
// differs on every machine, and on a Mac reading it means a Keychain prompt.
//
// On macOS, --use-mock-keychain makes Chrome use its fake keychain, whose
// password is "mock_password", derived with 1003 iterations: Chromium's
// crypto/apple/fake_keychain_v2.mm and
// components/os_crypt/async/browser/keychain_key_provider.mm, read 2026-09-26.
func derivedKey() []byte { return keyFor(runtime.GOOS) }

func keyFor(system string) []byte {
	password, iterations := "peanuts", 1
	if system == "darwin" {
		password, iterations = "mock_password", 1003
	}
	// The standard library's PBKDF2. It fails only for a key length out of
	// range, which 16 is not.
	key, err := pbkdf2.Key(sha1.New, password, []byte("saltysalt"), iterations, 16)
	if err != nil {
		panic(err)
	}
	return key
}

func (s *Session) decrypt(raw []byte) string {
	if len(raw) < 3 {
		return ""
	}
	switch string(raw[:3]) {
	case "v10", "v11":
	default:
		return string(raw) // stored in the clear
	}
	body := raw[3:]
	if len(body)%aes.BlockSize != 0 || len(body) == 0 {
		return ""
	}
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return ""
	}
	out := make([]byte, len(body))
	cipher.NewCBCDecrypter(block, []byte("                ")).CryptBlocks(out, body)

	if pad := int(out[len(out)-1]); pad > 0 && pad <= aes.BlockSize && pad <= len(out) {
		out = out[:len(out)-pad]
	}
	// Chrome 130+ puts 32 bytes of domain hash in front of the value. Nothing
	// announces it; it shows up as unprintable bytes, and a cookie value never
	// is.
	if len(out) > 32 && !printable(out[:32]) {
		out = out[32:]
	}
	return string(out)
}

func printable(b []byte) bool {
	for _, c := range b {
		if c < 0x20 || c > 0x7e {
			return false
		}
	}
	return true
}

// Cookies reads the profile's Google cookies, fresh, every time it is called.
// That is the point of this package, not an inefficiency.
func (s *Session) Cookies() (header string, names map[string]bool, err error) {
	var pairs []string
	names = map[string]bool{}
	err = s.query("Cookies",
		`SELECT name, value, encrypted_value FROM cookies WHERE host_key LIKE '%google.com'`,
		func(rows *sql.Rows) error {
			var name, plain string
			var encrypted []byte
			if err := rows.Scan(&name, &plain, &encrypted); err != nil {
				return err
			}
			value := plain
			if value == "" {
				value = s.decrypt(encrypted)
			}
			if value != "" {
				pairs = append(pairs, name+"="+value)
				names[name] = true
			}
			return nil
		})
	if err != nil {
		return "", nil, err
	}
	return strings.Join(pairs, "; "), names, nil
}

// DownloadURLs lists the addresses the browser's downloads finally came from,
// newest first: the last link of each redirect chain, as Chrome records it in
// the profile's History (table downloads_url_chains). A profile with no
// history yet has none.
func (s *Session) DownloadURLs() ([]string, error) {
	var urls []string
	err := s.query("History",
		`SELECT url FROM downloads_url_chains c
		 WHERE chain_index = (SELECT MAX(chain_index) FROM downloads_url_chains WHERE id = c.id)
		 ORDER BY id DESC`,
		func(rows *sql.Rows) error {
			var u string
			if err := rows.Scan(&u); err != nil {
				return err
			}
			urls = append(urls, u)
			return nil
		})
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return urls, err
}

// query runs a read-only query on one of the profile's SQLite files. The
// browser keeps them locked while it runs, so it works on a copy.
func (s *Session) query(file, query string, row func(*sql.Rows) error) error {
	source := filepath.Join(s.Profile, "Default", file)
	if _, err := os.Stat(source); err != nil {
		return fmt.Errorf("no %s in the profile: %w", file, err)
	}
	dir, err := os.MkdirTemp("", "homewend-profile-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	copyPath := filepath.Join(dir, file)
	if err := copyFile(source, copyPath); err != nil {
		return err
	}
	// A file in WAL mode keeps its newest rows in the -wal beside it until
	// Chrome checkpoints: on Chromium 152 on macOS, a finished download was in
	// History-wal and not in History (measured 2026-09-27). The copy is ours,
	// so SQLite is free to replay it into the copy.
	if err := copyFile(source+"-wal", copyPath+"-wal"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	db, err := sql.Open("sqlite", "file:"+copyPath)
	if err != nil {
		return fmt.Errorf("opening %s: %w", file, err)
	}
	defer db.Close()

	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("reading %s: %w", file, err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := row(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

func copyFile(source, dest string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// Get performs a request carrying the profile's current cookies and the headers
// of a real navigation.
//
// Those headers are not decoration: the download endpoint refuses anything
// whose Sec-Fetch-Dest is not "document", which is why a fetch() from inside
// the page bounces and a navigation does not. Measured 2026-09-23.
func (s *Session) Get(url string, extra map[string]string) (*http.Response, error) {
	cookies, _, err := s.Cookies()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", cookies)
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Referer", "https://takeout.google.com/")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	for key, value := range extra {
		req.Header.Set(key, value)
	}
	return s.client.Do(req)
}
