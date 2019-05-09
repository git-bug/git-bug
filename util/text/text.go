package text

import (
	"github.com/mattn/go-runewidth"
	"strings"
	"unicode/utf8"
)

// Force runewidth not to treat ambiguous runes as wide chars, so that things
// like unicode ellipsis/up/down/left/right glyphs can have correct runewidth
// and can be displayed correctly in terminals.
func init() {
	runewidth.DefaultCondition.EastAsianWidth = false
}

// Wrap a text for an exact line size
// Handle properly terminal color escape code
func Wrap(text string, lineWidth int) (string, int) {
	return WrapLeftPadded(text, lineWidth, 0)
}

// Wrap a text for an exact line size with a left padding
// Handle properly terminal color escape code
func WrapLeftPadded(text string, lineWidth int, leftPad int) (string, int) {
	var lines []string
	nbLine := 0
	pad := strings.Repeat(" ", leftPad)

	// tabs are formatted as 4 spaces
	text = strings.Replace(text, "\t", "    ", -1)
	// NOTE: text is first segmented into lines so that softwrapLine can handle.
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
//
func softwrapLine(line string, textWidth int) string {
	// NOTE: terminal escapes are stripped out of the line so the algorithm is
	// simpler. Do not try to mix them in the wrapping algorithm, as it can get
	// complicated quickly.
	line1, termEscapes := extractTermEscapes(line)

	chunks := segmentLine(line1)
	// Reverse the chunk array so we can use it as a stack.
	for i, j := 0, len(chunks)-1; i < j; i, j = i+1, j-1 {
		chunks[i], chunks[j] = chunks[j], chunks[i]
	}
	var line2 string = ""
	var width int = 0
	for len(chunks) > 0 {
		thisWord := chunks[len(chunks)-1]
		wl := wordLen(thisWord)
		if width+wl <= textWidth {
			line2 += chunks[len(chunks)-1]
			chunks = chunks[:len(chunks)-1]
			width += wl
			if width == textWidth && len(chunks) > 0 {
				// NOTE: new line begins when current line is full and there are more
				// chunks to come.
				line2 += "\n"
				width = 0
			}
		} else if wl > textWidth {
			// NOTE: By default, long words are splited to fill the remaining space.
			// But if the long words is the first non-space word in the middle of the
			// line, preceeding spaces shall not be counted in word spliting.
			splitWidth := textWidth - width
			if strings.HasSuffix(line2, "\n"+strings.Repeat(" ", width)) {
				splitWidth += width
			}
			left, right := splitWord(chunks[len(chunks)-1], splitWidth)
			chunks[len(chunks)-1] = right
			line2 += left + "\n"
			width = 0
		} else {
			line2 += "\n"
			width = 0
		}
	}

	line3 := applyTermEscapes(line2, termEscapes)
	return line3
}

// EscapeItem: Storage of terminal escapes in a line. 'item' is the actural
// escape command, and 'pos' is the index in the rune array where the 'item'
// shall be inserted back. For example, the escape item in "F\x1b33mox" is
// {"\x1b33m", 1}.
type escapeItem struct {
	item string
	pos  int
}

// Extract terminal escapes out of a line, returns a new line without terminal
// escapes and a slice of escape items. The terminal escapes can be inserted
// back into the new line at rune index 'item.pos' to recover the original line.
//
// Required: The line shall not contain "\n"
//
func extractTermEscapes(line string) (string, []escapeItem) {
	var termEscapes []escapeItem
	var line1 string

	pos := 0
	item := ""
	occupiedRuneCount := 0
	inEscape := false
	for i, r := range []rune(line) {
		if r == '\x1b' {
			pos = i
			item = string(r)
			inEscape = true
			continue
		}
		if inEscape {
			item += string(r)
			if r == 'm' {
				termEscapes = append(termEscapes, escapeItem{item, pos - occupiedRuneCount})
				occupiedRuneCount += utf8.RuneCountInString(item)
				inEscape = false
			}
			continue
		}
		line1 += string(r)
	}

	return line1, termEscapes
}

// Apply the extracted terminal escapes to the edited line. The only edit
// allowed is to insert "\n" like that in softwrapLine. Callers shall ensure
// this since this function is not able to check it.
func applyTermEscapes(line string, escapes []escapeItem) string {
	if len(escapes) == 0 {
		return line
	}

	var out string = ""

	currPos := 0
	currItem := 0
	for _, r := range line {
		if currItem < len(escapes) && currPos == escapes[currItem].pos {
			// NOTE: We avoid terminal escapes at the end of a line by move them one
			// pass the end of line, so that algorithms who trim right spaces are
			// happy. But algorithms who trim left spaces are still unhappy.
			if r == '\n' {
				out += "\n" + escapes[currItem].item
			} else {
				out += escapes[currItem].item + string(r)
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

	// Don't forget the trailing escape, if any.
	if currItem == len(escapes)-1 && currPos == escapes[currItem].pos {
		out += escapes[currItem].item
	}

	return out
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

// Rune categories
//
// These categories are so defined that each category forms a non-breakable
// chunk. It IS NOT the same as unicode code point categories.
//
const (
	none int = iota
	wideChar
	invisible
	shortUnicode
	space
	visibleAscii
)

// Determine the category of a rune.
func runeType(r rune) int {
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
