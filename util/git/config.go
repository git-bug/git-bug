package git

// Based off https://github.com/tcnksm/go-gitconfig which
// has a MIT license

import (
	"os/exec"
	"bytes"
	"io/ioutil"
	"syscall"
	"strings"
	"fmt"
)

type ErrNotFound struct {
	Key string
}

func (e *ErrNotFound) Error() string {
	return fmt.Sprintf("the key %s was not found", e.Key)
}

func GetConfig(key string) (string, error) {
	cmd := exec.Command("git", "config", "--get", "--null", key)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = ioutil.Discard

	err := cmd.Run()
	if exitError, ok := err.(*exec.ExitError); ok {
		if waitStatus, ok := exitError.Sys().(syscall.WaitStatus); ok {
			if waitStatus.ExitStatus() == 1 {
				return "", &ErrNotFound{Key: key}
			}
		}

		return "", err
	}

	return strings.TrimRight(stdout.String(), "\000"), nil
}

func SetConfig(key string, value string) error {
	cmd := exec.Command("git", "config", key, value)
	cmd.Stdout = ioutil.Discard
	cmd.Stderr = ioutil.Discard

	return cmd.Run()
}