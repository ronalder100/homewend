// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// DateKind is the difference between "when this was taken" and "when this
// reached you", and the reason it exists is that for a quarter of a real
// library nobody knows the first.
//
// A photo that arrived through WhatsApp has no EXIF — the app strips it when it
// recompresses — so the date Google records is when the file appeared on the
// phone. Every other tool calls that a capture date. It is not, and a library
// that says "taken 14 July 2019" about a picture someone forwarded is telling
// the user something it does not know.
type DateKind string

const (
	DateTaken    DateKind = "taken"    // a camera or a sidecar said so
	DateReceived DateKind = "received" // it arrived through a chat; nobody knows when it was shot
	DateGuessed  DateKind = "guessed"  // only the year, from Google's own folder
	DateUnknown  DateKind = "unknown"
)

// Kind says what the date on this capture really is.
func (c Capture) Kind() DateKind {
	switch c.Source {
	case FromSidecar:
		if c.Origin.FromMessaging() || c.Origin == OriginScreenshot {
			return DateReceived
		}
		return DateTaken
	case FromEXIF:
		return DateTaken
	case FromFolder:
		return DateGuessed
	default:
		return DateUnknown
	}
}

// Photo is one picture as the library knows it.
type Photo struct {
	Hash   string
	Path   string // relative to the library root, so the library can be moved
	Name   string
	Bytes  int64
	Taken  time.Time
	Source DateSource
	Kind   DateKind
	Origin Origin
	Albums []string
}

// Catalog is the index the interface reads: what there is, where it is, and
// what can be said about it.
//
// SQLite rather than a file of JSON because the grid has to scroll through tens
// of thousands of thumbnails and filter them without loading the lot, and
// because the driver is already here for reading Chrome's cookies. No cgo, so
// the app still cross-compiles to the three desktops from one machine.
type Catalog struct{ db *sql.DB }

const schema = `
CREATE TABLE IF NOT EXISTS photos (
  hash    TEXT PRIMARY KEY,
  path    TEXT NOT NULL,
  name    TEXT NOT NULL,
  bytes   INTEGER NOT NULL,
  taken   INTEGER,          -- unix seconds, NULL when nothing is known
  source  TEXT NOT NULL,
  kind    TEXT NOT NULL,
  origin  TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS album_members (
  album TEXT NOT NULL,
  hash  TEXT NOT NULL REFERENCES photos(hash),
  PRIMARY KEY (album, hash)
);
-- Where a photo came from: its sources, "<service>/<address>" for an
-- account ("google/ron@example.com"), "local/<folder>" for an import. A photo
-- that came from two is one photo with two; a photo recorded before sources
-- were kept has none, and is shown with every account.
CREATE TABLE IF NOT EXISTS owners (
  account TEXT NOT NULL,
  hash    TEXT NOT NULL,
  PRIMARY KEY (account, hash)
);
CREATE TABLE IF NOT EXISTS album_owners (
  account TEXT NOT NULL,
  album   TEXT NOT NULL,
  PRIMARY KEY (account, album)
);
-- The grid is always "newest first, maybe filtered": one index carries it.
CREATE INDEX IF NOT EXISTS photos_by_time   ON photos(taken DESC);
CREATE INDEX IF NOT EXISTS photos_by_origin ON photos(origin, taken DESC);
CREATE INDEX IF NOT EXISTS photos_by_path   ON photos(path);
`

// OpenCatalog opens, and creates if needed, the index at path.
func OpenCatalog(path string) (*Catalog, error) {
	dsn := "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	// On macOS SQLite picks its locking by filesystem, and on an SMB share it
	// picks AFP-style locks, which the Go SQLite cannot do: it panics on
	// fsctl. flock locks work there and on local disks alike, and go with the
	// process if it dies. Measured on a NAS share, 2026-09-27.
	if runtime.GOOS == "darwin" {
		dsn += "&vfs=unix-flock"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening the catalog: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("preparing the catalog: %w", err)
	}
	// A catalog from before sources named their service kept bare Google
	// addresses.
	if _, err := db.Exec(`UPDATE OR IGNORE owners SET account = 'google/' || account WHERE instr(account, '/') = 0;
UPDATE OR IGNORE album_owners SET account = 'google/' || account WHERE instr(account, '/') = 0;`); err != nil {
		db.Close()
		return nil, fmt.Errorf("naming the catalog's sources: %w", err)
	}
	return &Catalog{db: db}, nil
}

func (c *Catalog) Close() error { return c.db.Close() }

// Put records a photo, replacing whatever was known about it before. Safe to
// run again over a library that was already indexed: a second pass corrects,
// it does not duplicate.
func (c *Catalog) Put(photo Photo) error {
	var taken any
	if !photo.Taken.IsZero() {
		taken = photo.Taken.Unix()
	}
	_, err := c.db.Exec(
		`INSERT INTO photos (hash, path, name, bytes, taken, source, kind, origin)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(hash) DO UPDATE SET
		   path=excluded.path, name=excluded.name, bytes=excluded.bytes,
		   taken=excluded.taken, source=excluded.source, kind=excluded.kind,
		   origin=excluded.origin`,
		photo.Hash, photo.Path, photo.Name, photo.Bytes, taken,
		string(photo.Source), string(photo.Kind), string(photo.Origin))
	if err != nil {
		return fmt.Errorf("recording %s: %w", photo.Name, err)
	}
	for _, album := range photo.Albums {
		if _, err := c.db.Exec(
			`INSERT OR IGNORE INTO album_members (album, hash) VALUES (?, ?)`,
			album, photo.Hash); err != nil {
			return fmt.Errorf("adding %s to %s: %w", photo.Name, album, err)
		}
	}
	return nil
}

