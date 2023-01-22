package cmdtest

import (
	"regexp"
	"strings"
)

const ExpId = "\x07id\x07"
const ExpHumanId = "\x07human-id\x07"
const ExpTimestamp = "\x07timestamp\x07"
const ExpISO8601 = "\x07iso8601\x07"

const ExpOrgModeDate = "\x07org-mode-date\x07"

// MakeExpectedRegex transform a raw string of an expected output into a regex suitable for testing.
// Some markers like ExpId are available to substitute the appropriate regex for element that can vary randomly.
func MakeExpectedRegex(input string) string {
	var substitutes = map[string]string{
		ExpId:          `[0-9a-f]{64}`,
		ExpHumanId:     `[0-9a-f]{7}`,
		ExpTimestamp:   `[0-9]{7,10}`,
		ExpISO8601:     `\d{4}(-\d\d(-\d\d(T\d\d:\d\d(:\d\d)?(\.\d+)?(([+-]\d\d:\d\d)|Z)?)?)?)?`,
		ExpOrgModeDate: `\d\d\d\d-\d\d-\d\d [[:alpha:]]{3} \d\d:\d\d`,
	}

	escaped := []rune(regexp.QuoteMeta(input))

	var result strings.Builder
	var inSubstitute bool
	var substitute strings.Builder

	result.WriteString("^")

	for i := 0; i < len(escaped); i++ {
		r := escaped[i]
		if !inSubstitute && r == '\x07' {
			substitute.Reset()
			substitute.WriteRune(r)
			inSubstitute = true
			continue
		}
		if inSubstitute && r == '\x07' {
			substitute.WriteRune(r)
			sub, ok := substitutes[substitute.String()]
			if !ok {
				panic("unknown substitute: " + substitute.String())
			}
			result.WriteString(sub)
			inSubstitute = false
			continue
		}
		if inSubstitute {
			substitute.WriteRune(r)
		} else {
			result.WriteRune(r)
		}
	}

	result.WriteString("$")

	return result.String()
}
