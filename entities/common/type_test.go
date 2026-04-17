package common

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTypeZeroValueIsIssue(t *testing.T) {
	// Backward compat: pre-existing bugs with no recorded Type must deserialize
	// as IssueType (the Go zero value for int).
	var zero Type
	require.Equal(t, IssueType, zero)
	require.NoError(t, zero.Validate())
}

func TestTypeRoundTrip(t *testing.T) {
	for _, ty := range []Type{IssueType, PRType} {
		parsed, err := TypeFromString(ty.String())
		require.NoError(t, err)
		require.Equal(t, ty, parsed)
	}
}

func TestTypeFromStringAccepts(t *testing.T) {
	cases := map[string]Type{
		"issue":        IssueType,
		"ISSUE":        IssueType,
		"  issue  ":    IssueType,
		"bug":          IssueType,
		"pr":           PRType,
		"PR":           PRType,
		"pull-request": PRType,
		"pullrequest":  PRType,
	}
	for in, want := range cases {
		got, err := TypeFromString(in)
		require.NoError(t, err, in)
		require.Equal(t, want, got, in)
	}
}

func TestTypeFromStringRejectsUnknown(t *testing.T) {
	_, err := TypeFromString("discussion")
	require.Error(t, err)
}

func TestTypeValidate(t *testing.T) {
	require.NoError(t, IssueType.Validate())
	require.NoError(t, PRType.Validate())
	require.Error(t, Type(99).Validate())
}

func TestTypeGQLRoundTrip(t *testing.T) {
	cases := map[Type]string{
		IssueType: `"ISSUE"`,
		PRType:    `"PR"`,
	}
	for ty, want := range cases {
		var buf bytes.Buffer
		ty.MarshalGQL(&buf)
		require.Equal(t, want, buf.String())

		var decoded Type
		require.NoError(t, decoded.UnmarshalGQL(want[1:len(want)-1]))
		require.Equal(t, ty, decoded)
	}
}
