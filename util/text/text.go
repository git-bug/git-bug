package text

import (
	"github.com/mattn/go-runewidth"
	"strings"
	"unicode/utf8"
)

// Wrap a text for an exact line size
// Handle properly terminal color escape code
func Wrap(text string, lineWidth int) (string, int) {
	return WrapLeftPadded(text, lineWidth, 0)
}

// Wrap a text for an exact line size with a left padding
// Handle properly terminal color escape code
func WrapLeftPadded(text string, lineWidth int, leftPad int) (string, int) {
	pad := strings.Repeat(" ", leftPad)
	var lines []string
	nbLine := 0

	// tabs are formatted as 4 spaces
	text = strings.Replace(text, "\t", "    ", -1)
	for _, line := range strings.Split(text, "\n") {
		if line == "" || strings.TrimSpace(line) == "" {
			lines = append(lines, "")
			nbLine++
		} else {
			wrapped := softwrapLine(line, lineWidth-leftPad)
			firstLine := true
			for _, seg := range strings.Split(wrapped, "\n") {
				if firstLine {
					lines = append(lines, pad+strings.TrimRight(seg, " "))
					firstLine = false
				} else {
					lines = append(lines, pad+strings.TrimSpace(seg))
				}
				nbLine++
			}
		}
	}
	return strings.Join(lines, "\n"), nbLine
}

type EscapeItem struct {
	item string
	pos  int
}

func recordTermEscape(s string) (string, []EscapeItem) {
	var result []EscapeItem
	var newStr string

	pos := 0
	item := ""
	occupiedRuneCount := 0
	inEscape := false
	for i, r := range []rune(s) {
		if r == '\x1b' {
			pos = i
			item = string(r)
			inEscape = true
			continue
		}
		if inEscape {
			item += string(r)
			if r == 'm' {
				result = append(result, EscapeItem{item: item, pos: pos - occupiedRuneCount})
				occupiedRuneCount += utf8.RuneCountInString(item)
				inEscape = false
			}
			continue
		}
		newStr += string(r)
	}

	return newStr, result
}

func replayTermEscape(s string, sequence []EscapeItem) string {
	if len(sequence) == 0 {
		return string(s)
	}
	// Assume the original string contains no new line and the wrapped only insert
	// new lines. So that we can recover the position where we insert the term
	// escapes.
	var out string = ""

	currPos := 0
	currItem := 0
	for _, r := range []rune(s) {
		if currItem < len(sequence) && currPos == sequence[currItem].pos {
			if r == '\n' {
				out += "\n" + sequence[currItem].item
			} else {
				out += sequence[currItem].item + string(r)
				currPos++
			}
			currItem++
		} else {
			if r != '\n' {
				currPos++
			}
			out += string(r)
		}
	}

	return out
}

// Break a line into several lines so that each line consumes at most 'w' cells.
// Lines break at group of white spaces and multibyte chars. Nothing is removed
// from the line so that it behaves like a softwrap.
//
// Required: The line shall not contain '\n' (so it is a single line).
//
// WRAPPING ALGORITHM: The line is broken into non-breakable groups, then line
// breaks ("\n") is inserted between these groups so that the total length
// between breaks does not exceed the required width. Words that are longer than
// the width is broken into several words as `M+M+...+N`.
func softwrapLine(s string, w int) string {
	newStr, termSeqs := recordTermEscape(s)

	const (
		WIDE_CHAR     = iota
		INVISIBLE     = iota
		SHORT_UNICODE = iota
		SPACE         = iota
		VISIBLE_ASCII = iota
		NONE          = iota
	)

	// In order to simplify the terminal color sequence handling, we first strip
	// them out of the text and record their position, then do the wrap. After
	// that, we insert back these sequences.
	runeType := func(r rune) int {
		rw := runewidth.RuneWidth(r)
		if rw > 1 {
			return WIDE_CHAR
		} else if rw == 0 {
			return INVISIBLE
		} else if r > 127 {
			return SHORT_UNICODE
		} else if r == ' ' {
			return SPACE
		} else {
			return VISIBLE_ASCII
		}
	}

	var chunks []string
	var word string
	wordType := NONE
	flushWord := func() {
		chunks = append(chunks, word)
		word = ""
		wordType = NONE
	}
	for _, r := range []rune(newStr) {
		// A WIDE_CHAR itself constitutes a group.
		thisType := runeType(r)
		if thisType == WIDE_CHAR {
			if wordType != NONE {
				flushWord()
			}
			chunks = append(chunks, string(r))
			continue
		}
		// Other type of groups starts with a char of that type, and ends with a
		// char with different type or end of string.
		if thisType != wordType {
			if wordType != NONE {
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

	var line string = ""
	var width int = 0
	// Reverse the chunk array so we can use it as a stack.
	for i, j := 0, len(chunks)-1; i < j; i, j = i+1, j-1 {
		chunks[i], chunks[j] = chunks[j], chunks[i]
	}
	for len(chunks) > 0 {
		thisWord := chunks[len(chunks)-1]
		wl := wordLen(thisWord)
		if width+wl <= w {
			line += chunks[len(chunks)-1]
			chunks = chunks[:len(chunks)-1]
			width += wl
			if width == w && len(chunks) > 0 {
				line += "\n"
				width = 0
			}
		} else if wl > w {
			left, right := splitWord(chunks[len(chunks)-1], w)
			line += left + "\n"
			chunks[len(chunks)-1] = right
			width = 0
		} else {
			line += "\n"
			width = 0
		}
	}

	line = replayTermEscape(line, termSeqs)
	return line
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

func minInt(a, b int) int {
	if a > b {
		return b
	}
	return a
}