// PathOf returns where the photo with this hash lives, relative to the library
// root, and whether the catalog knows it at all.
func (c *Catalog) PathOf(hash string) (string, bool, error) {
	var path string
	err := c.db.QueryRow(`SELECT path FROM photos WHERE hash = ?`, hash).Scan(&path)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("reading the catalog: %w", err)
	}
	return path, true, nil
}

// PathTaken reports whether some photo is recorded at this relative path.
func (c *Catalog) PathTaken(path string) (bool, error) {
	var n int
	if err := c.db.QueryRow(`SELECT COUNT(*) FROM photos WHERE path = ?`, path).Scan(&n); err != nil {
		return false, fmt.Errorf("reading the catalog: %w", err)
	}
	return n > 0, nil
}

// addAlbum records that a photo belongs to an album, and the album to the
// account it came from. Separate from Put because a photo is placed once and
// may join several albums afterwards.
func (c *Catalog) addAlbum(album, hash, account string) error {
	_, err := c.db.Exec(
		`INSERT OR IGNORE INTO album_members (album, hash) VALUES (?, ?)`, album, hash)
	if err == nil && account != "" {
		_, err = c.db.Exec(
			`INSERT OR IGNORE INTO album_owners (account, album) VALUES (?, ?)`, account, album)
	}
	if err != nil {
		return fmt.Errorf("adding a photo to %s: %w", album, err)
	}
	return nil
}

// addOwner records that a photo came from an account. A photo already in the
// library gets one more owner when another account's export holds it too.
func (c *Catalog) addOwner(hash, account string) error {
	if account == "" {
		return nil
	}
	_, err := c.db.Exec(`INSERT OR IGNORE INTO owners (account, hash) VALUES (?, ?)`, account, hash)
	return err
}

// ownedBy is the SQL that keeps the photos of these accounts, and those of
// nobody's (recorded before owners were kept); nil accounts keeps them all.
func ownedBy(accounts []string) (string, []any) {
	if accounts == nil {
		return "", nil
	}
	clause := "(hash NOT IN (SELECT hash FROM owners)"
	var args []any
	if len(accounts) > 0 {
		clause += " OR hash IN (SELECT hash FROM owners WHERE account IN (?" + strings.Repeat(",?", len(accounts)-1) + "))"
		for _, a := range accounts {
			args = append(args, a)
		}
	}
	return clause + ")", args
}

// Filter is what the interface asks for: a page of the grid, narrowed.
type Filter struct {
	Origins []Origin // empty means all
	Album   string
	Year    string
	NoDate  bool // only the photos whose date nobody knows
	// Accounts keeps the photos of these accounts, by address; nil is all.
	Accounts []string
	Limit    int
	Offset   int
}

// Page returns photos newest first, narrowed by the filter.
func (c *Catalog) Page(f Filter) ([]Photo, error) {
	query := `SELECT hash, path, name, bytes, taken, source, kind, origin FROM photos`
	var where []string
	var args []any

	if len(f.Origins) > 0 {
		placeholders := ""
		for i, origin := range f.Origins {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
			args = append(args, string(origin))
		}
		where = append(where, "origin IN ("+placeholders+")")
	}
	if f.Album != "" {
		where = append(where,
			"hash IN (SELECT hash FROM album_members WHERE album = ?)")
		args = append(args, f.Album)
	}
	if f.Year != "" {
		// Compared as text on purpose: an unknown date must not silently match
		// a year, and strftime on NULL is NULL, which no year equals.
		where = append(where, "strftime('%Y', taken, 'unixepoch') = ?")
		args = append(args, f.Year)
	}
	if f.NoDate {
		where = append(where, "taken IS NULL")
	}
	if clause, more := ownedBy(f.Accounts); clause != "" {
		where = append(where, clause)
		args = append(args, more...)
	}
	for i, clause := range where {
		if i == 0 {
			query += " WHERE "
		} else {
			query += " AND "
		}
		query += clause
	}
	query += " ORDER BY taken DESC, name ASC"
	if f.Limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, f.Limit, f.Offset)
	}

	rows, err := c.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("reading the catalog: %w", err)
	}
	return scanPhotos(rows)
}

// photo is the record of one photo.
func (c *Catalog) photo(hash string) (Photo, error) {
	rows, err := c.db.Query(`SELECT hash, path, name, bytes, taken, source, kind, origin FROM photos WHERE hash = ?`, hash)
	if err != nil {
		return Photo{}, fmt.Errorf("reading the catalog: %w", err)
	}
	photos, err := scanPhotos(rows)
	if err != nil || len(photos) == 0 {
		return Photo{}, fmt.Errorf("reading the catalog: no photo %s: %v", hash, err)
	}
	return photos[0], nil
}

