package text

import "testing"

func TestLeftPadMaxLine(t *testing.T) {
	cases := []struct {
		input, output  string
		maxValueLength int
		leftPad        int
	}{
		{
			"foo",
			"foo ",
			4,
			0,
		},
		{
			"foofoofoo",
			"foo…",
			4,
			0,
		},
		{
			"foo",
			"foo       ",
			10,
			0,
		},
		{
			"foo",
			"  f…",
			4,
			2,
		},
		{
			"foofoofoo",
			"  foo…",
			6,
			2,
		},
		{
			"foo",
			"  foo     ",
			10,
			2,
		},
	}

	for i, tc := range cases {
		result := LeftPadMaxLine(tc.input, tc.maxValueLength, tc.leftPad)
		if result != tc.output {
			t.Fatalf("Case %d Input:\n\n`%s`\n\nExpected Output:\n\n`%s`\n\nActual Output:\n\n`%s`",
				i, tc.input, tc.output, result)
		}
	}
}
