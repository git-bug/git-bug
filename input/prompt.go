package input

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/MichaelMure/git-bug/bridge/core/auth"
	"github.com/MichaelMure/git-bug/util/colors"
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

// IsURL is a validator checking that the value is a fully formed URL
func IsURL(name string, value string) (string, error) {
	u, err := url.Parse(value)
	if err != nil {
		return fmt.Sprintf("%s is invalid: %v", name, err), nil
	}
	if u.Scheme == "" {
		return fmt.Sprintf("%s is missing a scheme", name), nil
	}
	if u.Host == "" {
		return fmt.Sprintf("%s is missing a host", name), nil
	}
	return "", nil
}

// Prompts

func Prompt(prompt, name string, validators ...PromptValidator) (string, error) {
	return PromptDefault(prompt, name, "", validators...)
}

func PromptDefault(prompt, name, preValue string, validators ...PromptValidator) (string, error) {
loop:
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
				continue loop
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

loop:
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
				continue loop
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
			_, _ = fmt.Fprintln(os.Stderr, "invalid input")
			continue
		}

		return index - 1, nil
	}
}

func PromptURLWithRemote(prompt, name string, validRemotes []string, validators ...PromptValidator) (string, error) {
	if len(validRemotes) == 0 {
		return Prompt(prompt, name, validators...)
	}

	sort.Strings(validRemotes)

	for {
		_, _ = fmt.Fprintln(os.Stderr, "\nDetected projects:")

		for i, remote := range validRemotes {
			_, _ = fmt.Fprintf(os.Stderr, "[%d]: %v\n", i+1, remote)
		}

		_, _ = fmt.Fprintf(os.Stderr, "\n[0]: Another project\n\n")
		_, _ = fmt.Fprintf(os.Stderr, "Select option: ")

		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", err
		}

		line = strings.TrimSpace(line)

		index, err := strconv.Atoi(line)
		if err != nil || index < 0 || index > len(validRemotes) {
			_, _ = fmt.Fprintln(os.Stderr, "invalid input")
			continue
		}

		// if user want to enter another project url break this loop
		if index == 0 {
			break
		}

		return validRemotes[index-1], nil
	}

	return Prompt(prompt, name, validators...)
}

func PromptCredential(target, name string, credentials []auth.Credential, choices []string) (auth.Credential, int, error) {
	if len(credentials) == 0 && len(choices) == 0 {
		return nil, 0, fmt.Errorf("no possible choice")
	}
	if len(credentials) == 0 && len(choices) == 1 {
		return nil, 0, nil
	}

	sort.Sort(auth.ById(credentials))

	for {
		_, _ = fmt.Fprintln(os.Stderr)

		offset := 0
		for i, choice := range choices {
			_, _ = fmt.Fprintf(os.Stderr, "[%d]: %s\n", i+1, choice)
			offset++
		}

		if len(credentials) > 0 {
			_, _ = fmt.Fprintln(os.Stderr)
			_, _ = fmt.Fprintf(os.Stderr, "Existing %s for %s:\n", name, target)

			for i, cred := range credentials {
				meta := make([]string, 0, len(cred.Metadata()))
				for k, v := range cred.Metadata() {
					meta = append(meta, k+":"+v)
				}
				sort.Strings(meta)
				metaFmt := strings.Join(meta, ",")

				fmt.Printf("[%d]: %s => (%s) (%s)\n",
					i+1+offset,
					colors.Cyan(cred.ID().Human()),
					metaFmt,
					cred.CreateTime().Format(time.RFC822),
				)
			}
		}

		_, _ = fmt.Fprintln(os.Stderr)
		_, _ = fmt.Fprintf(os.Stderr, "Select option: ")

		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		_, _ = fmt.Fprintln(os.Stderr)
		if err != nil {
			return nil, 0, err
		}

		line = strings.TrimSpace(line)
		index, err := strconv.Atoi(line)
		if err != nil || index < 1 || index > len(choices)+len(credentials) {
			_, _ = fmt.Fprintln(os.Stderr, "invalid input")
			continue
		}

		switch {
		case index <= len(choices):
			return nil, index - 1, nil
		default:
			return credentials[index-len(choices)-1], 0, nil
		}
	}
}
