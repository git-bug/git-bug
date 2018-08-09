package util

import (
	"strings"
	"testing"
)

func TestTextWrap(t *testing.T) {
	cases := []struct {
		Input, Output string
		Lim           int
	}{
		// A simple word passes through.
		{
			"foo",
			"foo",
			4,
		},
		// Word breaking
		{
			"foobarbaz",
			"foob\narba\nz",
			4,
		},
		// Lines are broken at whitespace.
		{
			"foo bar baz",
			"foo\nbar\nbaz",
			4,
		},
		// Word breaking
		{
			"foo bars bazzes",
			"foo\nbars\nbazz\nes",
			4,
		},
		// A word that would run beyond the width is wrapped.
		{
			"fo sop",
			"fo\nsop",
			4,
		},
		// A tab counts as 4 characters.
		{
			"foo\nb\t r\n baz",
			"foo\nb\n r\n baz",
			4,
		},
		// Trailing whitespace is removed after used for wrapping.
		// Runs of whitespace on which a line is broken are removed.
		{
			"foo    \nb   ar   ",
			"foo\n\nb\nar\n",
			4,
		},
		// An explicit line break at the end of the input is preserved.
		{
			"foo bar baz\n",
			"foo\nbar\nbaz\n",
			4,
		},
		// Explicit break are always preserved.
		{
			"\nfoo bar\n\n\nbaz\n",
			"\nfoo\nbar\n\n\nbaz\n",
			4,
		},
		// Ignore complete words with terminal color sequence
		{
			"foo \x1b[31mbar\x1b[0m baz",
			"foo\n\x1b[31mbar\x1b[0m\nbaz",
			4,
		},
		// Complete example:
		{
			" This is a list: \n\n\t* foo\n\t* bar\n\n\n\t* baz  \nBAM    ",
			" This\nis a\nlist:\n\n    *\nfoo\n    *\nbar\n\n\n    *\nbaz\nBAM\n",
			6,
		},
	}

	for i, tc := range cases {
		actual, lines := TextWrap(tc.Input, tc.Lim)
		if actual != tc.Output {
			t.Fatalf("Case %d Input:\n\n`%s`\n\nExpected Output:\n\n`%s`\n\nActual Output:\n\n`%s`",
				i, tc.Input, tc.Output, actual)
		}

		expected := len(strings.Split(tc.Output, "\n"))
		if expected != lines {
			t.Fatalf("Nb lines mismatch\nExpected:%d\nActual:%d",
				expected, lines)
		}
	}
}
