package util

import (
	"os"
	"syscall"
)

// ProcessIsRunning tell is a process is running
func ProcessIsRunning(pid int) bool {
	// never return no error in a unix system
	process, err := os.FindProcess(pid)

	if err != nil {
		return false
	}

	// Signal 0 doesn't do anything but allow testing the process
	err = process.Signal(syscall.Signal(0))

	// Todo: distinguish "you don't have access" and "process doesn't exist"

	return err == nil
}
