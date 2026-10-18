// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// getConsoleProcessList is not in golang.org/x/sys/windows.
var getConsoleProcessList = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetConsoleProcessList")

// doubleClicked reports whether the program was opened from Explorer rather
// than typed in a terminal. Opened that way it gets a console of its own,
// with no shell attached: GetConsoleProcessList counts the processes on the
// console, and a buffer too small for them all is answered with how many
// there are (learn.microsoft.com, "GetConsoleProcessList function").
func doubleClicked() bool {
	var ids [1]uint32
	n, _, _ := getConsoleProcessList.Call(uintptr(unsafe.Pointer(&ids[0])), uintptr(len(ids)))
	return n == 1
}
