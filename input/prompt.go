package input

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/MichaelMure/git-bug/util/interrupt"
	"golang.org/x/crypto/ssh/terminal"
)

func PromptValue(name string, preValue string) (string, error) {
	return promptValue(name, preValue, false)
}

func PromptValueRequired(name string, preValue string) (string, error) {
	return promptValue(name, preValue, true)
}

func promptValue(name string, preValue string, required bool) (string, error) {
	for {
		if preValue != "" {
			_, _ = fmt.Fprintf(os.Stderr, "%s [%s]: ", name, preValue)
		} else {
			_, _ = fmt.Fprintf(os.Stderr, "%s: ", name)
		}

		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", err
		}

		line = strings.TrimSpace(line)

		if preValue != "" && line == "" {
			return preValue, nil
		}

		if required && line == "" {
			_, _ = fmt.Fprintf(os.Stderr, "%s is empty\n", name)
			continue
		}

		return line, nil
	}
}

// PromptPassword performs interactive input collection to get the user password
// while halting echo on the controlling terminal.
func PromptPassword() (string, error) {
	termState, err := terminal.GetState(int(syscall.Stdin))
	if err != nil {
		return "", err
	}

	cancel := interrupt.RegisterCleaner(func() error {
		return terminal.Restore(int(syscall.Stdin), termState)
	})
	defer cancel()

	for {
		fmt.Print("password: ")
		bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
		// new line for coherent formatting, ReadPassword clip the normal new line
		// entered by the user
		fmt.Println()

		if err != nil {
			return "", err
		}

		if len(bytePassword) > 0 {
			return string(bytePassword), nil
		}

		fmt.Println("password is empty")
	}
}
