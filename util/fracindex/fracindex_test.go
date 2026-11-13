package fracindex

import (
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFracindex(t *testing.T) {
	largestInteger := "z" + strings.Repeat("z", 26)

	t.Run("validate", func(t *testing.T) {
		for _, position := range []string{
			"a0", "a0V", "Zz", "b12", "a1", "az", "b00", "Yzz",
			"A000000000000000000000000001", largestInteger, largestInteger + "V",
		} {
			require.NoError(t, Validate(position), position)
		}
		for _, position := range []string{
			"",
			"a",           // a gives an integer part of 2 characters
			"b1",          // b gives an integer part of 3 characters
			"a00", "a0V0", // a fractional part ending with 0
			"0a",  // not starting with a letter
			"a0~", // ~ is not a digit
			"a0é",
			"A00000000000000000000000000", // A followed by 26 0
		} {
			require.Error(t, Validate(position), position)
		}
	})

	t.Run("between", func(t *testing.T) {
		cases := []struct {
			before, after, want string
		}{
			// the table of Greenspan's implementation
			{"", "", "a0"},
			{"", "a0", "Zz"},
			{"a0", "", "a1"},
			{"a0", "a1", "a0V"},
			{"a0V", "a1", "a0l"},
			{"Zz", "a0", "ZzV"},
			{"Zz", "a1", "a0"},
			{"", "Y00", "Xzzz"},
			{"bzz", "", "c000"},
			{"a0", "a0V", "a0G"},
			{"a0", "a0G", "a08"},
			{"b125", "b129", "b127"},
			{"a0", "a1V", "a1"},
			{"Zz", "a01", "a0"},
			{"", "a0V", "a0"},
			{"", "b999", "b99"},
			{"", "A000000000000000000000000001", "A000000000000000000000000000V"},
			{"zzzzzzzzzzzzzzzzzzzzzzzzzzy", "", "zzzzzzzzzzzzzzzzzzzzzzzzzzz"},
			{"zzzzzzzzzzzzzzzzzzzzzzzzzzz", "", "zzzzzzzzzzzzzzzzzzzzzzzzzzzV"},
			// inserting between two concurrent inserts at a0V, after moving one
			{"a0V", "a0l", "a0d"},
			// the integer part alone when it is in between
			{"a1", "a3", "a2"},
			{"a0", "b00", "a1"},
			// stepping the integer part across a change of length or of sign
			{"az", "", "b00"},
			{"", "b00", "az"},
			{"Zz", "", "a0"},
			{"Yzz", "", "Z0"},
			{"", "Z0", "Yzz"},
			// stepping down onto the smallest integer part, which is not a position
			// alone: Greenspan's implementation returns it as is
			{"", "A" + strings.Repeat("0", 25) + "1", smallestInteger + "V"},
		}
		for _, c := range cases {
			position, err := Between(c.before, c.after)
			require.NoError(t, err, "between %q and %q", c.before, c.after)
			require.Equal(t, c.want, position, "between %q and %q", c.before, c.after)
			require.NoError(t, Validate(position))
		}

		for _, c := range [][2]string{
			{"a0", "a0"}, // no position between a position and itself
			{"a1", "a0"},
			{"V", ""}, // invalid positions
			{"", "a00"},
			{"a00", ""},
			{"a00", "a1"},
			{"0", "1"},
			{"", "A00000000000000000000000000"},
		} {
			_, err := Between(c[0], c[1])
			require.Error(t, err, "between %q and %q", c[0], c[1])
		}
	})

	t.Run("at the ends, repeatedly", func(t *testing.T) {
		// adding at the end, then at the start, steps the integer part across
		// several lengths, each position a valid integer part alone
		for _, atEnd := range []bool{true, false} {
			last := "a0"
			for range 10_000 {
				var position string
				var err error
				if atEnd {
					position, err = Between(last, "")
				} else {
					position, err = Between("", last)
				}
				require.NoError(t, err)
				require.NoError(t, Validate(position))
				integer, fraction := split(position)
				require.Empty(t, fraction, position)
				if atEnd {
					require.Less(t, last, integer)
				} else {
					require.Less(t, integer, last)
				}
				last = position
			}
		}
	})

	t.Run("between any positions", func(t *testing.T) {
		// positions of every integer length, whose integer part is often only 0 or
		// only z, where stepping it carries
		random := rand.New(rand.NewPCG(3, 4))
		letters := digits[10:]
		randomPosition := func() string {
			head := letters[random.IntN(len(letters))]
			length, _ := integerLength(head)
			position := []byte{head}
			edge := random.IntN(3)
			for range length - 1 {
				switch edge {
				case 0:
					position = append(position, '0')
				case 1:
					position = append(position, 'z')
				default:
					position = append(position, digits[random.IntN(len(digits))])
				}
			}
			fraction := random.IntN(4)
			for i := range fraction {
				// the fractional part doesn't end with 0
				low := 0
				if i == fraction-1 {
					low = 1
				}
				position = append(position, digits[low+random.IntN(len(digits)-low)])
			}
			if string(position) == smallestInteger {
				position = append(position, 'V')
			}
			return string(position)
		}

		for range 100_000 {
			a, b := randomPosition(), randomPosition()
			if a == b {
				continue
			}
			if a > b {
				a, b = b, a
			}
			// a and b, either end, and a with a position just after it
			nextToA := a + string(digits[1+random.IntN(len(digits)-1)])
			for _, c := range [][2]string{{a, b}, {"", a}, {b, ""}, {a, nextToA}} {
				before, after := c[0], c[1]
				position, err := Between(before, after)
				require.NoError(t, err, "between %q and %q", before, after)
				require.NoError(t, Validate(position), "between %q and %q", before, after)
				if before != "" {
					require.Less(t, before, position)
				}
				if after != "" {
					require.Less(t, position, after)
				}
			}
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
