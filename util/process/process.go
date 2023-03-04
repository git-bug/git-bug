package process

import (
	"os"
	"syscall"
)

// IsRunning tell is a process is running
func IsRunning(pid int) bool {
	// never return no error in a unix system
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Signal 0 doesn't do anything but allow testing the process
	err = process.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	if err.Error() == "os: process already finished" {
		return false
	}
	if errno, ok := err.(syscall.Errno); ok {
		switch errno {
		case syscall.ESRCH:
			return false
		case syscall.EPERM:
			return true
		}
	}
	return false
}
