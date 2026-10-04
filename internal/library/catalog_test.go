// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package library

import (
	"path/filepath"
	"testing"
	"time"
)

func TestCountsByYear(t *testing.T) {
	c, err := OpenCatalog(filepath.Join(t.TempDir(), "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	put := func(hash string, taken time.Time) {
		if err := c.Put(Photo{Hash: hash, Path: hash, Name: hash, Taken: taken}); err != nil {
			t.Fatal(err)
		}
	}
	put("a", time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC))
	put("b", time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC))
	put("c", time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC))
	put("d", time.Time{})

	got, err := c.CountsByYear(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got["2024"] != 2 || got["2019"] != 1 || got[""] != 1 || len(got) != 3 {
		t.Fatalf("got %v", got)
	}
}

func TestOwnersFilterThePhotos(t *testing.T) {
	c, err := OpenCatalog(filepath.Join(t.TempDir(), "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	y := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)
	for _, p := range []struct{ hash, owner string }{{"a", "ann@x"}, {"b", "bo@x"}, {"c", ""}, {"d", "ann@x"}} {
		c.Put(Photo{Hash: p.hash, Path: p.hash, Name: p.hash, Taken: y})
		c.addOwner(p.hash, p.owner)
	}
	c.addOwner("d", "bo@x") // in both exports
	c.addAlbum("Trip", "a", "ann@x")
	c.addAlbum("Kids", "b", "bo@x")

	count := func(accounts []string) int {
		got, err := c.CountsByYear(accounts)
		if err != nil {
			t.Fatal(err)
		}
		return got["2024"]
	}
	if count(nil) != 4 || count([]string{"ann@x"}) != 3 || count([]string{"bo@x"}) != 3 || count([]string{}) != 1 {
		t.Fatalf("all %d, ann %d, bo %d, nobody %d", count(nil), count([]string{"ann@x"}), count([]string{"bo@x"}), count([]string{}))
	}
	page, err := c.Page(Filter{Accounts: []string{"bo@x"}})
	if err != nil || len(page) != 3 {
		t.Fatalf("bo's page: %d, %v", len(page), err)
	}
	albums, err := c.AlbumsByOwner()
	if err != nil || len(albums) != 2 || albums[0] != (AlbumCount{"bo@x", "Kids", 1}) || albums[1] != (AlbumCount{"ann@x", "Trip", 1}) {
		t.Fatalf("albums %+v, %v", albums, err)
	}
}

func TestLatestAndAbout(t *testing.T) {
	c, err := OpenCatalog(filepath.Join(t.TempDir(), "catalog.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	for _, h := range []string{"a", "b", "c"} {
		c.Put(Photo{Hash: h, Path: h, Name: h})
	}
	c.addAlbum("Trip", "b", "ann@x")
	c.addOwner("b", "ann@x")
	latest, err := c.Latest(2)
	if err != nil || len(latest) != 2 || latest[0].Hash != "c" || latest[1].Hash != "b" {
		t.Fatalf("latest %+v, %v", latest, err)
	}
	albums, accounts, err := c.About("b")
	if err != nil || len(albums) != 1 || albums[0] != "Trip" || len(accounts) != 1 || accounts[0] != "ann@x" {
		t.Fatalf("about: %v %v %v", albums, accounts, err)
	}
}
