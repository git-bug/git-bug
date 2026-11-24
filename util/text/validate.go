package text

import (
	"unicode"
	"unicode/utf8"
)

// Empty tells if the string is blank: each of its code points is in blankTable.
// An empty string is blank. See values.md §7 in the spec.
func Empty(s string) bool {
	for _, r := range s {
		if !unicode.Is(blankTable, r) {
			return false
		}
	}
	return true
}

// blankTable is the fixed list of the code points of blank text: the spaces and
// line breaks, the control and format characters, the private-use characters and
// the characters that display as nothing. It doesn't follow the version of
// Unicode that Go knows, see values.md §7 in the spec.
var blankTable = &unicode.RangeTable{
	R16: []unicode.Range16{
		{Lo: 0x0000, Hi: 0x0020, Stride: 1},
		{Lo: 0x007F, Hi: 0x00A0, Stride: 1},
		{Lo: 0x00AD, Hi: 0x00AD, Stride: 1},
		{Lo: 0x034F, Hi: 0x034F, Stride: 1},
		{Lo: 0x0600, Hi: 0x0605, Stride: 1},
		{Lo: 0x061C, Hi: 0x061C, Stride: 1},
		{Lo: 0x06DD, Hi: 0x06DD, Stride: 1},
		{Lo: 0x070F, Hi: 0x070F, Stride: 1},
		{Lo: 0x0890, Hi: 0x0891, Stride: 1},
		{Lo: 0x08E2, Hi: 0x08E2, Stride: 1},
		{Lo: 0x115F, Hi: 0x1160, Stride: 1},
		{Lo: 0x1680, Hi: 0x1680, Stride: 1},
		{Lo: 0x17B4, Hi: 0x17B5, Stride: 1},
		{Lo: 0x180B, Hi: 0x180F, Stride: 1},
		{Lo: 0x2000, Hi: 0x200F, Stride: 1},
		{Lo: 0x2028, Hi: 0x202F, Stride: 1},
		{Lo: 0x205F, Hi: 0x206F, Stride: 1},
		{Lo: 0x2800, Hi: 0x2800, Stride: 1},
		{Lo: 0x3000, Hi: 0x3000, Stride: 1},
		{Lo: 0x3164, Hi: 0x3164, Stride: 1},
		{Lo: 0xE000, Hi: 0xF8FF, Stride: 1},
		{Lo: 0xFE00, Hi: 0xFE0F, Stride: 1},
		{Lo: 0xFEFF, Hi: 0xFEFF, Stride: 1},
		{Lo: 0xFFA0, Hi: 0xFFA0, Stride: 1},
		{Lo: 0xFFF0, Hi: 0xFFFB, Stride: 1},
	},
	R32: []unicode.Range32{
		{Lo: 0x110BD, Hi: 0x110BD, Stride: 1},
		{Lo: 0x110CD, Hi: 0x110CD, Stride: 1},
		{Lo: 0x13430, Hi: 0x1343F, Stride: 1},
		{Lo: 0x1BCA0, Hi: 0x1BCA3, Stride: 1},
		{Lo: 0x1D173, Hi: 0x1D17A, Stride: 1},
		{Lo: 0xE0000, Hi: 0xE0FFF, Stride: 1},
		{Lo: 0xF0000, Hi: 0xFFFFD, Stride: 1},
		{Lo: 0x100000, Hi: 0x10FFFD, Stride: 1},
	},
	LatinOffset: 3,
}

// Safe tells if the string is multi-line text: valid UTF-8, holding no character
// that isText forbids, except the tab, the line breaks and the line and paragraph
// separators. Its isolates pair up within each line. See values.md §6 in the spec.
func Safe(s string) bool {
	return isText(s, true)
}

// SafeOneLine tells if the string is text: valid UTF-8, a single line, holding
// no character that isForbidden lists, and whose isolates pair up. See values.md
// §5 in the spec.
func SafeOneLine(s string) bool {
	return isText(s, false)
}

func isText(s string, multiline bool) bool {
	if !utf8.ValidString(s) {
		return false
	}
	// the isolates open on the current line
	open := 0
	for _, r := range s {
		switch {
		case endsLine(r):
			if !multiline || open > 0 {
				return false
			}
		case r == '\t' || r == '\u2028':
			if !multiline {
				return false
			}
		case opensIsolate(r):
			open++
		case r == '\u2069':
			if open == 0 {
				return false
			}
			open--
		case isForbidden(r):
			return false
		}
	}
	return open == 0
}

// isForbidden tells if a character is never in text:
//   - the control characters, among them the line breaks and the tab;
//   - the line and paragraph separators;
//   - the bidi embeddings and overrides (U+202A-U+202E), which change the
//     direction of the text displayed after them. Bidi marks are allowed as RTL
//     text needs them, and isolates when they pair up;
//   - the surrogates, which valid UTF-8 doesn't hold;
//   - the noncharacters (U+FDD0-U+FDEF and U+xFFFE/U+xFFFF), which Unicode
//     reserves for internal use.
//
// Multi-line text allows the tab, the line breaks and the separators.
func isForbidden(r rune) bool {
	switch {
	case unicode.IsControl(r):
		return true
	case r == '\u2028' || r == '\u2029':
		return true
	case r >= '\u202a' && r <= '\u202e':
		return true
	case r >= 0xD800 && r <= 0xDFFF:
		return true
	case r >= '\ufdd0' && r <= '\ufdef':
		return true
	case r&0xFFFE == 0xFFFE:
		return true
	}
	return false
}

// endsLine tells if a character ends a line of multi-line text, for the isolates
// to pair up: a line feed, a carriage return or U+2029 PARAGRAPH SEPARATOR, as in
// the Unicode Bidirectional Algorithm.
func endsLine(r rune) bool {
	return r == '\n' || r == '\r' || r == '\u2029'
}

// opensIsolate tells if a character opens an isolate, which U+2069 closes.
func opensIsolate(r rune) bool {
	return r >= '\u2066' && r <= '\u2068'
}
