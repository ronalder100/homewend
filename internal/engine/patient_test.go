// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/ronalder/homewend/engine/internal/download"
)

func TestPatientlyWaitsOutTheNetwork(t *testing.T) {
	after = func(time.Duration) <-chan time.Time { return time.After(0) }
	defer func() { after = time.After }()
	calls := 0
	err := Patiently(context.Background(), nil, func() error {
		calls++
		switch {
		case calls < 5:
			return fmt.Errorf("reading the export list: %w", &net.OpError{Op: "read", Err: errors.New("connection reset by peer")})
		case calls < 8:
			return download.ErrIncomplete
		}
		return nil
	})
	if err != nil || calls != 8 {
		t.Errorf("got %v after %d calls, want success after 8", err, calls)
	}
}

func TestPatientlyReturnsWhatWaitingCannotCure(t *testing.T) {
	calls := 0
	err := Patiently(context.Background(), nil, func() error {
		calls++
		return download.ErrSessionExpired
	})
	if !errors.Is(err, download.ErrSessionExpired) || calls != 1 {
		t.Errorf("got %v after %d calls", err, calls)
	}
}
