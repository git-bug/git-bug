package graphql

import (
	"context"
	"fmt"

	"github.com/vektah/gqlgen/neelance/common"
	"github.com/vektah/gqlgen/neelance/query"
	"github.com/vektah/gqlgen/neelance/schema"
)

type ExecutableSchema interface {
	Schema() *schema.Schema

	Query(ctx context.Context, op *query.Operation) *Response
	Mutation(ctx context.Context, op *query.Operation) *Response
	Subscription(ctx context.Context, op *query.Operation) func() *Response
}

func CollectFields(doc *query.Document, selSet []query.Selection, satisfies []string, variables map[string]interface{}) []CollectedField {
	return collectFields(doc, selSet, satisfies, variables, map[string]bool{})
}

func collectFields(doc *query.Document, selSet []query.Selection, satisfies []string, variables map[string]interface{}, visited map[string]bool) []CollectedField {
	var groupedFields []CollectedField

	for _, sel := range selSet {
		switch sel := sel.(type) {
		case *query.Field:
			f := getOrCreateField(&groupedFields, sel.Alias.Name, func() CollectedField {
				f := CollectedField{
					Alias: sel.Alias.Name,
					Name:  sel.Name.Name,
				}
				if len(sel.Arguments) > 0 {
					f.Args = map[string]interface{}{}
					for _, arg := range sel.Arguments {
						if variable, ok := arg.Value.(*common.Variable); ok {
							if val, ok := variables[variable.Name]; ok {
								f.Args[arg.Name.Name] = val
							}
						} else {
							f.Args[arg.Name.Name] = arg.Value.Value(variables)
						}
					}
				}
				return f
			})

			f.Selections = append(f.Selections, sel.Selections...)
		case *query.InlineFragment:
			if !instanceOf(sel.On.Ident.Name, satisfies) {
				continue
			}

			for _, childField := range collectFields(doc, sel.Selections, satisfies, variables, visited) {
				f := getOrCreateField(&groupedFields, childField.Name, func() CollectedField { return childField })
				f.Selections = append(f.Selections, childField.Selections...)
			}

		case *query.FragmentSpread:
			fragmentName := sel.Name.Name
			if _, seen := visited[fragmentName]; seen {
				continue
			}
			visited[fragmentName] = true

			fragment := doc.Fragments.Get(fragmentName)
			if fragment == nil {
				// should never happen, validator has already run
				panic(fmt.Errorf("missing fragment %s", fragmentName))
			}

			if !instanceOf(fragment.On.Ident.Name, satisfies) {
				continue
			}

			for _, childField := range collectFields(doc, fragment.Selections, satisfies, variables, visited) {
				f := getOrCreateField(&groupedFields, childField.Name, func() CollectedField { return childField })
				f.Selections = append(f.Selections, childField.Selections...)
			}

		default:
			panic(fmt.Errorf("unsupported %T", sel))
		}
	}

	return groupedFields
}

type CollectedField struct {
	Alias      string
	Name       string
	Args       map[string]interface{}
	Selections []query.Selection
}

func instanceOf(val string, satisfies []string) bool {
	for _, s := range satisfies {
		if val == s {
			return true
		}
	}
	return false
}

func getOrCreateField(c *[]CollectedField, name string, creator func() CollectedField) *CollectedField {
	for i, cf := range *c {
		if cf.Alias == name {
			return &(*c)[i]
		}
	}

	f := creator()

	*c = append(*c, f)
	return &(*c)[len(*c)-1]
}
