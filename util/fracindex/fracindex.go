// Package fracindex places items in an order with fractional indexes: positions
// that leave room between any two of them, so that an item is inserted or moved by
// giving it a new position, without changing any other.
//
// The positions are the ones of David Greenspan's fractional indexing, see
// https://observablehq.com/@dgreensp/implementing-fractional-indexing. A position
// is a string of the digits 0-9, A-Z and a-z, whose order is also the order of
// their bytes, so that positions compared byte by byte sort as the numbers they
// stand for. It has two parts:
//
//   - an integer part, whose first character, a letter, gives its length, that
//     character included: 2 for a, 3 for b, up to 27 for z, and 2 for Z, 3 for Y,
//     up to 27 for A. The lowercase letters start the integers from a0 upward, the
//     uppercase ones those below it: Zy, Zz, a0, a1, ..., az, b00, b01.
//   - a fractional part: the rest of the string, possibly empty, not ending with 0.
//
// The smallest integer part, A followed by 26 0, is not a position on its own, so
// that there is always a position before another, after another, and between two
// different ones. Positions added one after another at either end step the
// integer part, and so stay short: a0, a1, a2.
package fracindex

import (
	"fmt"
	"strings"
)

// digits are the digits of a position, in increasing order, which is also the
// order of their bytes.
const digits = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// smallestInteger is the smallest integer part, A followed by 26 0, which is not a
// position on its own.
const smallestInteger = "A00000000000000000000000000"

// Validate checks a position: an integer part of the length its first letter
// gives, then a fractional part not ending with 0, all of the digits 0-9, A-Z and
// a-z.
func Validate(position string) error {
	if position == "" {
		return fmt.Errorf("empty position")
	}
	for i := 0; i < len(position); i++ {
		if strings.IndexByte(digits, position[i]) < 0 {
			return fmt.Errorf("position %q has a character other than 0-9, A-Z and a-z", position)
		}
	}
	length, ok := integerLength(position[0])
	if !ok {
		return fmt.Errorf("position %q doesn't start with a letter", position)
	}
	if len(position) < length {
		return fmt.Errorf("position %q is shorter than its integer part of %d characters", position, length)
	}
	if position == smallestInteger {
		return fmt.Errorf("position %q is the smallest integer part alone", position)
	}
	if len(position) > length && position[len(position)-1] == '0' {
		return fmt.Errorf("position %q has a fractional part ending with 0", position)
	}
	return nil
}

// Between returns a position between two positions, before sorting before after.
// An empty before stands for the start of the order, an empty after for its end.
func Between(before, after string) (string, error) {
	for _, position := range []string{before, after} {
		if position == "" {
			continue
		}
		if err := Validate(position); err != nil {
			return "", err
		}
	}
	if before != "" && after != "" && before >= after {
		return "", fmt.Errorf("no position between %q and %q", before, after)
	}

	switch {
	case before == "" && after == "":
		return "a0", nil

	case before == "":
		// the integer part of after alone if it has a fractional part, otherwise
		// the integer before it
		integer, fraction := split(after)
		switch {
		case integer == smallestInteger:
			return integer + mid("", fraction), nil
		case fraction != "":
			return integer, nil
		}
		lower := decrement(integer)
		if lower == smallestInteger {
			return lower + mid("", ""), nil
		}
		return lower, nil

	case after == "":
		// the integer after the one of before, if there is one
		integer, fraction := split(before)
		if higher, ok := increment(integer); ok {
			return higher, nil
		}
		return integer + mid(fraction, ""), nil

	default:
		integerBefore, fractionBefore := split(before)
		integerAfter, fractionAfter := split(after)
		if integerBefore == integerAfter {
			return integerBefore + mid(fractionBefore, fractionAfter), nil
		}
		// the integer after the one of before, if it is still before after
		if higher, ok := increment(integerBefore); ok && higher < after {
			return higher, nil
		}
		return integerBefore + mid(fractionBefore, ""), nil
	}
}

// integerLength returns the length of an integer part starting with head, false
// if head is not a letter.
func integerLength(head byte) (int, bool) {
	switch {
	case 'a' <= head && head <= 'z':
		return int(head-'a') + 2, true
	case 'A' <= head && head <= 'Z':
		return int('Z'-head) + 2, true
	default:
		return 0, false
	}
}

// split returns the integer and the fractional part of a valid position.
func split(position string) (integer, fraction string) {
	length, _ := integerLength(position[0])
	return position[:length], position[length:]
}

// increment returns the integer part following integer, false if integer is the
// largest one.
func increment(integer string) (string, bool) {
	head, body := integer[0], []byte(integer[1:])
	for i := len(body) - 1; i >= 0; i-- {
		d := strings.IndexByte(digits, body[i])
		if d < len(digits)-1 {
			body[i] = digits[d+1]
			return string(head) + string(body), true
		}
		body[i] = digits[0]
	}

	// every digit carried, and is now 0: the next integer is one digit longer
	// with a lowercase head, one digit shorter with an uppercase one
	switch head {
	case 'Z':
		return "a0", true
	case 'z':
		return "", false
	}
	head++
	if head > 'a' {
		body = append(body, digits[0])
	} else {
		body = body[1:]
	}
	return string(head) + string(body), true
}

// decrement returns the integer part preceding integer, which must be above the
// smallest one.
func decrement(integer string) string {
	head, body := integer[0], []byte(integer[1:])
	for i := len(body) - 1; i >= 0; i-- {
		d := strings.IndexByte(digits, body[i])
		if d > 0 {
			body[i] = digits[d-1]
			return string(head) + string(body)
		}
		body[i] = digits[len(digits)-1]
	}

	// every digit borrowed, and is now z: the previous integer is one digit
	// shorter with a lowercase head, one digit longer with an uppercase one
	if head == 'a' {
		return "Zz"
	}
	head--
	if head < 'Z' {
		body = append(body, digits[len(digits)-1])
	} else {
		body = body[1:]
	}
	return string(head) + string(body)
}

// mid returns a fractional part between the fractional parts before and after,
// before sorting before after. An empty before stands for the smallest fractional
// part, an empty after for the end of them.
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
