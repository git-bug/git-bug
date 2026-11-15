package text

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	t.Run("spec examples", func(t *testing.T) {
		// the examples of testdata/text.json in the spec; the lone surrogate,
		// which a Go string can't hold, is given as the bytes of its encoding
		cases := []struct {
			name      string
			input     string
			text      bool
			multiline bool
			blank     bool
		}{
			{"a word", "crash", true, true, false},
			{"empty", "", true, true, true},
			{"spaces only", "   ", true, true, true},
			{"a no-break space and an ideographic space", "\u00a0\u3000", true, true, true},
			{"a zero width space, of category Cf", "\u200b", true, true, true},
			{"a private-use character, of category Co", "\ue000", true, true, true},
			{"the last private-use character", "\U0010fffd", true, true, true},
			{"a Hangul filler, of category Lo, displays as nothing", "\u3164", true, true, true},
			{"a braille pattern blank, of category So, displays as nothing", "\u2800", true, true, true},
			{"a variation selector alone", "\ufe0f", true, true, true},
			{"a combining grapheme joiner, a default ignorable code point", "\u034f", true, true, true},
			{"a zero width no-break space, or byte order mark", "\ufeff", true, true, true},
			{"a soft hyphen", "\u00ad", true, true, true},
			{"reserved, a default ignorable code point: in the list", "\u2065", true, true, true},
			{"a followed by a tag character: a is not blank", "a\U000e0041", true, true, false},
			{"a zero width space between letters", "a\u200bb", true, true, false},
			{"a combining mark alone, of category M", "\u0301", true, true, false},
			{"é as U+00E9", "\u00e9pique", true, true, false},
			{"é as e followed by U+0301", "e\u0301pique", true, true, false},
			{"U+1F41B, 4 bytes in UTF-8", "\U0001f41b", true, true, false},
			{"the replacement character, of category So", "\ufffd", true, true, false},
			{"a code point unassigned in Unicode 17.0.0, outside the list", "\u0378", true, true, false},
			{"unassigned, the last code point of plane 1 before its noncharacters", "\U0001fffd", true, true, false},
			{"a right-to-left mark is allowed", "\u05d0\u200f", true, true, false},
			{"U+00A0, the first code point after the control characters", "\u00a0", true, true, true},
			{"U+202F, the first code point after the overrides", "\u202f", true, true, true},
			{"U+FDCF, the code point before the noncharacters", "\ufdcf", true, true, false},
			{"U+FDF0, the code point after the noncharacters", "\ufdf0", true, true, false},
			{"an isolate that is closed", "a\u2066b\u2069", true, true, false},
			{"nested isolates, each closed", "\u2067a\u2068b\u2069c\u2069", true, true, false},
			{"an isolate left open", "a\u2067b", false, false, false},
			{"U+2069 closes no isolate", "a\u2069b", false, false, false},
			{"U+2069 before the isolate it would close", "\u2069\u2067a", false, false, false},
			{"isolates closed on their own line", "\u2067a\u2069\n\u2068b\u2069", false, true, false},
			{"an isolate left open at the end of its line", "\u2067a\nb\u2069", false, false, false},
			{"U+2028 does not end a line", "\u2067a\u2028b\u2069", false, true, false},
			{"U+2029 ends a line", "\u2067a\u2029b\u2069", false, false, false},
			{"a tab", "a\tb", false, true, false},
			{"a line feed", "a\nb", false, true, false},
			{"a carriage return and a line feed", "a\r\nb", false, true, false},
			{"line breaks, a tab and a space", "\n\t ", false, true, true},
			{"a line separator: multi-line only", "a\u2028b", false, true, false},
			{"a line separator alone", "\u2028", false, true, true},
			{"a paragraph separator: multi-line only", "a\u2029b", false, true, false},
			{"a vertical tab, a control character", "a\vb", false, false, false},
			{"a NUL", "a\x00b", false, false, false},
			{"the last control character of the first range", "a\x1fb", false, false, false},
			{"DEL", "a\x7fb", false, false, false},
			{"the first control character of the second range", "a\u0080b", false, false, false},
			{"a next line, a control character", "a\u0085b", false, false, false},
			{"the last control character", "a\u009fb", false, false, false},
			{"a left-to-right embedding", "a\u202ab", false, false, false},
			{"a right-to-left override", "a\u202eb", false, false, false},
			{"a lone surrogate; a JSON decoder may replace it with U+FFFD, see values.md §2", "a\xed\xa0\x80b", false, false, false},
			{"the first noncharacter of the range", "a\ufdd0b", false, false, false},
			{"the last noncharacter of the range", "a\ufdefb", false, false, false},
			{"a noncharacter, the last but one code point of plane 0", "a\ufffeb", false, false, false},
			{"a noncharacter, the last code point of plane 0", "a\uffffb", false, false, false},
			{"U+1FFFE, a noncharacter", "a\U0001fffe", false, false, false},
			{"U+10FFFF, a noncharacter", "a\U0010ffff", false, false, false},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				require.Equal(t, c.text, SafeOneLine(c.input), "text")
				require.Equal(t, c.multiline, Safe(c.input), "multi-line text")
				require.Equal(t, c.blank, Empty(c.input), "blank")
			})
		}
	})

	t.Run("more examples", func(t *testing.T) {
		cases := []struct {
			name      string
			input     string
			text      bool
			multiline bool
		}{
			{"a bell, a control character", "a\x07b", false, false},
			{"invalid UTF-8", "\xA0\xA1", false, false},
			{"truncated UTF-8", "caf\xC3", false, false},
			// format characters are legitimate: emoji ZWJ sequences, ZWNJ in
			// Persian and Indic scripts
			{"an emoji ZWJ sequence", "\U0001F468\u200d\U0001F469\u200d\U0001F467", true, true},
			{"a ZWNJ", "می\u200cخوام", true, true},
			{"isolates closed on each line", "\u2066a\u2069\r\n\u2067b\u2069", false, true},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				require.Equal(t, c.text, SafeOneLine(c.input), "text")
				require.Equal(t, c.multiline, Safe(c.input), "multi-line text")
			})
		}
	})

	t.Run("cleanup", func(t *testing.T) {
		cases := []struct {
			name      string
			input     string
			multiline string
			text      string
		}{
			{"forbidden characters", "a\u202eb\ufffe\x07c", "abc", "abc"},
			{"invalid UTF-8", "a\xA0b", "a\ufffdb", "a\ufffdb"},
			{"line breaks and separators", " a\r\nb\u2028c\u2029d\te ", "a\nb\u2028c\u2029d\te", "abcde"},
			{"an isolate left open", "a\u2067b", "ab", "ab"},
			{"a close with no isolate open", "a\u2069b\u2066c\u2069", "ab\u2066c\u2069", "ab\u2066c\u2069"},
			{"nested isolates, the outer one left open", "\u2067a\u2068b\u2069c", "a\u2068b\u2069c", "a\u2068b\u2069c"},
			{"an isolate open across lines", "\u2067a\nb\u2069", "a\nb", "\u2067ab\u2069"},
			{"isolates left open on several lines", "\u2066a\u2067b\u2069\u2068\nc\u2067\u2068d\u2069\re\u2066", "a\u2067b\u2069\nc\u2068d\u2069\re", "a\u2067b\u2069c\u2068d\u2069e"},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				multiline := Cleanup(c.input)
				require.Equal(t, c.multiline, multiline, "multi-line text")
				require.True(t, Safe(multiline))
				text := CleanupOneLine(c.input)
				require.Equal(t, c.text, text, "text")
				require.True(t, SafeOneLine(text))
			})
		}
	})
}
