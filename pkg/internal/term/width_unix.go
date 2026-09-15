// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

//go:build !windows

package term

import (
	"os"
	"syscall"
	"unsafe"
)

func Width(f *os.File) int {
	var ws struct {
		rows, cols, x, y uint16
	}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&ws)))
	if errno != 0 {
		return 0
	}
	return int(ws.cols)
}
