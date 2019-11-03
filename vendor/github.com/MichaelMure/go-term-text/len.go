package text

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// Len return the length of a string in a terminal, while ignoring the terminal
// escape sequences.
func Len(text string) int {
	length := 0
	escape := false

	for _, char := range text {
		if char == '\x1b' {
			escape = true
		}
		if !escape {
			length += runewidth.RuneWidth(char)
		}
		if char == 'm' {
			escape = false
		}
	}

	return length
}

// MaxLineLen return the length in a terminal of the longest line, while
// ignoring the terminal escape sequences.
func MaxLineLen(text string) int {
	lines := strings.Split(text, "\n")

	max := 0

	for _, line := range lines {
		length := Len(line)
		if length > max {
			max = length
		}
	}

	return max
}
