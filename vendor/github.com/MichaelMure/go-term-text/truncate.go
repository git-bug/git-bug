package text

import "github.com/mattn/go-runewidth"

// TruncateMax truncate a line if its length is greater
// than the given length. Otherwise, the line is returned
// as is. If truncating occur, an ellipsis is inserted at
// the end.
// Handle properly terminal color escape code
func TruncateMax(line string, length int) string {
	if length <= 0 {
		return "…"
	}

	l := Len(line)
	if l <= length || l == 0 {
		return line
	}

	cleaned, escapes := ExtractTermEscapes(line)
	truncated := runewidth.Truncate(cleaned, length-1, "")

	return ApplyTermEscapes(truncated, escapes) + "…"
}
