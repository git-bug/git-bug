package text

import (
	"bytes"
	"strings"

	"github.com/mattn/go-runewidth"
)

// LeftPadMaxLine pads a line on the left by a specified amount and pads the
// string on the right to fill the maxLength.
// If the given string is too long, it is truncated with an ellipsis.
// Handle properly terminal color escape code
func LeftPadMaxLine(line string, length, leftPad int) string {
	cleaned, escapes := ExtractTermEscapes(line)

	scrWidth := runewidth.StringWidth(cleaned)
	// truncate and ellipse if needed
	if scrWidth+leftPad > length {
		cleaned = runewidth.Truncate(cleaned, length-leftPad, "…")
	} else if scrWidth+leftPad < length {
		cleaned = runewidth.FillRight(cleaned, length-leftPad)
	}

	rightPart := ApplyTermEscapes(cleaned, escapes)
	pad := strings.Repeat(" ", leftPad)

	return pad + rightPart
}

// LeftPad left pad each line of the given text
func LeftPadLines(text string, leftPad int) string {
	var result bytes.Buffer

	pad := strings.Repeat(" ", leftPad)

	lines := strings.Split(text, "\n")

	for i, line := range lines {
		result.WriteString(pad)
		result.WriteString(line)

		// no additional line break at the end
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}
