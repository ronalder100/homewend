// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package takeout

import (
	"encoding/base64"
	"errors"
	"fmt"
	"time"
)

// Driver is a page the program can read and press: see session.Page.
type Driver interface {
	Eval(expression string, out any) error
	Click(x, y float64) error
}

// ErrFormChanged means Takeout's form is not the one this code was written
// against. Nothing was sent: pressing on in a form we do not recognise could
// export the wrong thing, which is worse than exporting nothing.
var ErrFormChanged = errors.New("Takeout's export form has changed")

// Entry is one thing the content picker offers: an album, or a year.
type Entry struct {
	ID   string // the checkbox's value, Google's id for it
	Name string // as Google names it, in the account's language
}

// NotOfferedError means the picker has no entry for the year asked for. Nothing
// was sent; Offered is what the picker does have, for the user to choose from.
type NotOfferedError struct {
	Year    int
	Offered []Entry
}

func (e *NotOfferedError) Error() string {
	return fmt.Sprintf("Takeout offers no export of %d", e.Year)
}

// PickerWait is how long Google may take to fill the content picker. It
// answers the page's poll with the list of albums only when it has built it:
// 1 m 34 s to 2 m 3 s after the page loaded, six loads on 2026-09-26, nothing
// cached between loads.
const PickerWait = 5 * time.Minute

// The form, as read on 2026-09-26 at PhotosURL. It is found by what survives
// translation — Google's jsname, jsaction and data-id values, input names and
// values — never by a label.
//
//	step 1: one product listed, Google Photos, already ticked; the last
//	        button with jsaction h5M12e is "Next step". The content picker's
//	        button, jsname DNRMdf, is hidden and disabled until Google has
//	        sent the list of albums; then Google shows and enables it.
//	picker: a role=dialog, jsname XweNkc; one checkbox per album or year,
//	        its value Google's id for it; "Deselect all" is jsname Si7An; OK
//	        is data-id EBS5u, the id Google's own handler checks for.
//	step 2: radios named scheduleoptions, value 1 "Export once" (the
//	        default), 5 every month, 2 every two months; three comboboxes,
//	        jsname oYxtQd, for destination, file type (".zip" by default) and
//	        size ("2 GB"); the last h5M12e button is "Create export"
//
// The pages build inside shadow roots, so every lookup walks them.
const formScript = `
  const deep = (test, root = document, out = []) => {
    for (const el of root.querySelectorAll('*')) {
      if (test(el)) out.push(el)
      if (el.shadowRoot) deep(test, el.shadowRoot, out)
    }
    return out
  }
  const visible = (el) => el.getBoundingClientRect().width > 0
  const centre = (el) => {
    el.scrollIntoView({ block: 'center' })
    const r = el.getBoundingClientRect()
    return { x: r.left + r.width / 2, y: r.top + r.height / 2 }
  }
  const buttons = () => deep(el => el.tagName === 'BUTTON' && visible(el)
    && /\bh5M12e\b/.test(el.getAttribute('jsaction') || ''))
  const boxes = () => deep(el => el.tagName === 'INPUT' && el.type === 'checkbox' && visible(el))
  const schedule = () => deep(el => el.tagName === 'INPUT' && el.type === 'radio'
    && el.name === 'scheduleoptions' && visible(el))
  const combos = () => deep(el => el.getAttribute('role') === 'combobox'
    && el.getAttribute('jsname') === 'oYxtQd' && visible(el))
  const pickers = () => deep(el => el.getAttribute('jsname') === 'DNRMdf' && visible(el) && !el.disabled)
  const dialogs = () => deep(el => el.getAttribute('role') === 'dialog'
    && el.getAttribute('jsname') === 'XweNkc' && visible(el))
  const entries = (dialog) => deep(el => el.tagName === 'INPUT' && el.type === 'checkbox' && el.value, dialog)
`

type point struct{ X, Y float64 }

// step1 is what the first page shows once it has built.
type step1 struct {
	Boxes, Ticked int
	Next          *point
}

// picker is the content picker's button, once Google has enabled it.
type picker struct {
	Pickers int
	At      *point
}

// dialog is the open content picker.
type dialog struct {
	Dialogs int
	Entries []dialogEntry
}

type dialogEntry struct {
	ID, Name string
	Checked  bool
}

// step2 is what the second page shows once it has built.
type step2 struct {
	Once     *point // the "Export once" radio
	OnceSet  bool
	Combos   []string
	Create   *point
	Schedule int
}

