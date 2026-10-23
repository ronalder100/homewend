// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ronalder100/homewend/internal/gphotos"
	"github.com/ronalder100/homewend/internal/session"
)

// yearsFile keeps an account's last count, in its browser profile: a count
// takes a minute, and the years an account has change slowly.
const yearsFile = "homewend-years.json"

// YearCounts is how many photos and videos the account has on Google, by year.
type YearCounts struct {
	Counted time.Time      `json:"counted"`
	Years   []gphotos.Year `json:"years"`
}

// LoadYearCounts is the account's last count; nil when it was never counted.
func LoadYearCounts(account string) (*YearCounts, error) {
	dir, err := AccountDir(account)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, yearsFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var c YearCounts
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("reading %s: %w", yearsFile, err)
	}
	return &c, nil
}

// countYears counts the account on photos.google.com, out of sight, and keeps
// the count.
func countYears(ctx context.Context, sess *session.Session) error {
	page, err := sess.Headless(ctx, gphotos.URL)
	if err != nil {
		return err
	}
	defer page.Close()
	years, err := gphotos.CountYears(page)
	if err != nil {
		return err
	}
	return writeJSON(filepath.Join(sess.Profile, yearsFile), YearCounts{Counted: time.Now(), Years: years})
}
