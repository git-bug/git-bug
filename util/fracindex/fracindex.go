// Package fracindex places items in an order with fractional indexes: positions
// that leave room between any two of them, so that an item is inserted or moved by
// giving it a new position, without changing any other.
//
// A position is a non-empty string of the digits 0-9, A-Z and a-z, not ending with
// 0. It reads as the digits of a fraction between 0 and 1 in base 62. As the
// digits are in the order of their bytes, and no position ends with 0, positions
// compared byte by byte sort as their fractions do, and there is always one
// between two others: "bV" lies between "b" and "c", and "b0V" between "b" and
// "b1".
package fracindex

import (
	"fmt"
	"strings"
)

// digits are the digits of a position, in increasing order, which is also the
// order of their bytes.
const digits = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// Validate checks a position: a non-empty string of the digits 0-9, A-Z and a-z,
// not ending with 0.
func Validate(position string) error {
	if position == "" {
		return fmt.Errorf("empty position")
	}
	for i := 0; i < len(position); i++ {
		if strings.IndexByte(digits, position[i]) < 0 {
			return fmt.Errorf("position %q has a character other than 0-9, A-Z and a-z", position)
		}
	}
	if position[len(position)-1] == '0' {
		return fmt.Errorf("position %q ends with 0", position)
	}
	return nil
}

// Between returns a position between two valid positions, before sorting before
// after. An empty before stands for the start of the order, an empty after for its
// end.
func Between(before, after string) (string, error) {
	if before != "" && after != "" && before >= after {
		return "", fmt.Errorf("no position between %q and %q", before, after)
	}
	return mid(before, after), nil
}

// mid returns a position between before and after, as Between does, without
// checking them.
func mid(before, after string) string {
	// digit returns the digit of before at index i, which past its end is 0
	digit := func(i int) int {
		if i < len(before) {
			return strings.IndexByte(digits, before[i])
		}
		return 0
	}

	// keep the digits that before and after share
	if after != "" {
		n := 0
		for n < len(after) && digits[digit(n)] == after[n] {
			n++
		}
		if n > 0 {
			return after[:n] + mid(before[min(n, len(before)):], after[n:])
		}
	}

	// the first digits differ: take one in between if there is one
	low := digit(0)
	high := len(digits)
	if after != "" {
		high = strings.IndexByte(digits, after[0])
	}
	if high-low > 1 {
		return digits[(low+high+1)/2 : (low+high+1)/2+1]
	}

	// they follow each other: the first digit of after alone is in between if after
	// has more, otherwise keep the first digit of before and go one digit further
	if len(after) > 1 {
		return after[:1]
	}
	return digits[low:low+1] + mid(before[min(1, len(before)):], "")
}
