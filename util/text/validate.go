package text

import (
	"strings"
	"unicode"
)

// Empty tell if the string is considered empty once space
// and not graphics characters are removed
func Empty(s string) bool {
	trim := strings.TrimFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || !unicode.IsGraphic(r)
	})

	return trim == ""
}

// Safe will tell if a character in the string is considered unsafe
// Currently trigger on unicode control character except \n, \t and \r
func Safe(s string) bool {
	for _, r := range s {
		switch r {
		case '\t', '\r', '\n':
			continue
		}

		if unicode.IsControl(r) {
			return false
		}
	}

	return true
}
