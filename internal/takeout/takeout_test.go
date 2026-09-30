// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package takeout

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// record builds an export record the way /manage carries it: the fields this
// package reads in their positions, null or filler everywhere else.
func record(job string, created int64, parts, manifest string) string {
	fields := make([]string, 32)
	for i := range fields {
		fields[i] = "null"
	}
	fields[0] = `"ac.t.ta"`
	fields[fieldJob] = `"` + job + `"`
	fields[2] = `"January 2, 2026"`
	fields[fieldBytes] = "4387791"
	fields[7] = `[["photos","Google Photos",null,4314195]]`
	fields[fieldParts] = parts
	fields[fieldCreated] = fmt.Sprint(created)
	fields[fieldManifest] = manifest
	return "[" + strings.Join(fields, ",") + "]"
}

const (
	oldJob = "11111111-1111-1111-1111-111111111111"
	newJob = "22222222-2222-2222-2222-222222222222"
)

func TestExportsAreReadFromTheirRecords(t *testing.T) {
	live := record(newJob, 1767352183000,
		`[["takeout-20260102T110943Z-1-001.zip",2147483648,0,null,null,5],["IMG_0001.MOV",3000000000,1,null,null,5]]`,
		`["takeout-20260102T110943Z-001.zip",55620,0,null,null,5,null,"",1]`)
	expired := record(oldJob, 1766752183000, "null", "null")
	page := `<script>AF_initDataCallback({key: 'ds:0', data:[[[null,` + expired + `]]]});` +
		// A UUID that is not an export, as the real page carries one.
		`"33333333-3333-3333-3333-333333333333"` +
		`AF_initDataCallback({key: 'ds:1', data:[[[null,` + live + `],[null,` + expired + `]]]});</script>`

	exports, err := parseExports(page)
	if err != nil {
		t.Fatal(err)
	}
	if len(exports) != 2 || exports[0].Job != newJob || exports[1].Job != oldJob {
		t.Fatalf("got %+v; want the two exports, newest first", exports)
	}

	e := exports[0]
	if !e.Ready() || exports[1].Ready() {
		t.Errorf("ready: live %v, expired %v; want true, false", e.Ready(), exports[1].Ready())
	}
	if e.Bytes != 4387791 || e.Created.UnixMilli() != 1767352183000 {
		t.Errorf("bytes %d, created %v", e.Bytes, e.Created)
	}
	want := []Part{
		{Index: 0, Filename: "takeout-20260102T110943Z-1-001.zip", Size: 2147483648, Downloads: 0},
		{Index: 1, Filename: "IMG_0001.MOV", Size: 3000000000, Downloads: 1},
	}
	if fmt.Sprint(e.Parts) != fmt.Sprint(want) {
		t.Errorf("parts %+v; want %+v", e.Parts, want)
	}
	if m := e.Manifest; m.Index != 2 || m.Filename != "takeout-20260102T110943Z-001.zip" || m.Size != 55620 {
		t.Errorf("manifest %+v; want index 2, its name and size", m)
	}
}

func TestARecordThatCannotBeReadIsAnError(t *testing.T) {
	bad := []string{
		`["ac.t.ta","` + newJob + `"]`,
		record(newJob, 1, `[["../escape.zip",1,0]]`, "null"),
		record(newJob, 1, `[["a.zip","not a size",0]]`, "null"),
	}
	for _, page := range bad {
		if _, err := parseExports(page); err == nil {
			t.Errorf("no error for %.80s", page)
		}
	}
}

func TestUserFromDownload(t *testing.T) {
	cases := map[string]string{
		"https://takeout-download.usercontent.google.com/download/a.zip?j=x&i=0&user=123456&authuser=0": "123456",
		"https://takeout.google.com/takeout/download?j=x&i=0&user=98765432109876543210":                 "",
		"https://example.com/file.zip?user=1":                                                           "",
		"not a url %%":                                                                                  "",
	}
	for address, want := range cases {
		if got := UserFromDownload(address); got != want {
			t.Errorf("%s: got %q, want %q", address, got, want)
		}
	}
}

// answer is a Getter that gets the same response, whatever it asks for.
type answer struct {
	status   int
	location string
}

func (a answer) Get(string, map[string]string) (*http.Response, error) {
	header := http.Header{}
	if a.location != "" {
		header.Set("Location", a.location)
	}
	return &http.Response{StatusCode: a.status, Header: header, Body: io.NopCloser(strings.NewReader(""))}, nil
}

func TestSignedInAsksGoogle(t *testing.T) {
	cases := []struct {
		answer answer
		ok     bool
		err    bool
	}{
		{answer{status: 200}, true, false},
		{answer{302, "https://accounts.google.com/ServiceLogin?passive=1209600&continue=https://takeout.google.com/manage"}, false, false},
		// A redirect anywhere else, or a failure, says nothing about the session.
		{answer{302, "https://takeout.google.com/"}, false, true},
		{answer{status: 500}, false, true},
	}
	for _, c := range cases {
		ok, err := SignedIn(c.answer)
		if ok != c.ok || (err != nil) != c.err {
			t.Errorf("%+v: got %v, %v; want %v, error %v", c.answer, ok, err, c.ok, c.err)
		}
	}
}
