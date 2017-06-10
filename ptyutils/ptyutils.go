package ptyutils

import (
	"os"
	"syscall"
	"unsafe"
)

type winsize struct {
	row    uint16
	col    uint16
	xpixel uint16
	ypixel uint16
}

// SetWindowSize set's the PTY's window size
func SetWindowSize(file *os.File, width, height int) error {
	ws := &winsize{row: uint16(height), col: uint16(width)}
	_, _, err := syscall.Syscall(
		syscall.SYS_IOCTL,
		file.Fd(),
		uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(ws)),
	)
	if err != 0 {
		return err
	}
	return nil
}
