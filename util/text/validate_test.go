package text

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSafe(t *testing.T) {
	cases := []struct {
		name        string
		input       string
		safe        bool
		safeOneLine bool
	}{
		{"plain", "hello world", true, true},
		{"empty", "", true, true},
		{"multi-line", "a\nb\r\n\tc", true, false},
		{"control char", "a\x07b", false, false},
		{"invalid utf8", "\xA0\xA1", false, false},
		{"truncated utf8", "caf\xC3", false, false},
		{"replacement char", "a�b", true, true},
		// format characters (Cf) are legitimate: emoji ZWJ sequences, ZWNJ in
		// Persian/Indic scripts, bidi marks in RTL text
		{"emoji zwj sequence", "\U0001F468‍\U0001F469‍\U0001F467", true, true},
		{"zwnj", "می‌خوام", true, true},
		{"bidi mark", "abc‎def", true, true},
		{"bidi isolate", "abc⁧def⁩", true, true},
		{"bidi override", "abc‮def", false, false},
		{"bidi embedding", "abc‪def", false, false},
		{"noncharacter FDD0", "a﷐b", false, false},
		{"noncharacter FFFE", "a￾b", false, false},
		{"noncharacter supplementary", "a\U0010FFFFb", false, false},
		{"last char before noncharacters", "a�b\U0010FFFD", true, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.safe, Safe(c.input))
			require.Equal(t, c.safeOneLine, SafeOneLine(c.input))
		})
	}

	t.Run("cleanup output is safe", func(t *testing.T) {
		require.Equal(t, "ab⁧c", Cleanup("a‮b￾⁧c"))
		require.True(t, Safe(Cleanup("a\xA0\xA1b\x07‮﷐")))
		require.True(t, SafeOneLine(CleanupOneLine("a\xA0\xA1b\n‮﷐")))
	})
}
