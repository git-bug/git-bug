package input

import (
	"bufio"
	"fmt"
	"os"
	"strings"
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