// CreateExport asks for an export of the Google Photos of one year, or of all
// of them when year is 0, once, as .zip, on a page opened at PhotosURL. It returns once the request is sent;
// whether Google took it is for /manage to say.
func CreateExport(d Driver, year int) error {
	var one step1
	err := waitFor(d, "the first step", 30*time.Second, `(() => {`+formScript+`
		const next = buttons().pop()
		return { Boxes: boxes().length, Ticked: boxes().filter(b => b.checked).length,
		         Next: next ? centre(next) : null }
	})()`, &one, func() bool { return one.Boxes > 0 && one.Next != nil })
	if err != nil {
		return err
	}
	// Google Photos alone, as the address asks: anything else ticked would
	// export more than the user's photos.
	if one.Boxes != 1 || one.Ticked != 1 {
		return fmt.Errorf("%w: %d products listed, %d ticked, expected Google Photos alone", ErrFormChanged, one.Boxes, one.Ticked)
	}

	if err := choose(d, year); err != nil {
		return err
	}

	// The picker's closing moved the page: find Next step again.
	err = waitFor(d, "Next step after the content picker", 10*time.Second, `(() => {`+formScript+`
		const next = buttons().pop()
		return { Boxes: boxes().length, Ticked: boxes().filter(b => b.checked).length, Next: next ? centre(next) : null }
	})()`, &one, func() bool { return one.Next != nil })
	if err != nil {
		return err
	}
	if err := d.Click(one.Next.X, one.Next.Y); err != nil {
		return err
	}

	var two step2
	err = waitFor(d, "the second step", 30*time.Second, `(() => {`+formScript+`
		const once = schedule().find(r => r.value === '1')
		const create = buttons().pop()
		return { Schedule: schedule().length, Once: once ? centre(once) : null, OnceSet: !!(once && once.checked),
		         Combos: combos().map(c => (c.innerText || '').trim()), Create: create ? centre(create) : null }
	})()`, &two, func() bool { return two.Schedule > 0 })
	if err != nil {
		return err
	}
	if two.Schedule != 3 || two.Once == nil || len(two.Combos) != 3 || two.Create == nil {
		return fmt.Errorf("%w: %d frequency options, %d selects", ErrFormChanged, two.Schedule, len(two.Combos))
	}
	// The file type is left as Google offers it, and checked: the downloader
	// unpacks zips. The size, 2 GB when read, is left alone: any size works.
	if two.Combos[1] != ".zip" {
		return fmt.Errorf("%w: file type %q, expected .zip", ErrFormChanged, two.Combos[1])
	}
	if !two.OnceSet {
		if err := d.Click(two.Once.X, two.Once.Y); err != nil {
			return err
		}
		var set bool
		if err := d.Eval(`(() => {`+formScript+` const r = schedule().find(r => r.value === '1'); return !!(r && r.checked) })()`, &set); err != nil {
			return err
		}
		if !set {
			return fmt.Errorf("%w: could not choose to export once", ErrFormChanged)
		}
	}
	return d.Click(two.Create.X, two.Create.Y)
}

// choose opens the content picker, leaves only year ticked — or, for year 0,
// checks that everything is — and confirms.
//
// Nothing is forced: the picker's button is pressed only once Google has
// enabled it. Pressed earlier, forced visible, Google's handler reads a list
// it does not have yet and fails ("Cannot read properties of null"), measured
// 2026-09-26.
func choose(d Driver, year int) error {
	var p picker
	err := waitFor(d, "the content picker to be enabled", PickerWait, `(() => {`+formScript+`
		const p = pickers()
		return { Pickers: p.length, At: p.length === 1 ? centre(p[0]) : null }
	})()`, &p, func() bool { return p.Pickers > 0 })
	if err != nil {
		return err
	}
	if p.Pickers != 1 {
		return fmt.Errorf("%w: %d content pickers, expected Google Photos' alone", ErrFormChanged, p.Pickers)
	}

	var dl dialog
	look := `(() => {` + formScript + `
		const open = dialogs()
		if (open.length !== 1) return { Dialogs: open.length }
		return { Dialogs: 1, Entries: entries(open[0]).map(b => ({ ID: b.value, Name: b.name, Checked: b.checked })) }
	})()`
	// One press opens it. Once, on 2026-09-26, a press on the enabled button
	// did not; a second press is made only while no dialog is open, because
	// every press that lands opens its own copy.
	for press := 0; press < 2 && dl.Dialogs == 0; press++ {
		if err := d.Click(p.At.X, p.At.Y); err != nil {
			return err
		}
		err = waitFor(d, "the content picker to open", 15*time.Second, look, &dl, func() bool { return dl.Dialogs > 0 && len(dl.Entries) > 0 })
		if err != nil && !errors.Is(err, ErrFormChanged) {
			return err
		}
	}
	if dl.Dialogs != 1 || len(dl.Entries) == 0 {
		return fmt.Errorf("%w: the content picker did not open as one dialog with its entries (%d open)", ErrFormChanged, dl.Dialogs)
	}

	if year == 0 {
		// Everything is what Google ticks by default; it is checked here, not
		// assumed, because a partial selection would pass for a full export.
		for _, e := range dl.Entries {
			if !e.Checked {
				return fmt.Errorf("%w: not every album and year is ticked", ErrFormChanged)
			}
		}
		return confirm(d)
	}

	var want string
	offered := make([]Entry, len(dl.Entries))
	for i, e := range dl.Entries {
		offered[i] = Entry{ID: e.ID, Name: e.Name}
		if y, ok := YearOf(e.ID); ok && y == year {
			want = e.ID
		}
	}
	if want == "" {
		return &NotOfferedError{Year: year, Offered: offered}
	}

	if err := press(d, "Deselect all", `deep(el => el.getAttribute('jsname') === 'Si7An' && visible(el), dialogs()[0])[0]`); err != nil {
		return err
	}
	// Google clears the boxes asynchronously: the year is ticked only once
	// they are all clear, or the clearing could untick it again.
	var left int
	err = waitFor(d, "the content picker's boxes to clear", 10*time.Second, `(() => {`+formScript+` return entries(dialogs()[0]).filter(b => b.checked).length })()`, &left, func() bool { return left == 0 })
	if err != nil {
		return err
	}
	if err := press(d, "the year", `entries(dialogs()[0]).find(b => b.value === `+jsString(want)+`)`); err != nil {
		return err
	}
	// Exactly the year, or nothing is sent: a wrong selection would export
	// the wrong photos, or all of them, without anyone noticing.
	var ticked []string
	err = waitFor(d, "the year to be ticked", 5*time.Second, `(() => {`+formScript+` return entries(dialogs()[0]).filter(b => b.checked).map(b => b.value) })()`, &ticked, func() bool { return len(ticked) > 0 })
	if err != nil {
		return err
	}
	if len(ticked) != 1 || ticked[0] != want {
		return fmt.Errorf("%w: %d entries ticked, expected the year alone", ErrFormChanged, len(ticked))
	}
	return confirm(d)
}

