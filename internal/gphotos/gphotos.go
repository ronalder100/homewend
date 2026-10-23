// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package gphotos reads what Google Photos' own web app says about an
// account: how many photos and videos it holds, year by year. Takeout cannot
// say it (its content picker names the years and nothing more, measured
// 2026-10-20 and 2026-10-21).
package gphotos

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
)

// URL is the page whose session the counting borrows.
const URL = "https://photos.google.com/"

// Driver is the page, open at URL.
type Driver interface {
	Eval(expression string, out any) error
}

// ErrPageChanged means photos.google.com is not the page this code was
// written against, or not signed in: its session data is missing, or the
// timeline answers in another shape.
var ErrPageChanged = errors.New("Google Photos' page has changed")

// Year is how much of the account was taken in one year.
type Year struct {
	Year   int `json:"year"`
	Items  int `json:"items"`  // photos and videos
	Videos int `json:"videos"` // of them
}

// count pages through the timeline the web app shows, as it asks for it: the
// lcxiM RPC, posted to batchexecute with the page's own tokens, 500 items a
// page (about 490 come back), until no next page is named. Measured
// 2026-10-21 on one account: 33,745 items, 72 pages, 66 to 82 s. The shape
// follows Google-Photos-Toolkit (github.com/xob0t/Google-Photos-Toolkit,
// src/api/api.ts and parser.ts), checked against the answers:
//
//	request: [pageToken, null, 500, null, 1, source]; source 3 is the
//	         library and the archive, what Takeout exports; trash is apart
//	answer:  [items, nextPageToken, lastTimestamp]
//	item:    [2] taken (ms, UTC), [4] the time zone's offset (ms); the last
//	         field an object, with key 76647426 for a video
//
// The year is the local one, where the photo was taken: the 32 items of 2025
// counted so matched, one for one, the 32 photos of the 2025 export.
const count = `(async () => {
  const g = window.WIZ_global_data
  if (!g || !g.SNlM0e || !g.FdrFJe || !g.cfb2h || !g.eptZe) return { Missing: true }
  const call = async (data) => {
    const body = 'f.req=' + encodeURIComponent(JSON.stringify([[['lcxiM', JSON.stringify(data), null, 'generic']]])) +
      '&at=' + encodeURIComponent(g.SNlM0e) + '&'
    const q = new URLSearchParams({ rpcids: 'lcxiM', 'source-path': location.pathname, 'f.sid': g.FdrFJe, bl: g.cfb2h, pageId: 'none', rt: 'c' })
    const r = await fetch(location.origin + g.eptZe + 'data/batchexecute?' + q, { method: 'POST', credentials: 'include',
      headers: { 'content-type': 'application/x-www-form-urlencoded;charset=UTF-8' }, body })
    const line = (await r.text()).split('\n').find(l => l.includes('"wrb.fr"'))
    if (!line) throw new Error('answer ' + r.status + ' without wrb.fr')
    return JSON.parse(JSON.parse(line)[0][2])
  }
  const years = {}
  let token = null, pages = 0
  const seen = new Set()
  do {
    const d = await call([token, null, 500, null, 1, 3])
    if (!Array.isArray(d)) throw new Error('page ' + pages + ' is not a list')
    for (const it of d[0] || []) {
      if (typeof it[2] !== 'number') throw new Error('an item with no time')
      const y = new Date(it[2] + (typeof it[4] === 'number' ? it[4] : 0)).getUTCFullYear()
      const last = it[it.length - 1]
      const v = years[y] || (years[y] = [0, 0])
      v[0]++
      if (last && typeof last === 'object' && !Array.isArray(last) && last[76647426]) v[1]++
    }
    token = d[1] || null
    if (token && seen.has(token)) throw new Error('page ' + pages + ' names itself again')
    if (token) seen.add(token)
    pages++
  } while (token)
  return { Years: years, Pages: pages }
})()`

type answer struct {
	Missing bool
	Years   map[string][2]int
	Pages   int
}

// CountYears counts the account's photos and videos by the year they were
// taken, newest year first.
func CountYears(d Driver) ([]Year, error) {
	var a answer
	if err := d.Eval(count, &a); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrPageChanged, err)
	}
	return years(a)
}

func years(a answer) ([]Year, error) {
	if a.Missing {
		return nil, fmt.Errorf("%w: no session data on the page", ErrPageChanged)
	}
	out := []Year{}
	for k, v := range a.Years {
		y, err := strconv.Atoi(k)
		if err != nil {
			return nil, fmt.Errorf("%w: year %q", ErrPageChanged, k)
		}
		out = append(out, Year{Year: y, Items: v[0], Videos: v[1]})
	}
	sort.Slice(out, func(i, k int) bool { return out[i].Year > out[k].Year })
	return out, nil
}
