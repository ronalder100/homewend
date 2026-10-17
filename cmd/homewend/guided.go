// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"time"
)

// guided is what a double-click on Windows gets: homewend started with no
// arguments in a terminal of its own, by someone who has never typed a flag.
// It asks which year, runs takeout, which asks the rest, and waits before the
// window closes, since a window that vanishes on the last line is a program
// that looks broken.
func guided() int {
	ctx, stop := interrupted()
	cleanScreen(nil)
	var args []string
	for detail := text["guided year meta"]; ; detail = text["guided bad year"] {
		answer, err := field(ctx, text["guided year"], detail, "")
		if err != nil {
			stop()
			return newPrinter(false).fail(err)
		}
		if year, ok := parseYear(answer, time.Now().Year()); ok {
			if year != 0 {
				args = []string{strconv.Itoa(year)}
			}
			break
		}
	}
	stop()
	code := takeoutCommand(args)
	fmt.Print("\n" + text["guided done"])
	bufio.NewReader(os.Stdin).ReadString('\n')
	return code
}

// parseYear reads an answer to "which year": nothing means all of them (0),
// a year means that one, anything else is not an answer.
func parseYear(answer string, thisYear int) (int, bool) {
	if answer == "" {
		return 0, true
	}
	y, err := strconv.Atoi(answer)
	if err != nil || y < 1990 || y > thisYear {
		return 0, false
	}
	return y, true
}