// confirm presses the content picker's OK and waits for it to close.
func confirm(d Driver) error {
	if err := press(d, "OK", `deep(el => el.getAttribute('data-id') === 'EBS5u' && visible(el), dialogs()[0])[0]`); err != nil {
		return err
	}
	var open int
	return waitFor(d, "the content picker to close", 10*time.Second, `(() => {`+formScript+` return dialogs().length })()`, &open, func() bool { return open == 0 })
}

// press clicks the element finder finds, where it is at that moment. Two
// things move controls: bringing one into view moves the others, and the
// picker slides in as it opens. So the position is read until it holds still
// (measured 2026-09-26: "Deselect all" pressed while the picker was still
// opening missed, and pressed a second later worked).
func press(d Driver, what, finder string) error {
	var at, was *point
	for try := 0; try < 20; try++ {
		if err := d.Eval(`(() => {`+formScript+` const el = `+finder+`; return el ? centre(el) : null })()`, &at); err != nil {
			return err
		}
		if at == nil {
			return fmt.Errorf("%w: no %s in the content picker", ErrFormChanged, what)
		}
		if was != nil && *was == *at {
			return d.Click(at.X, at.Y)
		}
		was, at = at, nil
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("%w: %s kept moving", ErrFormChanged, what)
}

// YearOf reads the year out of a picker entry's id. Google's id for a year
// holds the year and nothing else — protobuf {2: {6: {1: year}}}, base64url:
// "EgUyAwjpDw" is 2025. Measured 2026-09-26 on one account, all 18 of its
// years, the page in English and in German; an album's id has another shape.
func YearOf(id string) (int, bool) {
	b, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil || len(b) < 6 || b[0] != 0x12 || int(b[1]) != len(b)-2 ||
		b[2] != 0x32 || int(b[3]) != len(b)-4 || b[4] != 0x08 {
		return 0, false
	}
	year, shift := 0, 0
	for i, c := range b[5:] {
		year |= int(c&0x7f) << shift
		shift += 7
		if c&0x80 == 0 {
			return year, i == len(b)-6
		}
	}
	return 0, false
}

// jsString quotes s for a script, as a JavaScript string literal.
func jsString(s string) string {
	return fmt.Sprintf("%q", s)
}

// waitFor evaluates expression into out until ready reports true; what names
// the wait in the error when it runs out. The pages
// draw their controls first and fill them in later, so a first look proves
// nothing.
func waitFor(d Driver, what string, limit time.Duration, expression string, out any, ready func() bool) error {
	for deadline := time.Now().Add(limit); ; time.Sleep(time.Second) {
		if err := d.Eval(expression, out); err != nil {
			return err
		}
		if ready() {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%w: waited %s for %s", ErrFormChanged, limit, what)
		}
	}
}
