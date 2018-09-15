package text

import (
	"bytes"
	"strings"
	"unicode/utf8"
)

// LeftPadMaxLine pads a string on the left by a specified amount and pads the string on the right to fill the maxLength
func LeftPadMaxLine(value string, maxValueLength, leftPad int) string {
	valueLength := utf8.RuneCountInString(value)
	if maxValueLength-leftPad >= valueLength {
		return strings.Repeat(" ", leftPad) + value + strings.Repeat(" ", maxValueLength-valueLength-leftPad)
	} else if maxValueLength-leftPad < valueLength {
		tmp := strings.Trim(value[0:maxValueLength-leftPad-3], " ") + "..."
		tmpLength := utf8.RuneCountInString(tmp)
		return strings.Repeat(" ", leftPad) + tmp + strings.Repeat(" ", maxValueLength-tmpLength-leftPad)
	}

	return value
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
