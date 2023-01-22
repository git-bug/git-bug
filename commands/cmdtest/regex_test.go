package cmdtest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMakeExpectedRegex(t *testing.T) {
	cases := []struct {
		sub  string
		text string
	}{
		{ExpId, "d96dc877077a571414168c946eb013035888715b561e75682cfae9ef785e3227"},
		{ExpHumanId, "d96dc87"},
		{ExpTimestamp, "1674368486"},
		{ExpISO8601, "2023-01-22T07:21:26+01:00"},
	}

	for _, tc := range cases {
		t.Run(tc.sub, func(t *testing.T) {
			require.Regexp(t, MakeExpectedRegex(tc.text), tc.text)
		})
	}
}
