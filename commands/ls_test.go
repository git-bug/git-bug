package commands

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_repairQuery(t *testing.T) {
	cases := []struct {
		args   []string
		output string
	}{
		{
			[]string{""},
			"",
		},
		{
			[]string{"foo"},
			"foo",
		},
		{
			[]string{"foo", "bar"},
			"foo bar",
		},
		{
			[]string{"foo bar", "baz"},
			"\"foo bar\" baz",
		},
		{
			[]string{"foo:bar", "baz"},
			"foo:bar baz",
		},
		{
			[]string{"foo:bar boo", "baz"},
			"foo:\"bar boo\" baz",
		},
	}

	for _, tc := range cases {
		require.Equal(t, tc.output, repairQuery(tc.args))
	}
}
