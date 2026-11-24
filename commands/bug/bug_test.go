package bugcmd

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/commands/bug/testenv"
	. "github.com/git-bug/git-bug/commands/cmdtest"
	"github.com/git-bug/git-bug/query"
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

func Test_completeQuery_sorting(t *testing.T) {
	// bugOptions mirrors the flag state after cobra parsing: the string fields
	// always hold a value (default or user supplied), while the Changed booleans
	// mark the flags actually given on the command line.
	cases := []struct {
		name         string
		queryStr     string
		opts         bugOptions
		expBy        query.OrderBy
		expDirection query.OrderDirection
	}{
		{
			name:         "no query keeps flag defaults",
			opts:         bugOptions{sortBy: "creation", sortDirection: "asc"},
			expBy:        query.OrderByCreation,
			expDirection: query.OrderAscending,
		},
		{
			name:         "query without sort keeps flag defaults",
			queryStr:     "status:open",
			opts:         bugOptions{sortBy: "creation", sortDirection: "asc"},
			expBy:        query.OrderByCreation,
			expDirection: query.OrderAscending,
		},
		{
			name:         "sort qualifier survives flag defaults",
			queryStr:     "status:open sort:edit-desc",
			opts:         bugOptions{sortBy: "creation", sortDirection: "asc"},
			expBy:        query.OrderByEdit,
			expDirection: query.OrderDescending,
		},
		{
			name:         "explicit flags override the sort qualifier",
			queryStr:     "sort:edit-desc",
			opts:         bugOptions{sortBy: "id", sortByChanged: true, sortDirection: "asc", sortDirectionChanged: true},
			expBy:        query.OrderById,
			expDirection: query.OrderAscending,
		},
		{
			name:         "one explicit flag mixes with the sort qualifier",
			queryStr:     "sort:edit-desc",
			opts:         bugOptions{sortBy: "id", sortByChanged: true, sortDirection: "asc"},
			expBy:        query.OrderById,
			expDirection: query.OrderDescending,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var q *query.Query
			if tc.queryStr == "" {
				q = query.NewQuery()
			} else {
				var err error
				q, err = query.Parse(tc.queryStr)
				require.NoError(t, err)
			}

			require.NoError(t, completeQuery(q, tc.opts))
			require.Equal(t, tc.expBy, q.OrderBy)
			require.Equal(t, tc.expDirection, q.OrderDirection)
		})
	}
}
