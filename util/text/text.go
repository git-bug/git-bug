package text

import (
	"bytes"
	"github.com/mattn/go-runewidth"
	"strings"
)

// Wrap a text for an exact line size
// Handle properly terminal color escape code
func Wrap(text string, lineWidth int) (string, int) {
	return WrapLeftPadded(text, lineWidth, 0)
}

// Wrap a text for an exact line size with a left padding
// Handle properly terminal color escape code
func WrapLeftPadded(text string, lineWidth int, leftPad int) (string, int) {
	var textBuffer bytes.Buffer
	nbLine := 0
	pad := strings.Repeat(" ", leftPad)

	// tabs are formatted as 4 spaces
	text = strings.Replace(text, "\t", "    ", 4)
	wrapped := wrapText(text, lineWidth-leftPad)
	for _, line := range strings.Split(wrapped, "\n") {
		textBuffer.WriteString(pad + line)
		textBuffer.WriteString("\n")
		nbLine++
	}
	return textBuffer.String(), nbLine
}

// Wrap text so that each line fills at most w cells. Lines break at word
// boundary or multibyte chars.
//
// Wrapping Algorithm: Treat the text as a sequence of words, with each word be
// an alphanumeric word, or a multibyte char. We scan through the text and
// construct the word, and flush the word into the paragraph once a word is
// ready. A word is ready when a word boundary is detected: a boundary char such
// as '\n', '\t', and ' ' is encountered; a multibyte char is found; or a
// multibyte to single-byte switch is encountered. '\n' is handled in a special
// manner.
func wrapText(s string, w int) string {
	word := ""
	out := ""

	width := 0
	firstWord := true
	isMultibyteWord := false

	flushWord := func() {
		wl := wordLen(word)
		if isMultibyteWord {
			if width+wl > w {
				out += "\n" + word
				width = wl
			} else {
				out += word
				width += wl
			}
		} else {
			if width == 0 {
				out += word
				width += wl
			} else if width+wl+1 > w {
				out += "\n" + word
				width = wl
			} else {
				out += " " + word
				width += wl + 1
			}
		}
		word = ""
	}

	for _, r := range []rune(s) {
		cw := runewidth.RuneWidth(r)
		if firstWord {
			word = string(r)
			isMultibyteWord = cw > 1
			firstWord = false
			continue
		}
		if r == '\n' {
			flushWord()
			out += "\n"
			width = 0
		} else if r == ' ' || r == '\t' {
			flushWord()
		} else if cw > 1 {
			flushWord()
			word = string(r)
			isMultibyteWord = true
			word = string(r)
		} else if cw == 1 && isMultibyteWord {
			flushWord()
			word = string(r)
			isMultibyteWord = false
		} else {
			word += string(r)
		}
	}
	// The text may end without newlines, ensure flushing it or we can lose the
	// last word.
	flushWord()

	return out
}

// wordLen return the length of a word, while ignoring the terminal escape
// sequences
func wordLen(word string) int {
	length := 0
	escape := false

	for _, char := range word {
		if char == '\x1b' {
			escape = true
		}
		if !escape {
			length += runewidth.RuneWidth(rune(char))
		}
		if char == 'm' {
			escape = false
		}
	}

	return length
}
