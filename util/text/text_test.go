package text

import (
	"strings"
	"testing"
)

func TestWrap(t *testing.T) {
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
			"foo\nb\n  r\n baz",
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
		// Handle words with colors sequence inside the word
		{
			"foo b\x1b[31mbar\x1b[0mr baz",
			"foo\nb\x1b[31mbar\n\x1b[0mr\nbaz",
			4,
		},
		// Break words with colors sequence inside the word
		{
			"foo bb\x1b[31mbar\x1b[0mr baz",
			"foo\nbb\x1b[31mba\nr\x1b[0mr\nbaz",
			4,
		},
		// Complete example:
		{
			" This is a list: \n\n\t* foo\n\t* bar\n\n\n\t* baz  \nBAM    ",
			" This\nis a\nlist:\n\n\n    *\nfoo\n    *\nbar\n\n\n    *\nbaz\nBAM\n",
			6,
		},
	}

	for i, tc := range cases {
		actual, lines := Wrap(tc.Input, tc.Lim)
		if actual != tc.Output {
			t.Fatalf("Case %d Input:\n\n`%s`\n\nExpected Output:\n\n`%s`\n\nActual Output:\n\n`%s`",
				i, tc.Input, tc.Output, actual)
		}

		expected := len(strings.Split(tc.Output, "\n"))
		if expected != lines {
			t.Fatalf("Case %d Nb lines mismatch\nExpected:%d\nActual:%d",
				i, expected, lines)
		}
	}
}

func TestWordLen(t *testing.T) {
	cases := []struct {
		Input  string
		Length int
	}{
		// A simple word
		{
			"foo",
			3,
		},
		// A simple word with colors
		{
			"\x1b[31mbar\x1b[0m",
			3,
		},
		// Handle prefix and suffix properly
		{
			"foo\x1b[31mfoobarHoy\x1b[0mbaaar",
			17,
		},
	}

	for i, tc := range cases {
		l := wordLen(tc.Input)
		if l != tc.Length {
			t.Fatalf("Case %d Input:\n\n`%s`\n\nExpected Output:\n\n`%d`\n\nActual Output:\n\n`%d`",
				i, tc.Input, tc.Length, l)
		}
	}
}

func TestSplitWord(t *testing.T) {
	cases := []struct {
		Input            string
		Length           int
		Result, Leftover string
	}{
		// A simple word passes through.
		{
			"foo",
			4,
			"foo", "",
		},
		// Cut at the right place
		{
			"foobarHoy",
			4,
			"foob", "arHoy",
		},
		// A simple word passes through with colors
		{
			"\x1b[31mbar\x1b[0m",
			4,
			"\x1b[31mbar\x1b[0m", "",
		},
		// Cut at the right place with colors
		{
			"\x1b[31mfoobarHoy\x1b[0m",
			4,
			"\x1b[31mfoob", "arHoy\x1b[0m",
		},
		// Handle prefix and suffix properly
		{
			"foo\x1b[31mfoobarHoy\x1b[0mbaaar",
			4,
			"foo\x1b[31mf", "oobarHoy\x1b[0mbaaar",
		},
		// Cut properly with length = 0
		{
			"foo",
			0,
			"", "foo",
		},
	}

	for i, tc := range cases {
		result, leftover := splitWord(tc.Input, tc.Length)
		if result != tc.Result || leftover != tc.Leftover {
			t.Fatalf("Case %d Input:\n\n`%s`\n\nExpected Output:\n\n`%s` - `%s`\n\nActual Output:\n\n`%s` - `%s`",
				i, tc.Input, tc.Result, tc.Leftover, result, leftover)
		}
	}
}
