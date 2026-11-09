package fracindex

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFracindex(t *testing.T) {
	t.Run("validate", func(t *testing.T) {
		for _, position := range []string{"V", "1", "0V", "b0V", "zzz", "Az9"} {
			require.NoError(t, Validate(position), position)
		}
		for _, position := range []string{"", "0", "b0", "b-c", "b c", "é"} {
			require.Error(t, Validate(position), position)
		}
	})

	t.Run("between", func(t *testing.T) {
		cases := []struct {
			before, after, want string
		}{
			// the examples of the package documentation
			{"b", "c", "bV"},
			{"b", "b1", "b0V"},
			// at the start, at the end, or in an empty order
			{"", "", "V"},
			{"", "1", "0V"},
			{"", "0V", "0G"},
			{"z", "", "zV"},
			{"zz", "", "zzV"},
			// with a digit in between
			{"a", "c", "b"},
			{"b", "bz", "bV"},
			{"bV", "c", "bl"},
		}
		for _, c := range cases {
			position, err := Between(c.before, c.after)
			require.NoError(t, err)
			require.Equal(t, c.want, position, "between %q and %q", c.before, c.after)
		}

		for _, c := range [][2]string{{"b", "b"}, {"c", "b"}} {
			_, err := Between(c[0], c[1])
			require.Error(t, err, "between %q and %q", c[0], c[1])
		}
	})

	t.Run("between, repeatedly", func(t *testing.T) {
		// insert at the start, at the end and in between, the same spot often, and
		// check every position is valid and in between its neighbours
		random := rand.New(rand.NewPCG(1, 2))
		var positions []string
		for range 2000 {
			i := len(positions)
			switch random.IntN(4) {
			case 0:
				i = 0
			case 1:
				i = min(len(positions), 5)
			case 2:
				i = random.IntN(len(positions) + 1)
			}

			var before, after string
			if i > 0 {
				before = positions[i-1]
			}
			if i < len(positions) {
				after = positions[i]
			}
			position, err := Between(before, after)
			require.NoError(t, err)
			require.NoError(t, Validate(position))
			require.Less(t, before, position)
			if after != "" {
				require.Less(t, position, after)
			}
			positions = slices.Insert(positions, i, position)
		}
		require.True(t, slices.IsSorted(positions))
	})
}
