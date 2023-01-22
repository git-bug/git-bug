package bugcmd

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/commands/bug/testenv"
	. "github.com/MichaelMure/git-bug/commands/cmdtest"
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

func TestBug_Format(t *testing.T) {
	const expOrgMode = `#+TODO: OPEN | CLOSED
* OPEN   ` + ExpHumanId + ` [` + ExpOrgModeDate + `] John Doe: this is a bug title ::
** Last Edited: [` + ExpOrgModeDate + `]
** Actors:
: ` + ExpHumanId + ` John Doe
** Participants:
: ` + ExpHumanId + ` John Doe
`

	const expJson = `[
    {
        "id": "` + ExpId + `",
        "human_id": "` + ExpHumanId + `",
        "create_time": {
            "timestamp": ` + ExpTimestamp + `,
            "time": "` + ExpISO8601 + `",
            "lamport": 2
        },
        "edit_time": {
            "timestamp": ` + ExpTimestamp + `,
            "time": "` + ExpISO8601 + `",
            "lamport": 2
        },
        "status": "open",
        "labels": null,
        "title": "this is a bug title",
        "actors": [
            {
                "id": "` + ExpId + `",
                "human_id": "` + ExpHumanId + `",
                "name": "John Doe",
                "login": ""
            }
        ],
        "participants": [
            {
                "id": "` + ExpId + `",
                "human_id": "` + ExpHumanId + `",
                "name": "John Doe",
                "login": ""
            }
        ],
        "author": {
            "id": "` + ExpId + `",
            "human_id": "` + ExpHumanId + `",
            "name": "John Doe",
            "login": ""
        },
        "comments": 1,
        "metadata": {}
    }
]
`

	cases := []struct {
		format string
		exp    string
	}{
		{"default", ExpHumanId + "\topen\tthis is a bug title                      John Doe        \n"},
		{"plain", ExpHumanId + "\topen\tthis is a bug title\n"},
		{"id", ExpId + "\n"},
		{"org-mode", expOrgMode},
		{"json", expJson},
	}

	for _, testcase := range cases {
		t.Run(testcase.format, func(t *testing.T) {
			env, _ := testenv.NewTestEnvAndBug(t)

			opts := bugOptions{
				sortDirection:       "asc",
				sortBy:              "creation",
				outputFormat:        testcase.format,
				outputFormatChanged: true, // disable auto-detect
			}

			require.NoError(t, runBug(env, opts, []string{}))
			require.Regexp(t, MakeExpectedRegex(testcase.exp), env.Out.String())
		})
	}
}
