package text

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
)

func Cleanup(text string) string {
	// windows new line, Github, really ?
	text = strings.Replace(text, "\r\n", "\n", -1)

	// remove all unicode control characters except
	// '\n', '\r' and '\t'
	t := runes.Remove(runes.Predicate(func(r rune) bool {
		switch r {
		case '\r', '\n', '\t':
			return false
		}
		return unicode.IsControl(r)
	}))
	sanitized, _, err := transform.String(t, text)
	if err != nil {
		// transform.String should never return an error as our transformer doesn't returns one.
		// Confirmed with fuzzing.
		panic(err)
	}

	// trim extra new line not displayed in the github UI but still present in the data
	return strings.TrimSpace(sanitized)
}

func CleanupOneLine(text string) string {
	// remove all unicode control characters *including*
	// '\n', '\r' and '\t'
	t := runes.Remove(runes.Predicate(unicode.IsControl))
	sanitized, _, err := transform.String(t, text)
	if err != nil {
		// transform.String should never return an error as our transformer doesn't returns one.
		// Confirmed with fuzzing.
		panic(err)
	}

	// trim extra new line not displayed in the github UI but still present in the data
	return strings.TrimSpace(sanitized)
}

func CleanupOneLineArray(texts []string) []string {
	for i := range texts {
		texts[i] = CleanupOneLine(texts[i])
	}
	return texts
}
