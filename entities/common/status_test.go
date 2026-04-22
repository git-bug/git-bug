package common

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStatusRoundTrip(t *testing.T) {
	for _, s := range []Status{OpenStatus, ClosedStatus, MergedStatus, DraftStatus} {
		parsed, err := StatusFromString(s.String())
		require.NoError(t, err)
		require.Equal(t, s, parsed)
	}
}

func TestStatusValidate(t *testing.T) {
	for _, s := range []Status{OpenStatus, ClosedStatus, MergedStatus, DraftStatus} {
		require.NoError(t, s.Validate())
	}
	require.Error(t, Status(0).Validate())
	require.Error(t, Status(99).Validate())
}

func TestStatusGQLRoundTrip(t *testing.T) {
	cases := map[Status]string{
		OpenStatus:   `"OPEN"`,
		ClosedStatus: `"CLOSED"`,
		MergedStatus: `"MERGED"`,
		DraftStatus:  `"DRAFT"`,
	}
	for s, want := range cases {
		var buf bytes.Buffer
		s.MarshalGQL(&buf)
		require.Equal(t, want, buf.String())

		var decoded Status
		require.NoError(t, decoded.UnmarshalGQL(want[1:len(want)-1]))
		require.Equal(t, s, decoded)
	}
}

func TestStatusFromStringRejectsUnknown(t *testing.T) {
	_, err := StatusFromString("flibble")
	require.Error(t, err)
}
