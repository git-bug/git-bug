package text

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// Force runewidth not to treat ambiguous runes as wide chars, so that things
// like unicode ellipsis/up/down/left/right glyphs can have correct runewidth
// and can be displayed correctly in terminals.
func init() {
	runewidth.DefaultCondition.EastAsianWidth = false
}

// Wrap a text for a given line size.
// Handle properly terminal color escape code
func Wrap(text string, lineWidth int) (string, int) {
	return WrapLeftPadded(text, lineWidth, 0)
}

// WrapLeftPadded wrap a text for a given line size with a left padding.
// Handle properly terminal color escape code
func WrapLeftPadded(text string, lineWidth int, leftPad int) (string, int) {
	pad := strings.Repeat(" ", leftPad)
	return WrapWithPad(text, lineWidth, pad)
}

// WrapWithPad wrap a text for a given line size with a custom left padding
// Handle properly terminal color escape code
func WrapWithPad(text string, lineWidth int, pad string) (string, int) {
	return WrapWithPadIndent(text, lineWidth, pad, pad)
}

// WrapWithPad wrap a text for a given line size with a custom left padding
// This function also align the result depending on the requested alignment.
// Handle properly terminal color escape code
func WrapWithPadAlign(text string, lineWidth int, pad string, align Alignment) (string, int) {
	return WrapWithPadIndentAlign(text, lineWidth, pad, pad, align)
}

// WrapWithPadIndent wrap a text for a given line size with a custom left padding
// and a first line indent. The padding is not effective on the first line, indent
// is used instead, which allow to implement indents and outdents.
// Handle properly terminal color escape code
func WrapWithPadIndent(text string, lineWidth int, indent string, pad string) (string, int) {
	return WrapWithPadIndentAlign(text, lineWidth, indent, pad, NoAlign)
}

// WrapWithPadIndentAlign wrap a text for a given line size with a custom left padding
// and a first line indent. The padding is not effective on the first line, indent
// is used instead, which allow to implement indents and outdents.
// This function also align the result depending on the requested alignment.
// Handle properly terminal color escape code
func WrapWithPadIndentAlign(text string, lineWidth int, indent string, pad string, align Alignment) (string, int) {
	var lines []string
	nbLine := 0

	// Start with the indent
	padStr := indent
	padLen := Len(indent)

	// tabs are formatted as 4 spaces
	text = strings.Replace(text, "\t", "    ", -1)

	// NOTE: text is first segmented into lines so that softwrapLine can handle.
	for i, line := range strings.Split(text, "\n") {
		// on the second line, use the padding instead
		if i == 1 {
			padStr = pad
			padLen = Len(pad)
		}

		if line == "" || strings.TrimSpace(line) == "" {
			// nothing in the line, we just add the non-empty part of the padding
			lines = append(lines, strings.TrimRight(padStr, " "))
			nbLine++
			continue
		}

		wrapped := softwrapLine(line, lineWidth-padLen)
		split := strings.Split(wrapped, "\n")

		if i == 0 && len(split) > 1 {
			// the very first line got wrapped
			// that means we need to switch to the normal padding
			// use the first wrapped line, ignore everything else and
			// wrap the remaining of the line with the normal padding.

			content := LineAlign(strings.TrimRight(split[0], " "), lineWidth-padLen, align)
			lines = append(lines, padStr+content)
			nbLine++
			line = strings.TrimPrefix(line, split[0])
			line = strings.TrimLeft(line, " ")

			padStr = pad
			padLen = Len(pad)
			wrapped = softwrapLine(line, lineWidth-padLen)
			split = strings.Split(wrapped, "\n")
		}

		for j, seg := range split {
			if j == 0 {
				// keep the left padding of the wrapped line
				content := LineAlign(strings.TrimRight(seg, " "), lineWidth-padLen, align)
				lines = append(lines, padStr+content)
			} else {
				content := LineAlign(strings.TrimSpace(seg), lineWidth-padLen, align)
				lines = append(lines, padStr+content)
			}
			nbLine++
		}
	}

	return strings.Join(lines, "\n"), nbLine
}

