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
		{`author:"René Descartes"`, true},

		{"actor:bernhard", true},
		{"participant:leonhard", true},

		{"label:hello", true},
		{`label:"Good first issue"`, true},

		{"title:titleOne", true},
		{`title:"Bug titleTwo"`, true},

		{"sort:edit", true},
		{"sort:unknown", false},
	}

	for _, test := range tests {
		_, err := ParseQuery(test.input)
		if (err == nil) != test.ok {
			t.Fatalf("Unexpected parse result, expected: %v, err: %v", test.ok, err)
		}
	}
}
