package text

import (
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
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
// Currently trigger on invalid UTF-8, on unicode control character
// except \n, \t and \r, and on unsafe format characters (see isUnsafeRune)
func Safe(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}

	for _, r := range s {
		switch r {
		case '\t', '\r', '\n':
			continue
		}

		if unicode.IsControl(r) || isUnsafeRune(r) {
			return false
		}
	}

	return true
}

// SafeOneLine will tell if a character in the string is considered unsafe
// Currently trigger on invalid UTF-8, on all unicode control character
// and on unsafe format characters (see isUnsafeRune)
func SafeOneLine(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}

	for _, r := range s {
		if unicode.IsControl(r) || isUnsafeRune(r) {
			return false
		}
	}

	return true
}

// isUnsafeRune tell if a non-control character should never appear in text:
//   - bidi embeddings and overrides (U+202A-U+202E), which can make text
//     display in a different order than it's stored. Bidi marks and isolates
//     are still allowed as they are legitimately used in RTL text.
//   - noncharacters (U+FDD0-U+FDEF and U+xFFFE/U+xFFFF), which by definition
//     are not meant for text interchange.
func isUnsafeRune(r rune) bool {
	switch {
	case r >= '‪' && r <= '‮':
		return true
	case r >= '﷐' && r <= '﷯':
		return true
	case r&0xFFFE == 0xFFFE:
		return true
	}
	return false
}

// ValidUrl will tell if the string contains what seems to be a valid URL
func ValidUrl(s string) bool {
	if strings.Contains(s, "\n") {
		return false
	}

	_, err := url.ParseRequestURI(s)
	return err == nil
}
