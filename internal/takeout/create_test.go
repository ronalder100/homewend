// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package takeout

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// form stands in for Takeout's pages: it answers each look at the page with
// what that page shows, and records every press.
type form struct {
	step1, picker, dialog, ticked, step2 string // the JSON each look returns
	clicks                               []point
}

func (f *form) Eval(expression string, out any) error {
	var answer string
	switch {
	case strings.Contains(expression, "Create: create"):
		answer = f.step2
	case strings.Contains(expression, "r.checked"):
		answer = "true"
	case strings.Contains(expression, "Pickers:"):
		answer = f.picker
	case strings.Contains(expression, "Entries:"):
		answer = f.dialog
	case strings.Contains(expression, "b.value === "):
		answer = `{"x":6,"y":6}`
	case strings.Contains(expression, "Si7An"):
		answer = `{"x":3,"y":3}`
	case strings.Contains(expression, "EBS5u"):
		answer = `{"x":4,"y":4}`
	case strings.Contains(expression, "entries(dialogs()[0]).filter(b => b.checked).length"):
		answer = "0"
	case strings.Contains(expression, "map(b => b.value)"):
		answer = f.ticked
	case strings.Contains(expression, "return dialogs().length"):
		answer = "0"
	default:
		answer = f.step1
	}
	return json.Unmarshal([]byte(answer), out)
}

func (f *form) Click(x, y float64) error {
	f.clicks = append(f.clicks, point{x, y})
	return nil
}

func (f *form) pressed(p point) bool {
	for _, c := range f.clicks {
		if c == p {
			return true
		}
	}
	return false
}

// Ids as Google sends them, read on 2026-09-26: two years and an album.
const (
	id2025 = "EgUyAwjpDw"
	id2026 = "EgUyAwjqDw"
	album  = "EhJKEAix95HUnAwRYSL6vIFJCl0"
)

var (
	next, pickerAt, deselect, ok, yearBox, once, create = point{1, 1}, point{2, 2}, point{3, 3}, point{4, 4}, point{6, 6}, point{7, 7}, point{8, 8}
)

func newForm() *form {
	return &form{
		step1:  `{"Boxes":1,"Ticked":1,"Next":{"X":1,"Y":1}}`,
		picker: `{"Pickers":1,"At":{"X":2,"Y":2}}`,
		dialog: `{"Dialogs":1,"Entries":[` +
			`{"ID":"` + album + `","Name":"10 giorni in Grecia","Checked":true},` +
			`{"ID":"` + id2025 + `","Name":"Photos from 2025","Checked":true},` +
			`{"ID":"` + id2026 + `","Name":"Photos from 2026","Checked":true}]}`,
		ticked: `["` + id2025 + `"]`,
		step2:  `{"Schedule":3,"Once":{"X":7,"Y":7},"OnceSet":true,"Combos":["Send download link via email",".zip","2 GB"],"Create":{"X":8,"Y":8}}`,
	}
}

func TestYearOfReadsGooglesYearIds(t *testing.T) {
	for id, want := range map[string]int{"EgUyAwjXDw": 2007, id2025: 2025, id2026: 2026} {
		if y, ok := YearOf(id); !ok || y != want {
			t.Errorf("YearOf(%s) = %d, %v; want %d", id, y, ok, want)
		}
	}
	for _, id := range []string{album, "", "not base64!", "EgUyAwjpDwA"} {
		if y, ok := YearOf(id); ok {
			t.Errorf("YearOf(%s) = %d; want no year", id, y)
		}
	}
}

func TestCreateExportChoosesTheYearAlone(t *testing.T) {
	f := newForm()
	if err := CreateExport(f, 2025); err != nil {
		t.Fatal(err)
	}
	want := []point{pickerAt, deselect, yearBox, ok, next, create}
	if len(f.clicks) != len(want) {
		t.Fatalf("pressed %v; want %v", f.clicks, want)
	}
	for i := range want {
		if f.clicks[i] != want[i] {
			t.Fatalf("pressed %v; want %v", f.clicks, want)
		}
	}
}

func TestCreateExportChoosesExportOnce(t *testing.T) {
	f := newForm()
	f.step2 = strings.Replace(f.step2, `"OnceSet":true`, `"OnceSet":false`, 1)
	if err := CreateExport(f, 2025); err != nil {
		t.Fatal(err)
	}
	if !f.pressed(once) {
		t.Errorf("pressed %v; want Export once pressed", f.clicks)
	}
}

func TestAYearNotOfferedSendsNothingAndSaysWhatIs(t *testing.T) {
	f := newForm()
	err := CreateExport(f, 2024)
	var notOffered *NotOfferedError
	if !errors.As(err, &notOffered) {
		t.Fatalf("got %v, want NotOfferedError", err)
	}
	if notOffered.Year != 2024 || len(notOffered.Offered) != 3 || notOffered.Offered[0].Name != "10 giorni in Grecia" {
		t.Errorf("got %+v; want 2024 and the three entries offered", notOffered)
	}
	if f.pressed(deselect) || f.pressed(create) {
		t.Errorf("pressed %v; want nothing past opening the picker", f.clicks)
	}
}

func TestCreateExportSendsNothingOnAFormItDoesNotKnow(t *testing.T) {
	cases := map[string]func(*form){
		"more than Photos ticked": func(f *form) { f.step1 = `{"Boxes":3,"Ticked":2,"Next":{"X":1,"Y":1}}` },
		"two pickers":             func(f *form) { f.picker = `{"Pickers":2}` },
		"more than the year left": func(f *form) { f.ticked = `["` + id2025 + `","` + album + `"]` },
		"not a zip":               func(f *form) { f.step2 = strings.Replace(f.step2, `".zip"`, `".tgz"`, 1) },
		"a fourth frequency":      func(f *form) { f.step2 = strings.Replace(f.step2, `"Schedule":3`, `"Schedule":4`, 1) },
	}
	for name, change := range cases {
		f := newForm()
		change(f)
		err := CreateExport(f, 2025)
		if !errors.Is(err, ErrFormChanged) {
			t.Errorf("%s: got %v, want ErrFormChanged", name, err)
		}
		if f.pressed(create) {
			t.Errorf("%s: Create export was pressed", name)
		}
	}
}

func TestCreateExportOfEverythingLeavesThePickerAsGoogleTicksIt(t *testing.T) {
	f := newForm()
	if err := CreateExport(f, 0); err != nil {
		t.Fatal(err)
	}
	if f.pressed(deselect) || f.pressed(yearBox) {
		t.Error("the selection was changed")
	}
	if !f.pressed(ok) || !f.pressed(create) {
		t.Error("the picker was not confirmed or the export not created")
	}
}

func TestCreateExportOfEverythingSendsNothingWhenSomethingIsUnticked(t *testing.T) {
	f := newForm()
	f.dialog = strings.Replace(f.dialog, `"Checked":true}]`, `"Checked":false}]`, 1)
	if err := CreateExport(f, 0); !errors.Is(err, ErrFormChanged) {
		t.Fatalf("got %v, want ErrFormChanged", err)
	}
	if f.pressed(create) {
		t.Error("the export was created")
	}
}
