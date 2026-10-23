// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package session

import (
	"context"
	"testing"
	"time"
)

// One browser at a time on a profile: the second waits for the first to let
// go, or gives up with its context; another profile does not wait.
func TestAProfileIsOpenedByOneBrowserAtATime(t *testing.T) {
	first, err := lockProfile(context.Background(), "p")
	if err != nil {
		t.Fatal(err)
	}
	if other, err := lockProfile(context.Background(), "q"); err != nil {
		t.Fatal(err)
	} else {
		other()
	}
	short, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := lockProfile(short, "p"); err == nil {
		t.Fatal("a second browser had the profile while the first held it")
	}
	got := make(chan func())
	go func() {
		next, _ := lockProfile(context.Background(), "p")
		got <- next
	}()
	first()
	first() // twice is once
	select {
	case next := <-got:
		next()
	case <-time.After(5 * time.Second):
		t.Fatal("the second browser never had its turn")
	}
}
