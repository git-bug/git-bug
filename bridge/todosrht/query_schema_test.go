package todosrht

import (
	"os"
	"testing"

	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestQueriesMatchCheckedInSchema(t *testing.T) {
	schemaText, err := os.ReadFile("schema.graphqls")
	if err != nil {
		t.Fatal(err)
	}
	schema, err := gqlparser.LoadSchema(&ast.Source{Name: "schema.graphqls", Input: string(schemaText)})
	if err != nil {
		t.Fatal(err)
	}
	for name, mutation := range map[string]string{
		"submit comment": submitCommentMutation,
		"submit ticket":  submitTicketMutation,
		"update status":  updateTicketStatusMutation,
		"update ticket":  updateTicketMutation,
		"create label":   createLabelMutation,
		"label ticket":   labelTicketMutation,
		"unlabel ticket": unlabelTicketMutation,
	} {
		t.Run(name, func(t *testing.T) {
			if _, errs := gqlparser.LoadQuery(schema, mutation); len(errs) != 0 {
				t.Fatal(errs)
			}
		})
	}
	for name, query := range map[string]string{
		"tracker": getTrackerByNameQuery,
		"tickets": getTicketsQuery,
		"events":  getEventsQuery,
		"labels":  getLabelsQuery,
	} {
		for _, ref := range []string{"tracker", "~owner/tracker"} {
			t.Run(name+"/"+ref, func(t *testing.T) {
				rendered, _, err := trackerQuery(query, ref)
				if err != nil {
					t.Fatal(err)
				}
				if _, errs := gqlparser.LoadQuery(schema, rendered); len(errs) != 0 {
					t.Fatal(errs)
				}
			})
		}
	}
}