// capture is what the record knows of when the photo was taken.
func (p Photo) capture() Capture { return Capture{When: p.Taken, Source: p.Source, Origin: p.Origin} }

// scanPhotos reads the rows of a SELECT of the photos' columns, in order.
func scanPhotos(rows *sql.Rows) ([]Photo, error) {
	defer rows.Close()
	var photos []Photo
	for rows.Next() {
		var photo Photo
		var taken sql.NullInt64
		var source, kind, origin string
		if err := rows.Scan(&photo.Hash, &photo.Path, &photo.Name, &photo.Bytes,
			&taken, &source, &kind, &origin); err != nil {
			return nil, err
		}
		if taken.Valid {
			photo.Taken = time.Unix(taken.Int64, 0).UTC()
		}
		photo.Source, photo.Kind, photo.Origin = DateSource(source), DateKind(kind), Origin(origin)
		photos = append(photos, photo)
	}
	return photos, rows.Err()
}

// CountsByOrigin is what the filter chips show: how many of each, so nobody
// clicks "WhatsApp" to find out it is empty.
func (c *Catalog) CountsByOrigin() (map[Origin]int, error) {
	rows, err := c.db.Query(`SELECT origin, COUNT(*) FROM photos GROUP BY origin`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[Origin]int{}
	for rows.Next() {
		var origin string
		var n int
		if err := rows.Scan(&origin, &n); err != nil {
			return nil, err
		}
		counts[Origin(origin)] = n
	}
	return counts, rows.Err()
}

// AlbumNames lists the albums, with how many photos each holds.
func (c *Catalog) AlbumNames() (map[string]int, error) {
	rows, err := c.db.Query(
		`SELECT album, COUNT(*) FROM album_members GROUP BY album ORDER BY album`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var album string
		var n int
		if err := rows.Scan(&album, &n); err != nil {
			return nil, err
		}
		counts[album] = n
	}
	return counts, rows.Err()
}

// CountsByYear is how many photos each year holds, and under "" those whose
// date nobody knows.
func (c *Catalog) CountsByYear(accounts []string) (map[string]int, error) {
	query := `SELECT COALESCE(strftime('%Y', taken, 'unixepoch'), ''), COUNT(*) FROM photos`
	clause, args := ownedBy(accounts)
	if clause != "" {
		query += " WHERE " + clause
	}
	rows, err := c.db.Query(query+" GROUP BY 1", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var year string
		var n int
		if err := rows.Scan(&year, &n); err != nil {
			return nil, err
		}
		counts[year] = n
	}
	return counts, rows.Err()
}

// CountsByOwner is how many photos each account brought, by address.
func (c *Catalog) CountsByOwner() (map[string]int, error) {
	rows, err := c.db.Query(`SELECT account, COUNT(*) FROM owners GROUP BY account`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var a string
		var n int
		if err := rows.Scan(&a, &n); err != nil {
			return nil, err
		}
		counts[a] = n
	}
	return counts, rows.Err()
}

// Latest are the photos recorded last, newest first: the first to arrive in
// a download are the last of these once it is over, the latest while it runs.
func (c *Catalog) Latest(n int) ([]Photo, error) {
	rows, err := c.db.Query(`SELECT hash, name FROM photos ORDER BY rowid DESC LIMIT ?`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Photo
	for rows.Next() {
		var p Photo
		if err := rows.Scan(&p.Hash, &p.Name); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// About says which albums a photo is in and which accounts it came from.
func (c *Catalog) About(hash string) (albums, accounts []string, err error) {
	read := func(query string) ([]string, error) {
		rows, err := c.db.Query(query, hash)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := []string{}
		for rows.Next() {
			var v string
			if err := rows.Scan(&v); err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, rows.Err()
	}
	if albums, err = read(`SELECT album FROM album_members WHERE hash = ? ORDER BY album`); err != nil {
		return nil, nil, err
	}
	accounts, err = read(`SELECT account FROM owners WHERE hash = ? ORDER BY account`)
	return albums, accounts, err
}

// AlbumCount is an album and the account it came from ("" when not known).
type AlbumCount struct {
	Account string
	Name    string
	Count   int
}

// AlbumsByOwner lists the albums, each under every account that brought it.
func (c *Catalog) AlbumsByOwner() ([]AlbumCount, error) {
	rows, err := c.db.Query(`
		SELECT COALESCE(o.account, ''), m.album, COUNT(*)
		FROM album_members m LEFT JOIN album_owners o ON o.album = m.album
		GROUP BY 1, 2 ORDER BY 2, 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AlbumCount
	for rows.Next() {
		var a AlbumCount
		if err := rows.Scan(&a.Account, &a.Name, &a.Count); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// relativeTo makes a library path portable, so moving the library to another
// disk does not invalidate the index.
func relativeTo(root, path string) string {
	if rel, err := filepath.Rel(root, path); err == nil {
		return filepath.ToSlash(rel)
	}
	return path
}
