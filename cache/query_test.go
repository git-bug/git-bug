package cache

import "testing"

func TestQueryParse(t *testing.T) {
	var tests = []struct {
		input string
		ok    bool
	}{
		{"gibberish", false},

		{"status:", false},

		{"status:open", true},
		{"status:closed", true},
		{"status:unknown", false},

		{"author:rene", true},
		// Todo: fix parsing
		// {"author:\"Rene Descartes\"", true},

	}

	for _, test := range tests {
		_, err := ParseQuery(test.input)
		if (err == nil) != test.ok {
			t.Fatalf("Unexpected parse result, expected: %v, err: %v", test.ok, err)
		}
	}
}
