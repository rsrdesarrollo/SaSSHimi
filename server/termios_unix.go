//go:build !windows

package server

import (
	"syscall"

	"golang.org/x/sys/unix"
)

func TermiosSaveStdin() *unix.Termios {
	termios, _ := unix.IoctlGetTermios(int(syscall.Stdin), unix.TCGETS)
	return termios
}

func TermiosRestoreStdin(value *unix.Termios) {
	unix.IoctlSetTermios(int(syscall.Stdin), unix.TCGETS, value)
}
