package query

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/query/ast"
)

func TestParse(t *testing.T) {
	var tests = []struct {
		input  string
		output *ast.Query
	}{
		{"gibberish", nil},
		{"status:", nil},
		{":value", nil},

		{"status:open", &ast.Query{
			Filters: ast.Filters{Status: []bug.Status{bug.OpenStatus}},
		}},
		{"status:closed", &ast.Query{
			Filters: ast.Filters{Status: []bug.Status{bug.ClosedStatus}},
		}},
		{"status:unknown", nil},

		{"author:rene", &ast.Query{
			Filters: ast.Filters{Author: []string{"rene"}},
		}},
		{`author:"René Descartes"`, &ast.Query{
			Filters: ast.Filters{Author: []string{"René Descartes"}},
		}},

		{"actor:bernhard", &ast.Query{
			Filters: ast.Filters{Actor: []string{"bernhard"}},
		}},
		{"participant:leonhard", &ast.Query{
			Filters: ast.Filters{Participant: []string{"leonhard"}},
		}},

		{"label:hello", &ast.Query{
			Filters: ast.Filters{Label: []string{"hello"}},
		}},
		{`label:"Good first issue"`, &ast.Query{
			Filters: ast.Filters{Label: []string{"Good first issue"}},
		}},

		{"title:titleOne", &ast.Query{
			Filters: ast.Filters{Title: []string{"titleOne"}},
		}},
		{`title:"Bug titleTwo"`, &ast.Query{
			Filters: ast.Filters{Title: []string{"Bug titleTwo"}},
		}},

		{"no:label", &ast.Query{
			Filters: ast.Filters{NoLabel: true},
		}},

		{"sort:edit", &ast.Query{
			OrderBy: ast.OrderByEdit,
		}},
		{"sort:unknown", nil},

		{`status:open author:"René Descartes" participant:leonhard label:hello label:"Good first issue" sort:edit-desc`,
			&ast.Query{
				Filters: ast.Filters{
					Status:      []bug.Status{bug.OpenStatus},
					Author:      []string{"René Descartes"},
					Participant: []string{"leonhard"},
					Label:       []string{"hello", "Good first issue"},
				},
				OrderBy:        ast.OrderByEdit,
				OrderDirection: ast.OrderDescending,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			query, err := Parse(tc.input)
			if tc.output == nil {
				assert.Error(t, err)
				assert.Nil(t, query)
			} else {
				assert.NoError(t, err)
				if tc.output.OrderBy != 0 {
					assert.Equal(t, tc.output.OrderBy, query.OrderBy)
				}
				if tc.output.OrderDirection != 0 {
					assert.Equal(t, tc.output.OrderDirection, query.OrderDirection)
				}
				assert.Equal(t, tc.output.Filters, query.Filters)
			}
		})
	}
}
