// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
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
	case FromSidecar, FromVideo:
		// A video sent through a chat is made again on the way: its header
		// says when it was sent, not when it was filmed.
		if c.Origin.FromMessaging() || c.Origin == OriginScreenshot {
			return DateReceived
		}
		return DateTaken
	case FromEXIF, FromGiven:
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

// catalogDSN is how the index at path is opened, with options after the
// path's "?".
func catalogDSN(path, options string) string {
	dsn := "file:" + path + "?" + options
	// On macOS SQLite picks its locking by filesystem, and on an SMB share it
	// picks AFP-style locks, which the Go SQLite cannot do: it panics on
	// fsctl. flock locks work there and on local disks alike, and go with the
	// process if it dies. Measured on a NAS share, 2026-09-27.
	if runtime.GOOS == "darwin" {
		dsn += "&vfs=unix-flock"
	}
	return dsn
}

// OpenCatalog opens, and creates if needed, the index at path.
func OpenCatalog(path string) (*Catalog, error) {
	db, err := sql.Open("sqlite", catalogDSN(path, "_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"))
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
	if err := clockDates(db, path); err != nil {
		db.Close()
		return nil, fmt.Errorf("turning the catalog's dates into clocks: %w", err)
	}
	return &Catalog{db: db}, nil
}

// clocksSince is the catalog's user_version from which every date in it is
// a clock. Before, Google's dates were kept as instants in UTC while the
// camera's were clocks, and the grid, which reads a clock, showed a photo
// from Takeout one or two hours off, near midnight on the wrong day.
const clocksSince = 1

// clockDates turns the instants an older catalog kept for Google's dates
// into the clocks of this computer's zone, the one the photos were filed
// by, once: a copy of the catalog as it was is kept beside it first.
//
// The window opens the catalog for several requests at once, and a command
// in a terminal may too: the turn is made under the write lock, taken at
// the start (BEGIN IMMEDIATE), and the version read again under it, so one
// of them turns the dates and the others find it done. A deferred
// transaction would read, then fail to write on a snapshot grown old.
func clockDates(db *sql.DB, path string) error {
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil || version >= clocksSince {
		return err
	}
	ctx := context.Background()
	backup, err := keepBefore(db, path)
	if err != nil {
		return err
	}
	if backup != "" {
		defer os.Remove(backup) // left only when it was not needed
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return err
	}
	done := false
	defer func() {
		if !done {
			conn.ExecContext(ctx, `ROLLBACK`)
		}
	}()
	if err := conn.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil || version >= clocksSince {
		return err
	}
	// The copy taken before the lock is of the catalog as it still is: no
	// one turned the dates since, or the version would say so. One kept by
	// an earlier run that stopped halfway is as good, and stays.
	if backup != "" && !exists(path+".before-clocks") {
		if err := os.Rename(backup, path+".before-clocks"); err != nil {
			return err
		}
	}
	// One statement, run inside SQLite: a library of a hundred thousand
	// photos turned row by row from here held the lock for seconds, longer
	// than the others wait for it. The zone's offset changes a few times a
	// year; each span between two changes is one branch of the CASE.
	var first, last sql.NullInt64
	if err := conn.QueryRowContext(ctx, `SELECT MIN(taken), MAX(taken) FROM photos WHERE source = ? AND taken IS NOT NULL`, string(FromSidecar)).Scan(&first, &last); err != nil {
		return err
	}
	if first.Valid {
		if _, err := conn.ExecContext(ctx, `UPDATE photos SET taken = taken + `+offsets(first.Int64, last.Int64, time.Local)+` WHERE source = ? AND taken IS NOT NULL`, string(FromSidecar)); err != nil {
			return err
		}
	}
	if _, err := conn.ExecContext(ctx, fmt.Sprintf(`PRAGMA user_version = %d`, clocksSince)); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, `COMMIT`); err != nil {
		return err
	}
	done = true
	return nil
}

