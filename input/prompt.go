package input

import (
	"bufio"
	"errors"
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
			_, _ = fmt.Fprintf(os.Stderr, "invalid input")
			continue
		}

		return index, nil
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
			_, _ = fmt.Fprintf(os.Stderr, "invalid input")
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

var ErrDirectPrompt = errors.New("direct prompt selected")
var ErrInteractiveCreation = errors.New("interactive creation selected")

func PromptCredential(target, name string, credentials []auth.Credential) (auth.Credential, error) {
	if len(credentials) == 0 {
		return nil, nil
	}

	sort.Sort(auth.ById(credentials))

	for {
		_, _ = fmt.Fprintf(os.Stderr, "[1]: enter my %s\n", name)

		_, _ = fmt.Fprintln(os.Stderr)
		_, _ = fmt.Fprintf(os.Stderr, "Existing %s for %s:", name, target)

		for i, cred := range credentials {
			meta := make([]string, 0, len(cred.Metadata()))
			for k, v := range cred.Metadata() {
				meta = append(meta, k+":"+v)
			}
			sort.Strings(meta)
			metaFmt := strings.Join(meta, ",")

			fmt.Printf("[%d]: %s => (%s) (%s)\n",
				i+2,
				colors.Cyan(cred.ID().Human()),
				metaFmt,
				cred.CreateTime().Format(time.RFC822),
			)
		}

		_, _ = fmt.Fprintln(os.Stderr)
		_, _ = fmt.Fprintf(os.Stderr, "Select option: ")

		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		_, _ = fmt.Fprintln(os.Stderr)
		if err != nil {
			return nil, err
		}

		line = strings.TrimSpace(line)
		index, err := strconv.Atoi(line)
		if err != nil || index < 1 || index > len(credentials)+1 {
			_, _ = fmt.Fprintln(os.Stderr, "invalid input")
			continue
		}

		switch index {
		case 1:
			return nil, ErrDirectPrompt
		default:
			return credentials[index-2], nil
		}
	}
}

func PromptCredentialWithInteractive(target, name string, credentials []auth.Credential) (auth.Credential, error) {
	sort.Sort(auth.ById(credentials))

	for {
		_, _ = fmt.Fprintf(os.Stderr, "[1]: enter my %s\n", name)
		_, _ = fmt.Fprintf(os.Stderr, "[2]: interactive %s creation\n", name)

		if len(credentials) > 0 {
			_, _ = fmt.Fprintln(os.Stderr)
			_, _ = fmt.Fprintf(os.Stderr, "Existing %s for %s:", name, target)

			for i, cred := range credentials {
				meta := make([]string, 0, len(cred.Metadata()))
				for k, v := range cred.Metadata() {
					meta = append(meta, k+":"+v)
				}
				sort.Strings(meta)
				metaFmt := strings.Join(meta, ",")

				fmt.Printf("[%d]: %s => (%s) (%s)\n",
					i+2,
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
			return nil, err
		}

		line = strings.TrimSpace(line)
		index, err := strconv.Atoi(line)
		if err != nil || index < 1 || index > len(credentials)+1 {
			_, _ = fmt.Fprintln(os.Stderr, "invalid input")
			continue
		}

		switch index {
		case 1:
			return nil, ErrDirectPrompt
		case 2:
			return nil, ErrInteractiveCreation
		default:
			return credentials[index-3], nil
		}
	}
}
