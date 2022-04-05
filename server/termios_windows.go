//go:build windows

package server

func TermiosSaveStdin() int {
	return 0
}

func TermiosRestoreStdin(value int) {
}