// offsets is an SQL expression of taken, an instant in Unix seconds between
// first and last: the seconds that turn it into the clock of zone, as
// asClock does. Only numbers made here go in it.
func offsets(first, last int64, zone *time.Location) string {
	var b strings.Builder
	at := time.Unix(first, 0).In(zone)
	for {
		_, offset := at.Zone()
		_, end := at.ZoneBounds()
		// Past the zone's last written rule (2037) Go may end a span where
		// it began: what lies beyond takes the offset found there.
		if end.IsZero() || !end.After(at) || end.Unix() > last {
			// One offset over them all, as in UTC: no CASE to choose.
			if b.Len() == 0 {
				return strconv.Itoa(offset)
			}
			fmt.Fprintf(&b, " ELSE %d END", offset)
			return b.String()
		}
		if b.Len() == 0 {
			b.WriteString("CASE")
		}
		fmt.Fprintf(&b, " WHEN taken < %d THEN %d", end.Unix(), offset)
		at = end
	}
}

// keepBefore copies the catalog, when it has dates to turn and no copy is
// kept yet, into a file of its own beside it: "" when none is needed. Each
// opening writes its own, so two never write the same file, and a copy cut
// short (a crash, a full disk) is never the one kept.
func keepBefore(db *sql.DB, path string) (string, error) {
	if exists(path + ".before-clocks") {
		return "", nil
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM photos WHERE source = ? AND taken IS NOT NULL`, string(FromSidecar)).Scan(&n); err != nil || n == 0 {
		return "", err
	}
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".before-clocks-*")
	if err != nil {
		return "", err
	}
	f.Close()
	// VACUUM INTO writes into an empty file, and reads one snapshot of the
	// catalog: a copy as it was at one moment.
	if _, err := db.Exec(`VACUUM INTO ?`, f.Name()); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// OpenCatalogReadOnly opens the index at path to look at it: no schema, no
// migration, nothing written. A look before an import changes nothing in the
// library, and a library on a read-only share can still be looked at.
//
// Read-only alone would not do it: in WAL mode SQLite creates the -shm and
// -wal files beside the catalog, and fails where it cannot (measured 09/11).
// So a catalog with no -wal beside it is opened immutable. One with a -wal
// holds rows not yet in the catalog file — a download running, or one
// stopped by closing its terminal — that immutable would not see: it is
// opened read-only, and the files it needs are already there. A -wal
// without its -shm (a library copied without it) would need one created:
// the catalog file is read alone, and what only the -wal holds is missed.
func OpenCatalogReadOnly(path string) (*Catalog, error) {
	options := "mode=ro&immutable=1"
	if exists(path+"-wal") && exists(path+"-shm") {
		options = "mode=ro&_pragma=busy_timeout(5000)"
	}
	db, err := sql.Open("sqlite", catalogDSN(path, options))
	if err != nil {
		return nil, fmt.Errorf("opening the catalog: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("opening the catalog: %w", err)
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
func (p Photo) capture() Capture {
	return Capture{When: p.Taken, Source: p.Source, Origin: p.Origin, clock: true}
}

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

// forget drops a photo whose file is gone from the library, so that a copy
// brought in again is placed and counted as new.
func (c *Catalog) forget(hash string) error {
	_, err := c.db.Exec(`DELETE FROM photos WHERE hash = ?`, hash)
	return err
}

// setDate records a date given to a photo already in the library.
func (c *Catalog) setDate(hash string, capture Capture) error {
	_, err := c.db.Exec(`UPDATE photos SET taken = ?, source = ?, kind = ? WHERE hash = ?`,
		capture.When.Unix(), string(capture.Source), string(capture.Kind()), hash)
	return err
}

// setPath records that a photo is now at path, under name.
func (c *Catalog) setPath(hash, path, name string) error {
	_, err := c.db.Exec(`UPDATE photos SET path = ?, name = ? WHERE hash = ?`, path, name, hash)
	return err
}
