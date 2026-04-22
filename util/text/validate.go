package text

import (
	"net/url"
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

// SafeOneLine will tell if a character in the string is considered unsafe
// Currently trigger on all unicode control character
func SafeOneLine(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}

	return true
}

// ValidUrl reports whether the string is a well-formed URL using a web-safe
// scheme. Only http and https are accepted — javascript:, data:, file: and
// similar are rejected because the string may later be rendered into an
// <a href> or <img src> in the web UI, where those schemes enable XSS or
// leak local files.
func ValidUrl(s string) bool {
	if strings.Contains(s, "\n") {
		return false
	}
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return true
	}
	return false
}
