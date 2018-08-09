package util

import (
	"bytes"
	"regexp"
	"strings"
)

func WordWrap(text string, lineWidth int) (string, int) {
	words := strings.Fields(strings.TrimSpace(text))
	if len(words) == 0 {
		return "", 1
	}
	lines := 1
	wrapped := words[0]
	spaceLeft := lineWidth - len(wrapped)
	for _, word := range words[1:] {
		if len(word)+1 > spaceLeft {
			wrapped += "\n" + word
			spaceLeft = lineWidth - len(word)
			lines++
		} else {
			wrapped += " " + word
			spaceLeft -= 1 + len(word)
		}
	}

	return wrapped, lines
}

func TextWrap(text string, lineWidth int) (string, int) {
	var textBuffer bytes.Buffer
	var lineBuffer bytes.Buffer
	nbLine := 1
	firstLine := true

	// tabs are formatted as 4 spaces
	text = strings.Replace(text, "\t", "    ", 4)

	re := regexp.MustCompile(`(\x1b\[\d+m)?([^\x1b]*)(\x1b\[\d+m)?`)

	for _, line := range strings.Split(text, "\n") {
		spaceLeft := lineWidth

		if !firstLine {
			textBuffer.WriteString("\n")
			nbLine++
		}

		firstWord := true

		for _, word := range strings.Split(line, " ") {
			prefix := ""
			suffix := ""

			matches := re.FindStringSubmatch(word)
			if matches != nil && (matches[1] != "" || matches[3] != "") {
				// we have a color escape sequence
				prefix = matches[1]
				word = matches[2]
				suffix = matches[3]
			}

			if spaceLeft > len(word) {
				if !firstWord {
					lineBuffer.WriteString(" ")
					spaceLeft -= 1
				}
				lineBuffer.WriteString(prefix + word + suffix)
				spaceLeft -= len(word)
				firstWord = false
			} else {
				if len(word) > lineWidth {
					for len(word) > 0 {
						l := minInt(spaceLeft, len(word))
						part := prefix + word[:l]
						prefix = ""
						word = word[l:]

						lineBuffer.WriteString(part)
						textBuffer.Write(lineBuffer.Bytes())
						lineBuffer.Reset()

						if len(word) > 0 {
							textBuffer.WriteString("\n")
							nbLine++
						}

						spaceLeft = lineWidth
					}
				} else {
					textBuffer.WriteString(strings.TrimRight(lineBuffer.String(), " "))
					textBuffer.WriteString("\n")
					lineBuffer.Reset()
					lineBuffer.WriteString(prefix + word + suffix)
					firstWord = false
					spaceLeft = lineWidth - len(word)
					nbLine++
				}
			}
		}
		textBuffer.WriteString(strings.TrimRight(lineBuffer.String(), " "))
		lineBuffer.Reset()
		firstLine = false
	}

	return textBuffer.String(), nbLine
}

func minInt(a, b int) int {
	if a > b {
		return b
	}
	return a
}