// Break a line into several lines so that each line consumes at most
// 'textWidth' cells.  Lines break at groups of white spaces and multibyte
// chars. Nothing is removed from the original text so that it behaves like a
// softwrap.
//
// Required: The line shall not contain '\n'
//
// WRAPPING ALGORITHM: The line is broken into non-breakable chunks, then line
// breaks ("\n") are inserted between these groups so that the total length
// between breaks does not exceed the required width. Words that are longer than
// the textWidth are broken into pieces no longer than textWidth.
func softwrapLine(line string, textWidth int) string {
	escaped, escapes := ExtractTermEscapes(line)

	chunks := segmentLine(escaped)
	// Reverse the chunk array so we can use it as a stack.
	for i, j := 0, len(chunks)-1; i < j; i, j = i+1, j-1 {
		chunks[i], chunks[j] = chunks[j], chunks[i]
	}

	// for readability, minimal implementation of a stack:

	pop := func() string {
		result := chunks[len(chunks)-1]
		chunks = chunks[:len(chunks)-1]
		return result
	}

	push := func(chunk string) {
		chunks = append(chunks, chunk)
	}

	peek := func() string {
		return chunks[len(chunks)-1]
	}

	empty := func() bool {
		return len(chunks) == 0
	}

	var out strings.Builder

	// helper to write in the output while interleaving the escape
	// sequence at the correct places.
	// note: the final algorithm will add additional line break in the original
	// text. Those line break are *not* fed to this helper so the positions don't
	// need to be offset, which make the whole thing much easier.
	currPos := 0
	currItem := 0
	outputString := func(s string) {
		for _, r := range s {
			for currItem < len(escapes) && currPos == escapes[currItem].Pos {
				out.WriteString(escapes[currItem].Item)
				currItem++
			}
			out.WriteRune(r)
			currPos++
		}
	}

	width := 0

	for !empty() {
		wl := Len(peek())

		if width+wl <= textWidth {
			// the chunk fit in the available space
			outputString(pop())
			width += wl
			if width == textWidth && !empty() {
				// only add line break when there is more chunk to come
				out.WriteRune('\n')
				width = 0
			}
		} else if wl > textWidth {
			// words too long for a full line are split to fill the remaining space.
			// But if the long words is the first non-space word in the middle of the
			// line, preceding spaces shall not be counted in word splitting.
			splitWidth := textWidth - width
			if strings.HasSuffix(out.String(), "\n"+strings.Repeat(" ", width)) {
				splitWidth += width
			}
			left, right := splitWord(pop(), splitWidth)
			// remainder is pushed back to the stack for next round
			push(right)
			outputString(left)
			out.WriteRune('\n')
			width = 0
		} else {
			// normal line overflow, we add a line break and try again
			out.WriteRune('\n')
			width = 0
		}
	}

	// Don't forget the trailing escapes, if any.
	for currItem < len(escapes) && currPos >= escapes[currItem].Pos {
		out.WriteString(escapes[currItem].Item)
		currItem++
	}

	return out.String()
}

// Segment a line into chunks, where each chunk consists of chars with the same
// type and is not breakable.
func segmentLine(s string) []string {
	var chunks []string

	var word string
	wordType := none
	flushWord := func() {
		chunks = append(chunks, word)
		word = ""
		wordType = none
	}

	for _, r := range s {
		// A WIDE_CHAR itself constitutes a chunk.
		thisType := runeType(r)
		if thisType == wideChar {
			if wordType != none {
				flushWord()
			}
			chunks = append(chunks, string(r))
			continue
		}
		// Other type of chunks starts with a char of that type, and ends with a
		// char with different type or end of string.
		if thisType != wordType {
			if wordType != none {
				flushWord()
			}
			word = string(r)
			wordType = thisType
		} else {
			word += string(r)
		}
	}
	if word != "" {
		flushWord()
	}

	return chunks
}

type RuneType int

// Rune categories
//
// These categories are so defined that each category forms a non-breakable
// chunk. It IS NOT the same as unicode code point categories.
const (
	none RuneType = iota
	wideChar
	invisible
	shortUnicode
	space
	visibleAscii
)

// Determine the category of a rune.
func runeType(r rune) RuneType {
	rw := runewidth.RuneWidth(r)
	if rw > 1 {
		return wideChar
	} else if rw == 0 {
		return invisible
	} else if r > 127 {
		return shortUnicode
	} else if r == ' ' {
		return space
	} else {
		return visibleAscii
	}
}

// splitWord split a word at the given length, while ignoring the terminal escape sequences
func splitWord(word string, length int) (string, string) {
	runes := []rune(word)
	var result []rune
	added := 0
	escape := false

	if length == 0 {
		return "", word
	}

	for _, r := range runes {
		if r == '\x1b' {
			escape = true
		}

		width := runewidth.RuneWidth(r)
		if width+added > length {
			// wide character made the length overflow
			break
		}

		result = append(result, r)

		if !escape {
			added += width
			if added >= length {
				break
			}
		}

		if r == 'm' {
			escape = false
		}
	}

	leftover := runes[len(result):]

	return string(result), string(leftover)
}
