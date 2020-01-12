package input

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/MichaelMure/git-bug/util/interrupt"
)

// PromptValidator is a validator for a user entry
// If complaint is "", value is considered valid, otherwise it's the error reported to the user
// If err != nil, a terminal error happened
type PromptValidator func(name string, value string) (complaint string, err error)

// Required is a validator preventing a "" value
func Required(name string, value string) (string, error) {
	if value == "" {
		return fmt.Sprintf("%s is empty", name), nil
	}
	return "", nil
}

func Prompt(prompt, name string, validators ...PromptValidator) (string, error) {
	return PromptDefault(prompt, name, "", validators...)
}

func PromptDefault(prompt, name, preValue string, validators ...PromptValidator) (string, error) {
	for {
		if preValue != "" {
			_, _ = fmt.Fprintf(os.Stderr, "%s [%s]: ", prompt, preValue)
		} else {
			_, _ = fmt.Fprintf(os.Stderr, "%s: ", prompt)
		}

		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", err
		}

		line = strings.TrimSpace(line)

		if preValue != "" && line == "" {
			line = preValue
		}

		for _, validator := range validators {
			complaint, err := validator(name, line)
			if err != nil {
				return "", err
			}
			if complaint != "" {
				_, _ = fmt.Fprintln(os.Stderr, complaint)
				continue
			}
		}

		return line, nil
	}
}

func PromptPassword(prompt, name string, validators ...PromptValidator) (string, error) {
	termState, err := terminal.GetState(syscall.Stdin)
	if err != nil {
		return "", err
	}

	cancel := interrupt.RegisterCleaner(func() error {
		return terminal.Restore(syscall.Stdin, termState)
	})
	defer cancel()

	for {
		_, _ = fmt.Fprintf(os.Stderr, "%s: ", prompt)

		bytePassword, err := terminal.ReadPassword(syscall.Stdin)
		// new line for coherent formatting, ReadPassword clip the normal new line
		// entered by the user
		fmt.Println()

		if err != nil {
			return "", err
		}

		pass := string(bytePassword)

		for _, validator := range validators {
			complaint, err := validator(name, pass)
			if err != nil {
				return "", err
			}
			if complaint != "" {
				_, _ = fmt.Fprintln(os.Stderr, complaint)
				continue
			}
		}

		return pass, nil
	}
}

func PromptChoice(prompt string, choices []string) (int, error) {
	for {
		for i, choice := range choices {
			_, _ = fmt.Fprintf(os.Stderr, "[%d]: %s\n", i+1, choice)
		}
		_, _ = fmt.Fprintf(os.Stderr, "%s: ", prompt)

		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		fmt.Println()
		if err != nil {
			return 0, err
		}

		line = strings.TrimSpace(line)

		index, err := strconv.Atoi(line)
		if err != nil || index < 1 || index > len(choices) {
			fmt.Println("invalid input")
			continue
		}

		return index, nil
	}
}
