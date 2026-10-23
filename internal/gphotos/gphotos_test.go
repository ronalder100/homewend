// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package gphotos

import (
	"encoding/json"
	"errors"
	"testing"
)

// page answers the count with what it was given.
type page string

func (p page) Eval(_ string, out any) error { return json.Unmarshal([]byte(p), out) }

func TestCountYearsNewestFirst(t *testing.T) {
	got, err := CountYears(page(`{"Years":{"2019":[6690,685],"2025":[32,0],"2007":[2,0]},"Pages":72}`))
	if err != nil {
		t.Fatal(err)
	}
	want := []Year{{2025, 32, 0}, {2019, 6690, 685}, {2007, 2, 0}}
	if len(got) != len(want) {
		t.Fatalf("got %v; want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v; want %v", got, want)
		}
	}
}

func TestCountYearsWithoutTheSessionSaysThePageChanged(t *testing.T) {
	if _, err := CountYears(page(`{"Missing":true}`)); !errors.Is(err, ErrPageChanged) {
		t.Fatalf("got %v; want ErrPageChanged", err)
	}
}
