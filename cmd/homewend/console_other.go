// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build !windows

package main

// doubleClicked is false: elsewhere the program is run from a shell.
func doubleClicked() bool { return false }
