// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package term

import (
	"os"
	"syscall"
	"unsafe"
)

var bufferInfo = syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleScreenBufferInfo")

type coord struct {
	x, y int16
}

type rect struct {
	left, top, right, bottom int16
}

type screen struct {
	size    coord
	cursor  coord
	attrs   uint16
	window  rect
	maxSize coord
}

func Width(f *os.File) int {
	var s screen
	ok, _, _ := bufferInfo.Call(f.Fd(), uintptr(unsafe.Pointer(&s)))
	if ok == 0 {
		return 0
	}
	return int(s.window.right-s.window.left) + 1
}
