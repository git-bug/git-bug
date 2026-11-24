package text

import (
	"slices"
	"strings"
)

// Cleanup turns a string into multi-line text (see Safe): it replaces invalid
// UTF-8 with U+FFFD, removes the characters that multi-line text forbids and the
// isolates that don't pair up within their line, and trims the spaces around it.
func Cleanup(text string) string {
	// windows new line, Github, really ?
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.TrimSpace(cleanup(text, true))
}

// CleanupOneLine turns a string into text (see SafeOneLine): it replaces invalid
// UTF-8 with U+FFFD, removes the characters that text forbids, line breaks
// included, and the isolates that don't pair up, and trims the spaces around it.
func CleanupOneLine(text string) string {
	return strings.TrimSpace(cleanup(text, false))
}

func CleanupOneLineArray(texts []string) []string {
	for i := range texts {
		texts[i] = CleanupOneLine(texts[i])
	}
	return texts
}

func cleanup(text string, multiline bool) string {
	// ranging over a string reads invalid UTF-8 as U+FFFD
	kept := make([]rune, 0, len(text))
	// the indexes in kept of the isolates open on the current line
	var open []int
	// dropOpen marks the isolates left open at the end of a line, removed once
	// the whole text is read: removing each in place would copy the rest of the
	// text every time
	dropOpen := func() {
		for _, i := range open {
			kept[i] = dropped
		}
		open = open[:0]
	}
	for _, r := range text {
		switch {
		case multiline && endsLine(r):
			dropOpen()
		case multiline && (r == '\t' || r == '\u2028'):
			// allowed in multi-line text, though isForbidden lists them
		case opensIsolate(r):
			open = append(open, len(kept))
		case r == '\u2069':
			if len(open) == 0 {
				continue
			}
			open = open[:len(open)-1]
		case isForbidden(r):
			continue
		}
		kept = append(kept, r)
	}
	dropOpen()
	return string(slices.DeleteFunc(kept, func(r rune) bool { return r == dropped }))
}

// dropped marks a rune that cleanup removes. Ranging over a string never yields
// it, as it reads invalid UTF-8 as U+FFFD.
const dropped rune = -1
