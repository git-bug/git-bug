package text

import (
	"bytes"
	"fmt"
	"github.com/mattn/go-runewidth"
	"strings"
)

// LeftPadMaxLine pads a string on the left by a specified amount and pads the
// string on the right to fill the maxLength
func LeftPadMaxLine(text string, length, leftPad int) string {
	var rightPart string = text

	scrWidth := runewidth.StringWidth(text)
	// truncate and ellipse if needed
	if scrWidth+leftPad > length {
		rightPart = runewidth.Truncate(text, length-leftPad, "...")
	} else if scrWidth+leftPad < length {
		rightPart = runewidth.FillRight(text, length-leftPad)
	}

	return fmt.Sprintf("%s%s",
		strings.Repeat(" ", leftPad),
		rightPart,
	)
}

// LeftPad left pad each line of the given text
func LeftPad(text string, leftPad int) string {
	var result bytes.Buffer

	pad := strings.Repeat(" ", leftPad)

	for _, line := range strings.Split(text, "\n") {
		result.WriteString(pad)
		result.WriteString(line)
		result.WriteString("\n")
	}

	return result.String()
}
