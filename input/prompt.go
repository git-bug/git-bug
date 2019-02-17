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
			fmt.Printf("%s [%s]: ", name, preValue)
		} else {
			fmt.Printf("%s: ", name)
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
			fmt.Printf("%s is empty\n", name)
			continue
		}

		return line, nil
	}
}
