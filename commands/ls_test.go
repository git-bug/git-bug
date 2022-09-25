package commands

import (
	"encoding/json"
	"fmt"
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

func TestLs_Format(t *testing.T) {
	const expOrgMode = `^#+TODO: OPEN | CLOSED
[*] OPEN   [0-9a-f]{7} \[\d\d\d\d-\d\d-\d\d [[:alpha:]]{3} \d\d:\d\d\] John Doe: this is a bug title ::
[*]{2} Last Edited: \[\d\d\d\d-\d\d-\d\d [[:alpha:]]{3} \d\d:\d\d\]
[*]{2} Actors:
: [0-9a-f]{7} John Doe
[*]{2} Participants:
: [0-9a-f]{7} John Doe
$`

	cases := []struct {
		format string
		exp    string
	}{
		{"default", "^[0-9a-f]{7}\topen\tthis is a bug title                               \tJohn Doe       \t\n$"},
		{"plain", "^[0-9a-f]{7} \\[open\\] this is a bug title\n$"},
		{"compact", "^[0-9a-f]{7} open this is a bug title                            John Doe\n$"},
		{"id", "^[0-9a-f]{64}\n$"},
		{"org-mode", expOrgMode},
	}

	for _, testcase := range cases {
		opts := lsOptions{
			sortDirection: "asc",
			sortBy:        "creation",
			outputFormat:  testcase.format,
		}

		name := fmt.Sprintf("with %s format", testcase.format)

		t.Run(name, func(t *testing.T) {
			env, _ := newTestEnvAndBug(t)

			require.NoError(t, runLs(env.env, opts, []string{}))
			require.Regexp(t, testcase.exp, env.out.String())
		})
	}

	t.Run("with JSON format", func(t *testing.T) {
		opts := lsOptions{
			sortDirection: "asc",
			sortBy:        "creation",
			outputFormat:  "json",
		}

		env, _ := newTestEnvAndBug(t)

		require.NoError(t, runLs(env.env, opts, []string{}))

		bugs := []JSONBugExcerpt{}
		require.NoError(t, json.Unmarshal(env.out.Bytes(), &bugs))

		require.Len(t, bugs, 1)
	})
}
