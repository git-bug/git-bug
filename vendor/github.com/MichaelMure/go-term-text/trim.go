package text

import (
	"strings"
	"unicode"
)

// TrimSpace remove the leading and trailing whitespace while ignoring the
// terminal escape sequences.
// Returns the number of trimmed space on both side.
func TrimSpace(line string) string {
	cleaned, escapes := ExtractTermEscapes(line)

	// trim left while counting
	left := 0
	trimmed := strings.TrimLeftFunc(cleaned, func(r rune) bool {
		if unicode.IsSpace(r) {
			left++
			return true
		}
		return false
	})

	trimmed = strings.TrimRightFunc(trimmed, unicode.IsSpace)

	escapes = OffsetEscapes(escapes, -left)
	return ApplyTermEscapes(trimmed, escapes)
}
