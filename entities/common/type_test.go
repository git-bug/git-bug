package common

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKindZeroValueIsIssue(t *testing.T) {
	// Backward compat: pre-existing bugs with no recorded Kind must deserialize
	// as IssueKind (the Go zero value for int).
	var zero Kind
	require.Equal(t, IssueKind, zero)
	require.NoError(t, zero.Validate())
}

func TestKindRoundTrip(t *testing.T) {
	for _, k := range []Kind{IssueKind, PRKind} {
		parsed, err := KindFromString(k.String())
		require.NoError(t, err)
		require.Equal(t, k, parsed)
	}
}

func TestKindFromStringAccepts(t *testing.T) {
	cases := map[string]Kind{
		"issue":        IssueKind,
		"ISSUE":        IssueKind,
		"  issue  ":    IssueKind,
		"bug":          IssueKind,
		"pr":           PRKind,
		"PR":           PRKind,
		"pull-request": PRKind,
		"pullrequest":  PRKind,
	}
	for in, want := range cases {
		got, err := KindFromString(in)
		require.NoError(t, err, in)
		require.Equal(t, want, got, in)
	}
}

func TestKindFromStringRejectsUnknown(t *testing.T) {
	_, err := KindFromString("discussion")
	require.Error(t, err)
}

func TestKindValidate(t *testing.T) {
	require.NoError(t, IssueKind.Validate())
	require.NoError(t, PRKind.Validate())
	require.Error(t, Kind(99).Validate())
}

func TestKindGQLRoundTrip(t *testing.T) {
	cases := map[Kind]string{
		IssueKind: `"ISSUE"`,
		PRKind:    `"PR"`,
	}
	for k, want := range cases {
		var buf bytes.Buffer
		k.MarshalGQL(&buf)
		require.Equal(t, want, buf.String())

		var decoded Kind
		require.NoError(t, decoded.UnmarshalGQL(want[1:len(want)-1]))
		require.Equal(t, k, decoded)
	}
}
