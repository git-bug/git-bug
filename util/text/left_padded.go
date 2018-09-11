package text

import (
	"strings"
	"unicode/utf8"
)

// LeftPaddedString pads a string on the left by a specified amount and pads the string on the right to fill the maxLength
func LeftPaddedString(value string, maxValueLength, padAmount int) string {
	valueLength := utf8.RuneCountInString(value)
	if maxValueLength-padAmount >= valueLength {
		return strings.Repeat(" ", padAmount) + value + strings.Repeat(" ", maxValueLength-valueLength-padAmount)
	} else if maxValueLength-padAmount < valueLength {
		tmp := strings.Trim(value[0:maxValueLength-padAmount-3], " ") + "..."
		tmpLength := utf8.RuneCountInString(tmp)
		return strings.Repeat(" ", padAmount) + tmp + strings.Repeat(" ", maxValueLength-tmpLength-padAmount)
	}

	return value
}
