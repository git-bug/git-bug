package text

import (
	"strings"
)

type Alignment int

const (
	NoAlign Alignment = iota
	AlignLeft
	AlignCenter
	AlignRight
)

// LineAlign align the given line as asked and apply the needed padding to match the given
// lineWidth, while ignoring the terminal escape sequences.
// If the given lineWidth is too small to fit the given line, it's returned without
// padding, overflowing lineWidth.
func LineAlign(line string, lineWidth int, align Alignment) string {
	switch align {
	case NoAlign:
		return line
	case AlignLeft:
		return LineAlignLeft(line, lineWidth)
	case AlignCenter:
		return LineAlignCenter(line, lineWidth)
	case AlignRight:
		return LineAlignRight(line, lineWidth)
	}
	panic("unknown alignment")
}

// LineAlignLeft align the given line on the left while ignoring the terminal escape sequences.
// If the given lineWidth is too small to fit the given line, it's returned without
// padding, overflowing lineWidth.
func LineAlignLeft(line string, lineWidth int) string {
	return TrimSpace(line)
}

// LineAlignCenter align the given line on the center and apply the needed left
// padding, while ignoring the terminal escape sequences.
// If the given lineWidth is too small to fit the given line, it's returned without
// padding, overflowing lineWidth.
func LineAlignCenter(line string, lineWidth int) string {
	trimmed := TrimSpace(line)
	totalPadLen := lineWidth - Len(trimmed)
	if totalPadLen < 0 {
		totalPadLen = 0
	}
	pad := strings.Repeat(" ", totalPadLen/2)
	return pad + trimmed
}

// LineAlignRight align the given line on the right and apply the needed left
// padding to match the given lineWidth, while ignoring the terminal escape sequences.
// If the given lineWidth is too small to fit the given line, it's returned without
// padding, overflowing lineWidth.
func LineAlignRight(line string, lineWidth int) string {
	trimmed := TrimSpace(line)
	padLen := lineWidth - Len(trimmed)
	if padLen < 0 {
		padLen = 0
	}
	pad := strings.Repeat(" ", padLen)
	return pad + trimmed
}
