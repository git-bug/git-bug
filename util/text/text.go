package text

import (
	"bytes"
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
	var lineBuffer bytes.Buffer
	nbLine := 1
	firstLine := true
	pad := strings.Repeat(" ", leftPad)

	// tabs are formatted as 4 spaces
	text = strings.Replace(text, "\t", "    ", 4)

	for _, line := range strings.Split(text, "\n") {
		spaceLeft := lineWidth - leftPad

		if !firstLine {
			textBuffer.WriteString("\n")
			nbLine++
		}

		firstWord := true

		for _, word := range strings.Split(line, " ") {
			wordLength := wordLen(word)

			if !firstWord {
				lineBuffer.WriteString(" ")
				spaceLeft -= 1

				if spaceLeft <= 0 {
					textBuffer.WriteString(pad + strings.TrimRight(lineBuffer.String(), " "))
					textBuffer.WriteString("\n")
					lineBuffer.Reset()
					spaceLeft = lineWidth - leftPad
					nbLine++
					firstLine = false
				}
			}

			// Word fit in the current line
			if spaceLeft >= wordLength {
				lineBuffer.WriteString(word)
				spaceLeft -= wordLength
				firstWord = false
			} else {
				// Break a word longer than a line
				if wordLength > lineWidth {
					for wordLength > 0 && len(word) > 0 {
						l := minInt(spaceLeft, wordLength)
						part, leftover := splitWord(word, l)
						word = leftover
						wordLength = wordLen(word)

						lineBuffer.WriteString(part)
						textBuffer.WriteString(pad)
						textBuffer.Write(lineBuffer.Bytes())
						lineBuffer.Reset()

						spaceLeft -= l

						if spaceLeft <= 0 {
							textBuffer.WriteString("\n")
							nbLine++
							spaceLeft = lineWidth - leftPad
						}
					}
				} else {
					// Normal break
					textBuffer.WriteString(pad + strings.TrimRight(lineBuffer.String(), " "))
					textBuffer.WriteString("\n")
					lineBuffer.Reset()
					lineBuffer.WriteString(word)
					firstWord = false
					spaceLeft = lineWidth - wordLength
					nbLine++
				}
			}
		}

		textBuffer.WriteString(pad + strings.TrimRight(lineBuffer.String(), " "))
		lineBuffer.Reset()
		firstLine = false
	}

	return textBuffer.String(), nbLine
}

func wordLen(word string) int {
	length := 0
	escape := false

	for _, char := range word {
		if char == '\x1b' {
			escape = true
		}

		if !escape {
			length++
		}

		if char == 'm' {
			escape = false
		}
	}

	return length
}

func splitWord(word string, length int) (string, string) {
	result := ""
	added := 0
	escape := false

	if length == 0 {
		return "", word
	}

	for _, char := range word {
		if char == '\x1b' {
			escape = true
		}

		result += string(char)

		if !escape {
			added++
			if added == length {
				break
			}
		}

		if char == 'm' {
			escape = false
		}
	}

	leftover := word[len(result):]

	return result, leftover
}

func minInt(a, b int) int {
	if a > b {
		return b
	}
	return a
}
